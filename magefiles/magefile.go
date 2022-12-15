//go:build mage

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	cmd "github.com/rosskirkpat/kscaler/cmd"
	"github.com/rosskirkpat/kscaler/magetools"
)

var Default = Build
var g *magetools.Go
var version string
var commit string
var artifactOutput = filepath.Join("artifacts")

func Version() error {
	c, err := magetools.GetCommit()
	if err != nil {
		return err
	}
	commit = c

	dt := os.Getenv("GIT_TAG")
	isClean, err := magetools.IsGitClean()
	if err != nil {
		return err
	}
	if dt != "" && isClean {
		version = dt
		return nil
	}

	tag, err := magetools.GetLatestTag()
	if err != nil {
		return err
	}
	if tag != "" && isClean {
		version = tag
		return nil
	}

	version = commit
	if !isClean {
		version = commit + "-dirty"
		log.Printf("[Version] dirty version encountered: %s \n", version)
	}
	// check if this is a release version and fail if the version contains `dirty`
	if strings.Contains(version, "dirty") && os.Getenv("GIT_TAG") != "" || tag != "" {
		return fmt.Errorf("[Version] releases require a non-dirty tag: %s", version)
	}
	log.Printf("[Version] version: %s \n", version)

	return nil
}

func Setup() {
	mg.Deps(Version)
	osList := []string{"windows", "linux"}
	archList := []string{"arm64", "amd64"}
	for os := range osList {
		for arch := range archList {
			g = magetools.NewGo(arch, os, version, commit, "0", "1")
		}
	}
	gwindowsamd64 = magetools.NewGo("amd64", "windows", version, commit, "0", "1")
	gwindowsarm64 = magetools.NewGo("arm64", "windows", version, commit, "0", "1")
	glinuxamd64 = magetools.NewGo("amd64", "linux", version, commit, "0", "1")
	glinuxarm64 = magetools.NewGo("arm64", "linux", version, commit, "0", "1")
}

func Dependencies() error {
	mg.Deps(Setup)
	return g.Mod("download")
}

func Validate() error {
	envs := map[string]string{"GOOS": "windows", "ARCH": "amd64", "CGO_ENABLED": "0", "MAGEFILE_VERBOSE": "1"}

	log.Printf("[Validate] Running: golangci-lint \n")
	if err := sh.RunWithV(envs, "golangci-lint", "run"); err != nil {
		return err
	}

	log.Printf("[Validate] Running: go fmt \n")
	if err := sh.RunWithV(envs, "go", "fmt", "./..."); err != nil {
		return err
	}

	log.Printf("validate has completed successfully \n")
	return nil
}

func Build() error {
	mg.Deps(Clean, Dependencies, Validate)
	kscaleOutput := filepath.Join("bin", "kscale.exe")

	log.Printf("[Build] Building kscale version: %s \n", version)
	log.Printf("[Build] Output: %s \n", kscaleOutput)
	if err := g.Build(flags, "cmd/resources.go", kscaleOutput); err != nil {
		return err
	}
	log.Printf("[Build] successfully built kscale version %s \n", version)

	log.Printf("[Build] now staging build artifacts \n")
	if err := os.MkdirAll(artifactOutput, os.ModePerm); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(artifactOutput, "kscale.exe"), kscaleOutput); err != nil {
		return err
	}
	if err := sh.Copy(filepath.Join(artifactOutput, "kscale"), kscaleOutput); err != nil {
		return err
	}
	if err := sh.Copy(filepath.Join(artifactOutput, "kscale-windows-amd64.tgz"), kscaleOutput); err != nil {
		return err
	}
	if err := sh.Copy(filepath.Join(artifactOutput, "kscale-linux-amd64.tgz"), kscaleOutput); err != nil {
		return err
	}

	log.Printf("[Build] all required build artifacts have been staged \n")
	files, err := os.ReadDir(artifactOutput)
	if err != nil {
		return err
	}

	if len(files) != 3 {
		return errors.New("[Build] a required build artifact is missing, exiting now \n")
	}

	var artifacts strings.Builder
	for _, file := range files {
		artifacts.WriteString(file.Name() + " ,")
	}

	log.Printf("[Build] artifacts copied: %s \n", artifacts.String())

	return nil
}

func Test() error {
	mg.Deps(Build)
	log.Printf("[Test] Testing kscale version %s \n", version)
	if err := g.Test(flags, "./..."); err != nil {
		return err
	}
	log.Printf("[Test] successfully tested kscale version %s \n", version)
	return nil
}

func CI() {
	mg.Deps(Test)
}

func flags(version string, commit string) string {
	return fmt.Sprintf(`-s -w -X github.com/rosskirkpat/kscale/pkg/defaults.AppVersion=%s -X github.com/rosskirkpat/kscale /pkg/defaults.AppCommit=%s -extldflags "-static"`, version, commit)
}

func Docs() error {
	fmt.Println("Generating Docs...")
	if err := cmd.SetupDocs("./docs/cmd/"); err != nil {
		return err
	}
	return nil
}
func ValidateDocs() error {
	fmt.Println("Validating Docs...")
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	if string(out) != "" {
		return fmt.Errorf("Found changes while generating docs, please commit docs changes and try again")
	}

	return nil
}

func InstallDeps() error {
	fmt.Println("Installing Deps...")
	installs := []string{
		"github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.0",
		"github.com/goreleaser/goreleaser@latest",
	}

	for _, pkg := range installs {
		fmt.Printf("Installing %s\n", pkg)
		cmd := exec.Command("go", "install", pkg)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func Clean() error {
	fmt.Println("Cleaning...")
	if err := sh.Rm(artifactOutput); err != nil {
		return err
	}
	return sh.Rm("bin")
	rmFiles := []string{
		"./kscale.exe",
		"./coverage.txt",
	}
	for _, f := range rmFiles {
		if err := os.Remove(f); err != nil {
			if !strings.Contains(err.Error(), "The system cannot find the file specified") {
				return err
			}
		}
	}

	return nil
}

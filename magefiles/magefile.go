//go:build mage

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/rosskirkpat/kscaler/magetools"
)

var (
	Default        = Build
	g              *magetools.Go
	gs             []*magetools.Go
	version        string
	commit         string
	artifactOutput = filepath.Join("artifacts")
)

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
	mg.Deps(Version, Clean)
	osList := []string{"windows", "linux", "darwin"}
	archList := []string{"arm64", "amd64"}
	for _, osname := range osList {
		for _, arch := range archList {
			g = magetools.NewGo(arch, osname, version, commit, "0", "1")
			gs = append(gs, g)
		}
	}
	//gwindowsamd64 = magetools.NewGo("amd64", "windows", version, commit, "0", "1")
	//gwindowsarm64 = magetools.NewGo("arm64", "windows", version, commit, "0", "1")
	//glinuxamd64 = magetools.NewGo("amd64", "linux", version, commit, "0", "1")
	//glinuxarm64 = magetools.NewGo("arm64", "linux", version, commit, "0", "1")
}

func Dependencies() error {
	mg.Deps(Setup, InstallDeps)
	return g.Mod("download")
}

func Validate() error {
	envs := map[string]string{"GOOS": "darwin", "ARCH": "arm64", "CGO_ENABLED": "0", "MAGEFILE_VERBOSE": "1"}

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
	mg.Deps(Dependencies, Validate)
	cmdGenOutput := filepath.Join("bin", "cmd-gen")
	if err := os.MkdirAll("bin", os.ModePerm); err != nil {
		return err
	}

	log.Printf("[Build] Building cmd-gen\n")
	if err := g.Build(flags, "internal/cmd-gen/main.go", cmdGenOutput); err != nil {
		return err
	}
	log.Printf("[Build] successfully built cmd-gen\n")
	if err := g.Generate("github.com/rosskirkpat/kscaler/cmd"); err != nil {
		return err
	}
	log.Printf("[Build] successfully generated cobra sub-commands\n")

	log.Printf("[Build] Building multi-platform kscaler binaries for version: %s \n", version)

	var isRelease bool
	parsedVersion, err := semver.Parse(version)
	if err != nil {
		log.Printf("[Build] failed to parse version %s, assuming this is NOT a release: %s\n", version, err)
		log.Printf("[Build] will not stage release binaries in artifacts/ dir\n")
	} else {
		log.Printf("[Build] detected release %s, will stage release binaries in artifacts/ dir\n", parsedVersion.FinalizeVersion())
		isRelease = true
		if err := os.MkdirAll("artifacts/windows", os.ModePerm); err != nil {
			return err
		}
		if err := os.MkdirAll("artifacts/linux", os.ModePerm); err != nil {
			return err
		}
		if err := os.MkdirAll("artifacts/darwin", os.ModePerm); err != nil {
			return err
		}
	}
	for _, gg := range gs {
		kscalerOutput := filepath.Join("bin", gg.OS, "kscaler"+"-"+gg.Arch)
		if err := gg.Build(flags, "github.com/rosskirkpat/kscaler/cmd", kscalerOutput); err != nil {
			return err
		}
		if isRelease {
			dirs, file := path.Split(kscalerOutput)
			osName := strings.Split(dirs, "/")[1]
			target := path.Join("artifacts", osName, file)
			log.Printf("[Build] copying %s to %s\n", kscalerOutput, target)

			if err := sh.Copy(target, kscalerOutput); err != nil {
				log.Printf("[Build] failed to copy %s to %s\n", kscalerOutput, target)
				return err
			}
		}
	}

	if !isRelease {
		return nil
	}

	log.Printf("[Build] all required build artifacts have been staged \n")
	files, err := os.ReadDir("artifacts")
	if err != nil {
		log.Printf("[Build] failed to read artifacts dir: %v\n", err)
		return err
	}
	var parsedFiles []string
	for _, file := range files {
		if file.IsDir() {
			nested, err := os.ReadDir(path.Join("artifacts", file.Name()))
			if err != nil {
				log.Printf("[Build] failed to read nested dir %s: %v\n", file.Name(), err)
				return err
			}
			for _, nf := range nested {
				if !nf.IsDir() {
					parsedFiles = append(parsedFiles, path.Join("artifacts", file.Name(), nf.Name()))
				}
			}
		}
	}

	if len(parsedFiles) != 6 {
		return fmt.Errorf("[Build] a required build artifact is missing, expected 6, found %d, exiting now \n", len(parsedFiles))
	}

	var artifacts strings.Builder
	for _, file := range parsedFiles {
		artifacts.WriteString(file + "\n")
	}

	log.Printf("[Build] all artifacts successfully staged:\n%s", artifacts.String())

	return nil
}

func Test() error {
	mg.Deps(Build)
	log.Printf("[Test] Testing kscaler version %s\n", version)
	if err := g.Test(flags, "./..."); err != nil {
		return err
	}
	log.Printf("[Test] successfully tested kscaler version %s \n", version)
	return nil
}

func CI() {
	mg.Deps(Test)
}

func flags(version string, commit string) string {
	return fmt.Sprintf(`-s -w -X github.com/rosskirkpat/kscaler/pkg/kscaler.AppVersion=%s -X github.com/rosskirkpat/kscaler/pkg/kscaler.AppCommit=%s -extldflags "-static"`, version, commit)
}

// TODO (rk) fix docs?

//
//func Docs() error {
//	fmt.Println("Generating Docs...")
//	if err := SetupDocs("./docs/cmd/"); err != nil {
//		return err
//	}
//	return nil
//}
//
//func SetupDocs(path string) error {
//	stat, err := os.Stat(path)
//	if err != nil && !os.IsNotExist(err) {
//		return err
//	}
//
//	if stat != nil && stat.IsDir() {
//		if err := os.RemoveAll(path); err != nil {
//			return err
//		}
//	}
//
//	if err := os.Mkdir(path, 0600); err != nil {
//		return err
//	}
//
//	return doc.GenMarkdownTree(rootCmd, path)
//}

func ValidateDocs() error {
	fmt.Println("Validating Docs...")
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	if string(out) != "" {
		return fmt.Errorf("found changes while generating docs, please commit docs changes and try again")
	}

	return nil
}

func InstallDeps() error {
	fmt.Println("Installing Deps...")
	installs := []string{
		"github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.1",
		//"github.com/goreleaser/goreleaser@latest",
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
	if err := sh.Rm("artifacts"); err != nil {
		return err
	}
	return sh.Rm("bin")
}

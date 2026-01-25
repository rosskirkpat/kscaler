package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cleanupCmd = &cobra.Command{
		Use:     "cleanup [-f | --force] [--kubeconfig /path/to/kube/config]",
		Aliases: []string{"clean"},
		Short:   "perform a cleanup of every resource created by kscale",
		Run:     cleanup,
	}

	CleanupFuncs = []func(ctx context.Context, cli ctrlclient.Client) error{
		cleanupDeployment,
		cleanupNamespace,
		cleanupNode,
		cleanupClusterRole,
		cleanupClusterRoleBinding,
		cleanupPod,
		cleanupRole,
		cleanupRoleBinding,
		cleanupSecret,
		cleanupService,
		cleanupServiceAccount,
		cleanupStatefulSet,
	}
)

func askConfirm(input io.Reader) bool {
	var res string
	if _, err := fmt.Fscanln(input, &res); err != nil {
		return false
	}
	if strings.EqualFold(res, "y") || strings.EqualFold(res, "yes") {
		return true
	}
	return false
}

func cleanup(cmd *cobra.Command, args []string) {
	logrus.Traceln("trace starting")
	logrus.Debugln("debug starting")
	logrus.Printf("beginning cleanup")
	//Ask for confirmation before force removing delegation
	if ok, _ := cmd.Flags().GetBool("force"); !ok {
		cmd.Println("\nAre you sure you want to cleanup? (yes/no)")
		confirmed := askConfirm(os.Stdin)
		if !confirmed {
			logrus.Fatalf("Canceling %v cleanup.", DefaultName)
		}
		cmd.Println("Confirmed `yes` from user input")
	} else {
		cmd.Println("user supplied --force flag, skipping confirmation before attempting clean up")
	}

	ctx, cancel := context.WithCancelCause(rootCmd.Context())
	defer cancel(errors.New("cleanup context error"))

	var kubeconfig string
	if kc, ok := os.LookupEnv("KUBECONFIG"); !ok {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	} else {
		kubeconfig = kc
	}

	logrus.Infof("[cleanup] using kubeconfig: %s\n", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatal(err)
	}
	config.QPS = -1
	config.Burst = -1
	cli, err := ctrlclient.New(config, ctrlclient.Options{
		Scheme: scheme,
	})
	if err != nil || cli == nil {
		logrus.Fatalf("[cleanup] error encountered: %s\n", err)
	}

	for _, fn := range CleanupFuncs {
		err := fn(ctx, cli)
		if err != nil {
			logrus.Errorf("cleanup func encountered an unexpected error: %s\n", err)
			continue
		}
	}
}

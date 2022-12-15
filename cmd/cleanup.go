package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rosskirkpat/kscaler/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cleanupCmd = &cobra.Command{
		Use:     "cleanup [-f | --force] [--kubeconfig /path/to/kube/config]",
		Aliases: []string{"clean"},
		Short:   "perform a cleanup of every resource created by kscale",
		Run:     cleanup,
	}
	kubeconfig string
	resource   string
	ns         string
	force      bool
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
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", DefaultKubeConfig, "path to the kubeconfig file")
	cmd.Flags().StringVarP(&ns, "namespace", "n", DefaultNamespace, "namespace to cleanup resources in")
	cmd.Flags().StringVarP(&resource, "resource", "r", "all", "type of resource to cleanup [Default: all]")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force resource cleanup")
	//cmd.AddCommand(cleanupCmd)
	logrus.Traceln("trace starting")
	logrus.Debugln("debug starting")
	logrus.Printf("beginning cleanup")
	// TODO: find each object labeled by kscale and remove it
	// Ask for confirmation before force removing delegation
	if ok, _ := cmd.Flags().GetBool("force"); !ok {
		cmd.Println("\nAre you sure you want to cleanup? (yes/no)")
		confirmed := askConfirm(os.Stdin)
		if !confirmed {
			logrus.Fatalf("Canceling %v cleanup.", DefaultName)
		}
	} else {
		cmd.Println("Confirmed `yes` from user input")
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		60,
	)
	defer cancel()

	c := client.Client{}
	i := int64(300)
	listOpts := v1.ListOptions{
		LabelSelector:  KScaleCreatedByLabel,
		TimeoutSeconds: &i,
	}
	err := c.DeleteCollection(ctx, KScaleConfig{}.Namespace, v1.DeleteOptions{}, listOpts)
	if err != nil {
		cmd.Println("")
		cmd.Println("[cleanup] error encountered")
		logrus.Fatalf(err.Error())
	}

}

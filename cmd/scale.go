package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	scaleCmd = &cobra.Command{
		Use:   "resource [-n 1000] [--kubeconfig /path/to/kube/config]",
		Short: "scale specified resource",
		Run:   scale,
	}
)

func scale(cmd *cobra.Command, args []string) {
	v := viper.GetViper()
	logrus.Traceln("trace starting")
	logrus.Debugln("debug starting")
	cmd.Flags().IntP("number", "n", 1, "Number of resources to create")
	cmd.Flags().String(v.GetString("resource"), "", "Type of resource to create")
	logrus.Printf("creating (%v) resources of type (%v)", viper.GetInt("number"), viper.GetString("resource"))
	err := Parser()
	if err != nil {
		cmd.Println("")
		cmd.Println("[scale] error encountered")
		logrus.Fatalf(err.Error())
	}
}

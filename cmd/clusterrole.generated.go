package main

// Code generated by cmd-gen; DO NOT EDIT.

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//const clusterRole = "clusterRole"

var (
	clusterRoleCmd = &cobra.Command{
		Use:   "clusterRole [-n 1000] [--kubeconfig /path/to/kube/config]",
		Short: "scale clusterRole",
		Run:   scaleClusterRole,
		Hidden: true,
	}
	pclusterRole string
)


func scaleClusterRole(cmd *cobra.Command, args []string) {
	var k KScaleConfig
	logrus.Printf("cmd args: (%v)", cmd.Args)
	cmd.Args = rootCmd.Args
	logrus.Printf("root args: (%v)", rootCmd.Args)
	logrus.Traceln("trace starting")
	logrus.Debugln("debug starting")
	//cmd.PersistentFlags().StringVarP(&number, "number", "n", "", "Number of clusterRole to create")
	//cmd.Flags().StringVarP(&number, "number", "n", "", "Number of clusterRole to create")
	logrus.Printf("args: (%v)", args)
	logrus.Printf("request: %v\n", k.Scale.Request)
	logrus.Printf("getint: %v\n", viper.GetInt("request"))
	logrus.Printf("check number 1: %v\n", viper.GetInt("number"))
	cmd.Flags().Set("number", string(k.Scale.Request))
	logrus.Printf("check number 2: %v\n", viper.GetInt("number"))
	viper.Set("request", KScaleConfig{}.Scale.Request)
	logrus.Printf("check request: %v\n", viper.GetInt("request"))
	logrus.Printf("[scaleClusterRole] attempting to create (%v) resources of type (%v)\n", viper.GetInt("request"), clusterRole)
	
	err := Parser()
	if err != nil {
		cmd.Println("")
		cmd.Println("[scaleClusterRole] error encountered")
		logrus.Fatalf(err.Error())
	}
}


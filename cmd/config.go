package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rosskirkpat/kscaler/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	DefaultAppDirectory     = path.Join(os.Getenv("HOME"), DefaultName)
	DefaultConfigFile       = path.Join(DefaultAppDirectory, "config.yaml")
	DefaultKubeConfig       = path.Join(os.Getenv("HOME"), ".kube/config")
	TemporaryConfigLocation = path.Join(DefaultAppDirectory, "config.tmp")
	KScaleCreatedByLabel    = fmt.Sprintf("created-by=%s", DefaultName)
	cmdConfig               = &cobra.Command{
		Use:     "config",
		Short:   "Config commands: kscale config --help",
		Long:    `Config commands: kscale config <command>`,
		Aliases: []string{"c"},
	}
)

type KScaleConfig struct {
	Scale     Scaler `json:"scale" yaml:"scale" mapstructure:"scale"`
	Namespace string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
	LogLevel  string `json:"loglevel,logLevel,log_level" yaml:"loglevel, logLevel, log_level" mapstructure:"loglevel, log_level, logLevel"`
	client    client.Client
	Type      ResourceType
	Obj       runtime.Object
}

type Scaler struct {
	Resource  string `json:"resource" yaml:"resource" mapstructure:"resource"`
	Namespace string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
	Request   int    `json:"request" yaml:"request" mapstructure:"request"`
}

type ResourceType struct {
	namespace             corev1.Namespace
	secret                corev1.Secret
	serviceAccount        corev1.ServiceAccount
	pod                   corev1.Pod
	node                  corev1.Node
	persistentVolumeClaim corev1.PersistentVolumeClaim
	statefulSet           appsv1.StatefulSet
	role                  rbacv1.Role
	roleBinding           rbacv1.RoleBinding
	clusterRoleBinding    rbacv1.ClusterRoleBinding
	clusterRole           rbacv1.ClusterRole
	service               corev1.Service
}

func LoadConfig() *KScaleConfig {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Warnf("[initConfig] error detecting UserHomeDir: %v", err)
		}
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath(fmt.Sprintf("$HOME/.%s", DefaultName))
		viper.AddConfigPath(fmt.Sprintf("/etc/%s", DefaultName))
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kscale")
	}

	var config KScaleConfig

	if err := viper.ReadInConfig(); err == nil {
		logrus.Infof("Using config file: %s", viper.ConfigFileUsed())
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			logrus.Debugf("config file (%s) changed at (%s)", e.Name, time.Now())
		})
		//if err := viper.ReadInConfig(); err != nil {
		//	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		//		// KScaleConfig file not found
		//	} else {
		//		// KScaleConfig file was found but another error was produced
		//	}
		//}
	} else {
		logrus.Warnf("[LoadConfig] error when reading config file: %v", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		logrus.Printf("[LoadConfig] unable to decode config file to struct: %v", err)
	}
	return &config
}

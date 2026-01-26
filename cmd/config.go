package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rosskirkpat/kscaler/pkg/kscaler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetes2 "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	DefaultAppDirectory     = path.Join(os.Getenv("HOME"), kscaler.DefaultName)
	DefaultConfigFile       = path.Join(DefaultAppDirectory, "config.yaml")
	DefaultKubeConfig       = path.Join(os.Getenv("HOME"), ".kube/config")
	TemporaryConfigLocation = path.Join(DefaultAppDirectory, "config.tmp")
	cmdConfig               = &cobra.Command{
		Use:     "config",
		Short:   "Config commands: kscaler config --help",
		Long:    `Config commands: kscaler config <command>`,
		Aliases: []string{"c"},
	}
)

type KScalerConfig struct {
	Scale          Scaler `json:"scale" yaml:"scale" mapstructure:"scale"`
	Namespace      string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
	LogLevel       string `json:"loglevel" yaml:"loglevel" mapstructure:"loglevel"`
	kubeconfigPath string
	Obj            runtime.Object
	RestConfig     *rest.Config
	Clientset      *kubernetes2.Clientset
}

type Scaler struct {
	Resource  string `json:"resource" yaml:"resource" mapstructure:"resource"`
	Namespace string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
	Request   int    `json:"request" yaml:"request" mapstructure:"request"`
}

func LoadConfig() *KScalerConfig {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Warnf("[initConfig] error detecting UserHomeDir: %v\n", err)
		}
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath(fmt.Sprintf("$HOME/.%s", kscaler.DefaultName))
		viper.AddConfigPath(fmt.Sprintf("/etc/%s", kscaler.DefaultName))
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kscaler")
	}

	var config KScalerConfig

	if err := viper.ReadInConfig(); err == nil {
		logrus.Infof("Using config file: %s\n", viper.ConfigFileUsed())
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			logrus.Debugf("config file (%s) changed at (%s)\n", e.Name, time.Now())
		})
	} else {
		logrus.Warnf("[LoadConfig] error when reading config file, using default configuration: %v\n", err)
		return nil
	}
	if err := viper.Unmarshal(&config); err != nil {
		logrus.Errorf("[LoadConfig] unable to decode config file to struct: %v\n", err)
		return nil
	}
	return &config
}

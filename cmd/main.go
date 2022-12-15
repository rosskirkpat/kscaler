package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	runtime2 "runtime"
	"runtime/debug"
	"syscall"

	"github.com/rosskirkpat/kscaler/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	namespace             = "namespace"
	node                  = "node"
	clusterRole           = "clusterrole"
	clusterRoleBinding    = "clusterRoleBinding"
	persistentVolumeClaim = "persistentvolumeclaim"
	pod                   = "pod"
	role                  = "role"
	roleBinding           = "rolebinding"
	secret                = "secret"
	service               = "service"
	serviceAccount        = "serviceaccount"
	statefulSet           = "statefulset"
	sleBusyBoxImage       = "registry.suse.com/bci/bci-busybox:15.4"
	DefaultName           = "kscale"
	DefaultNamespace      = "kscale"
	defaultPodCommand     = "sleep 30;exit"
	maxSecretSize         = 1023
)

var (
	t        = new(bool)
	f        = new(bool)
	i        = new(int32)
	cfgFile  string
	debugVar bool
	traceVar bool
	quietVar bool
	rootCmd  = &cobra.Command{
		Use:   "kscale",
		Short: "Perform resource scale testing",
		Long:  "Perform customizable resource scale testing against any kubernetes cluster",
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logrus.Errorf("[Execute] did not successfully complete: %v", err)
		os.Exit(1)
	}
}

func SetupDocs(path string) error {
	stat, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if stat != nil && stat.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	if err := os.Mkdir(path, 0600); err != nil {
		return err
	}

	return doc.GenMarkdownTree(rootCmd, path)
}

func initConfig() {
	err := viper.SafeWriteConfigAs(TemporaryConfigLocation)
	if err != nil {
		logrus.Warnf("[initConfig] failed to safely write configuration to %v", TemporaryConfigLocation)
		backupConfig := TemporaryConfigLocation + fmt.Sprint(rand.IntnRange(1, 1000))
		err = viper.SafeWriteConfigAs(backupConfig)
		if err == nil {
			logrus.Infof("[initConfig] successfully backed up running config to %v", backupConfig)
		}
		logrus.Warnf("[initConfig] failed to safely write configuration to %v", backupConfig)
	}
	logrus.Debugf("[initConfig] successfully backed up running configuration to %v", TemporaryConfigLocation)

	if debugVar || traceVar {
		if debugVar {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if traceVar {
			logrus.SetLevel(logrus.TraceLevel)
		}
	} else {
		logrus.SetFormatter(&easy.Formatter{
			LogFormat: "%msg%",
		})
	}
	if quietVar {
		logrus.SetOutput(ioutil.Discard)
	}
	viper.AutomaticEnv()

}

func Parser() error {
	//v := viper.GetViper()
	//r := v.Get("resource")
	k := LoadConfig()
	c := k.client
	*t = true
	*f = false
	*i = int32(k.Scale.Request)

	switch k.Scale.Resource {
	case clusterRole:
		err := k.Scale.Scale(c, k, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
				Namespace:    DefaultNamespace},
		})
		if err != nil {
			return err
		}
	case clusterRoleBinding:
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
				Namespace:    DefaultNamespace,
			},
		}
		err := k.Scale.Scale(c, k, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
				Namespace:    DefaultNamespace,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: cr.GroupVersionKind().Group,
				Kind:     cr.GroupVersionKind().Kind,
				Name:     cr.Name,
			},
		})
		if err != nil {
			return err
		}
	case namespace:
		err := k.Scale.Scale(c, k, &corev1.Namespace{})
		if err != nil {
			return err
		}
	case node:
		err := k.Scale.Scale(c, k, &corev1.Node{})
		if err != nil {
			return err
		}
	case persistentVolumeClaim:
		err := k.Scale.Scale(c, k, &corev1.PersistentVolumeClaim{})
		if err != nil {
			return err
		}
	case pod:
		err := k.Scale.Scale(c, k, &corev1.Pod{})
		if err != nil {
			return err
		}
	case role:
		err := k.Scale.Scale(c, k, &rbacv1.Role{})
		if err != nil {
			return err
		}
	case roleBinding:
		err := k.Scale.Scale(c, k, &rbacv1.RoleBinding{})
		if err != nil {
			return err
		}
	case secret:
		d := make(map[string][]byte)
		for i := 0; i < k.Scale.Request; i++ {
			// add one to ensure we do not generate a 0-byte sized secret
			n := rand.Intn(maxSecretSize) + 1
			d["data"] = make([]byte, n)
		}

		err := k.Scale.Scale(c, k, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
				Namespace:    DefaultNamespace,
			},
			Immutable: t,
			Data:      d,
			Type:      corev1.SecretTypeOpaque,
		})
		if err != nil {
			return err
		}

	case service:
		err := k.Scale.Scale(c, k, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
			},
			Spec: corev1.ServiceSpec{
				ExternalName: DefaultName,
			},
		})
		if err != nil {
			return err
		}
	case serviceAccount:
		err := k.Scale.Scale(c, k, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
			},
			AutomountServiceAccountToken: t,
		})
		if err != nil {
			return err
		}
	case statefulSet:
		p := new(bool)
		*p = false
		err := k.Scale.Scale(c, k, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: i,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: DefaultName,
						Namespace:    DefaultNamespace,
					},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{{
							Name:            DefaultName,
							Image:           sleBusyBoxImage,
							Command:         []string{defaultPodCommand},
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &corev1.SecurityContext{
								Privileged:               f,
								RunAsNonRoot:             t,
								ReadOnlyRootFilesystem:   t,
								AllowPrivilegeEscalation: f,
							},
						},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("[Parser] resource type (%+v) was unexpected", k.Scale.Resource)
	}
	return nil
}

func (s *Scaler) Scale(c client.Client, config *KScaleConfig, obj runtime.Object) error {
	// TODO: validation
	// Validate the path first

	r := config.Scale.Request
	jobs := make(chan int, r)
	results := make(chan int, r)
	for w := 1; w <= (runtime2.NumCPU() / 2); w++ {
		go s.kscaler(c, obj, jobs, results)
	}

	//for i := 0; i < r; i++ {
	//	err := c.Create(c.Ctx, s.Namespace, obj, obj, metav1.CreateOptions{})
	//	if err != nil {
	//		logrus.Errorf("failed during create: %v", err)
	//		break
	//	}
	//}
	for j := 1; j <= r; j++ {
		jobs <- j
	}
	close(jobs)

	for a := 1; a <= r; a++ {
		<-results
	}
	logrus.Infof("finished creating (%v) resources of kind (%v)",
		r,
		obj.GetObjectKind().GroupVersionKind().Kind)
	return nil
}

func (s *Scaler) kscaler(c client.Client, obj runtime.Object, jobs <-chan int, results chan<- int) {
	// Set up a channel to listen to for interrupt signals
	var runChan = make(chan os.Signal, 1)

	// Set up a context to allow for graceful server shutdowns in the event
	// of an OS interrupt (defers the cancel just in case)
	ctx, cancel := context.WithTimeout(
		context.Background(),
		60,
	)
	defer cancel()
	signal.Notify(runChan, os.Interrupt, syscall.SIGABRT)
	for j := range jobs {
		err := c.Create(ctx, s.Namespace, obj, obj, metav1.CreateOptions{
			FieldManager: DefaultName,
		})
		if err != nil {
			logrus.Errorf("failed during create: %v", err)
			break
		}
		results <- j * 2
	}
	// Block on this channel listening for those previously defined syscalls assign
	// to variable, so we can let the user know why the server is shutting down
	interrupt := <-runChan

	// If we get one of the pre-prescribed syscalls, gracefully terminate the server
	// while alerting the user
	logrus.Infof("kscale is shutting down due to %+v\n", interrupt)
}

func main() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kscale.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debugVar, "debug", "d", false, "Turn on debug verbosity")
	rootCmd.PersistentFlags().BoolVarP(&traceVar, "trace", "t", false, "Turn on trace verbosity")
	rootCmd.PersistentFlags().BoolVarP(&quietVar, "quiet", "q", false, "Turn off all output.")
	rootCmd.PersistentFlags().IntVarP(&number, "request", "n", 0, "Number of resources to create")
	rootCmd.AddCommand(cmdConfig)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(scaleCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(podCmd)
	rootCmd.AddCommand(clusterRoleCmd)
	rootCmd.AddCommand(clusterRoleBindingCmd)
	rootCmd.AddCommand(namespaceCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(persistentVolumeClaimCmd)
	rootCmd.AddCommand(roleCmd)
	rootCmd.AddCommand(roleBindingCmd)
	rootCmd.AddCommand(secretCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(serviceAccountCmd)
	rootCmd.AddCommand(statefulSetCmd)
	if Version == "" {
		i, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		Version = i.Main.Version
	}
	Execute()
}

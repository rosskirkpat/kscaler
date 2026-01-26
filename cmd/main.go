package main

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rosskirkpat/kscaler/pkg/kscaler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	clientgo "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cfgFile  string
	debugVar bool
	traceVar bool
	quietVar bool
	number   int
	force    bool

	rootCmd = &cobra.Command{
		Use:              "kscaler",
		Short:            "Perform resource scale testing",
		Long:             "Perform customizable resource scale testing against any kubernetes cluster",
		TraverseChildren: true,
	}

	scheme = runtime.NewScheme()
)

func init() {
	runtimeutil.Must(corev1.AddToScheme(scheme))
	runtimeutil.Must(rbacv1.AddToScheme(scheme))
	runtimeutil.Must(appsv1.AddToScheme(scheme))
	runtimeutil.Must(api.AddToScheme(scheme))
	runtimeutil.Must(clientgo.AddToScheme(scheme))

	if err := SetupDocs("./docs/cmd/"); err != nil {
		logrus.Errorf("Error setting up docs: %v", err)
	}
}

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
	logrus.SetFormatter(&logrus.TextFormatter{})
	if debugVar {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if traceVar {
		logrus.SetLevel(logrus.TraceLevel)
	}
	if quietVar {
		logrus.SetOutput(io.Discard)
	}

	config := LoadConfig()
	if config == nil {
		logrus.Errorf("[InitConfig] loaded config is nil, attempting fallback")
	} else {
		logrus.Infof("[InitConfig] loaded config:\n%+v\n%+v\n%+v\n%+v\n", config.Scale, config.Namespace, config.LogLevel, config.kubeconfigPath)
		viper.AutomaticEnv()
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("[InitConfig] error getting current working directory: %v", err)
	} else {
		TemporaryConfigLocation = path.Join(cwd, ".kscaler.conf")
	}

	TemporaryConfigLocation = "./.kscaler.conf"

	viper.NewWithOptions(viper.KeyDelimiter("_"))
	err = viper.SafeWriteConfigAs(TemporaryConfigLocation)
	if err != nil {
		logrus.Warnf("[initConfig] failed to safely write configuration to %v\n", TemporaryConfigLocation)
		backupConfig := TemporaryConfigLocation + fmt.Sprint(rand.IntnRange(1, 1000))
		err = viper.SafeWriteConfigAs(backupConfig)
		if err == nil {
			logrus.Infof("[initConfig] successfully backed up running config to %v\n", backupConfig)
		}
		logrus.Warnf("[initConfig] failed to safely write configuration to %v\n", backupConfig)
	}
	logrus.Debugf("[initConfig] successfully backed up running configuration to %v\n", TemporaryConfigLocation)

	viper.AutomaticEnv()
}

func Parser(k *KScalerConfig) error {
	replicas := ptr.To(int32(k.Scale.Request))

	var kubeconfig string
	if kc, ok := os.LookupEnv("KUBECONFIG"); !ok {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	} else {
		kubeconfig = kc
	}

	logrus.Infof("[Parser] using kubeconfig: %s\n", kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatal(err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("could not create kubernetes clientset: %w", err)
	}

	// TODO (rk) allow tuning qps and burst values via env vars and/or cli flags
	config.QPS = 250
	config.Burst = 500
	cli, err := ctrlclient.New(config, ctrlclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return err
	}
	k.RestConfig = config
	k.Clientset = clientset

	kscalerNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kscaler.DefaultNamespace,
			Labels: map[string]string{"created-by": kscaler.DefaultName},
		},
	}
	if err := cli.Create(context.Background(), kscalerNamespace); errors.FilterOut(err, apierrors.IsAlreadyExists) != nil {
		return err
	}

	switch k.Scale.Resource {

	case kscaler.ClusterRoleName:
		if err := k.Scale.Scale(cli, k, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Namespace:    kscaler.DefaultNamespace},
		}); err != nil {
			return err
		}

	case kscaler.ClusterRoleBindingName:
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
			},
		}
		err := k.Scale.Scale(cli, k, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
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
	case kscaler.DeploymentName:
		if err := k.Scale.Scale(cli, k, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Namespace:    kscaler.DefaultNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: replicas,
				Template: kscaler.PodTemplateSpec,
			},
		}); err != nil {
			return err
		}

	case kscaler.NamespaceName:
		err := k.Scale.Scale(cli, k, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Labels: map[string]string{
					"app": kscaler.DefaultName,
				},
			},
		})
		if err != nil {
			return err
		}

	case kscaler.NodeName:
		if err := k.Scale.Scale(cli, k, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Labels: map[string]string{
					"node.alpha.kubernetes.io/ttl":                           "0s",
					"volumes.kubernetes.io/controller-managed-attach-detach": "false",
				},
			},
			Spec:   corev1.NodeSpec{},
			Status: corev1.NodeStatus{},
		}); err != nil {
			return err
		}

	case kscaler.PersistentVolumeClaimName:
		err := k.Scale.Scale(cli, k, &corev1.PersistentVolumeClaim{})
		if err != nil {
			return err
		}

	case kscaler.PodName:
		err := k.Scale.Scale(cli, k, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Namespace:    kscaler.DefaultNamespace,
			},
			Spec: kscaler.PodSpec,
		})
		if err != nil {
			return err
		}

	case kscaler.RoleName:
		err := k.Scale.Scale(cli, k, &rbacv1.Role{})
		if err != nil {
			return err
		}

	case kscaler.RoleBindingName:
		err := k.Scale.Scale(cli, k, &rbacv1.RoleBinding{})
		if err != nil {
			return err
		}

	case kscaler.SecretName:
		d := make(map[string][]byte)
		for i := 0; i < k.Scale.Request; i++ {
			// add one to ensure we do not generate a 0-byte sized secret
			n := rand.Intn(kscaler.MaxSecretSize) + 1
			d["data"] = make([]byte, n)
		}

		err := k.Scale.Scale(cli, k, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Namespace:    kscaler.DefaultNamespace,
			},
			Immutable: ptr.To(true),
			Data:      d,
			Type:      corev1.SecretTypeOpaque,
		})
		if err != nil {
			return err
		}

	case kscaler.ServiceName:

		err := k.Scale.Scale(cli, k, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
			},
			Spec: corev1.ServiceSpec{
				ExternalName: kscaler.DefaultName,
			},
		})
		if err != nil {
			return err
		}
	case kscaler.ServiceAccountName:

		err := k.Scale.Scale(cli, k, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
			},
			AutomountServiceAccountToken: ptr.To(true),
		})
		if err != nil {
			return err
		}
	case kscaler.StatefulSetName:
		p := new(bool)
		*p = false

		err := k.Scale.Scale(cli, k, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: kscaler.DefaultName + "-",
				Namespace:    kscaler.DefaultNamespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To(int32(1)),
				Template: kscaler.PodTemplateSpec,
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

func (s *Scaler) Scale(c ctrlclient.Client, config *KScalerConfig, obj ctrlclient.Object) error {
	// TODO: validation
	// Validate the path first

	jobs := config.Scale.Request
	// TODO (rk) pass through context from root command
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	successfulJobs := 0
	totalRetriesRequired := 0
	for job := 0; job < jobs; job++ {
		wg.Add(1)
		object := obj.DeepCopyObject().(ctrlclient.Object)
		object.SetLabels(map[string]string{
			"created-by": kscaler.DefaultName,
		})
		maxRetries := 5
		go func() {
			defer wg.Done()
			for attempt := 1; attempt <= maxRetries; attempt++ {
				if err := s.scaleOne(ctx, c, object, job); err == nil {
					successfulJobs++
					return
				}
				logrus.Errorf("job %d failed, retry attempt %d/%d", job, attempt, maxRetries)
				totalRetriesRequired++
			}
			logrus.Errorf("job %d reached maximum number of retries (%d), job will be skipped", job, maxRetries)
		}()
	}
	t0 := time.Now()
	wg.Wait()

	logrus.Infof("kscaler action took %v", time.Since(t0))
	logrus.Infof("%d/%d jobs were successful with %d retries required", successfulJobs, jobs, totalRetriesRequired)
	logrus.Infof("finished creating %v resources of kind (%T)", successfulJobs, obj)
	logrus.Info("kscaler is shutting down")

	return nil
}

func (s *Scaler) scaleOne(ctx context.Context, c ctrlclient.Client, obj ctrlclient.Object, job int) error {
	if obj == nil {
		logrus.Errorf("[Scaler] Scale job %d called with nil resource", job)
		return nil
	}
	obj.SetResourceVersion("")

	if fmt.Sprintf("%T", obj) == "*v1.Node" {
		obj.(*corev1.Node).Spec.ProviderID = fmt.Sprintf("kscaler://%d", job)
	}

	err := c.Create(ctx, obj, &ctrlclient.CreateOptions{
		FieldManager: kscaler.DefaultName,
	})
	if err != nil {
		logrus.Errorf("job %d failed during create: %v\n", job, err)
		return err
	}
	if fmt.Sprintf("%T", obj) == "*v1.Node" {
		newNode := &corev1.Node{}
		err := c.Get(ctx, types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		}, newNode)
		if err != nil {
			logrus.Errorf("job %d failed during create: %v\n", job, err)
			return err
		}
		orig := newNode.DeepCopy()
		//no := obj.DeepCopyObject().(*corev1.Node)
		newNode.Status = corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:                     apiresource.MustParse("16"),
				corev1.ResourceMemory:                  apiresource.MustParse("32Gi"),
				corev1.ResourceEphemeralStorage:        apiresource.MustParse("100Gi"),
				corev1.ResourceHugePagesPrefix + "1Gi": apiresource.MustParse("0"),
				corev1.ResourceHugePagesPrefix + "2Mi": apiresource.MustParse("0"),
				corev1.ResourcePods:                    apiresource.MustParse("110"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:                     apiresource.MustParse("16"),
				corev1.ResourceMemory:                  apiresource.MustParse("32Gi"),
				corev1.ResourceEphemeralStorage:        apiresource.MustParse("100Gi"),
				corev1.ResourceHugePagesPrefix + "1Gi": apiresource.MustParse("0"),
				corev1.ResourceHugePagesPrefix + "2Mi": apiresource.MustParse("0"),
				corev1.ResourcePods:                    apiresource.MustParse("110"),
			},
			Conditions: []corev1.NodeCondition{
				{
					LastHeartbeatTime:  metav1.Now(),
					LastTransitionTime: metav1.Now(),
					Message:            "kubelet has sufficient memory available",
					Reason:             "KubeletHasSufficientMemory",
					Status:             "False",
					Type:               corev1.NodeMemoryPressure,
				},
				{
					LastHeartbeatTime:  metav1.Now(),
					LastTransitionTime: metav1.Now(),
					Message:            "kubelet has no disk pressure",
					Reason:             "KubeletHasNoDiskPressure",
					Status:             "False",
					Type:               corev1.NodeDiskPressure,
				},
				{
					LastHeartbeatTime:  metav1.Now(),
					LastTransitionTime: metav1.Now(),
					Message:            "kubelet has sufficient PID available",
					Reason:             "KubeletHasSufficientPID",
					Status:             "False",
					Type:               corev1.NodePIDPressure,
				},
				{
					LastHeartbeatTime:  metav1.Now(),
					LastTransitionTime: metav1.Now(),
					Message:            "kubelet is posting ready status",
					Reason:             "KubeletReady",
					Status:             "True",
					Type:               corev1.NodeReady,
				},
			},
			Addresses: []corev1.NodeAddress{},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: int32(10250),
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture:            "amd64",
				BootID:                  string(uuid.NewUUID()),
				ContainerRuntimeVersion: "docker://19.3.12",
				KernelVersion:           "4.19.76-fakelinux",
				KubeProxyVersion:        "v1.19.1", //nolint:staticcheck //deprecated, but we should continue to use it until the api removes it
				KubeletVersion:          "v1.19.1",
				MachineID:               string(uuid.NewUUID()),
				SystemUUID:              string(uuid.NewUUID()),
				OperatingSystem:         "linux",
				OSImage:                 "Fake Kubernetes Image",
			},
			Images: []corev1.ContainerImage{},
		}

		// use fake kubelet IP
		newNode.Status.Addresses = append(newNode.Status.Addresses, corev1.NodeAddress{
			Address: "127.0.0.1",
			Type:    corev1.NodeInternalIP,
		})

		//// use fake kubelet hostname
		//newNode.Status.Addresses = append(newNode.Status.Addresses, corev1.NodeAddress{
		//	Address: GetNodeHost(newNode.Name),
		//	Type:    corev1.NodeHostName,
		//})

		err = c.Status().Patch(ctx, newNode, ctrlclient.MergeFrom(orig))
		if err != nil {
			return err
		}

		// remove not ready taints
		orig = newNode.DeepCopy()
		newNode.Spec.Taints = []corev1.Taint{}
		err = c.Patch(ctx, newNode, ctrlclient.MergeFrom(orig))
		if err != nil {
			return err
		}
	}

	if obj.GetName() == "" {
		logrus.Errorf("job %d reported success yet has no resource name\n", job)
	}
	logrus.Infof("job %d succeeded, created resource %T/%s", job, obj, obj.GetName())
	return nil
}

func main() {
	cobra.OnInitialize(initConfig)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(stderrors.New("root command context error"))
	rootCmd.SetContext(ctx)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kscaler.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debugVar, "debug", "d", false, "Turn on debug verbosity")
	rootCmd.PersistentFlags().BoolVarP(&traceVar, "trace", "t", false, "Turn on trace verbosity")
	rootCmd.PersistentFlags().BoolVarP(&quietVar, "quiet", "q", false, "Turn off all output.")
	rootCmd.PersistentFlags().IntVarP(&number, "number", "n", 0, "Number of resources to create")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force resource cleanup")
	rootCmd.AddCommand(cmdConfig)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(podCmd)
	rootCmd.AddCommand(clusterRoleCmd)
	rootCmd.AddCommand(clusterRoleBindingCmd)
	rootCmd.AddCommand(namespaceCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(persistentVolumeClaimCmd)
	rootCmd.AddCommand(roleCmd)
	rootCmd.AddCommand(roleBindingCmd)
	secretCmd.Flags().IntVarP(&secretSize, "size", "s", 1, "Secret size")
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

//func GetNodeHost(nodeName string) string {
//	return strings.ReplaceAll(nodeName, ".", "-") + "." + "nodes.kscaler.home.arpa"
//}

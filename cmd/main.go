package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	//"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	scheme2 "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deployment            = "deployment"
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
	busyboxImage          = "busybox"
	DefaultName           = "kscaler"
	DefaultNamespace      = "kscaler"
	maxSecretSize         = 1023
)

var (
	t                 = new(bool)
	f                 = new(bool)
	i                 = new(int32)
	cfgFile           string
	debugVar          bool
	traceVar          bool
	quietVar          bool
	number            int
	defaultPodCommand = []string{"sleep", "30"}
	scheme            = runtime.NewScheme()

	rootCmd = &cobra.Command{
		Use:   "kscaler",
		Short: "Perform resource scale testing",
		Long:  "Perform customizable resource scale testing against any kubernetes cluster",
	}
)

func init() {
	runtimeutil.Must(corev1.AddToScheme(scheme))
	runtimeutil.Must(rbacv1.AddToScheme(scheme))
	runtimeutil.Must(appsv1.AddToScheme(scheme))
	runtimeutil.Must(api.AddToScheme(scheme))
	runtimeutil.Must(scheme2.AddToScheme(scheme))
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
		logrus.SetOutput(io.Discard)
	}
	viper.AutomaticEnv()

}

func Parser(k *KScalerConfig) error {
	//v := viper.GetViper()
	//r := v.Get("resource")
	//logrus.Infof("[Parser] resource: %v\n", r)
	//k := LoadConfig()
	// use the current context in kubeconfig

	//c := k.client
	*t = true
	*f = false
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

	//apiConfig, err := clientcmd.LoadFromFile(kubeconfig)
	//if err != nil {
	//	return err
	//}
	//clientset := kubernetes.NewForConfigOrDie(config)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatal(err)
	}
	config.QPS = -1
	config.Burst = -1
	cli, err := ctrlclient.New(config, ctrlclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return err
	}

	//config.GroupVersion = &resourcev1beta1.SchemeGroupVersion
	//config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	//restClient, err := rest.RESTClientFor(config)
	//if err != nil {
	//	logrus.Fatal(err)
	//}
	if err := cli.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultNamespace}}); errors.FilterOut(err, errors2.IsAlreadyExists) != nil {
		return err
	}

	switch k.Scale.Resource {
	case clusterRole:

		//gvr := schema.GroupVersionResource{
		//	Group:    rbacv1.GroupName,
		//	Version:  rbacv1.SchemeGroupVersion.Version,
		//	Resource: "ClusterRole",
		//}
		//k.client = client.NewClient(gvr, "ClusterRole", false, clientset.RESTClient(), time.Minute)
		if err := k.Scale.Scale(cli, k, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				Namespace:    DefaultNamespace},
		}); err != nil {
			return err
		}
	case clusterRoleBinding:
		//gvr := schema.GroupVersionResource{
		//	Group:    rbacv1.GroupName,
		//	Version:  rbacv1.SchemeGroupVersion.Version,
		//	Resource: "ClusterRoleBinding",
		//}
		//k.client = client.NewClient(gvr, "ClusterRoleBinding", false, clientset.RESTClient(), time.Minute)

		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				//Namespace:    DefaultNamespace,
			},
		}
		err := k.Scale.Scale(cli, k, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				//Namespace:    DefaultNamespace,
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
	case deployment:
		if err := k.Scale.Scale(cli, k, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				Namespace:    DefaultNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: replicas,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: DefaultName + "-",
						Namespace:    DefaultNamespace,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            DefaultName,
							Image:           busyboxImage,
							Command:         defaultPodCommand,
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                ptr.To(int64(2000)),
								RunAsGroup:               ptr.To(int64(2000)),
								Privileged:               ptr.To(false),
								RunAsNonRoot:             ptr.To(false),
								ReadOnlyRootFilesystem:   ptr.To(true),
								AllowPrivilegeEscalation: ptr.To(false),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
						},
					},
				},
			},
		}); err != nil {
			return err
		}
	case namespace:
		//namespaceClient := clientset.CoreV1().Namespaces()
		//cli := clientset.RESTClient()
		//gvr := schema.GroupVersionResource{
		//	Group:    corev1.SchemeGroupVersion.Group,
		//	Version:  corev1.SchemeGroupVersion.Version,
		//	Resource: "Namespace",
		//}
		//k.client = client.NewClient(gvr, "Namespace", false, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				Labels: map[string]string{
					"app": DefaultName,
				},
			},
		})
		if err != nil {
			return err
		}
	case node:
		//gvr := schema.GroupVersionResource{
		//	Group:    corev1.GroupName,
		//	Version:  corev1.SchemeGroupVersion.Version,
		//	Resource: "Node",
		//}
		//k.client = client.NewClient(gvr, "Node", false, clientset.RESTClient(), time.Minute)

		if err := k.Scale.Scale(cli, k, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				Labels: map[string]string{
					"app":                          DefaultName,
					"node.alpha.kubernetes.io/ttl": "60s",
				},
			},
			Spec: corev1.NodeSpec{
				PodCIDR:       "10.42.0.0/16",
				Unschedulable: true,
			},
			Status: corev1.NodeStatus{
				Phase: corev1.NodePending,
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
						Reason: "KubeletReady",
					},
				},
			},
		}); err != nil {
			return err
		}
	case persistentVolumeClaim:
		//gvr := schema.GroupVersionResource{
		//	Group:    corev1.GroupName,
		//	Version:  corev1.SchemeGroupVersion.Version,
		//	Resource: "PersistentVolumeClaim",
		//}
		//k.client = client.NewClient(gvr, "PersistentVolumeClaim", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &corev1.PersistentVolumeClaim{})
		if err != nil {
			return err
		}
	case pod:
		//gvr := schema.GroupVersionResource{
		//	Group:    corev1.GroupName,
		//	Version:  corev1.SchemeGroupVersion.Version,
		//	Resource: "Pod",
		//}
		//k.client = client.NewClient(gvr, "Pod", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				Namespace:    DefaultNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:            DefaultName,
					Image:           busyboxImage,
					Command:         defaultPodCommand,
					ImagePullPolicy: "IfNotPresent",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:                ptr.To(int64(2000)),
						RunAsGroup:               ptr.To(int64(2000)),
						Privileged:               ptr.To(false),
						RunAsNonRoot:             ptr.To(false),
						ReadOnlyRootFilesystem:   ptr.To(true),
						AllowPrivilegeEscalation: ptr.To(false),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
				},
			},
		})
		if err != nil {
			return err
		}
	case role:
		//gvr := schema.GroupVersionResource{
		//	Group:    rbacv1.GroupName,
		//	Version:  rbacv1.SchemeGroupVersion.Version,
		//	Resource: "Role",
		//}
		//k.client = client.NewClient(gvr, "Role", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &rbacv1.Role{})
		if err != nil {
			return err
		}
	case roleBinding:
		//gvr := schema.GroupVersionResource{
		//	Group:    rbacv1.GroupName,
		//	Version:  rbacv1.SchemeGroupVersion.Version,
		//	Resource: "RoleBinding",
		//}
		//k.client = client.NewClient(gvr, "RoleBinding", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &rbacv1.RoleBinding{})
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
		//gvr := schema.GroupVersionResource{
		//	Group:    corev1.GroupName,
		//	Version:  corev1.SchemeGroupVersion.Version,
		//	Resource: "Secret",
		//}
		//k.client = client.NewClient(gvr, "Secret", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
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
		//	gvr := schema.GroupVersionResource{
		//		Group:    corev1.GroupName,
		//		Version:  corev1.SchemeGroupVersion.Version,
		//		Resource: "Service",
		//	}
		//	k.client = client.NewClient(gvr, "Service", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
			},
			Spec: corev1.ServiceSpec{
				ExternalName: DefaultName,
			},
		})
		if err != nil {
			return err
		}
	case serviceAccount:
		//gvr := schema.GroupVersionResource{
		//	Group:    corev1.GroupName,
		//	Version:  corev1.SchemeGroupVersion.Version,
		//	Resource: "ServiceAccount",
		//}
		//k.client = client.NewClient(gvr, "ServiceAccount", false, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
			},
			AutomountServiceAccountToken: t,
		})
		if err != nil {
			return err
		}
	case statefulSet:
		p := new(bool)
		*p = false
		//gvr := schema.GroupVersionResource{
		//	Group:    appsv1.GroupName,
		//	Version:  appsv1.SchemeGroupVersion.Version,
		//	Resource: "StatefulSet",
		//}
		//k.client = client.NewClient(gvr, "StatefulSet", true, clientset.RESTClient(), time.Minute)

		err := k.Scale.Scale(cli, k, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: DefaultName + "-",
				Namespace:    DefaultNamespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: i,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: DefaultName + "-",
						Namespace:    DefaultNamespace,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            DefaultName,
							Image:           busyboxImage,
							Command:         defaultPodCommand,
							ImagePullPolicy: "IfNotPresent",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                ptr.To(int64(2000)),
								RunAsGroup:               ptr.To(int64(2000)),
								Privileged:               ptr.To(false),
								RunAsNonRoot:             ptr.To(false),
								ReadOnlyRootFilesystem:   ptr.To(true),
								AllowPrivilegeEscalation: ptr.To(false),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
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

func (s *Scaler) Scale(c ctrlclient.Client, config *KScalerConfig, obj ctrlclient.Object) error {
	//func (s *Scaler) Scale(c *client.Client, config *KScalerConfig, obj runtime.Object) error {
	// TODO: validation
	// Validate the path first

	jobs := config.Scale.Request
	//done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//finish := make(chan struct{})
	var wg sync.WaitGroup
	//originalObj := obj.DeepCopyObject().(ctrlclient.Object)
	for job := 0; job < jobs; job++ {
		wg.Add(1)
		//object := originalObj.DeepCopyObject().(ctrlclient.Object)
		go func() {
			defer wg.Done()
			if err := s.scaleOne(ctx, c, obj, job); err != nil {
				logrus.Errorf("failed during create: %v\n", err)
				return
			}
		}()
	}
	t0 := time.Now()
	wg.Wait()

	//for i := 0; i < r; i++ {
	//	wg.Add(1)
	//	//go func() {
	//	select {
	//	default:
	//		go func() {
	//			err := s.scaleOne(ctx, c, obj, i)
	//			if err != nil {
	//				logrus.Errorf("failed during create: %v\n", err)
	//			}
	//		}()
	//	case <-finish:
	//		break
	//	}
	//	wg.Done()
	//	//}()
	//}
	//wg.Wait()
	//close(finish)
	logrus.Infof("scale action took %v \n", time.Since(t0))
	logrus.Infof("finished creating (%v) resources of kind (%T)\n", jobs, obj)
	logrus.Info("kscaler is shutting down")
	//jobs := make(chan int, r)
	//results := make(chan int, r)
	//for w := 1; w <= (runtime2.NumCPU() / 2); w++ {
	//	go s.kscaler(c, obj, r, results)
	//go s.kscaler(c, obj, jobs, results)
	//}

	//s.kscaler(c, obj, r, results)

	//go func(ctx context.Context) {
	//	defer cancel()
	//	for {
	//		//select {
	//		//case <-done:
	//		//	return
	//		//default:
	//		//	s.kscaler(c, obj, r, done)
	//		//}
	//		select {
	//		case <-ctx.Done(): // if cancel() execute
	//			done <- struct{}{}
	//			return
	//		default:
	//			s.kscaler(ctx, c, obj, r)
	//		}
	//	}
	//}(ctx)
	//close(done)
	//
	//var runChan = make(chan os.Signal, 1)
	//signal.Notify(runChan)

	// Block on this channel listening for those previously defined syscalls assign
	// to variable, so we can let the user know why the server is shutting down

	//<-runChan

	// If we get one of the pre-prescribed syscalls, gracefully terminate the server
	// while alerting the user

	//for i := 0; i < r; i++ {
	//	err := c.Create(c.Ctx, s.Namespace, obj, obj, metav1.CreateOptions{})
	//	if err != nil {
	//		logrus.Errorf("failed during create: %v", err)
	//		break
	//	}
	//}
	//for j := 1; j <= r; j++ {
	//	jobs <- j
	//}
	//close(jobs)

	//for a := 1; a <= r; a++ {
	//	<-results
	//}
	//close(results)
	return nil
}

func (s *Scaler) scaleOne(ctx context.Context, c ctrlclient.Client, obj ctrlclient.Object, job int) error {
	obj.SetResourceVersion("")

	if fmt.Sprintf("%T", obj) == "*v1.Node" {
		obj.(*corev1.Node).Spec.ProviderID = fmt.Sprintf("kscaler://%d", job)
	}
	//fmt.Printf("obj: %+v\n", obj)
	//fmt.Printf("obj kind: %v\n", obj.GetObjectKind())
	//if obj.GetObjectKind().GroupVersionKind().Kind == "Node" {
	//	logrus.Infof("setting node providerID\n")
	//	obj.(*corev1.Node).Spec.ProviderID = fmt.Sprintf("kscaler://%d", job)
	//}
	err := c.Create(ctx, obj, &ctrlclient.CreateOptions{
		FieldManager: DefaultName,
	})
	if err != nil {
		logrus.Errorf("job %d failed during create: %v\n", job, err)
		return err
	}
	//results <- j * 2
	logrus.Infof("job %d succeeded, created resource %T/%s\n", job, obj, obj.GetName())
	return nil
}

func (s *Scaler) kscaler(ctx context.Context, c ctrlclient.Client, obj ctrlclient.Object, jobs int) {
	//func (s *Scaler) kscaler(c ctrlclient.Client, obj ctrlclient.Object, jobs int, results chan<- int) {
	//func (s *Scaler) kscaler(c ctrlclient.Client, obj ctrlclient.Object, jobs <-chan int, results chan<- int) {
	//func (s *Scaler) kscaler(c *client.Client, obj runtime.Object, jobs <-chan int, results chan<- int) {
	// Set up a channel to listen to for interrupt signals
	//var runChan = make(chan os.Signal, 1)
	//signal.Notify(runChan)
	//
	//runChan := make(chan os.Signal, 1)
	// Set up a context to allow for graceful server shutdowns in the event
	// of an OS interrupt (defers the cancel just in case)
	//ctx, cancel := context.WithTimeout(
	//	context.Background(),
	//	60,
	//)
	//defer cancel()

	//ctx := context.Background()
	//signal.Notify(runChan, os.Interrupt, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	for j := range jobs {
		select {
		case <-ctx.Done():
			logrus.Info("all jobs done")
			return
		default:
			err := s.scaleOne(ctx, c, obj, j)
			if err != nil {
				logrus.Errorf("failed during create: %v\n", err)
				return
			}
		}
	}

	originalObj := obj.DeepCopyObject().(ctrlclient.Object)
	for j := range jobs {
		obj = originalObj.DeepCopyObject().(ctrlclient.Object)
		if obj.GetObjectKind().GroupVersionKind().Kind == "Node" {
			obj.(*corev1.Node).Spec.ProviderID = fmt.Sprintf("kscaler://%d", j)
		}
		err := c.Create(ctx, obj, &ctrlclient.CreateOptions{
			FieldManager: DefaultName,
		})
		if err != nil {
			logrus.Errorf("failed during create: %v\n", err)
			//logrus.Infof("job failed, jobs left: %v\n", len(jobs))
			//results <- -1
			break
		}
		//results <- j * 2
		logrus.Infof("job %d succeeded, created resource %T/%s\n", j, obj, obj.GetName())
	}
	ctx.Done()
	//done <- true

	//// Block on this channel listening for those previously defined syscalls assign
	//// to variable, so we can let the user know why the server is shutting down
	//
	//interrupt := <-runChan
	//// If we get one of the pre-prescribed syscalls, gracefully terminate the server
	//// while alerting the user
	//logrus.Infof("kscaler is shutting down due to %+v\n", interrupt)
}

func main() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kscaler.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debugVar, "debug", "d", false, "Turn on debug verbosity")
	rootCmd.PersistentFlags().BoolVarP(&traceVar, "trace", "t", false, "Turn on trace verbosity")
	rootCmd.PersistentFlags().BoolVarP(&quietVar, "quiet", "q", false, "Turn off all output.")
	rootCmd.PersistentFlags().IntVarP(&number, "number", "n", 0, "Number of resources to create")
	rootCmd.AddCommand(cmdConfig)
	rootCmd.AddCommand(cleanupCmd)
	//rootCmd.AddCommand(scaleCmd)
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

	//var kubeconfig *string
	//if home := homedir.HomeDir(); home != "" {
	//	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	//} else {
	//	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	//}

	//flag.Parse()

	//rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("kubeconfig"))

	Execute()
}

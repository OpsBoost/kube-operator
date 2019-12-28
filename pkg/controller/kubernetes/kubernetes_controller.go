package kubernetes

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	appv1alpha1 "github.com/opsboost/kube-operator/pkg/apis/app/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kubernetes")

func MustNewKubeClient() kubernetes.Interface {
	cfg, err := InClusterConfig()
	if err != nil {
		panic(err)
	}
	return kubernetes.NewForConfigOrDie(cfg)
}

func InClusterConfig() (*rest.Config, error) {
	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Kubernetes Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKubernetes{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kubernetes-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Kubernetes
	err = c.Watch(&source.Kind{Type: &appv1alpha1.Kubernetes{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Kubernetes
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1alpha1.Kubernetes{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKubernetes implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKubernetes{}

// ReconcileKubernetes reconciles a Kubernetes object
type ReconcileKubernetes struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Kubernetes object and makes changes based on the state read
// and what is in the Kubernetes.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKubernetes) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Kubernetes")

	// Fetch the Kubernetes instance
	instance := &appv1alpha1.Kubernetes{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	pod := newPodForCR(instance)

	// Set Kubernetes instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

func getServiceForServiceName(serviceName string, namespace string, kubecli kubernetes.Interface) (*corev1.Service, error) {
	listOptions := metav1.ListOptions{}
	svcs, err := kubecli.CoreV1().Services(namespace).List(listOptions)
	if err != nil {
		log.Error(err, "Service not found")
	}
	for _, svc := range svcs.Items {
		if strings.Contains(svc.Name, serviceName) {
			fmt.Fprintf(os.Stdout, "service name: %v\n", svc.Name)
			return &svc, nil
		}
	}
	return nil, errors.NewBadRequest("cannot find service for deployment")
}

func getPodsForSvc(svc *corev1.Service, namespace string, kubecli kubernetes.Interface) (*corev1.PodList, error) {
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := kubecli.CoreV1().Pods(namespace).List(listOptions)
	if err != nil {
		log.Error(err, "Pods not found")
	}
	for _, pod := range pods.Items {
		fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)
	}
	return pods, nil
}

// newPodForCR returns a integrated kubernetes control plane pod with the same name/namespace as the cr
func newPodForCR(cr *appv1alpha1.Kubernetes) *corev1.Pod {
	labels := map[string]string{
		"kubernetes": cr.Name,
	}

	podVolumes := []corev1.Volume{}

	kubernetesApiServer := corev1.Container{
		Name:    "kube-apiserver",
		Image:   fmt.Sprintf("k8s.gcr.io/hyperkube:v%s", cr.Spec.Version),
		Command: []string{"kube-apiserver", "--etcd-servers=http://localhost:2379", "--authorization-mode=AlwaysAllow"},
		Env: []corev1.EnvVar{
			{
				Name: "KUBERNETES_SERVICE_HOST",
			},
			{
				Name: "KUBERNETES_SERVICE_PORT",
			},
			{
				Name: "KUBERNETES_SERVICE_HTTPS_PORT",
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "api",
				ContainerPort: 6443,
				Protocol:      "TCP",
			},
		},
	}

	var etcdServiceNs string
	if cr.Spec.EtcdServiceNamespace != "" {
		etcdServiceNs = "default"
	} else {
		etcdServiceNs = cr.Spec.EtcdServiceNamespace
	}

	kubecli := MustNewKubeClient()
	etcdService, _ := getServiceForServiceName(cr.Spec.EtcdService, etcdServiceNs, kubecli)
	etcdPods, _ := getPodsForSvc(etcdService, etcdServiceNs, kubecli)

	etcdCommand := []string{"etcd", "grpc-proxy", "start", "--listen-addr=127.0.0.1:2379", "--endpoints="}

	etcdPeers := []string{}

	for _, pod := range etcdPods.Items {
		etcdPeers = append(etcdPeers, fmt.Sprintf("%s%s.svc:2379", pod.Name, pod.Namespace))
	}

	etcdCommand = append(etcdCommand, strings.Join(etcdPeers, ","))

	//etcdCommand = []string{"etcd", "grpc-proxy", "start", "--listen-addr=127.0.0.1:2379", fmt.Sprintf("--endpoints=%s.%s.svc:2379", cr.Spec.EtcdService, cr.Namespace)}
	etcdVolumes := []corev1.VolumeMount{}

	if cr.Spec.EtcdNamespace != "" {
		etcdCommand = append(etcdCommand, fmt.Sprintf("--namespace=%s", cr.Spec.EtcdNamespace))
	} else {
		etcdCommand = append(etcdCommand, fmt.Sprintf("--namespace=%s", cr.Name))
	}

	if cr.Spec.EtcdClientSecretRef != "" {
		volumeName := "etcd-peer-tls"
		podVolumes = append(podVolumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cr.Spec.EtcdClientSecretRef,
				},
			},
		})
		etcdVolumes = append(etcdVolumes, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			MountPath: fmt.Sprintf("/%s", volumeName),
		})
		etcdCommand = append(etcdCommand, fmt.Sprintf("--cert-file=/%s/etcd-client.crt", volumeName))
		etcdCommand = append(etcdCommand, fmt.Sprintf("--key-file=/%s/etcd-client.key", volumeName))
		etcdCommand = append(etcdCommand, fmt.Sprintf("--trusted-ca-file=/%s/etcd-client-ca.crt", volumeName))
	}

	etcdProxy := corev1.Container{
		Name:         "etcd",
		Image:        "quay.io/coreos/etcd:v3.4.3",
		Command:      etcdCommand,
		VolumeMounts: etcdVolumes,
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Volumes: podVolumes,
			Containers: []corev1.Container{
				kubernetesApiServer,
				etcdProxy,
			},
		},
	}
}

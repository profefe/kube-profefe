package kubeutil

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	// AllNamespace when you use the  kubernetes client in go an empty string
	// means all namespace
	AllNamespace = ""
	// ProfefeEnabledAnnotation is the annotation used to discrimanate when a
	// pod has to be profiled or not.
	ProfefeEnabledAnnotation = "profefe.com/enable"
)

// GetPodByName returns a single pod with the profefe annotation enabled
func GetPodByName(clientset kubernetes.Interface, namespace, name string, opt metav1.GetOptions) (*v1.Pod, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(name, opt)
	if err != nil {
		return nil, err
	}
	enabled, ok := pod.Annotations[ProfefeEnabledAnnotation]
	if ok && enabled == "true" && pod.Status.Phase == v1.PodRunning {
		return pod, nil
	}
	return nil, fmt.Errorf("Pod not found or it does not have the right annotations")
}

// GetNamespaceFromKubernetesFlags returns the namespace combining the
// namespace flags from genericclioptions.ConfigFlags and the Allnamespace
// option in genericclioptions.ResourceBuilderFlags
func GetNamespaceFromKubernetesFlags(
	configFlag *genericclioptions.ConfigFlags,
	rbFlags *genericclioptions.ResourceBuilderFlags) string {

	namespace := "default"
	if configFlag.Namespace != nil && *configFlag.Namespace != "" {
		namespace = *configFlag.Namespace
	}
	if rbFlags.AllNamespaces != nil && *rbFlags.AllNamespaces {
		namespace = AllNamespace
	}
	return namespace
}

// GetSelectedPods returns all the pods with the profefe annotation enabled
// filtered by the selected labels
func GetSelectedPods(clientset kubernetes.Interface,
	namespace string,
	listOpt metav1.ListOptions) ([]v1.Pod, error) {

	target := []v1.Pod{}
	pods, err := clientset.CoreV1().Pods(namespace).List(listOpt)
	if err != nil {
		return target, err
	}
	for _, pod := range pods.Items {
		enabled, ok := pod.Annotations[ProfefeEnabledAnnotation]
		if ok && enabled == "true" && pod.Status.Phase == v1.PodRunning {
			target = append(target, pod)
		}
	}
	return target, nil
}

type PortForwardAPodRequest struct {
	RestConfig *rest.Config
	Pod        v1.Pod
	LocalPort  int
	Streams    genericclioptions.IOStreams
	StopCh     <-chan struct{}
	ReadyCh    chan struct{}
}

// PortForwardAPod forwards the port specificed from the profefe port
// annotation locally.
func PortForwardAPod(req PortForwardAPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.Pod.Namespace, req.Pod.Name)
	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return err
	}

	port := 6060
	if rawPort, ok := req.Pod.Annotations["profefe.com/port"]; ok && rawPort != "" {
		port, err = strconv.Atoi(rawPort)
		if err != nil {
			return err
		}
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	// TODO: port selection needs to be dynamic based on availability
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, port)}, req.StopCh, req.ReadyCh, req.Streams.Out, req.Streams.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}

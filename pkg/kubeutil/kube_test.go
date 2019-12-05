package kubeutil

import (
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetSelectedPods(t *testing.T) {
	t.Parallel()
	data := []struct {
		clientset         kubernetes.Interface
		countExpectedPods int
		inputNamespace    string
		listOpt           metav1.ListOptions
		err               error
	}{
		// Pods are in the system but they do not match the creteria
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "influxdb-v2",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace:    "default",
			countExpectedPods: 0,
		},
		// there are not pods in the default namespace with the right annotation and in status running
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "influxdb-v2",
					Namespace: "hola",
					Annotations: map[string]string{
						ProfefeEnabledAnnotation: "true",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "none",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace:    "default",
			countExpectedPods: 0,
		},
		// there is a pod in the default namespace with the right annotation and in status running
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "influxdb-v2",
					Namespace: "default",
					Annotations: map[string]string{
						ProfefeEnabledAnnotation: "true",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "none",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace:    "default",
			countExpectedPods: 1,
		},
	}

	for _, single := range data {
		t.Run("", func(single struct {
			clientset         kubernetes.Interface
			countExpectedPods int
			inputNamespace    string
			listOpt           metav1.ListOptions
			err               error
		}) func(t *testing.T) {
			return func(t *testing.T) {
				pods, err := GetSelectedPods(single.clientset, single.inputNamespace, single.listOpt)
				if err != nil {
					if single.err == nil {
						t.Fatalf(err.Error())
					}
					if !strings.EqualFold(single.err.Error(), err.Error()) {
						t.Fatalf("expected err: %s got err: %s", single.err, err)
					}
				} else {
					if len(pods) != single.countExpectedPods {
						t.Fatalf("expected %d pods, got %d", single.countExpectedPods, len(pods))
					}
				}
			}
		}(single))
	}
}

func TestGetPodByName(t *testing.T) {
	t.Parallel()
	data := []struct {
		clientset      kubernetes.Interface
		expectedPod    v1.Pod
		inputNamespace string
		inputName      string
		err            error
	}{
		// Pods are in the system but the one we requested does not exists
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "influxdb-v2",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace: "default",
			inputName:      "hello",
			err:            fmt.Errorf("pods \"hello\" not found"),
		},
		// The required pod is in the system but it does not have the right annotation
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "hello",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace: "default",
			inputName:      "hello",
			err:            fmt.Errorf("Pod not found or it does not have the right annotations"),
		},
		// The required pod is in the system with the right annotation but not in running state
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello",
					Namespace: "default",
					Annotations: map[string]string{
						ProfefeEnabledAnnotation: "true",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace: "default",
			err:            fmt.Errorf("Pod not found or it does not have the right annotations"),
			inputName:      "hello",
		},
		// The required pod is in the system in running state with the right annotation
		{
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hello",
					Namespace: "default",
					Annotations: map[string]string{
						ProfefeEnabledAnnotation: "true",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			}, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "chronograf",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			}),
			inputNamespace: "default",
			inputName:      "hello",
		},
	}

	for _, single := range data {
		t.Run("", func(single struct {
			clientset      kubernetes.Interface
			expectedPod    v1.Pod
			inputNamespace string
			inputName      string
			err            error
		}) func(t *testing.T) {
			return func(t *testing.T) {
				pod, err := GetPodByName(single.clientset, single.inputNamespace, single.inputName, metav1.GetOptions{})
				if err != nil {
					if single.err == nil {
						t.Fatalf(err.Error())
					}
					if !strings.EqualFold(single.err.Error(), err.Error()) {
						t.Fatalf("expected err: %s got err: %s", single.err, err)
					}
				} else {
					if _, ok := pod.Annotations[ProfefeEnabledAnnotation]; !ok {
						t.Errorf("profefe.com/enable annotation is not there but it is supposed to be part of the pod spec")
					}
				}
			}
		}(single))
	}
}

func TestGetNamespaceFromKubernetesFlags(t *testing.T) {
	t.Parallel()
	data := []struct {
		configFlags       *genericclioptions.ConfigFlags
		rbFlags           *genericclioptions.ResourceBuilderFlags
		expectedNamespace string
	}{
		{
			// An empty namespace returns default when all namespace is not set
			&genericclioptions.ConfigFlags{
				Namespace: func() *string {
					n := ""
					return &n
				}(),
			},
			&genericclioptions.ResourceBuilderFlags{},
			"default",
		},
		{
			&genericclioptions.ConfigFlags{},
			&genericclioptions.ResourceBuilderFlags{},
			"default",
		},
		{
			// An empty namespace returns an empty namespace when all namespace is true
			&genericclioptions.ConfigFlags{
				Namespace: func() *string {
					n := ""
					return &n
				}(),
			},
			&genericclioptions.ResourceBuilderFlags{
				AllNamespaces: func() *bool {
					t := true
					return &t
				}(),
			},
			"",
		},
		{
			&genericclioptions.ConfigFlags{
				Namespace: func() *string {
					n := "hello"
					return &n
				}(),
			},
			&genericclioptions.ResourceBuilderFlags{},
			"hello",
		},
	}

	for _, s := range data {
		t.Run("", func(single struct {
			configFlags       *genericclioptions.ConfigFlags
			rbFlags           *genericclioptions.ResourceBuilderFlags
			expectedNamespace string
		}) func(t *testing.T) {
			return func(t *testing.T) {
				n := GetNamespaceFromKubernetesFlags(s.configFlags, s.rbFlags)
				if n != s.expectedNamespace {
					t.Errorf("expected %s got %s", s.expectedNamespace, n)
				}
			}
		}(s))
	}
}

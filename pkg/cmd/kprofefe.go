package cmd

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gianarb/kube-profefe/pkg/kubeutil"
	"github.com/gianarb/kube-profefe/pkg/pprofutil"
	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewKProfefeCmd(logger *zap.Logger, streams genericclioptions.IOStreams) *cobra.Command {
	flags := pflag.NewFlagSet("kprofefe", pflag.ExitOnError)
	pflag.CommandLine = flags

	kubeConfigFlags := genericclioptions.NewConfigFlags(false)
	kubeResouceBuilderFlags := genericclioptions.NewResourceBuilderFlags()

	if ProfefeHostPort == "" {
		ProfefeHostPort = "http://localhost:10100"
	}

	cmd := &cobra.Command{
		Use:   "kprofefe",
		Short: "kprofefe collects profiles from inside a kubernetes cluster",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.SetOutput(streams.ErrOut)
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			logger = logger.With(
				zap.Strings("args", args),
				zap.String("profefe-hostport", ProfefeHostPort),
			)
			var config *rest.Config
			var err error

			config, err = rest.InClusterConfig()
			if err != nil {
				config, err = kubeConfigFlags.ToRESTConfig()
			}
			if err != nil {
				logger.Fatal("Impossible to retrive a kubernetes config", zap.Error(err))
			}
			if config == nil {
				logger.Fatal("Impossible to retrive a kubernetes config")
			}
			// creates the clientset
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				logger.Fatal("Kubernetes Client creation failed", zap.Error(err))
			}

			// Contains the pool of pods that we need to gather profiles from
			selectedPods := []corev1.Pod{}

			namespace := kubeutil.GetNamespaceFromKubernetesFlags(kubeConfigFlags, kubeResouceBuilderFlags)

			logger = logger.With(
				zap.String("namespace", namespace),
			)

			// If the arguments are more than zero we should check by pod name
			// (args == resourceName)
			if len(args) > 0 {
				for _, podName := range args {
					pod, err := kubeutil.GetPodByName(clientset, namespace, podName, metav1.GetOptions{})
					if err != nil {
						logger.Warn("Pod not found", zap.Error(err))
						continue
					}
					selectedPods = append(selectedPods, *pod)
				}
			} else {
				selectedPods, err = kubeutil.GetSelectedPods(clientset, namespace, metav1.ListOptions{
					LabelSelector: *kubeResouceBuilderFlags.LabelSelector,
				})
				if err != nil {
					logger.Fatal("Error retrieving list of pods from kubernetes api", zap.Error(err))
				}
			}

			// If the selectedPods are zero there is nothing to do.
			if len(selectedPods) == 0 {
				logger.Fatal("No pods to profile")
			}

			logger.Info("Starting to profile...", zap.Int("selected_pods_count", len(selectedPods)))

			pClient := profefe.NewClient(profefe.Config{
				HostPort: ProfefeHostPort,
			}, http.Client{})

			wg := sync.WaitGroup{}
			wg.Add(10)

			poolC := make(chan corev1.Pod)
			for ii := 0; ii < 10; ii++ {
				go func(c chan corev1.Pod) {
					for {
						pod, more := <-c
						if more == false {
							wg.Done()
							return
						}
						do(ctx, logger, pClient, pod)
					}
				}(poolC)

			}

			for _, target := range selectedPods {
				poolC <- target
			}

			close(poolC)
			wg.Wait()
			logger.Info("It is all done bye...")
		},
	}

	flags.AddFlagSet(cmd.PersistentFlags())
	flags.StringVar(&ProfefeHostPort, "profefe-hostport", "http://localhost:10100", `where profefe is located`)
	kubeConfigFlags.AddFlags(flags)
	kubeResouceBuilderFlags.WithLabelSelector("")
	kubeResouceBuilderFlags.WithAllNamespaces(false)
	kubeResouceBuilderFlags.AddFlags(flags)

	return cmd
}

func do(ctx context.Context, l *zap.Logger, pClient *profefe.Client, target corev1.Pod) {
	logger := l.With(zap.String("pod", target.Name))
	targetPort := pprofutil.GetProfefePortByPod(target)
	profiles, err := pprofutil.GatherAllByPod(context.Background(), fmt.Sprintf("http://%s", target.Status.PodIP), target, targetPort)
	if err != nil {
		panic(err)
	}
	for profileType, profile := range profiles {
		profefeType := profefe.NewProfileTypeFromString(profileType.String())
		if profefeType == profefe.UnknownProfile {
			logger.Warn("Unknown profile type it can not be sent to profefe. Skip this profile")
			continue
		}
		saved, err := pClient.SavePprof(context.Background(), profefe.SavePprofRequest{
			Profile:    profile,
			Service:    target.Name,
			InstanceID: target.Status.HostIP,
			Type:       profefeType,
			Labels: map[string]string{
				"namespace": target.Namespace,
				"from":      "kube-profefe",
			},
		})
		if err != nil {
			logger.Warn("Unknown profile type it can not be sent to profefe. Skip this profile", zap.Error(err))
		} else {
			logger.Info("Profile stored in profefe.", zap.String("id", saved.Body.ID), zap.String("profefe_profile_type", profefeType.String()), zap.String("url", fmt.Sprintf("%s/api/0/profiles/%s", ProfefeHostPort, saved.Body.ID)))
		}
	}
}

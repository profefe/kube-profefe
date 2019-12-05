package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gianarb/kube-profefe/pkg/kubeutil"
	"github.com/gianarb/kube-profefe/pkg/pprofutil"
	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewKProfefeCmd(streams genericclioptions.IOStreams) *cobra.Command {
	flags := pflag.NewFlagSet("kubectl-profefe", pflag.ExitOnError)
	pflag.CommandLine = flags

	kubeConfigFlags := genericclioptions.NewConfigFlags(false)
	kubeResouceBuilderFlags := genericclioptions.NewResourceBuilderFlags()

	cmd := &cobra.Command{
		Use:   "kprofefe",
		Short: "kprofefe collects profiles from inside a kubernetes cluster",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.SetOutput(streams.ErrOut)
		},
		Run: func(cmd *cobra.Command, args []string) {
			var config *rest.Config
			var err error

			config, err = rest.InClusterConfig()
			if err != nil {
				config, err = kubeConfigFlags.ToRESTConfig()
			}
			if err != nil {
				panic(err)
			}
			if config == nil {
				panic("woww")
			}
			// creates the clientset
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err.Error())
			}

			// Contains the pool of pods that we need to gather profiles from
			selectedPods := []corev1.Pod{}

			namespace := kubeutil.GetNamespaceFromKubernetesFlags(kubeConfigFlags, kubeResouceBuilderFlags)

			// If the arguments are more than zero we should check by pod name
			// (args == resourceName)
			if len(args) > 0 {
				for _, podName := range args {
					pod, err := kubeutil.GetPodByName(clientset, namespace, podName, metav1.GetOptions{})
					if err != nil {
						println(err.Error())
						continue
					}
					selectedPods = append(selectedPods, *pod)
				}
			} else {
				selectedPods, err = kubeutil.GetSelectedPods(clientset, namespace, metav1.ListOptions{
					LabelSelector: *kubeResouceBuilderFlags.LabelSelector,
				})
				if err != nil {
					println(err.Error())
					os.Exit(1)
				}
			}

			// If the selectedPods are zero there is nothing to do.
			if len(selectedPods) == 0 {
				println("there are not pod that matches your research")
				os.Exit(1)
			}

			println(fmt.Sprintf("selected %d pod/s", len(selectedPods)))

			pClient := profefe.NewClient(profefe.Config{
				HostPort: ProfefeHostPort,
			}, http.Client{})

			for _, target := range selectedPods {
				targetPort := pprofutil.GetProfefePortByPod(target)
				profiles, err := pprofutil.GatherAllByPod(context.Background(), fmt.Sprintf("http://%s", target.Status.PodIP), target, targetPort)
				if err != nil {
					panic(err)
				}
				for profileType, profile := range profiles {
					profefeType := profefe.NewProfileTypeFromString(profileType.String())
					if profefeType == profefe.UnknownProfile {
						println("unknown profile type: :" + profileType.String())
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
						println(fmt.Sprintf("%s type=%s profile_type=%s", err.Error(), profefeType, profile.PeriodType.Type))
					} else {
						println(fmt.Sprintf("%s/api/0/profiles/%s type=%s", ProfefeHostPort, saved.Body.ID, profefeType))
					}
				}
			}
		},
	}

	flags.AddFlagSet(cmd.PersistentFlags())
	flags.StringVar(&ProfefeHostPort, "profefe-hostport", "http://localhost:10100", `where profefe is located`)
	kubeConfigFlags.AddFlags(flags)
	kubeResouceBuilderFlags.WithLabelSelector("")
	kubeResouceBuilderFlags.AddFlags(flags)

	return cmd
}

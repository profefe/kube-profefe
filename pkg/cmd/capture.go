package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gianarb/kube-profefe/pkg/kubeutil"
	"github.com/gianarb/kube-profefe/pkg/pprofutil"
	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/google/pprof/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultForwardHost = "http://localhost"
)

var (
	OutputDir        string
	ProfefeHostPort  string
	ProfefeHostPortE string
)

func NewCaptureCmd(configFlag *genericclioptions.ConfigFlags, rbFlags *genericclioptions.ResourceBuilderFlags, streams genericclioptions.IOStreams) *cobra.Command {
	captureCmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture gathers profiles for a pod or a set of them. If can filter by namespace and via label selector.",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.SetOutput(streams.ErrOut)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			//TODO: Validate the argument. It is a list of podName.
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			config, err := configFlag.ToRESTConfig()
			if err != nil {
				println(err.Error())
				os.Exit(1)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				println(err.Error())
				os.Exit(1)
			}

			// Contains the pool of pods that we need to gather profiles from
			selectedPods := []corev1.Pod{}

			namespace := kubeutil.GetNamespaceFromKubernetesFlags(configFlag, rbFlags)

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
					LabelSelector: *rbFlags.LabelSelector,
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

			for _, target := range selectedPods {
				stopCh := make(chan struct{}, 1)
				readyCh := make(chan struct{})
				var berr, bout bytes.Buffer
				buffErr := bufio.NewWriter(&berr)
				buffOut := bufio.NewWriter(&bout)

				// TODO: make it smater
				localPort := 9000
				for ; localPort < 9009; localPort++ {
					ln, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
					if err != nil {
						continue
					}
					ln.Close()
					break
				}

				go func() {
					err = kubeutil.PortForwardAPod(kubeutil.PortForwardAPodRequest{
						RestConfig: config,
						Pod:        target,
						LocalPort:  localPort,
						Streams: genericclioptions.IOStreams{
							In:     os.Stdin,
							Out:    buffOut,
							ErrOut: buffErr,
						},
						StopCh:  stopCh,
						ReadyCh: readyCh,
					})
					if err != nil {
						panic(err)
					}
				}()

				select {
				case <-readyCh:
					break
				}

				println("gathering profiles for pod: " + target.Name)

				profiles, err := pprofutil.GatherAllByPod(context.Background(), DefaultForwardHost, target, localPort)
				if err != nil {
					panic(err)
				}

				var pClient *profefe.Client
				if ProfefeHostPortE != "" {
					pClient = profefe.NewClient(profefe.Config{
						HostPort: ProfefeHostPortE,
					}, http.Client{})
				}

				err = writeProfiles(context.Background(), pClient, profiles, target)
				if err != nil {
					panic(err)
				}

				close(stopCh)
				buffErr.Flush()
				buffOut.Flush()
			}
		},
	}
	flagsCapture := pflag.NewFlagSet("kubectl-profefe-capture", pflag.ExitOnError)
	flagsCapture.StringVar(&OutputDir, "output-dir", "/tmp", "Directory where to place the profiles")
	flagsCapture.StringVar(&ProfefeHostPortE, "profefe-hostport", "", `Where profefe
is (eg http://localhost:10100). If not set the profiles will be store in
your /tmp directory. When set the profiles will be only pushed in
profefe.`)
	captureCmd.Flags().AddFlagSet(flagsCapture)
	return captureCmd
}

func writeProfiles(ctx context.Context, pClient *profefe.Client, profiles map[pprofutil.Profile]*profile.Profile, target corev1.Pod) error {
	for profileType, profile := range profiles {
		profefeType := profefe.NewProfileTypeFromString(profileType.String())
		if profefeType == profefe.UnknownProfile {
			println("unknown profile type: :" + profile.PeriodType.Type)
			continue
		}
		// if the profefe client is not null the profile needs to be pushed to
		// profefe server, otherwise it is written into a file locally
		if pClient != nil {
			req := profefe.SavePprofRequest{
				Profile:    profile,
				Service:    target.Name,
				InstanceID: target.Status.HostIP,
				Type:       profefeType,
				Labels: map[string]string{
					"namespace": target.Namespace,
					"from":      "kube-profefe",
				},
			}
			if serviceName, ok := target.Annotations["profefe.com/service"]; ok && serviceName != "" {
				req.Service = serviceName
				req.Labels["pod"] = target.Name
			}
			saved, err := pClient.SavePprof(context.Background(), req)
			if err != nil {
				println(fmt.Sprintf("%s type=%s profile_type=%s", err.Error(), profefeType, profile.PeriodType.Type))
			} else {
				println(fmt.Sprintf("%s/api/0/profiles/%s type=%s", ProfefeHostPortE, saved.Body.ID, profefeType))
			}
		} else {
			f, err := os.OpenFile(
				fmt.Sprintf("%s/profile-%s-%s-%d.pb.gz", OutputDir, profileType, target.Name, time.Now().UTC().Unix()),
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0644)
			if err != nil {
				println(err.Error())
				continue
			}
			err = profile.Write(f)
			if err != nil {
				println(err.Error())
				continue
			}
		}
	}
	return nil
}

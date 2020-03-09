package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gianarb/kube-profefe/pkg/kubeutil"
	"github.com/gianarb/kube-profefe/pkg/pprofutil"
	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/google/pprof/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// totPodLenKey represents the number of pods that has to be scraped in this iteration
	totPodLenKey = key.New("profefe.com/count_pods")
	// totPerGoroutePodsLen represents the number of pods that a particualr goroutine handled
	totPerGoroutePodsLen = key.New("profefe.com/count_pods_per_goroutine")
	JaegerAddress        string
)

var Tracer string

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
		RunE: func(cmd *cobra.Command, args []string) error {
			if Tracer == "jaeger" {
				logger.Info("Used the tracer output jaeger")
				// Create Jaeger Exporter
				exporter, err := jaeger.NewExporter(
					jaeger.WithCollectorEndpoint(JaegerAddress),
					jaeger.WithProcess(jaeger.Process{
						ServiceName: "kprofefe",
					}),
				)
				if err != nil {
					log.Fatal(err)
				}

				// For demoing purposes, always sample. In a production application, you should
				// configure this to a trace.ProbabilitySampler set at the desired
				// probability.
				tp, err := sdktrace.NewProvider(
					sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
					sdktrace.WithSyncer(exporter))
				if err != nil {
					log.Fatal(err)
				}
				global.SetTraceProvider(tp)
				defer exporter.Flush()
			}

			tracer := global.TraceProvider().Tracer("kprofefe")

			ctx, span := tracer.Start(context.Background(), "kprofefe.RunE")
			defer span.End()

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
				logger.Fatal("Impossible to retrieve a kubernetes config", zap.Error(err))
			}
			if config == nil {
				logger.Fatal("Impossible to retrieve a kubernetes config")
			}
			// creates the clientset
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				logger.Error("Kubernetes Client creation failed", zap.Error(err))
				return err
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
					logger.Error("Error retrieving list of pods from kubernetes api", zap.Error(err))
					return err
				}
			}

			span.SetAttributes(totPodLenKey.Int(len(selectedPods)))

			// If the selectedPods are zero there is nothing to do.
			if len(selectedPods) == 0 {
				logger.Info("No pods to profile")
				return nil
			}

			logger.Info("Starting to profile...", zap.Int("selected_pods_count", len(selectedPods)))

			pClient := profefe.NewClient(profefe.Config{
				HostPort: ProfefeHostPort,
			}, http.Client{})

			wg := sync.WaitGroup{}
			wg.Add(10)

			poolC := make(chan corev1.Pod)
			for ii := 0; ii < 10; ii++ {
				go func(c chan corev1.Pod, ctx context.Context, ii int) {
					tracer.WithSpan(ctx, fmt.Sprintf("goroutine-%d", ii), func(ctx context.Context) error {
						nPod := 0
						for {
							pod, more := <-c
							if more == false {
								logger.Info("there are not pods to process. Closing goroutine...")
								wg.Done()
								return nil
							}
							nPod++
							trace.SpanFromContext(ctx).SetAttributes(totPerGoroutePodsLen.Int(nPod))
							ctx, cancel := context.WithTimeout(ctx, time.Second*40)
							defer cancel()
							do(ctx, logger, pClient, pod)
						}
					})
				}(poolC, ctx, ii)
			}

			for _, target := range selectedPods {
				poolC <- target
			}

			close(poolC)
			wg.Wait()
			logger.Info("It is all done bye...")
			return nil
		},
	}

	flags.AddFlagSet(cmd.PersistentFlags())
	flags.StringVar(&Tracer, "tracer", "dev", `where to send telemetry`)
	flags.StringVar(&ProfefeHostPort, "profefe-hostport", "http://localhost:10100", `where profefe is located`)
	flags.StringVar(&JaegerAddress, "--tracer.jaeger-address", "http://localhost:14268/api/traces", "Set the destionation for your traces")
	kubeConfigFlags.AddFlags(flags)
	kubeResouceBuilderFlags.WithLabelSelector("")
	kubeResouceBuilderFlags.WithAllNamespaces(false)
	kubeResouceBuilderFlags.AddFlags(flags)

	return cmd
}

func do(ctx context.Context, l *zap.Logger, pClient *profefe.Client, target corev1.Pod) {
	logger := l.With(zap.String("pod", target.Name))
	targetPort := pprofutil.GetProfefePortByPod(target)
	var profiles map[pprofutil.Profile]*profile.Profile
	var err error
	profiles, err = pprofutil.GatherAllByPod(ctx, logger, fmt.Sprintf("http://%s", target.Status.PodIP), target, targetPort)
	if err != nil {
		logger.Error("impossible to gather profiles", zap.Error(err))
		return
	}
	for profileType, profile := range profiles {
		profefeType := profefe.NewProfileTypeFromString(profileType.String())
		if profefeType == profefe.UnknownProfile {
			logger.Warn("Unknown profile type it can not be sent to profefe. Skip this profile", zap.String("profile_type", profileType.String()))
			continue
		}
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
			logger.Warn("Unknown profile type it can not be sent to profefe. Skip this profile", zap.Error(err))
		} else {
			logger.Info("Profile stored in profefe.", zap.String("id", saved.Body.ID), zap.String("profefe_profile_type", profefeType.String()), zap.String("url", fmt.Sprintf("%s/api/0/profiles/%s", ProfefeHostPort, saved.Body.ID)))
		}
	}
}

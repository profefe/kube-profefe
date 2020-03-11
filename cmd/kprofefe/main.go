package main

import (
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/gianarb/kube-profefe/pkg/cmd"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/trace/stdout"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	exporter, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)

	logger, _ := zap.NewDevelopment()

	rootCmd := cmd.NewKProfefeCmd(logger, genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	err = rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

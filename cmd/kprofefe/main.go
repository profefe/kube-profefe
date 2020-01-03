package main

import (
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/gianarb/kube-profefe/pkg/cmd"
	"go.uber.org/zap"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	logger, _ := zap.NewDevelopment()
	rootCmd := cmd.NewKProfefeCmd(logger, genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	rootCmd.Execute()
}

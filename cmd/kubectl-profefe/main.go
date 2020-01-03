package main

import (
	"os"

	"github.com/gianarb/kube-profefe/pkg/cmd"
	"go.uber.org/zap"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	logger, _ := zap.NewDevelopment()
	rootCmd := cmd.NewProfefeCmd(logger, genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	rootCmd.Execute()
}

package main

import (
	"os"

	"github.com/gianarb/kube-profefe/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	rootCmd := cmd.NewKProfefeCmd(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	rootCmd.Execute()
}

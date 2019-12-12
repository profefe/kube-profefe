package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ProfefeCmdOptions struct {
	configFlags *genericclioptions.ConfigFlags

	genericclioptions.IOStreams
}

func NewProfefeCmd(streams genericclioptions.IOStreams) *cobra.Command {
	flags := pflag.NewFlagSet("kubectl-profefe", pflag.ExitOnError)
	pflag.CommandLine = flags

	kubeConfigFlags := genericclioptions.NewConfigFlags(false)
	kubeResouceBuilderFlags := genericclioptions.NewResourceBuilderFlags()

	rootCmd := &cobra.Command{
		Use:   "kubectl-profefe",
		Short: "It is a kubectl plugin that you can use to retrieve and manage profiles in Go.",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.SetOutput(streams.ErrOut)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	flags.AddFlagSet(rootCmd.PersistentFlags())
	kubeConfigFlags.AddFlags(flags)
	kubeResouceBuilderFlags.WithLabelSelector("")
	kubeResouceBuilderFlags.WithAllNamespaces(false)
	kubeResouceBuilderFlags.AddFlags(flags)

	captureCmd := NewCaptureCmd(kubeConfigFlags, kubeResouceBuilderFlags, streams)
	flagsCapture := pflag.NewFlagSet("kubectl-profefe-capture", pflag.ExitOnError)
	flagsCapture.StringVar(&OutputDir, "output-dir", "/tmp", "Directory where to place the profiles")
	captureCmd.Flags().AddFlagSet(flagsCapture)
	rootCmd.AddCommand(captureCmd)
	rootCmd.AddCommand(NewGetCmd())

	return rootCmd
}

package cmd

import (
	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/spf13/cobra"
)

func NewGetProfileTypesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "profile-types",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, v := range profefe.AllProfileTypes() {
				println(v)
			}
			return nil
		},
	}
	return cmd
}

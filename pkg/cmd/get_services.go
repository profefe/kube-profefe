package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewGetServicesCmd() *cobra.Command {
	flags := pflag.NewFlagSet("kprofefe", pflag.ExitOnError)

	cmd := &cobra.Command{
		Use: "services",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pClient := profefe.NewClient(profefe.Config{
				HostPort: ProfefeHostPort,
			}, http.Client{})

			resp, err := pClient.GetServices(ctx)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), "Services:\n")
			for _, v := range resp.Body {
				fmt.Fprintf(cmd.OutOrStdout(), "\t %s\n", v)
			}
			return nil
		},
	}

	flags.AddFlagSet(cmd.PersistentFlags())
	flags.StringVar(&ProfefeHostPort, "profefe-hostport", "http://localhost:10100", `where profefe is located`)

	cmd.Flags().AddFlagSet(flags)

	return cmd
}

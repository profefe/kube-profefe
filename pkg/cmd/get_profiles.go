package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	service     string
	profileType string
	from        time.Duration
	to          time.Duration
)

func NewGetProfilesCmd() *cobra.Command {
	flags := pflag.NewFlagSet("kprofefe", pflag.ExitOnError)

	cmd := &cobra.Command{
		Use: "profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pClient := profefe.NewClient(profefe.Config{
				HostPort: ProfefeHostPort,
			}, http.Client{})

			req := profefe.GetProfilesRequest{}
			req.Service = service

			if profileType != "" {
				pt := profefe.NewProfileTypeFromString(profileType)
				if pt != profefe.UnknownProfile {
					req.Type = pt
				}
			}

			req.To = time.Now().UTC()
			req.From = req.To.Add(-from).UTC()

			resp, err := pClient.GetProfiles(ctx, req)
			if err != nil {
				return err
			}

			for _, v := range resp.Body {
				println(fmt.Sprintf("ID: %s Type: %s Service: %s CreateAt: %s", v.ID, v.Type, v.Service, v.CreatedAt.Format(time.RFC1123)))
			}
			return nil
		},
	}

	flags.AddFlagSet(cmd.PersistentFlags())
	flags.StringVar(&ProfefeHostPort, "profefe-hostport", "http://localhost:10100", `where profefe is located`)
	flags.StringVar(&profileType, "profile-type", "cpu", `The pprof profiles to retrieve`)
	flags.StringVar(&service, "service", "", ``)
	flags.DurationVar(&from, "from", 24*time.Hour, ``)
	flags.DurationVar(&to, "to", 0, ``)

	cmd.Flags().AddFlagSet(flags)

	return cmd
}

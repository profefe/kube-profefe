package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gianarb/kube-profefe/pkg/profefe"
	"github.com/google/pprof/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewLoadCmd() *cobra.Command {
	var serviceName string
	flags := pflag.NewFlagSet("load", pflag.ExitOnError)
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load a profile you have locally to profefe",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("You have to specify at least one profile as argument")
			}

			pClient := profefe.NewClient(profefe.Config{
				HostPort: ProfefeHostPort,
			}, http.Client{})

			for _, path := range args {
				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("Error (%s): %s", path, err)
				}
				p, err := profile.Parse(file)
				if err != nil {
					return fmt.Errorf("Error (%s): %s", path, err)
				}

				profefeType := profefe.NewProfileTypeFromString(p.PeriodType.Type)
				if profefeType == profefe.UnknownProfile {
					return fmt.Errorf("Error (%s) Unknown profile type it can not be sent to profefe. Skip this profile", path)
				}

				req := profefe.SavePprofRequest{
					Profile: p,
					Service: serviceName,
					InstanceID: func() string {
						h, err := os.Hostname()
						if err != nil {
							return "local"
						}
						return h
					}(),
					Type: profefeType,
				}
				saved, err := pClient.SavePprof(context.Background(), req)
				if err != nil {
					return fmt.Errorf("Error (%s): %s", err, path)
				} else {
					println(fmt.Sprintf("Profile (%s) stored in profefe: %s/api/0/profiles/%s", path, ProfefeHostPort, saved.Body.ID))
				}
			}
			return nil
		},
	}
	flags.StringVar(&ProfefeHostPort, "profefe-hostport", "http://localhost:10100", `where profefe is located`)
	flags.StringVar(&serviceName, "service", "", `The service name`)
	cmd.Flags().AddFlagSet(flags)
	return cmd
}

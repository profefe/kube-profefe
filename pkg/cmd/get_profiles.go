package cmd

import (
	"context"
	"errors"
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
	fromRaw     string
	toRaw       string

	ErrFromAheadOfTo = fmt.Errorf("From value ahead of to")
)

func fromStringToTime(now time.Time, raw string) (t time.Time, err error) {
	var d time.Duration
	// If raw is a duration substract it from now and return
	if d, err = time.ParseDuration(raw); err == nil {
		t = now.Add(d)
		return
	}
	// At this point is is not a duration so it has to be a valid RFC3339
	t, err = time.Parse(time.RFC3339, raw)
	return
}

func fromRawRangeToTime(now time.Time, fromRaw, toRaw string) (from, to time.Time, err error) {
	from, err = fromStringToTime(now, fromRaw)
	if err != nil {
		return
	}

	to, err = fromStringToTime(now, toRaw)
	if err != nil {
		return
	}
	if to.Sub(from) <= 0 {
		return time.Time{}, time.Time{}, ErrFromAheadOfTo
	}

	return
}
func NewGetProfilesCmd() *cobra.Command {
	var (
		from time.Time
		to   time.Time
	)

	flags := pflag.NewFlagSet("kprofefe", pflag.ExitOnError)
	cmd := &cobra.Command{
		Use: "profiles",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			now := time.Now()

			from, to, err = fromRawRangeToTime(now, fromRaw, toRaw)
			if err != nil {
				if errors.Is(err, ErrFromAheadOfTo) {
					return fmt.Errorf("from %s is ahead of to %s", from.Format(time.RFC3339), to.Format(time.RFC3339))
				}
				return err
			}

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

			req.From = from.UTC()
			req.To = to.UTC()

			fmt.Printf("FROM: %s - TO: %s\n", req.From.Format(time.RFC1123), req.To.Format(time.RFC1123))

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
	flags.StringVar(&fromRaw, "from", "24h", "Point in time for the first profile got gathered. From can be a duration or a RFC3339 (eg https://validator.w3.org/feed/docs/error/InvalidRFC3339Date.html)")
	flags.StringVar(&toRaw, "to", "0m", "Point in time for the first profile got gathered. From can be a duration or a RFC3339 (eg https://validator.w3.org/feed/docs/error/InvalidRFC3339Date.html)")

	cmd.Flags().AddFlagSet(flags)

	return cmd
}

package pprofutil

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/google/pprof/profile"
)

const (
	DefaultProfilePath = "/debug/pprof"
	DefaultProfilePort = 6060
)

func GatherAllByPod(ctx context.Context, logger *zap.Logger, host string, pod corev1.Pod, forwardedPort int) (map[Profile]*profile.Profile, error) {
	path := DefaultProfilePath
	if rawPath, ok := pod.Annotations["profefe.com/path"]; ok && rawPath != "" {
		path = rawPath
	}
	return GatherAll(ctx, logger, fmt.Sprintf("%s:%d%s", host, forwardedPort, path))
}

// GatherAll downloads all profile types from address.
func GatherAll(ctx context.Context, logger *zap.Logger, addr string) (map[Profile]*profile.Profile, error) {
	type res struct {
		prof        *profile.Profile
		profileType Profile
		err         error
	}
	profileTypes := Profiles()
	profiles := make(chan res, len(profileTypes))

	for _, p := range profileTypes {
		go func(ctx context.Context, addr string, p Profile) {
			prof, err := Gather(ctx, addr, p)
			if err != nil {
				logger.With(zap.String("profefe_profile_type", p.String())).With(zap.Error(err)).Error("Impossible to gather the profile")
			}
			profiles <- res{prof, p, err}
		}(ctx, addr, p)
	}

	profs := map[Profile]*profile.Profile{}
	for i := 0; i < len(profileTypes); i++ {
		p := <-profiles
		if p.prof != nil {
			profs[p.profileType] = p.prof
		}
	}

	close(profiles)
	return profs, nil
}

func GetProfefePortByPod(pod corev1.Pod) int {
	port := DefaultProfilePort
	if rawPort, ok := pod.Annotations["profefe.com/port"]; ok && rawPort != "" {
		if i, err := strconv.Atoi(rawPort); err == nil {
			port = i
		}
	}
	return port
}

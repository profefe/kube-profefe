package pprofutil

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/pprof/profile"
)

const (
	DefaultProfilePath = "/debug/pprof"
	DefaultProfilePort = 6060
)

func GatherAllByPod(ctx context.Context, host string, pod corev1.Pod, forwardedPort int) (map[Profile]*profile.Profile, error) {
	path := DefaultProfilePath
	if rawPath, ok := pod.Annotations["profefe.com/path"]; ok && rawPath != "" {
		path = rawPath
	}
	return GatherAll(ctx, fmt.Sprintf("%s:%d%s", host, forwardedPort, path))
}

// GatherAll downloads all profile types from address.
func GatherAll(ctx context.Context, addr string) (map[Profile]*profile.Profile, error) {
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
			profiles <- res{prof, p, err}
		}(ctx, addr, p)
	}

	var err error
	profs := map[Profile]*profile.Profile{}
	for i := 0; i < len(profileTypes); i++ {
		p := <-profiles
		if p.prof != nil {
			profs[p.profileType] = p.prof
		}
		if p.err != nil {
			err = p.err
		}
	}

	return profs, err
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

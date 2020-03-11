package pprofutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/google/pprof/profile"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/plugin/httptrace"
)

// Profile defines the types of profiles available.
type Profile int

// NewProfile parses the reader and returns a profile
func NewProfile(r io.Reader) (*profile.Profile, error) {
	return profile.Parse(r)
}

const (
	// Allocs returns a sampling of all past memory allocations for addr at
	// addr/debug/pprof/allocs.
	Allocs Profile = iota
	// Block are stack traces that led to blocking on sync primitives.
	Block
	// Goroutine are stack traces of all current goroutines.
	Goroutine
	// Heap is a sampling of memory allocations of live objects.
	Heap
	// Mutex are stack traces of holders of contended mutexes.
	Mutex
	//ThreadCreate are Stack traces that led to the creation of new OS threads.
	ThreadCreate
	// CPUProfile returns a CPU profile for addr at addr/debug/pprof/profile?seconds=seconds.
	CPUProfile
)

func (p Profile) String() string {
	switch p {
	case Allocs:
		return "allocs"
	case Block:
		return "block"
	case Goroutine:
		return "goroutine"
	case Heap:
		return "heap"
	case Mutex:
		return "mutex"
	case ThreadCreate:
		return "threadcreate"
	case CPUProfile:
		return "cpu"
	default:
		return ""
	}
}

// Profiles returns all the profile types possible to Gather.
func Profiles() []Profile {
	return []Profile{
		Allocs,
		Block,
		Goroutine,
		Heap,
		Mutex,
		ThreadCreate,
		CPUProfile,
	}
}

// Gather downloads a profile from the address.
func Gather(ctx context.Context, addr string, p Profile) (prof *profile.Profile, err error) {
	// TODO: give some way to give the profile duration
	if p == CPUProfile {
		prof, err = cpuprofile(ctx, addr, 30*time.Second)
	} else {
		path := fmt.Sprintf("/debug/pprof/%s", p)
		prof, err = gatherAt(ctx, addr, path)
	}

	if err != nil {
		return nil, err
	}

	// Well, this is to add a bit of context for the line protocol encoder.
	prof.Comments = append(prof.Comments,
		fmt.Sprintf("type=%s", p.String()),
		fmt.Sprintf("url=%s", addr),
	)
	return prof, err
}

func gatherAt(ctx context.Context, addr, path string) (*profile.Profile, error) {
	u, err := join(addr, path)
	if err != nil {
		return nil, err
	}

	return get(ctx, u)
}

// If seconds is 0 then it will default to 30s.
func cpuprofile(ctx context.Context, addr string, seconds time.Duration) (*profile.Profile, error) {
	u, err := join(addr, "/debug/pprof/profile")
	if err != nil {
		return nil, err
	}

	if seconds == 0 {
		seconds = 30 * time.Second
	}
	q := u.Query()
	q.Set("seconds", fmt.Sprintf("%.0f", seconds.Seconds()))
	u.RawQuery = q.Encode()
	return get(ctx, u)
}

func join(addr string, path string) (*url.URL, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	u.Path = path
	return u, nil
}

func get(ctx context.Context, u *url.URL) (*profile.Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	tr := global.TraceProvider().Tracer("pprof/client")

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])

	var p *profile.Profile

	err = tr.WithSpan(ctx, fmt.Sprintf("%s:%d %s", file, line, f.Name()), func(ctx context.Context) error {
		var err error
		ctx, req = httptrace.W3C(ctx, req)
		httptrace.Inject(ctx, req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()
		p, err = NewProfile(resp.Body)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

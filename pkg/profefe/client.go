package profefe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/pprof/profile"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/plugin/httptrace"
)

type ProfileType int8

const timeFormat = "2006-01-02T15:04:05"

const (
	UnknownProfile ProfileType = iota
	CPUProfile
	HeapProfile
	BlockProfile
	MutexProfile
	GoroutineProfile
	ThreadcreateProfile

	OtherProfile = 127
)

func AllProfileTypes() []string {
	return []string{
		UnknownProfile.String(),
		CPUProfile.String(),
		HeapProfile.String(),
		BlockProfile.String(),
		MutexProfile.String(),
		GoroutineProfile.String(),
		ThreadcreateProfile.String(),
	}
}

func (ptype ProfileType) String() string {
	switch ptype {
	case UnknownProfile:
		return "unknown"
	case CPUProfile:
		return "cpu"
	case HeapProfile:
		return "heap"
	case BlockProfile:
		return "block"
	case MutexProfile:
		return "mutex"
	case GoroutineProfile:
		return "goroutine"
	case ThreadcreateProfile:
		return "threadcreate"
	case OtherProfile:
		return "other"
	}
	return fmt.Sprintf("%d", ptype)
}

func NewProfileTypeFromString(s string) ProfileType {
	s = strings.TrimSpace(s)
	switch s {
	case "cpu":
		return CPUProfile
	case "heap":
		return HeapProfile
	case "block":
		return BlockProfile
	case "mutex":
		return MutexProfile
	case "goroutine":
		return GoroutineProfile
	case "threadcreate":
		return ThreadcreateProfile
	case "other":
		return OtherProfile
	default:
		return UnknownProfile
	}
}

type Config struct {
	HostPort  string
	UserAgent string
}

type Client struct {
	Config
	http.Client
}

const (
	ProfileTypeCPU       = "cpu"
	ProfileTypeHeap      = "heap"
	ProfileTypeBlock     = "block"
	ProfileTypeMutex     = "mutex"
	ProfileTypeGoroutine = "goroutine"
	ProfileTypeOther     = "other"
)

func GetProfileType() []string {
	return []string{
		ProfileTypeCPU,
		ProfileTypeHeap,
		ProfileTypeBlock,
		ProfileTypeMutex,
		ProfileTypeGoroutine,
		ProfileTypeOther,
	}
}

// GET
// /api/0/profiles?service=<service>&type=<type>from=<created_from>&to=<created_to>&labels=<key=value,key=value>
func (c *Client) GetProfiles(ctx context.Context, req GetProfilesRequest) (*GetProfilesResponse, error) {
	buf := bytes.NewBuffer([]byte{})
	r, err := http.NewRequestWithContext(ctx, "GET", c.HostPort+"/api/0/profiles", buf)

	q := r.URL.Query()
	q.Add("from", req.From.Format(timeFormat))
	q.Add("to", req.To.Format(timeFormat))
	q.Add("type", req.Type.String())
	q.Add("service", req.Service)
	labels := ""
	isLabels := false
	// Set labels as part of the profile and push them down to profefe
	for k, v := range req.Labels {
		isLabels = true
		labels = labels + "," + k + "=" + v
	}
	if isLabels {
		q.Add("labels", labels)
	}
	r.URL.RawQuery = q.Encode()

	resp, err := c.makeHTTPRequest(r)
	defer resp.Body.Close()
	rr := &GetProfilesResponse{}

	err = json.NewDecoder(resp.Body).Decode(rr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return rr, nil
	}
	return nil, fmt.Errorf(rr.Error)
}

type GetProfilesRequest struct {
	// service name
	Service string
	// cpu, heap, block, mutex, or goroutine
	Type ProfileType
	// a set of key-value pairs, e.g. "region=europe-west3,dc=fra,ip=1.2.3.4,version=1.0"
	Labels map[string]string

	From, To time.Time
}

type GetProfilesResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  []struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Service   string    `json:"service"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"body"`
}

// https://github.com/profefe/profefe#save-pprof-data
// POST /api/0/profiles?service=<service>&instance_id=<iid>&type=[cpu|heap]&labels=<key=value,key=value>
// body pprof.pb.gz
func (c *Client) SavePprof(ctx context.Context, req SavePprofRequest) (*SavePprofResponse, error) {
	buf := bytes.NewBuffer([]byte{})
	req.Profile.Write(buf)
	r, err := http.NewRequest("POST", c.HostPort+"/api/0/profiles", buf)
	if err != nil {
		return nil, err
	}

	q := r.URL.Query()
	q.Add("service", req.Service)
	q.Add("instance_id", req.InstanceID)
	q.Add("type", req.Type.String())
	labels := ""
	isLabels := false
	// Set labels as part of the profile and push them down to profefe
	for k, v := range req.Labels {
		isLabels = true
		labels = labels + "," + k + "=" + v
		req.Profile.SetLabel(k, []string{v})
	}
	if isLabels {
		q.Add("labels", labels)
	}
	r.URL.RawQuery = q.Encode()

	r.Header.Add("UserAgent", c.UserAgent)
	r.Header.Add("Content-Type", "application/octet-stream")

	resp, err := c.makeHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rr := &SavePprofResponse{}

	err = json.NewDecoder(resp.Body).Decode(rr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return rr, nil
	}
	return nil, fmt.Errorf(rr.Error)
}

type SavePprofRequest struct {
	Profile *profile.Profile
	// service name
	Service string
	// an identifier of running instance
	InstanceID string
	// cpu, heap, block, mutex, or goroutine
	Type ProfileType
	// a set of key-value pairs, e.g. "region=europe-west3,dc=fra,ip=1.2.3.4,version=1.0"
	Labels map[string]string
}

type SavePprofResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Service   string    `json:"service"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"body"`
}

// GET
// /api/0/services
func (c *Client) GetServices(ctx context.Context) (*GetServicesResponse, error) {
	buf := bytes.NewBuffer([]byte{})
	r, err := http.NewRequestWithContext(ctx, "GET", c.HostPort+"/api/0/services", buf)

	resp, err := c.makeHTTPRequest(r)
	defer resp.Body.Close()
	rr := &GetServicesResponse{}

	err = json.NewDecoder(resp.Body).Decode(rr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return rr, nil
	}
	return nil, fmt.Errorf(rr.Error)
}

type GetServicesResponse struct {
	Body  []string
	Error string `json:"error"`
}

func NewClient(config Config, httpClient http.Client) *Client {
	if config.HostPort == "" {
		config.HostPort = "http://localhost:10100"
	}
	if config.UserAgent == "" {
		config.UserAgent = "kubectl-profefe"
	}
	return &Client{config, httpClient}
}

func (c *Client) makeHTTPRequest(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	tr := global.TraceProvider().Tracer("pprof/client")

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])

	err := tr.WithSpan(r.Context(), fmt.Sprintf("%s:%d %s", file, line, f.Name()), func(ctx context.Context) error {
		var err error
		ctx, r = httptrace.W3C(ctx, r)
		httptrace.Inject(ctx, r)
		resp, err = c.Do(r)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, nil
}

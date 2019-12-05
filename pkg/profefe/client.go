package profefe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/pprof/profile"
)

type ProfileType int8

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

// https://github.com/profefe/profefe#save-pprof-data
// POST /api/0/profiles?service=<service>&instance_id=<iid>&type=[cpu|heap]&labels=<key=value,key=value>
// body pprof.pb.gz
func (c *Client) SavePprof(ctx context.Context, req SavePprofRequest) (*SavePprofResponse, error) {
	buf := bytes.NewBuffer([]byte{})
	req.Profile.Write(buf)
	labels := ""
	for k, v := range req.Labels {
		labels = labels + "," + k + "=" + v
	}
	r, err := http.NewRequest("POST", c.HostPort+"/api/0/profiles", buf)
	if err != nil {
		return nil, err
	}

	q := r.URL.Query()
	q.Add("service", req.Service)
	q.Add("instance_id", req.InstanceID)
	q.Add("type", req.Type.String())
	r.URL.RawQuery = q.Encode()

	r.Header.Add("UserAgent", c.UserAgent)
	r.Header.Add("Content-Type", "application/octet-stream")

	resp, err := c.Do(r)
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

func NewClient(config Config, httpClient http.Client) *Client {
	if config.HostPort == "" {
		config.HostPort = "http://localhost:10100"
	}
	if config.UserAgent == "" {
		config.UserAgent = "kubectl-profefe"
	}
	return &Client{config, httpClient}
}

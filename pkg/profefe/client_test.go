package profefe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestSavePprof(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "profefe/profefe:git-10551f2",
		ExposedPorts: []string{"10100/tcp"},
		WaitingFor:   wait.ForLog("server is running"),
	}
	nginxC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}
	defer nginxC.Terminate(ctx)
	ip, err := nginxC.Host(ctx)
	if err != nil {
		t.Error(err)
	}
	port, err := nginxC.MappedPort(ctx, "10100")
	if err != nil {
		t.Error(err)
	}

	client := NewClient(Config{
		HostPort:  fmt.Sprintf("http://%s:%d", ip, port.Int()),
		UserAgent: "testcontaners",
	}, http.Client{})

	f, err := os.Open("../../test/pprof.profefe.samples.cpu.001.pb.gz")
	if err != nil {
		t.Fatal(err)
	}
	p, err := profile.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	funcName := runtime.FuncForPC(reflect.ValueOf(TestSavePprof).Pointer()).Name()

	resp, err := client.SavePprof(ctx, SavePprofRequest{
		Profile:    p,
		Service:    funcName,
		InstanceID: funcName,
		Type:       CPUProfile,
		Labels: map[string]string{
			"hello": "dude",
		}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Body.Service != funcName {
		t.Errorf("expected serviec name %s got %s", funcName, resp.Body.Service)
	}
}

func TestGetServices(t *testing.T) {
	ctx := context.Background()

	pprofServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{
  "code": 200,
  "body": [
    "first",
    "second"
  ]
}`))
	}))

	client := NewClient(Config{
		HostPort:  pprofServer.URL,
		UserAgent: "test",
	}, *pprofServer.Client())

	rr, err := client.GetServices(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rr.Body[0] != "first" {
		t.Errorf("expect first got %s", rr.Body[0])
	}
}

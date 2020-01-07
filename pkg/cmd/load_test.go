package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestLoadProfileTest(t *testing.T) {
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

	cmd := NewLoadCmd()
	cmd.SetArgs([]string{
		"--profefe-hostport",
		fmt.Sprintf("http://%s:%d", ip, port.Int()),
		"--service",
		"test",
		"../../test/pprof.profefe.samples.cpu.001.pb.gz",
	})
	err = cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}
}

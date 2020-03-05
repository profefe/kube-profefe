package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetServices(t *testing.T) {
	pprofServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{
  "code": 200,
  "body": [
    "first",
    "second"
  ]
}`))
	}))

	buf := new(bytes.Buffer)
	cmd := NewGetServicesCmd()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--profefe-hostport",
		pprofServer.URL,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}
	exp := `Services:
     first
     second`

	out := buf.String()
	if strings.Contains(out, exp) {
		t.Errorf(`expected "%s" got "%s"`, exp, out)
	}
}

//Services:
//first
//second

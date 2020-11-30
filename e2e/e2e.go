package e2e

import (
	"io/ioutil"
	"os"
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
)

/* Contains utilities to:
 * 1. bootstrap a kind cluster and to stop it
 * 2. deploy profefe
 * 3. Deploy some other application/s for testing
 * 5. Deploying kprofefe
 */

func CreateCluster(t *testing.T, name string) (*kubernetes.Clientset, func() error) {
	provider := cluster.NewProvider()
	if err := provider.Create(name); err != nil {
		t.Error(err)
		return nil, func() error { return nil }
	}

	tmpfile, err := ioutil.TempFile("", name)
	if err != nil {
		t.Error(err)
		return nil, func() error { return nil }
	}
	t.Log(tmpfile.Name())

	config, err := clientcmd.BuildConfigFromFlags("", tmpfile.Name())
	if err != nil {
		t.Error(err)
		return nil, func() error { return nil }
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, func() error { return nil }
	}

	return clientset, func() error {
		if err := provider.Delete(name, tmpfile.Name()); err != nil {
			return err
		}
		return os.Remove(tmpfile.Name())
	}
}

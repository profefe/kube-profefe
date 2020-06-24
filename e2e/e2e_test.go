package e2e

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateCluster(t *testing.T) {
	clientset, shutdown := CreateCluster(t, "random")
	time.Sleep(10 * time.Second)
	if err := shutdown(); err != nil {
		t.Fatal(err)
	}
	n, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal(n.Kind)
}

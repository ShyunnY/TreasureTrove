package kube

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var cli, _ = NewClient(ClientConfig{
	Host:       "",
	Port:       "",
	KubeConfig: "D:\\treasure\\istio\\01_istio_init\\webhook\\kubeconfig",
})

func TestClientConn(t *testing.T) {

	cli, err := NewClient(ClientConfig{
		Host:       "",
		Port:       "",
		KubeConfig: "D:\\treasure\\istio\\01_istio_init\\webhook\\kubeconfig",
	})
	assert.NoError(t, err)
	assert.NotNil(t, cli)

	cs, err := cli.ClientSet()
	assert.NoError(t, err)
	assert.NotNil(t, cs)

}

func TestGetConfigMap(t *testing.T) {

	cliset, err := cli.ClientSet()
	assert.NoError(t, err)

	list, err := cliset.CoreV1().ConfigMaps(corev1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)

	for _, cm := range list.Items {
		fmt.Printf("cm_name: %+v\n", cm.Name)
	}

}

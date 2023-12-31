package webhooks

import (
	"fishnet-inject/kube"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"testing"
)

func TestCmInformer(t *testing.T) {

	client, err := kube.NewClient(kube.ClientConfig{})
	assert.NoError(t, err)

	cliset, err := client.ClientSet()
	assert.NoError(t, err)

	informer := NewConfigMapInformer(
		cliset,
		corev1.NamespaceDefault,
		func(cm *corev1.ConfigMap) {
			assert.NotNil(t, cm)
			assert.Equal(t, InjectorConfigMapKey, cm.Name, "configmap not match")
		},
	)

	informer.Run(wait.NeverStop)
}

package webhooks

import (
	"fishnet-inject/kube"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"testing"
)

func TestWatcherRun(t *testing.T) {

	client, err := kube.NewClient(kube.ClientConfig{})
	assert.NoError(t, err)

	cliset, err := client.ClientSet()
	assert.NoError(t, err)

	configKey := "inject_config"
	watcher := NewConfigMapWatch(
		cliset,
		"configmap watcher",
		corev1.NamespaceDefault,
		configKey,
	)
	assert.NotNil(t, watcher)

	watcher.SetCallback(func(config *Config) error {

		assert.NotNil(t, config)
		return nil
	})

	watcher.Run(wait.NeverStop)

}

func TestReadConfigMap(t *testing.T) {

	var configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: fishnet-config
data: 
  inject_config: |
    valueConfig:
      injectProbe: false
      labels:
        abc: efg
`
	cm := &corev1.ConfigMap{}
	err := yaml.Unmarshal([]byte(configMap), cm)
	assert.NoError(t, err)

	// success
	configKey := "inject_config"
	config, err := readConfigMap(cm, configKey)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// configKey not found
	configKey = "sidecar_config"
	config, err = readConfigMap(cm, configMap)
	assert.Error(t, err)
	assert.Nil(t, config)

}

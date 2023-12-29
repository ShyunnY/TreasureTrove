package webhooks

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
)

// Watcher
// 调用configMap informer进行获取cm
// 通过回调函数, 当配置更新后, 将新配置传入回调函数中
type Watcher interface {
	SetCallback(func(*Config) error)

	Run(stop <-chan struct{})
}

type ConfigMapCallback func(*corev1.ConfigMap)

type ConfigMapWatcher struct {
	callbacks         ConfigMapCallback
	handler           func(*Config) error
	name              string
	namespace         string
	configKey         string
	configMapInformer *ConfigMapInformer
}

// SetCallback
// 必须要在Run(stop <-chan struct{})启动之前设置handler
func (c *ConfigMapWatcher) SetCallback(handler func(*Config) error) {
	c.handler = handler
}

func NewConfigMapWatch(client *kubernetes.Clientset, name, namespace, configKey string) Watcher {

	w := &ConfigMapWatcher{
		name:      name,
		namespace: namespace,
		configKey: configKey,
	}

	w.configMapInformer = NewConfigMapInformer(client, namespace, func(configMap *corev1.ConfigMap) {

		fishnetConfig, err := readConfigMap(configMap, configKey)
		if err != nil {
			log.Println("error: ", err)
			// TODO: 我们需要使用默认值??
		}

		if w.handler != nil {
			if err := w.handler(fishnetConfig); err != nil {
				log.Println(err)
			}
		}

	})

	return w
}

func readConfigMap(configMap *corev1.ConfigMap, configKey string) (*Config, error) {

	// 1.读取configMap中配置
	config, exist := configMap.Data[configKey]
	if !exist {
		return nil, fmt.Errorf("configmap not found data, key: %s", configKey)
	}

	// 2.反序列化configMap的配置
	injectConfig, err := unmarshalConfig([]byte(config))
	if err != nil {
		return nil, err
	}
	return injectConfig, nil
}

func (c *ConfigMapWatcher) Run(stop <-chan struct{}) {

	c.configMapInformer.Run(stop)

	log.Println("configmap watcher closed")
}

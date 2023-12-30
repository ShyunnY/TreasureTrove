package webhooks

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"log"
)

// Watcher
// 调用configMap informer进行获取cm
// 通过回调函数, 当配置更新后, 将新配置传入回调函数中
type Watcher interface {
	SetCallback(func(*Config) error)

	Run(stop <-chan struct{})

	Get() (*Config, error)
}

type ConfigMapCallback func(*corev1.ConfigMap)

type ConfigMapWatcher struct {
	callbacks         ConfigMapCallback
	handler           func(*Config) error
	name              string
	namespace         string
	configKey         string
	configMapInformer *ConfigMapInformer

	cliset *kubernetes.Clientset
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
		cliset:    client,
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

func (c *ConfigMapWatcher) Get() (*Config, error) {

	configMap, err := c.cliset.
		CoreV1().
		ConfigMaps(c.namespace).
		Get(context.TODO(), InjectorConfigMapKey, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(
				schema.ParseGroupResource("configmaps"),
				InjectorConfigMapKey,
			)
		} else {
			return nil, err
		}
	}

	return readConfigMap(configMap, c.configKey)
}

func (c *ConfigMapWatcher) Run(stop <-chan struct{}) {
	log.Println("configmap watcher start")
	c.configMapInformer.Run(stop)

	log.Println("configmap watcher closed")
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

package webhooks

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"time"
)

var (
	notResync time.Duration = 0
)

// ConfigMapInformer
// 使用informer监视对应configmap
// add/update, 将获取的cm返回
// delete, 重建上一次的配置
type ConfigMapInformer struct {
	informer cache.SharedIndexInformer
}

func NewConfigMapInformer(client *kubernetes.Clientset, namespace string, callbacks func(*corev1.ConfigMap)) *ConfigMapInformer {

	ci := &ConfigMapInformer{}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		client,
		notResync,
		informers.WithNamespace(namespace),
	)
	configMapInformer := informerFactory.Core().V1().ConfigMaps().Informer()
	configMapInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			// 我们仅对指定configMap感兴趣
			if cm, ok := obj.(*corev1.ConfigMap); !ok {
				log.Println("this is no configmap!")
				return false
			} else if cm.Name != InjectorConfigMapKey {
				return false
			}
			return true
		},
		Handler: cache.ResourceEventHandlerFuncs{
			// TODO: 后续我们也许会对add,update进行区分, 目前接收到新增/更新我们都对配置进行重建
			AddFunc: func(obj interface{}) {
				// filter已经帮我们进行过滤了, 所以我们直接转换成configmap
				cm := obj.(*corev1.ConfigMap)
				callbacks(cm)
			},
			UpdateFunc: func(_, obj interface{}) {
				cm := obj.(*corev1.ConfigMap)
				callbacks(cm)
			},

			// TODO: 当fishnet-cm被删除后, 我们应该进行重建
			DeleteFunc: nil,
		},
	})

	ci.informer = configMapInformer

	return ci
}

func (cmi *ConfigMapInformer) Run(stop <-chan struct{}) {
	// 使用空struct作为占位符, 内存仅为1byte

	log.Println("configmapInformer start watch configmap")
	cmi.informer.Run(stop)

}

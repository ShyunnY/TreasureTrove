package webhooks

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"webhook/kube"
)

// ConfigMapInformer
// TODO: 1.使用informer监视对应configmap
// TODO: 2. add/update, 将获取的cm返回
// TODO: 3. delete, 返回默认的配置, 同时创建新的cm(默认配置)
// TODO: 4. 进行有条件的过滤
// TODO: 抽象一个队列出来, 外部从队列中获取配置
type ConfigMapInformer struct {
}

func cm() {
	client, _ := kube.NewClient(kube.ClientConfig{})
	cliset, _ := client.ClientSet()
	factory := informers.NewSharedInformerFactoryWithOptions(
		cliset,
		0, // 我们不需要重新同步所有资源
	)

	cmInformer := factory.Core().V1().ConfigMaps().Informer()
	_, err := cmInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			cm, ok := obj.(*corev1.ConfigMap)
			if !ok {
				return false
			}

			return cm.Name == "fishnet-cm"
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cm, ok := obj.(*corev1.ConfigMap)
				if !ok {
					panic("no configmap!")
				}
				fmt.Printf("add cm name: %+v\n", cm.Name)
			},
			UpdateFunc: func(_, obj interface{}) {
				cm, ok := obj.(*corev1.ConfigMap)
				if !ok {
					panic("no configmap!")
				}
				fmt.Printf("update cm name: %+v\n", cm.Name)
			},
		},
	})
	if err != nil {
		panic(err)
	}

	cmInformer.Run(wait.NeverStop)

}

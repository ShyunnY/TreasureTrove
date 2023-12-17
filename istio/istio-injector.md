## Istio Injector分析

Istio可以配置成自动向Pod注入sidecar容器和init容器, 使Pod加入ServiceMesh中.

其中Istio使用Kubernetes的MutatingWebhook进行自动注入. 



我们来看一下Istio中关于自动注入的Injector Webhook源码



**NewWebhook**主要功能:

+ 创建webhook实例
+ 获取sidecar注入相关配置
+ 创建一个Multicast广播(可以在一个Watcher上执行多个回调handler)
+ 添加一个监听配置(sidecarConfig,valuesConfig,meshConfig)变更的handler
+ 注册webhook的端点

```go
func NewWebhook(p WebhookParameters) (*Webhook, error) {
	...
    
	// 1. 创建一个webhook
	wh := &Webhook{
		watcher:    p.Watcher,
		meshConfig: p.Env.Mesh(),
		env:        p.Env,
		revision:   p.Revision,
	}
	
    ...
    
	// TODO 2. 创建一个广播
	mc := NewMulticast(p.Watcher, wh.GetConfig)

	// TODO 3. 添加一个配置变更时进行处理的handler
	mc.AddHandler(wh.updateConfig)
	
	wh.MultiCast = mc

	// TODO 4. 获取要注入的sidecar和values配置
	sidecarConfig, valuesConfig, err := p.Watcher.Get()
	if err != nil {
		return nil, err
	}

	// 5. 将sidecar配置和valuesConfig设置到webhook
	if err := wh.updateConfig(sidecarConfig, valuesConfig); err != nil {
		log.Errorf("failed to process webhook config: %v", err)
	}

	// 6. 注册自动sidecar注入的MutatingWebhook的端点
	p.Mux.HandleFunc("/inject", wh.serveInject)
	p.Mux.HandleFunc("/inject/", wh.serveInject)

	// TODO 7. 注册一个meshConfig修改时的回调handler
	p.Env.Watcher.AddMeshHandler(func() {
		wh.mu.Lock()
        // 将新的meshConfig设置到webhook的meshConfig上
		wh.meshConfig = p.Env.Mesh()
		wh.mu.Unlock()
	})

	return wh, nil
}
```




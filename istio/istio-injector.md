

## Istio Debug

我们在后续的Istio源码阅读之旅中, 会阅读大量的源码. 我们可以在源码中看到执行逻辑, 但是有些上下文参数对象我们却看不到. 这个时候就需要我们能够有一种方式通过IDE进行远程调试.

Go提供了**Devle**工具可以让我们进行*remote debug*, 即istiod运行在k8s集群中我们依旧可以进行打断点的方式进行debug. 这方便了我们阅读源码, 同时让我们可以更好的看到代码中上下文环境的值.



首先我们拉取istio repository

```shell
$ git init
$ git pull https://github.com/istio/istio.git
```



随后我们对源码进行编译. 这将会把istio源码构建成一个可调试的镜像

```shell
$ make DEBUG=1 docker.pilot
```

> 最好在一台有加速器的主机上操作, 这过程会拉取gcr.io仓库的镜像, 有可能会导致超时
>
> 这里有个坑, 我第一次直接下载代码源文件, 然后执行make命令, 这会导致缺失tag的错误信息. 这个tag实际上会查找git revision的值. 所以我们通过git拉取代码再进行make构建

此时我们可以基于make构建出来的调试镜像, 构建一个含有**Devle**工具的镜像

这里我就简单的将Devle二进制文件拷贝进容器

```dockerfile
FROM registry.cn-guangzhou.aliyuncs.com/shyunn/istio-debug:1.20

WORKDIR /usr/local/bin/

COPY . .

EXPOSE 40000

ENTRYPOINT ["/usr/local/bin/dlv","--continue", "--accept-multiclient","--listen=:40000", "--check-go-version=false","--headless=true", "--api-version=2","--log=true", "--log-output=debugger,debuglineerr,gdbwire,lldbout,rpc","exec", "/usr/local/bin/pilot-discovery", "--"]
```

我们可以将镜像推送到私服中, 然后修改istiod Deployment资源文件

```shell
$ kubectl edit deployment istiod -n istio-system

# image修改为我们刚刚构建出来的镜像, 然后添加安全上下文配置, 让delve可以进行执行
# yaml
...
image: xxx  # 修改为自己构建的镜像
securityContext:
  allowPrivilegeEscalation: true
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: false
  runAsGroup: 1337
  runAsNonRoot: true
  runAsUser: 1337
...
```

最后我们将端口进行转发, 就可以在goland中配置remote debug了

```shell
$ kubectl port-forward -n istio-system --address localhost deployment/istiod 40000:40000
```





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




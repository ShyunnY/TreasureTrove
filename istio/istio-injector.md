

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





## Istio-iptables

这个组件是Istio进行注入时, 在initContainer中使用组件, 主要是用于对iptables路由表的更改

我们首先进入源码`/istio/istio-master/pilot/cmd/pilot-agent`目录对`main.go`进行编译, 编译出pilot-agent命令行工具.

此时可以直接在主机上进行测试, pilot-agent的istio-iptables实际上就是使用了iptables, 对Pod内的iptables nat表进行修改, 增加了不同的路由规则将入站流量重定向到15006端口, 出站流量重定向15001端口. 所有流量都交由Sidecar(Envoy)进行处理.



我们来看一下pilot-agent istio-iptables的flag参数:

```shell
istio-iptables is responsible for setting up port forwarding for Istio Sidecar.

Usage:
  pilot-agent istio-iptables [flags]

Flags:
      --capture-all-dns                             Instead of only capturing DNS traffic to DNS server IP, capture all DNS traffic at port 53. This setting is only effective when redirect dns is enabled.
      --cni-mode                                    Whether to run as CNI plugin.
      --drop-invalid                                Enable invalid drop in the iptables rules.
  -n, --dry-run                                     Do not call any external dependencies like iptables.
      --dual-stack                                  Enable ipv4/ipv6 redirects for dual-stack.
  -p, --envoy-port string                           Specify the envoy port to which redirect all TCP traffic. (default "15001")
  -h, --help                                        help for istio-iptables
  -z, --inbound-capture-port string                 Port to which all inbound TCP traffic to the pod/VM should be redirected to. (default "15006")
  -e, --inbound-tunnel-port string                  Specify the istio tunnel port for inbound tcp traffic. (default "15008")
      --iptables-probe-port uint16                  Set listen port for failure detection. (default 15002)
      --iptables-trace-logging                      Insert tracing logs for each iptables rules, using the LOG chain.
      --iptables-version string                     version of iptables command. If not set, this is automatically detected.
  -c, --istio-exclude-interfaces string             Comma separated list of NIC (optional). Neither inbound nor outbound traffic will be captured.
  -m, --istio-inbound-interception-mode string      The mode used to redirect inbound connections to Envoy, either "REDIRECT" or "TPROXY".
  -b, --istio-inbound-ports string                  Comma separated list of inbound ports for which traffic is to be redirected to Envoy (optional). The wildcard character "*" can be used to configure redirection for all ports. An empty list will disable.
  -t, --istio-inbound-tproxy-mark string             (default "1337")
  -r, --istio-inbound-tproxy-route-table string      (default "133")
  -d, --istio-local-exclude-ports string            Comma separated list of inbound ports to be excluded from redirection to Envoy (optional). Only applies when all inbound traffic (i.e. "*") is being redirected.
  -o, --istio-local-outbound-ports-exclude string   Comma separated list of outbound ports to be excluded from redirection to Envoy.
  -q, --istio-outbound-ports string                 Comma separated list of outbound ports to be explicitly included for redirection to Envoy.
  -i, --istio-service-cidr string                   Comma separated list of IP ranges in CIDR form to redirect to envoy (optional). The wildcard character "*" can be used to redirect all outbound traffic. An empty list will disable all outbound.
  -x, --istio-service-exclude-cidr string           Comma separated list of IP ranges in CIDR form to be excluded from redirection. Only applies when all  outbound traffic (i.e. "*") is being redirected.
  -k, --kube-virt-interfaces string                 Comma separated list of virtual interfaces whose inbound traffic (from VM) will be treated as outbound.
      --network-namespace string                    The network namespace that iptables rules should be applied to.
      --probe-timeout duration                      Failure detection timeout. (default 5s)
  -g, --proxy-gid string                            Specify the GID of the user for which the redirection is not applied (same default value as -u param).
  -u, --proxy-uid string                            Specify the UID of the user for which the redirection is not applied. Typically, this is the UID of the proxy container.
      --redirect-dns                                Enable capture of dns traffic by istio-agent.
  -f, --restore-format                              Print iptables rules in iptables-restore interpretable format. (default true)
      --run-validation                              Validate iptables.
      --skip-rule-apply                             Skip iptables apply.

Global Flags:
      --log_as_json                   Whether to format output as JSON or in plain console-friendly format
      --log_caller string             Comma-separated list of scopes for which to include caller information, scopes can be any of [ads, adsc, all, authn, authorization, ca, cache, citadelclient, controllers, default, delta, dns, gateway, gcecred, googleca, googlecas, grpcgen, healthcheck, ingress status, iptables, klog, kube, mockcred, model, monitoring, retry, sds, security, serviceentry, spiffe, status, stsclient, stsserver, token, trustBundle, validation, wasm, wle, xdsproxy]
      --log_output_level string       Comma-separated minimum per-scope logging level of messages to output, in the form of <scope>:<level>,<scope>:<level>,... where scope can be one of [ads, adsc, all, authn, authorization, ca, cache, citadelclient, controllers, default, delta, dns, gateway, gcecred, googleca, googlecas, grpcgen, healthcheck, ingress status, iptables, klog, kube, mockcred, model, monitoring, retry, sds, security, serviceentry, spiffe, status, stsclient, stsserver, token, trustBundle, validation, wasm, wle, xdsproxy] and level can be one of [debug, info, warn, error, fatal, none] (default "default:info")
      --log_rotate string             The path for the optional rotating log file
      --log_rotate_max_age int        The maximum age in days of a log file beyond which the file is rotated (0 indicates no limit) (default 30)
      --log_rotate_max_backups int    The maximum number of log file backups to keep before older files are deleted (0 indicates no limit) (default 1000)
      --log_rotate_max_size int       The maximum size in megabytes of a log file beyond which the file is rotated (default 104857600)
      --log_stacktrace_level string   Comma-separated minimum per-scope logging level at which stack traces are captured, in the form of <scope>:<level>,<scope:level>,... where scope can be one of [ads, adsc, all, authn, authorization, ca, cache, citadelclient, controllers, default, delta, dns, gateway, gcecred, googleca, googlecas, grpcgen, healthcheck, ingress status, iptables, klog, kube, mockcred, model, monitoring, retry, sds, security, serviceentry, spiffe, status, stsclient, stsserver, token, trustBundle, validation, wasm, wle, xdsproxy] and level can be one of [debug, info, warn, error, fatal, none] (default "default:none")
      --log_target stringArray        The set of paths where to output the log. This can be any path as well as the special values stdout and stderr (default [stdout])
      --vklog Level                   number for the log level verbosity. Like -v flag. ex: --vklog=9
```

看到上面的参数, 是不是感觉头都大了. 我们掌握几个常用的参数是如何使用

```shell
istio-iptables is responsible for setting up port forwarding for Istio Sidecar.

Usage:
  pilot-agent istio-iptables [flags]

Flags:
--capture-all-dns  # 不是仅捕获到DNS服务器IP的DNS流量, 而是捕获端口53处的所有DNS流量. 此设置仅在启用重定向DNS时有效
--cni-mode  # 是否作为CNI插件进行运行
--drop-invalid  # 在iptables开启无效drop
-n, --dry-run   # 不要调用任何外部依赖项，例如iptables
--dual-stack    # 开启ipv4/ipv6双栈重定向
-p, --envoy-port string  # 指定将所有TCP流量重定向到envoy的端口, 例如出站流量. 默认为 15001
-z, --inbound-capture-port string  # 指定将所有入站的TCP流量重定向到envoy的端口. 默认为 15006
-e, --inbound-tunnel-port string   # 指定入站TCP流量的隧道端口. 默认为 15008
--iptables-probe-port uint16       # 故障检测的端口. 默认为 15002 (Probe探针配置)
--iptables-trace-logging  # 使用Log规则在每个istio设置的iptables规则上插入跟踪日志. (尽量不要开启, 这会导致cpu和存储的额外开销, 调试阶段可以开启)
--iptables-version string  # iptables的版本. 默认会自动进行检测
-c, --istio-exclude-interfaces string   # NIC列表, 入站和出站流量都不会对其进行捕获.
-m, --istio-inbound-interception-mode string # 入站流量重定向到Envoy的模式, 可选"REDIRECT/TPROXY". 默认使用TPROXY
-b, --istio-inbound-ports string  # 将入站流量重定向到Envoy的网络端口(就是访问哪些端口的流量将会被重定向). 使用"*"匹配所有端口, 空列表代表禁用. (默认是空列表)
-t, --istio-inbound-tproxy-mark string # 默认为1337
-r, --istio-inbound-tproxy-route-table string # 默认为133
-d, --istio-local-exclude-ports string  # 重定向到Envoy的入站流量排除的端口.(就是入站流量访问哪些端口将不会被重定向). 仅当所有入站端口("*")被重定向时才适用.
-o, --istio-local-outbound-ports-exclude string  # # 重定向到Envoy的出站流量排除的端口.(就是出站流量访问哪些端口将不会被重定向). 
-q, --istio-outbound-ports string  # 显式指定需要重定向到Envoy出站流量的端口
-i, --istio-service-cidr string  # 重定向到Envoy的出站流量CIDR IP地址,  使用"*"匹配所有端口
-x, --istio-service-exclude-cidr string # 排除重定向到Envoy的出站流量CIDR IP地址,  使用"*"匹配所有端口
-k, --kube-virt-interfaces string  # 虚拟接口列表, 其入站流量将视为出站流量
--network-namespace string  # 应用iptables规则的网络空间
--probe-timeout duration   # 故障检测超时(Probe探针配置)
-g, --proxy-gid string  # 不应用重定向的GID
-u, --proxy-uid string  # 不应用重定向的UID, 这一般是与容器UID相同.
--redirect-dns  # 开启istio-agent捕获DNS流量
-f, --restore-format  # 以iptables-resotre格式打印pilot-agent设置的iptbales规则
--run-validation # 验证iptables
--skip-rule-apply   # 跳过iptables应用

Global Flags:
--log_as_json  # 将日志输出格式化为JSON或普通控制台友好的格式
--log_output_level string   # 要输出的最小日志级别列表, 格式为 <scope>:<level>,<scope>:<level>,... scope可选值: [ads, adsc, all, authn, authorization, ca, cache, citadelclient, controllers, default, delta, dns, gateway, gcecred, googleca, googlecas, grpcgen, healthcheck, ingress status, iptables, klog, kube, mockcred, model, monitoring, retry, sds, security, serviceentry, spiffe, status, stsclient, stsserver, token, trustBundle, validation, wasm, wle, xdsproxy],  level可选值:  [debug, info, warn, error, fatal, none] (默认为 "default:info")
--log_target stringArray  # 日志输出的target. 这可以是任何路径以及特殊值stdout和stderr(默认是"Stdout")
--vklog Level  # 日志级别详细程度的数字。就像 -v 标志一样。例如：--vklog=9（默认“0”）
...
```

我们可以使用`--dry-run`看看pilot-agent会为我们生成哪些iptables规则

```shell
# 生成iptables规则
$ ./pilot-agent istio-iptables --log_as_json --dry-run
2023-12-19T13:10:05.770489Z     info    Istio iptables environment:
ENVOY_PORT=
INBOUND_CAPTURE_PORT=
ISTIO_INBOUND_INTERCEPTION_MODE=
ISTIO_INBOUND_TPROXY_ROUTE_TABLE=
ISTIO_INBOUND_PORTS=
ISTIO_OUTBOUND_PORTS=
ISTIO_LOCAL_EXCLUDE_PORTS=
ISTIO_EXCLUDE_INTERFACES=
ISTIO_SERVICE_CIDR=
ISTIO_SERVICE_EXCLUDE_CIDR=
ISTIO_META_DNS_CAPTURE=
INVALID_DROP=

2023-12-19T13:10:05.770540Z     info    Istio iptables variables:
IPTABLES_VERSION=
PROXY_PORT=15001
PROXY_INBOUND_CAPTURE_PORT=15006
PROXY_TUNNEL_PORT=15008
PROXY_UID=1337		# proxy uid和gid默认被设置为1337
PROXY_GID=1337
INBOUND_INTERCEPTION_MODE=
INBOUND_TPROXY_MARK=1337
INBOUND_TPROXY_ROUTE_TABLE=133
INBOUND_PORTS_INCLUDE=
INBOUND_PORTS_EXCLUDE=
OUTBOUND_OWNER_GROUPS_INCLUDE=*
OUTBOUND_OWNER_GROUPS_EXCLUDE=
OUTBOUND_IP_RANGES_INCLUDE=
OUTBOUND_IP_RANGES_EXCLUDE=
OUTBOUND_PORTS_INCLUDE=
OUTBOUND_PORTS_EXCLUDE=
KUBE_VIRT_INTERFACES=
ENABLE_INBOUND_IPV6=false
DUAL_STACK=false
DNS_CAPTURE=false
DROP_INVALID=false
CAPTURE_ALL_DNS=false
DNS_SERVERS=[],[]
NETWORK_NAMESPACE=
CNI_MODE=false
EXCLUDE_INTERFACES=

2023-12-19T13:10:05.770599Z     info    Running iptables-restore with the following input:
* nat
# 新增了4条自定义链
-N ISTIO_INBOUND
-N ISTIO_REDIRECT
-N ISTIO_IN_REDIRECT
-N ISTIO_OUTPUT
# pilot-agent设置的流量拦截规则
-A ISTIO_INBOUND -p tcp --dport 15008 -j RETURN
-A ISTIO_REDIRECT -p tcp -j REDIRECT --to-ports 15001
-A ISTIO_IN_REDIRECT -p tcp -j REDIRECT --to-ports 15006
-A OUTPUT -p tcp -j ISTIO_OUTPUT
-A ISTIO_OUTPUT -o lo -s 127.0.0.6/32 -j RETURN
-A ISTIO_OUTPUT -o lo ! -d 127.0.0.1/32 -p tcp ! --dport 15008 -m owner --uid-owner 1337 -j ISTIO_IN_REDIRECT
-A ISTIO_OUTPUT -o lo -m owner ! --uid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -m owner --uid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -o lo ! -d 127.0.0.1/32 -p tcp ! --dport 15008 -m owner --gid-owner 1337 -j ISTIO_IN_REDIRECT
-A ISTIO_OUTPUT -o lo -m owner ! --gid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -m owner --gid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -d 127.0.0.1/32 -j RETURN
COMMIT
2023-12-19T13:10:05.770614Z     info    iptables-restore --noflush
2023-12-19T13:10:05.770618Z     info    Running ip6tables-restore with the following input:

2023-12-19T13:10:05.770621Z     info    ip6tables-restore --noflush
2023-12-19T13:10:05.770626Z     info    iptables-save 
2023-12-19T13:10:05.770629Z     info    skipping configuring routes due to dry run mode
```

实际上在Pod, istio首先通过initContainer容器对Pod的iptables定义对应的流量拦截规则, 与istio在Pod中注入的Envoy Sidecar一起工作, 形成了入站出站流量劫持功能(将流量都交由Istio的数据面(Envoy)进行处理). 

> 有可能你会对这些默认端口很迷惑, 并不知道这些端口是干啥用的. 这里有[传送门](https://istio.io/latest/zh/docs/ops/deployment/requirements/), 你可以查阅一下对应的端口是干什么的.



我们有了上面对`pilot-agent istio-iptables`用法的讲解, 接下来我们直接看一下实际上istio注入的initContainer是如何使用的

```yaml
# 准备一个简单的Pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: hellopod
spec:
  containers:
    - name: hello
      image: "fake.docker.io/google-samples/hello-go-gke:1.0"
      ports:
        - name: http
          containerPort: 80
---
# 应用在istio mesh后, 我们查看该Pod的yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    ...
  labels:
    ...
  name: hellopod
spec:
  containers:
  ...
  
  # initContainer (用于初始化iptables的容器)
  initContainers:
  - args:
    - istio-iptables
    - -p
    - "15001"
    - -z
    - "15006"
    - -u
    - "1337"
    - -m
    - REDIRECT
    - -i
    - '*'
    - -x
    - ""
    - -b
    - '*'
    - -d
    - 15090,15021,15020
    - --log_output_level=default:info
    image: gcr.io/istio-testing/proxyv2:latest
    name: istio-init
    resources:
      limits:
        cpu: "2"
        memory: 1Gi
      requests:
        cpu: 100m
        memory: 128Mi
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        add:
        - NET_ADMIN
        - NET_RAW
        drop:
        - ALL
      privileged: false
      readOnlyRootFilesystem: false
      runAsGroup: 0
      runAsNonRoot: false
      runAsUser: 0
...
```

istio自动注入了initContainers配置, 我们从该initContainers启动参数可以看到. pilot-agent在Pod的init阶段对Pod内网络iptables做了以下操作:

```shell
# initContainer的启动命令和参数等价于下列
$ pilot-agent istio-iptables -p "15001" -z "15006" -u "1337" -m REDIRECT -i '*' -x "" -b '*' -d 15090,15021,15020 --log_output_level=default:info

# 我们可以使用--dry-run看一下其定义的iptables规则
$ pilot-agent istio-iptables -p "15001" -z "15006" -u "1337" -m REDIRECT -i '*' -x "" -b '*' -d 15090,15021,15020 --log_output_level=default:info --dry-run

# 我们只看iptables规则部分
* nat
# pilot创建的四张表
-N ISTIO_INBOUND
-N ISTIO_REDIRECT
-N ISTIO_IN_REDIRECT
-N ISTIO_OUTPUT

# istio进行流量劫持的规则
-A ISTIO_INBOUND -p tcp --dport 15008 -j RETURN
-A ISTIO_REDIRECT -p tcp -j REDIRECT --to-ports 15001
-A ISTIO_IN_REDIRECT -p tcp -j REDIRECT --to-ports 15006
-A PREROUTING -p tcp -j ISTIO_INBOUND
-A ISTIO_INBOUND -p tcp --dport 15090 -j RETURN
-A ISTIO_INBOUND -p tcp --dport 15021 -j RETURN
-A ISTIO_INBOUND -p tcp --dport 15020 -j RETURN
# +将15090,15020,15021,15008端口以外的流量全部重定向到15006
-A ISTIO_INBOUND -p tcp -j ISTIO_IN_REDIRECT
-A OUTPUT -p tcp -j ISTIO_OUTPUT
-A ISTIO_OUTPUT -o lo -s 127.0.0.6/32 -j RETURN
-A ISTIO_OUTPUT -o lo ! -d 127.0.0.1/32 -p tcp ! --dport 15008 -m owner --uid-owner 1337 -j ISTIO_IN_REDIRECT
-A ISTIO_OUTPUT -o lo -m owner ! --uid-owner 1337 -j RETURN
# Envoy代理的"真实"出站流量, 将其直接路由出去
-A ISTIO_OUTPUT -m owner --uid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -m owner --gid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -o lo ! -d 127.0.0.1/32 -p tcp ! --dport 15008 -m owner --gid-owner 1337 -j ISTIO_IN_REDIRECT
-A ISTIO_OUTPUT -o lo -m owner ! --gid-owner 1337 -j RETURN
-A ISTIO_OUTPUT -d 127.0.0.1/32 -j RETURN
# 将所有出站流量重定向到15001
-A ISTIO_OUTPUT -j ISTIO_REDIRECT
COMMIT
```

> 我们可以将上述命令解释为: pilot-agent设置了以下的iptables规则. 出站流量都重定向Envoy的15001端口, 入站的流量都重定向Envoy的15006端口, 使用重定向的方式将入站流量转至Envoy, 所有的出站流量的IP网段都将重定向到Envoy, 所有的入站流量都重定向到Envoy, 同时排除15090,15021,15020端口入站流量的重定向, 将用户UID设置为1337.
>
> 在Istio注入的Sidecar或者initContainer中, **UID/GID都喜欢使用1337**.
>
> 上述规则中端口含义如下:
>
> + 15008: HBONE mTLS 隧道端口
> + 15001: Envoy 出站
> + 15006: Envoy 入站
> + 15090: 用于安全网络的 HBONE 端口
> + 15020: 从Istio代理、Envoy和应用程序合并的Prometheus遥测
> + 15021: 健康检查





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

serveInject主要功能:

+ 

```go
```




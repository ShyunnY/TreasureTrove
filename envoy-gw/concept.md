## Envoy Gateway

Envoy Gateway是一个开源项目，用于将[Envoy Proxy](https://www.envoyproxy.io/)作为独立或基于 Kubernetes 的应用程序网关进行管理。[Gateway API](https://gateway-api.sigs.k8s.io/)资源用于动态供应和配置托管Envoy代理.

Envoy Gateway项目的高级目标是通过支持多种入口和**L7/L4**流量路由用例的富有表现力、可扩展、面向角色的API来降低使用难度.



### Goal

**目标**

EnvoyGateway的目标：

+ 具有表现力的API(使用KubernetesGatewayAPI即可操作)
+ 所有环境下均可使用(Kubernetes和非Kubernetes环境)
+ 可拓展性(可以使用xDS API进行二次开发)



在基于Envoy进行拓展时, 我们常会听见两个术语:

- 控制平面: 用于提供应用程序网关和路由功能的相互关联的软件组件的集合。控制平面由 Envoy Gateway实现，并提供管理数据平面的服务。这些服务在[组件](https://gateway.envoyproxy.io/v0.6.0/design/system-design/#components)部分中有详细介绍。
- 数据平面: 提供智能应用程序级流量路由，并作为一个或多个Envoy代理实现。

> 如果有使用过istio, 对这两个名词应该很熟悉



**架构**

![image-20240106135724566](assets/image-20240106135724566.png)

从架构图中我们可以看到两个比较重要的部分: **静态配置**和**动态配置**

**静态配置**

静态配置用于在启动Envoy Gateway时的配置. 例如: 更改GatewayClass控制器名称、配置Provider等。目前，Envoy Gateway仅支持通过配置文件进行配置。如果未提供配置文件,Envoy Gateway将使用默认配置参数启动。

**动态配置**

动态配置基于声明**数据平面**的期望状态并使用**协调循环将实际状态驱动到期望状态**的概念. 数据平面的期望状态被定义为提供以下服务的Kubernetes资源：

- [基础设施管理: 管理数据平面基础设施，即部署、升级等。此配置通过GatewayClass](https://gateway-api.sigs.k8s.io/concepts/api-overview/#gatewayclass)和[Gateway](https://gateway-api.sigs.k8s.io/concepts/api-overview/#gateway)资源表示。可以引用`EnvoyProxy`[自定义资源](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)对gatewayclass.spec.parametersRef修改数据平面基础设施默认参数，例如使用`ClusterIP`服务而不是`LoadBalancer`服务来公开Envoy网络端点。
- 流量路由: 定义如何处理对后端服务的应用程序级请求。例如，将对`www.baidu.com`的所有 HTTP 请求路由到运行 Web 服务器的后端服务。此配置通过[HTTPRoute](https://gateway-api.sigs.k8s.io/concepts/api-overview/#httproute)和[TLSRoute](https://gateway-api.sigs.k8s.io/concepts/api-overview/#tlsroute)资源来表达，这些资源匹配、过滤流量并将流量路由到[后端](https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.BackendObjectReference)。尽管后端可以是任何有效的 Kubernetes Group/Kind资源，但Envoy Gateway仅支持[Service](https://kubernetes.io/docs/concepts/services-networking/service/)引用.







### Components

**Provider**

Provider是一个基础设施组件，Envoy Gateway调用它来建立其运行时配置、解析服务、持久数据等. Provider在Envoy Gateway启动时通过静态配置进行配置.

**Kubernetes Provider**

- 使用 Kubernetes 风格的控制器来协调构成 [动态配置的](https://gateway.envoyproxy.io/v0.6.0/design/system-design/#dynamic-configuration)Kubernetes 资源。
- 通过 Kubernetes API CRUD 操作管理数据平面。
- 使用 Kubernetes 进行服务发现。
- 使用 etcd（通过 Kubernetes API）来保存数据。

**File Provider**

- 使用文件观察器来观察定义数据平面配置的目录中的文件。
- 通过调用内部 API 来管理数据平面，例如`CreateDataPlane()`.
- 使用主机的 DNS 进行服务发现。
- 如果需要，本地文件系统用于保存数据。



**Resource Watcher(资源监听器)**

Resource Watcher监视用于建立和维护Envoy Gateway动态配置的资源. Watch资源的机制是特定于Provider的，例如通知程序、缓存等用于Kubernetes Provider。

**NOTE:** Resource Watcher使用配置的*Provider*作为输入，并将资源提供给资源转换器作为输出。



**Resource Translator(资源转换器)**

Resource Translator转换外部资源, e.g. GatewayClass, 从Resource Watcher转换为Intermediate Representation (IR)中间表示

- 从Resource Watcher中转换基础设施特定的resources/fields到Infra IR.
- 从Resource Watcher中转换Proxy代理的resources/fields到xDS IR.

**Note:** 资源转换器是作为package`Translator`中的API类型实现的`gatewayapi`.

实际上就是将各类不同的资源进行相互转换. 例: 将gateway-api资源转换为xDS-api资源, 将gateway-api转为infra-api资源等.



**Intermediate Representation (IR)**

Intermediate Representation定义了将外部资源转换为的内部数据模型。这使得Envoy Gateway能够与用于动态配置的外部资源解耦。IR包括用作Infra Manager输入的Infra IR和用作 xDS Translator输入的xDS IR.

**Infra IR**: 用作托管数据平面基础设施的内部定义。 

**xDS IR**: 用作托管数据平面xDS配置的内部定义。



**xDS Server**

xDS Server是基于[Go Control Plane](https://github.com/envoyproxy/go-control-plane)的 xDS gRPC-Server. Go控制平面实现了Delta xDS服务器协议，并负责使用xDS来配置数据平面



**Infra Manager** 

基础设施管理器（Infra Manager）是一个特定于Provider的组件.负责管理以下基础设施：

**数据平面**: 管理运行托管Envoy代理所需的所有基础设施。例如，在Kubernetes集群中运行Envoy需要CRUD部署、服务等资源。 

**辅助控制平面**: 用于实施需要与托管Envoy代理进行外部集成的应用程序Gateway功能的可选基础设施。例如, 全局速率限制需要预配和配置Envoy速率限制服务和速率限制过滤器。此类功能通过 Custom Route Filters扩展向用户公开。 基础设施管理器使用*基础设施中间表示（Infra IR）*作为输入，以管理数据平面基础设施。



### Watching Components

Envoy Gateway由多个在进程中通信的组件构成。其中一些组件（即提供者）监视外部资源，并将它们所看到的内容"发布(Sub)"供其他组件消费；另一些组件观察其他组件发布的内容并对其进行操作（例如，资源转换器观察提供者发布的内容，然后发布其自己的结果，由另一组件观察）。一些内部发布的结果会被多个组件消费。

为了促进这种通信，使用了 watchable 库。watchable.Map 类型非常类似于标准库的 sync.Map 类型，但支持 .Subscribe（和 .SubscribeSubset）方法，以促进发布/订阅模式。



**Pub**

我们通信的许多内容都自然地具有名称，可以是一个简单的"name"字符串，也可以是一个"name/namespace"元组。由于 watchable.Map是有类型的，因此为每种类型的事物都有一个Map是有意义的（非常类似于如果我们使用本机的Go map）。例如，可能由Kubernetes Provider写入并由IR转换器读取的结构体。

Kubernetes 提供者通过调用 `table.Thing.Store(name, val)` 和 `table.Thing.Delete(name)`更新表，通过使用与当前值深度相等的值（通常使用 `reflect.DeepEqual`，但您也可以实现自己的`.Equal`方法）更新Map键是一个空操作；这不会为订阅者触发事件。这很方便，因此发布者不需要跟踪太多状态；它不需要知道'我已经发布过这个东西吗?'，它只需`.Store`其数据，watchable就会处理正确的事情。

```go
type ResourceTable struct {
    // gateway classes are cluster-scoped; no namespace
    GatewayClasses watchable.Map[string, *gwapiv1.GatewayClass]

    // gateways are namespace-scoped, so use a k8s.io/apimachinery/pkg/types.NamespacedName as the map key.
    Gateways watchable.Map[types.NamespacedName, *gwapiv1.Gateway]

    HTTPRoutes watchable.Map[types.NamespacedName, *gwapiv1.HTTPRoute]
}
```



**Sub**


同时，Translator和其他感兴趣的组件通过`table.Thing.Subscribe`（或`table.Thing.SubscribeSubset`，如果它们只关心一些特定的Thing）来订阅它。因此,Translator的 goroutine可能如下所示。 

```go
func(ctx context.Context) error {
    for snapshot := range k8sTable.HTTPRoutes.Subscribe(ctx) {
        fullState := irInput{
           GatewayClasses: k8sTable.GatewayClasses.LoadAll(),
           Gateways:       k8sTable.Gateways.LoadAll(),
           HTTPRoutes:     snapshot.State,
        }
        translate(irInput)
    }
}
```

通过`.Subscribe`获取的更新，可以通过`snapshot.State`获取订阅的Map的完整视图；但必须显式读取其他映射。与`sync.Map`类似,`watchable.Map`是线程安全的；虽然`.Subscribe`是一个方便的运行时机的方法，但可以在没有订阅的情况下使用`.Load`等方法。

**NOTE:** 可以有任意数量的订阅者. 同样,可以有任意数量的发布者`.Store`事物，但最好为每个Map只有一个发布者.

从`.Subscribe`返回的通道立即可读，其中包含了在调用`.Subscribe`时映射存在的快照；并且在每次`.Store`或`.Delete`改变Map时再次可读。如果在读取之间发生多次变异（或者如果在 `.Subscribe` 和第一次读取之间发生变异），它们会被合并成一个快照进行读取；`snapshot.State` 是最新的完整状态，而`snapshot.Updates`是导致此快照与上次读取不同的每个变异的列表. 这样，订阅者就不需要担心如果他们无法跟上来自发布者的更改的速度, 会积累多少积压.

> 通过snapshot快照设计, 即使发布者在不断发布, 订阅者也不用担心数据在更新过程会导致不可读. snapshot.Updates将会一直合并新配置.

如果在调用 `.Subscribe` 之前Map包含任何内容，那么第一次读取将不包括那些预先存在的项的 `snapshot.Updates` 条目；如果您正在使用 `snapshot.Update` 而不是 `snapshot.State`，那么必须为您的第一次读取添加特殊处理。我们有一个实用函数 `./internal/message.HandleSubscription` 来帮助处理这种情况。
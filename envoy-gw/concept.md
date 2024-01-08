## Envoy Gateway

Envoy Gateway是一个开源项目，用于将[Envoy Proxy](https://www.envoyproxy.io/)作为独立或基于 Kubernetes 的应用程序网关进行管理。[Gateway API](https://gateway-api.sigs.k8s.io/)资源用于动态供应和配置托管Envoy代理.

Envoy Gateway项目的高级目标是通过支持多种入口和**L7/L4**流量路由用例的富有表现力、可扩展、面向角色的API来降低使用难度.

为了便于我们阅读及理解后续概念, 我们首先明确以下两个概念:

+ **EnvoyProxy**: Gateway实例, 在EnvoyGateway中也被称为数据平面(data-plane), 它是实际上处理南北流量的实例
+ **EnvoyGateway**: Gateway控制器, 在EnvoyGateway中也被成为控制平面(control-plane), 它实际上是一个k8s控制器, 它会处理gateway-api请求, 以及动态分发EnvoyProxy的配置.

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

#### **Provider**

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



#### **Resource Watcher(资源监听器)**

Resource Watcher监视用于建立和维护Envoy Gateway动态配置的资源. Watch资源的机制是特定于Provider的，例如通知程序、缓存等用于Kubernetes Provider。

**NOTE:** Resource Watcher使用配置的*Provider*作为输入，并将资源提供给资源转换器作为输出。



#### **Resource Translator(资源转换器)**

Resource Translator转换外部资源, e.g. GatewayClass, 从Resource Watcher转换为Intermediate Representation (IR)中间表示

- 从Resource Watcher中转换基础设施特定的resources/fields到Infra IR.
- 从Resource Watcher中转换Proxy代理的resources/fields到xDS IR.

**Note:** 资源转换器是作为package`Translator`中的API类型实现的`gatewayapi`.

实际上就是将各类不同的资源进行相互转换. 例: 将gateway-api资源转换为xDS-api资源, 将gateway-api转为infra-api资源等.



#### **Intermediate Representation (IR中间表示)**

Intermediate Representation定义了将外部资源转换为的内部数据模型。这使得Envoy Gateway能够与用于动态配置的外部资源解耦。IR包括用作Infra Manager输入的Infra IR和用作 xDS Translator输入的xDS IR.

**Infra IR**: 用作托管数据平面基础设施的内部定义。 

**xDS IR**: 用作托管数据平面xDS配置的内部定义。



**xDS Server**

xDS Server是基于[Go Control Plane](https://github.com/envoyproxy/go-control-plane)的 xDS gRPC-Server. Go控制平面实现了Delta xDS服务器协议，并负责使用xDS来配置数据平面



#### **Infra Manager** 

基础设施管理器（Infra Manager）是一个特定于Provider的组件.负责管理以下基础设施：

**数据平面**: 管理运行托管Envoy代理所需的所有基础设施。例如，**在Kubernetes集群中运行Envoy需要CRUD实例部署、服务等资源。** 

**辅助控制平面**: 用于实施需要与托管Envoy代理进行外部集成的应用程序Gateway功能的可选基础设施。例如, 全局速率限制需要预配和配置Envoy速率限制服务和速率限制过滤器。此类功能通过 Custom Route Filters扩展向用户公开。 基础设施管理器使用*基础设施中间表示（Infra IR）*作为输入，以管理数据平面基础设施。



### Watching Components

Envoy Gateway由多个在进程中通信的组件构成。其中一些组件（即提供者）监视外部资源，并将它们所看到的内容"发布(Sub)"供其他组件消费；另一些组件观察其他组件发布的内容并对其进行操作（例如，资源转换器观察提供者发布的内容，然后发布其自己的结果，由另一组件观察）。一些内部发布的结果会被多个组件消费。

为了促进这种通信，使用了 watchable 库。watchable.Map 类型非常类似于标准库的 sync.Map 类型，但支持 .Subscribe（和 .SubscribeSubset）方法，以促进发布/订阅模式。



#### **Pub**

我们通信的许多内容都自然地具有名称，可以是一个简单的"name"字符串，也可以是一个"name/namespace"元组。由于 watchable.Map是有类型的，因此为每种类型的事物都有一个Map是有意义的（非常类似于如果我们使用本机的Go map）。例如，可能由Kubernetes Provider写入并由IR转换器读取的结构体。

Kubernetes 提供者通过调用 `table.Thing.Store(name, val)` 和 `table.Thing.Delete(name)`更新表，通过使用与当前值深度相等的值（通常使用 `reflect.DeepEqual`，但您也可以实现自己的`.Equal`方法）更新Map键是一个空操作；这不会为订阅者触发事件。这很方便，因此发布者不需要跟踪太多状态；它不需要知道'我已经发布过这个东西吗?'，它只需`.Store`其数据，watchable就会处理正确的事情。

```go
type ResourceTable struct {
    // gateway classes are cluster-scoped; no namespace
    // 由于GatewayClass是集群资源, 所以并不需要使用namespace
    GatewayClasses watchable.Map[string, *gwapiv1.GatewayClass]

    // gateways are namespace-scoped, so use a k8s.io/apimachinery/pkg/types.NamespacedName as the map key.
    Gateways watchable.Map[types.NamespacedName, *gwapiv1.Gateway]

    HTTPRoutes watchable.Map[types.NamespacedName, *gwapiv1.HTTPRoute]
}
```



#### **Sub**


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





### Gateway Translator

Gateway API将外部资源（例如GatewayClass）从配置的Provider程序转换为中间表示（IR）



#### 输入输出

Translator接受一组输入, 并将输入通过内部转换为输出.

**Gateway API Translator的主要输入**：

- GatewayClass、Gateway、HTTPRoute、TLSRoute、Service、ReferenceGrant、Namespace 和 Secret 资源。

**Gateway API Translator的输出**：

- Xds和infra internal (IR)。
- GatewayClass、Gateway、HTTPRoutes的状态更新



#### **Listeners兼容性**

每个网关中的监听器必须具有唯一的主机名、端口和协议组合。实现可以按端口分组，然后如果实现确定组中的监听器是"兼容的"，则将每组监听器**合并**为单个监听器。

**Note:** Envoy Gateway不会跨多个Gateway进行合并Listeners(即使他们是兼容的)

兼容性规则:

+ 具有兼容Listener的Gateway（相同端口和协议，不同主机名）
+ 具有兼容Listener的Gateway（相同端口和协议，一个指定主机名, 一个未指定主机名）
+ 具有不兼容Listener的Gateway（相同端口和协议，相同主机名）
+ 具有不兼容Listener的Gateway（均不指定主机名）



#### 计算状态

Gateway API规定了每个资源的一组丰富的状态字段和条件。为了达到符合性，Envoy Gateway必须计算受管资源的适当状态字段和条件。

状态是为以下情况计算和设置的：

1. 受管的GatewayClass（gatewayclass.status.conditions）。
2. 每个受管的Gateway，基于其监听器的状态（gateway.status.conditions）。对于Kubernetes提供者，还包括Envoy Deployment和Service的状态，以计算Gateway的状态。
3. 每个Gateway的监听器（gateway.status.listeners）。
4. 每个Route的ParentRef（route.status.parents）。

Gateway API Translator负责在**将Gateway API资源转换为IR并通过消息总线发布状态**时计算状态条件。状态管理器订阅这些状态消息并使用配置的Provider*更新资源状态*。例如，状态管理器使用Kubernetes客户端在Kubernetes API服务器上更新资源状态



#### Context Structure

为了在Translator过程中存储、访问和操作信息，使用了一组上下文结构体。这些结构体包装了特定的Gateway API类型，并添加了额外的字段和方法以支持处理.



`GatewayContext`

```go
// wrap Gateway
type GatewayContext struct {
	// The managed Gateway
    // 当前context管理的Gateway实例
	*v1beta1.Gateway

	// A list of Gateway ListenerContexts.
    // 当前Gateway下的ListenerContext列表
	listeners []*ListenerContext
}
```



`ListenerContext`

```go
// wrap Gateway.Listener
type ListenerContext struct {
    
    // The Gateway listener.
    // gateway.Listener配置
	*v1beta1.Listener

	// The Gateway this Listener belongs to.
    // 当前Listener属于的Gateway
	gateway           *v1beta1.Gateway

	// An index used for managing this listener in the list of Gateway listeners.
    // 用于在Gateway Listener列表中管理该监听器的索引
	listenerStatusIdx int

	// Only Routes in namespaces selected by the selector may be attached
	// to the Gateway this listener belongs to.
    // 只有由Selector选定的命名空间中的Router才能附加到拥有该Listener的Gateway
	namespaceSelector labels.Selector

	// The TLS Secret for this Listener, if applicable.
    // Listener的TLS Secret(如果存在)
	tlsSecret         *v1.Secret
}
```



`RouterContext`

```go
// RouteContext表示可以引用Gateway对象的通用Route对象（HTTPRoute、TLSRoute）等
type RouteContext interface {
	client.Object

	// GetRouteType returns the Kind of the Route object, HTTPRoute,
	// TLSRoute, TCPRoute, UDPRoute etc.
    // 获取Router的类型
	GetRouteType() string

	// GetHostnames returns the hosts targeted by the Route object.
    // 获取Router的hosts
	GetHostnames() []string

	// GetParentReferences returns the ParentReference of the Route object.
    // 获取Router的父引用ParentReference
	GetParentReferences() []v1beta1.ParentReference

	// GetRouteParentContext returns RouteParentContext by using the Route
	// objects' ParentReference.
    // 获取Router的父引用ParentReferenceContext
	GetRouteParentContext(forParentRef v1beta1.ParentReference) *RouteParentContext
}
```







### Controller Metrics

目前，Envoy Gateway控制平面提供log和控制器运行时metrics,但没有任何trace。日志通过我们的专有库（`internal/logging`由`zap`进行填充）进行管理并写入`/dev/stdout`.

控制平面的指标：

+ 支持Prometheus metrics的**PULL**模式, 并将这些metrics公开在管理地址上。
+ 支持Prometheus metrics的**PUSH**模式，从而通过gRPC或HTTP将指标发送到OpenTelemetry Stats接收器(Sink)中.

#### **标准**

Envoy Gateway的指标将建立在[OpenTelemetry](https://opentelemetry.io/)标准的基础上。所有指标都将通过[openTelemetry SDK](https://opentelemetry.io/docs/specs/otel/metrics/sdk/)进行配置，该SDK提供可连接到各种后端的中性库.



#### 可扩展性 

Envoy Gateway支持PULL/PUSH模式的指标，默认情况下通过Prometheus导出指标。

此外，Envoy Gateway还可以使用OTEL gRPC指标导出器和OTEL HTTP指标导出器导出指标，通过grpc/http将指标推送到远程OTEL收集器。

用户可以通过两种方式扩展这些功能：

+ **下游收集**: 基于导出的数据，其他工具可以根据需要收集、处理和导出遥测数据。一些示例包括：PULL模式中的指标：OTEL收集器可以抓取Prometheus并导出到X。 PUSH模式中的指标：OTEL收集器可以接收OTEL gRPC/HTTP导出器的指标并导出到X。 虽然上述示例涉及OTEL收集器，但还有许多其他可用的系统。

+ **供应商扩展：** <u>OTEL库允许注册提供者/处理程序</u>。虽然我们将提供Envoy Gateway可扩展性中提到的默认选项（通过Prometheus进行PULL，通过OTEL HTTP指标导出器进行PUSH），但我们可以轻松地允许Envoy Gateway的定制构建插入替代项，如果默认选项不符合其需求。例如，用户可能更喜欢通过OTLP gRPC指标导出器而不是HTTP指标导出器编写指标。这是完全可以接受的,而且几乎不可能阻止。<u>OTEL有注册其提供者/导出器的方式</u>，而Envoy Gateway可以确保其使用方式不过于困难，以便更轻松地替换不同的提供者/导出器。

> 换句话说, 我们可以在下游收集上选用不同的组件(非侵入式), 或者在代码埋点中使用不同的Handler进行处理(侵入式). 选择需要看具体使用场景.



#### 类型定义

我们可以看一下EnvoyGateway中是如何定义Metrics类型:

`EnvoyGatewayTelemetry`

```go
// EnvoyGatewayTelemetry defines telemetry configurations for envoy gateway control plane.
// Control plane will focus on metrics observability telemetry and tracing telemetry later.
// EnvoyGatewayTelemetry定义了Envoy Gateway控制平面的遥测配置, 控制平面将在后续专注于度量观测遥测和跟踪遥测。
type EnvoyGatewayTelemetry struct {
	// Metrics defines metrics configuration for envoy gateway.
    // Metrics定义了Envoy Gateway关于metrics的配置
	Metrics *EnvoyGatewayMetrics `json:"metrics,omitempty"`
}
```

`EnvoyGatewayMetrics`

```go
// EnvoyGatewayMetrics defines control plane push/pull metrics configurations.
// EnvoyGatewayMetrics定义了控制平面push/pull指标的策略
type EnvoyGatewayMetrics struct {
	// Sinks defines the metric sinks where metrics are sent to.
    // Sink定义的是指标应该发送的地方. (我们可以在这进行拓展下游收集器) (push策略)
	Sinks []EnvoyGatewayMetricSink `json:"sinks,omitempty"`
    
	// Prometheus defines the configuration for prometheus endpoint.
    // 定义Prometheus的端点配置.  (pull策略)
	Prometheus *EnvoyGatewayPrometheusProvider `json:"prometheus,omitempty"`
}
```

`EnvoyGatewayMetricSink`

```go
// EnvoyGatewayMetricSink defines control plane
// metric sinks where metrics are sent to.

// EnvoyGatewayMetricSink定义了控制面需要将指标发送到哪个组件上.
type EnvoyGatewayMetricSink struct {
	// Type defines the metric sink type.
	// EG control plane currently supports OpenTelemetry.
	// +kubebuilder:validation:Enum=OpenTelemetry
	// +kubebuilder:default=OpenTelemetry
    
    // 定义了指标Sink的类型, 目前EnvoyGateway仅支持OTEL
	Type MetricSinkType `json:"type"`
    
	// OpenTelemetry defines the configuration for OpenTelemetry sink.
	// It's required if the sink type is OpenTelemetry.
    
    // OTEL的相关配置. (需要将Type设置为OpenTelemetry)
	OpenTelemetry *EnvoyGatewayOpenTelemetrySink `json:"openTelemetry,omitempty"`
}
```

`EnvoyGatewayOpenTelemetrySink`

```go
// otel sink配置
type EnvoyGatewayOpenTelemetrySink struct {
	// Host define the sink service hostname.
    
    // otel collector的host
	Host string `json:"host"`
    
	// Protocol define the sink service protocol.
	// +kubebuilder:validation:Enum=grpc;http
    
    // otel collector的protocol协议. 可选grpc/http
	Protocol string `json:"protocol"`
	// Port defines the port the sink service is exposed on.
	//
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=4317
    
    // otel collector的port 默认为4317
	Port int32 `json:"port,omitempty"`
}
```

`EnvoyGatewayPrometheusProvider`

```go
// EnvoyGatewayPrometheusProvider will expose prometheus endpoint in pull mode.

// EnvoyGatewayPrometheusProvider将暴露端点让Prometheus抓取
type EnvoyGatewayPrometheusProvider struct {
	// Disable defines if disables the prometheus metrics in pull mode.
	
    // 控制Prometheus 开启/关闭
	Disable bool `json:"disable,omitempty"`
}
```



我们看完上面有关控制平面Metrics类型定义, 再来看一下在K8s环境中如何配置控制面:

禁用Promtheus, 将指标以push方式推送到otel collector上

```yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyGateway
gateway:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
logging:
  level: null
  default: info
provider:
  type: Kubernetes
telemetry:
  # 与上面metrics的定义相联系
  metrics:
    prometheus:
      disable: false
    sinks:
      - type: OpenTelemetry
        openTelemetry:
          host: otel-collector.monitoring.svc.cluster.local
          port: 4318
          protocol: http
```



### Backend TrafficPolicy

BackendTrafficPolicy是一种隐含的层次结构类型的API，可用于扩展Gateway API。它可以针对Gateway或xRoute（HTTPRoute/GRPCRoute等）进行定位。当定位到Gateway时，它将将BackendTrafficPolicy中配置的设置应用于该Gateway的所有子xRoute资源。如果一个BackendTrafficPolicy定位到一个xRoute，并且不同的BackendTrafficPolicy定位到该路由所属的Gateway，那么将以定位到*xRoute资源的策略中的配置为冲突中的胜出*配置



例如以下配置, 定位到`king=HTTPRoute`的配置胜出, 将会覆盖gateway级别的配置.

```yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: default-ipv-policy
  namespace: default
spec:
  protocols:
    enableIPv6: false
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: eg
    namespace: default
---
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: ipv6-support-policy
  namespace: default
spec:
  protocols:
    enableIPv6: true
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: ipv6-route
    namespace: default
```



#### features-fields

以下是BackendTrafficPolicy部分字段以及其代表的功能

- Protocol configuration  协议配置
- Circuit breaking  熔断
- Retries  重试
- Keep alive probes  保持探针存活
- Health checking  健康检查
- Load balancing  负载均衡
- Rate limit  速率限制



#### design

+ 该API仅支持单个targetRef, 并且只能绑定到`Gateway`或`Route`（HTTPRoute/GRPCRoute等）资源。 

+ :star: 此API资源必须属于与其目标资源相同的命名空间。 
+ 每个特定Listener（部分）在Gateway中只能附加*一个*策略资源。 
+ 如果策略定位到资源但无法Attach到它，应该在策略状态字段中使用Conflicted=True条件反映此信息。
+ 如果多个策略定位到同一资源，则最旧的资源（基于创建时间戳）将附加到Gateway Listeners，其他资源将不会附加。 
+ 如果策略A具有一个targetRef，其中包含一个sectionName，即它定位到Gateway中的特定Listener，并且策略B有一个targetRef，它定位到同一个Gateway， 则策略A将应用到targetRef.SectionName中定义的特定Listener。 策略B将应用于Gateway中的其余Listeners。策略B将具有一个附加的状态条件Overridden=True



### Bootstrap Design

Bootstrap配置提供了需要允许高级用户指定他们自定义的Envoy Bootstrap配置，而不是使用Envoy Gateway中定义的默认Bootstrap配置。这使得高级用户能够扩展Envoy Gateway并支持他们的定制用例，例如设置Trace和Stats配置，这些配置在Envoy Gateway中并未提供支持



我们可以:

+ 定义一个API字段，以允许用户指定自定义的引导配置
+ 使用`egctl`工具，使用户能够生成默认的引导配置，并验证其自定义的引导配置

利用现有的EnvoyProxy资源，可以通过使用**parametersRef**字段将其附加到GatewayClass，并在资源内部定义一个Bootstrap字段。如果设置了此字段，其值将用作由Envoy Gateway创建的所有受管Envoy代理的引导配置。也就是说, 我们可以声明一个EnvoyProxy配置, 在声明GatewayClass的时候通过<u>parametersRef字段指明EnvoyProxy</u>配置.



举个例子: 假设我们希望修改Envoy实例的镜像,我们可以这样做:

1. 声明一个EnvoyProxy配置

   ```yaml
   apiVersion: gateway.envoyproxy.io/v1alpha1
   kind: EnvoyProxy
   metadata:
     name: custom-proxy-config
     namespace: envoy-gateway-system
   spec:
     # 配置Envoy Gateway实例的Bootstrap
     bootstrap: ...
     provider:
       type: Kubernetes
       kubernetes:
         # 指定Envoy Gateway实例的镜像
         envoyDeployment:
           container:
             image: bitnami/envoy:1.28.0
   ```

2. 修改GatewayClass配置 

   ```yaml
   apiVersion: gateway.networking.k8s.io/v1
   kind: GatewayClass
   metadata:
     name: eg
   spec:
     controllerName: gateway.envoyproxy.io/gatewayclass-controller
     # 通过parameterRef引用上面配置的EnvoyProxy
     parametersRef:
       group: gateway.envoyproxy.io
       kind: EnvoyProxy
       name: custom-proxy-config
       namespace: envoy-gateway-system
   ```



最后我们再看一下EnvoyProxy关于Bootstrap类型定义

```go
// EnvoyProxySpec定义了EnvoyProxy状态
type EnvoyProxySpec struct {
	// Bootstrap定义Envoy引导配置为YAML字符串。
    // 访问 https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/bootstrap/v3/bootstrap.proto#envoy-v3-api-msg-config-bootstrap-v3-bootstrap
    // 了解有关语法的更多信息。
    // 如果设置了此字段，则该字段用作托管的Envoy代理群的引导配置，而不是由Envoy Gateway设置的默认引导配置。
    // 在Bootstrap中，一些必须与xDS服务器（Envoy Gateway）通信并从中接收xDS资源的字段是不可配置的，设置它们将导致`EnvoyProxy`资源被拒绝。
	Bootstrap *string `json:"bootstrap,omitempty"`
}
```





### Client Traffic Policy

ClientTrafficPolicy允许系统管理员配置Envoy代理服务器与下游客户端的行为方式. 这与BackendTrafficPolicy策略相反. 一个作用在`Proxy -> Upstram`, 另一个作用在`Downstream -> Proxy`



#### features-fields

+ Downstream ProxyProtocol 下游代理协议
+ Downstream Keep Alives 下游Keepalive
+ IP Blocking IP黑名单
+ Downstream HTTP3 下游HTTP3



#### design

+ ClientTrafficPolicy API仅支持单个targetRef，并且只能绑定到一个Gateway资源。
+ ClientTrafficPolicy API资源必须属于与Gateway资源*相同*的命名空间。

+ 每个特定的Listener（部分）在Gateway中只能附加一个策略资源。如果策略的目标是某个资源，但无法附加到该资源，则此信息应通过Policy Status字段以Conflicted=True的条件反映。

+ 如果多个策略都针对相同的资源，基于创建时间戳的最旧资源将附加到Gateway的Listeners，而其他资源则不会。

+ 如果Policy A具有一个targetRef，其中包含一个sectionName，即它针对Gateway内的特定Listener，而Policy B具有一个targetRef，它针对整个Gateway，则：
  + Policy A将应用/附加到targetRef.SectionName中定义的特定Listener。
  + Policy B将应用于Gateway内的其余Listeners。Policy B将具有一个附加的状态条件，即Overridden=True



#### TODO: **Config**

我们可以看一下ClientTrafficPolicy配置及其字段:

```yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: ClientTrafficPolicy
metadata:
  name: enable-proxy-protocol-policy
  namespace: default
spec:
  # targetRef是此策略附加到的资源的名称。此策略和TargetRef必须位于同一命名空间中，此策略才能生效并应用于Gateway
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: eg
    namespace: default
  enableProxyProtocol: true
```





### TODO: ConfigApi

关于control-plane控制平面的配置(由于当前版本EnvoyGateway不包含元数据字段，因为它当前表示为静态配置文件而不是Kubernetes资源).

我们可以通过`$ kubectl get cm envoy-gateway-config -n envoy-gateway-system -o yaml`查看EnvoyGateway的配置.

> 由于 EnvoyGateway 不显示状态，因此 EnvoyGatewaySpec 是内联的







### Egctl

`egctl`是一个可以从EnvoyProxy和Gateway收集配置信息，分析系统配置以诊断EnvoyGateway中的任何问题的命令行工具.

`egctl`的语法如下:

```shell
$ egctl [command] [entity] [name] [flags]
```

其中:

- command：指定要对一个或多个资源执行的操作，例如`config`。`version`
- entity：指定正在执行操作的实体，例如`envoy-proxy`或`envoy-gateway`。
- name：指定指定实例的名称。
- flags：指定可选标志。例如，您可以使用`-c`或`--config`标志来指定安装值。

|   Operation    |        Syntax         |                  Description                   |
| :------------: | :-------------------: | :--------------------------------------------: |
|   `version`    |    `egctl version`    |               打印当前egctl版本                |
|    `config`    | `egctl config ENTITY` | 从EnvoyProxy和EnvoyGateway检索有关proxy的配置  |
|   `analyze`    |    `egctl analyze`    |      分析EnvoyGateway配置以及打印校验信息      |
| `experimental` | `egctl experimental`  | 用于实验功能的子命令。这些并不能保证向后兼容性 |



举几个栗子:

```shell
# 检索QuickStart.yaml中gateway配置的EnvoyProxy
# 我们去掉冗余数据后可以看到, EnvoyProxy监听10080端口, 我们在Gateway中配置的Listener.port实际上是配置在EnvoyProxy前面的Service上.
# 然后EnvoyProxy使用xDS中的rDS进行路由匹配, 如果匹配, 再将流量转发到cDS上游
$ egctl config envoy-proxy listener envoy-default-eg-gw-63522087-7b5fdfc667-7xmk8 -o yaml
envoy-gateway-system:
  envoy-default-eg-gw-63522087-7b5fdfc667-7xmk8:
    dynamicListeners:
    - activeState:
        listener:
          '@type': type.googleapis.com/envoy.config.listener.v3.Listener
          address:
            socketAddress:
              address: 0.0.0.0
              portValue: 10080
          defaultFilterChain:
            filters:
            - name: envoy.filters.network.http_connection_manager
              typedConfig:
                '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                httpFilters:
                - name: envoy.filters.http.router
                  typedConfig:
                    '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                rds:
                  configSource:
                    ads: {}
                    resourceApiVersion: V3
                  routeConfigName: default/eg-gw/http
                statPrefix: http
                upgradeConfigs:
                - upgradeType: websocket
                useRemoteAddress: true
          name: default/eg-gw/http

$ 
```





### Observability

EnvoyGateway也提供了可观测性三板斧: **链路, 日志, 指标**

#### Log

Envoy支持将可扩展的访问日志记录到不同的目标（sinks），如文件、gRPC等。Envoy支持使用预定义字段以及任意的HTTP请求和响应头来定制访问日志格式。Envoy支持多个内置的访问日志过滤器和在运行时注册的扩展过滤器。

Envoy Gateway利用Gateway API来配置受管理的Envoy代理。Gateway API定义了核心、扩展和实现特定的API支持级别，供实现者（例如Envoy Gateway）暴露功能。由于访问日志未在核心或扩展API中涵盖，因此EG应提供一种易于配置的访问日志格式和每个Envoy代理的目标（sinks）。

> 我们可以使用自定义的Sink接收EnvoyProxy产生的日志.

当前:

1. **访问日志Sink支持：**
   - 文件：将访问日志记录到文件的功能。
   - OpenTelemetry后端：将访问日志发送到OpenTelemetry后端的能力。
2. **cel：**
   - 基于CEL表达式实现访问日志过滤器。这表明打算通过使用CEL表达式作为过滤访问日志的条件，实现更多灵活性



我们可以配置以下内容:

1. 将访问日志记录到文件：
   - 针对EnvoyProxy的配置，设定使访问日志被记录到文件的设置。
2. 将访问日志发送到OpenTelemetry后端：
   - 针对EnvoyProxy的配置，设定使访问日志被发送到OpenTelemetry后端的设置。
3. 为EnvoyProxy配置多个访问日志提供程序：
   - 针对EnvoyProxy的配置，设定多个访问日志提供程序的设置。

package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fishnet-inject/kube"
	"fishnet-inject/render"
	"fishnet-inject/sugar"
	"fishnet-inject/webhooks/tpls"
	"gomodules.xyz/jsonpatch/v2"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sort"
	"strconv"
	"sync"
)

type InjectorWebhook struct {
	*admission.Webhook
}

func NewInjectorWebhook() (*InjectorWebhook, error) {
	injectWh := &InjectorWebhook{}

	client, connErr := kube.NewClient(kube.ClientConfig{})
	if connErr != nil {
		return nil, connErr
	}

	cliset, err := client.ClientSet()
	if connErr != nil {
		return nil, err
	}

	// default watch "mesh" namespace config
	injector := NewInjector(cliset, configNamespace, "inject_config")

	wh := &webhook.Admission{
		Handler: injector,
	}
	injectWh.Webhook = wh

	return injectWh, nil
}

// Injector
// TODO: 我们需要确保幂等性
// TODO: 还需要合理进行错误处理
type Injector struct {
	Config *Config
	mux    *sync.RWMutex

	cliset *kubernetes.Clientset

	namespace string
	configKey string
	once      *sync.Once
	watcher   Watcher
}

func NewInjector(cliset *kubernetes.Clientset, namespace, configKey string) *Injector {

	injector := &Injector{
		cliset:    cliset,
		namespace: namespace,
		configKey: configKey,
		mux:       &sync.RWMutex{},
		once:      &sync.Once{},
	}

	// 初始化watcher
	injector.initWatcher(context.TODO())

	return injector
}

func (ij *Injector) initWatcher(ctx context.Context) {

	// 注册watcher handler
	// 我们注册一个获取配置的回调函数
	// 当ConfigMap更新的时候, 我们希望同时更新Webhook的Injector配置
	watcher := NewConfigMapWatch(ij.cliset, "configmap-watcher", ij.namespace, ij.configKey)
	watcher.SetCallback(ij.updateConfig)
	ij.watcher = watcher

	// 在初始化的时候, 我们阻塞的获取ConfigMap中的配置
	// 如果用户并未提供相关的配置, 我们将使用默认的配置
	ij.once.Do(func() {
		var err error
		var config *Config
		if config, err = ij.watcher.Get(); err != nil {
			sugar.Warn("injector set default config")
			_ = ij.updateConfig(NewDefaultConfig())

		} else if err = ij.updateConfig(config); err != nil {

			sugar.Error("injector update config error: ", err)
		}
	})

	go func() {
		watcher.Run(ctx.Done())
	}()
}

func (ij *Injector) Handle(ctx context.Context, req admission.Request) admission.Response {

	// 反序列化pod
	pod := &corev1.Pod{}
	if err := json.Unmarshal(req.Object.Raw, pod); err != nil {
		log.Println("fishnet injector webhook is unable to deserialize the pod, error: ", err.Error())
		return admission.Denied("unable deserialize pod")
	}

	sugar.Infof("webhook receive the pod '%s' that needs to be injected", pod.Name)
	return ij.injectLogic(pod, req)
}

func (ij *Injector) injectLogic(originPod *corev1.Pod, req admission.Request) admission.Response {

	// potential check
	potentialCheck(originPod, req)

	// check if the pod should be injected
	if !checkInject(originPod) {
		return webhook.Allowed("")
	}

	// render template
	mergePod, err := runTemplate(originPod, ij.getConfig())
	if err != nil {
		sugar.Error("run template error: ", err)
		return admission.Allowed("")
	}

	// post process pod handler
	if err := postProcessPod(mergePod, ij.getConfig()); err != nil {
		return admission.Allowed("")
	}

	// build patch for delta pod and origin pod
	patch, err := createJSONPatch(originPod, mergePod)
	if err != nil {
		return admission.Allowed("")
	}

	return admission.Patched("inject success", patch...)
}

func postProcessPod(mergePod *corev1.Pod, config *Config) error {

	// robustness checking
	if mergePod.Annotations == nil {
		mergePod.Annotations = map[string]string{}
	}
	if mergePod.Labels == nil {
		mergePod.Labels = map[string]string{}
	}

	// set sidecarEnv
	if len(config.ValueConfig.SidecarEnv) > 0 {
		if container := findSidecarContainer(ProxyContainerName, mergePod.Spec.Containers); container == nil {
			return nil
		} else {
			setContainerEnv(container, config.ValueConfig.SidecarEnv)
		}
	}

	// 2. TODO: 设置Prometheus配置

	// 设置Probe
	overwriteProbe(mergePod, config)

	// 设置其余元数据
	if err := applyMetadata(mergePod, config); err != nil {
		return err
	}

	// 重排容器顺序
	if err := reorderContainer(mergePod, config); err != nil {
		sugar.Error("reorder pod: %s container error: %v", mergePod.Name, err)
		return err
	}

	return nil
}

// 对containers进行重排序
// 后续也许需要对initContainers进行重排序
func reorderContainer(pod *corev1.Pod, config *Config) error {

	// 默认将envoyproxy放在最后
	proxyLocation := moveLast

	if config.ValueConfig.AfterProxyStart {
		sugar.Info("proxy container start before application")
		proxyLocation = moveFirst
	}

	containers := []corev1.Container{}
	var proxy *corev1.Container
	for _, c := range pod.Spec.Containers {
		c := c
		if c.Name == ProxyContainerName {
			proxy = &c
		} else {
			containers = append(containers, c)
		}
	}

	if proxy == nil {
		return nil
	}

	switch proxyLocation {
	case moveFirst:
		containers = append([]corev1.Container{*proxy}, containers...)
	case moveLast:
		containers = append(containers, *proxy)
	}
	pod.Spec.Containers = containers

	return nil
}

func overwriteProbe(pod *corev1.Pod, config *Config) {

	if config.ValueConfig.ProxyProbe == nil {
		return
	}

	if !shouldOverwriteProbe(pod.Annotations) {
		return
	}

	proxy := findSidecarContainer(ProxyContainerName, pod.Spec.Containers)
	if proxy == nil {
		return
	}
	proxy.StartupProbe = config.ValueConfig.ProxyProbe.StartupProbe
	proxy.ReadinessProbe = config.ValueConfig.ProxyProbe.ReadinessProbe

	return
}

func shouldOverwriteProbe(annotations map[string]string) bool {
	for key, val := range annotations {
		if key == sidecarOverwriteAnnotation {
			if overwrite, err := strconv.ParseBool(val); err == nil {
				return overwrite
			}
		}
	}

	return false
}

func applyMetadata(pod *corev1.Pod, config *Config) error {

	for key, val := range config.ValueConfig.Annotations {
		pod.Annotations[key] = val
	}

	for key, val := range config.ValueConfig.Labels {
		pod.Labels[key] = val
	}

	if _, ok := pod.Annotations[sidecarInjectAnnotation]; !ok {
		pod.Annotations[sidecarInjectAnnotation] = "true"
	}

	// set status metadata
	if _, ok := pod.Annotations[sidecarInjectStatus]; !ok {
		status := &SidecarInjectStatus{}

		status.InitContainers = append(status.InitContainers, config.td.InitContainerImage)
		status.Containers = append(status.Containers, config.td.SidecarImage)
		status.Revision = "default" //TODO:  currently set to the default

		if raw, err := json.Marshal(status); err != nil {
			return err
		} else {
			pod.Annotations[sidecarInjectStatus] = string(raw)
		}

	}

	return nil
}

func (ij *Injector) updateConfig(injectConfig *Config) error {

	ij.mux.Lock()
	defer ij.mux.Unlock()

	ij.Config = injectConfig
	sugar.Debug("sync injector config success")

	return nil
}

func (ij *Injector) getConfig() *Config {
	ij.mux.RLock()
	defer ij.mux.RUnlock()

	return ij.Config
}

// 将外部envs设置到给定的容器上
// 如果容器中env与外部env有同名env, 则外部的env将会覆盖容器中已存在的env
// example:
// container1.env: a,b,c,d
// newEnvs: b,f
// result: containers.env: a,b(newEnvs),c,d
func setContainerEnv(container *corev1.Container, newEnvs map[string]string) {

	envVars := make([]corev1.EnvVar, 0)

	for _, env := range container.Env {
		if _, found := newEnvs[env.Name]; !found {
			envVars = append(envVars, env)
		}
	}

	keys := make([]string, len(newEnvs))
	for key := range newEnvs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := newEnvs[key]
		envVars = append(envVars, corev1.EnvVar{
			Name:      key,
			Value:     val,
			ValueFrom: nil,
		})
	}

	container.Env = envVars
}

func findSidecarContainer(name string, containers []corev1.Container) *corev1.Container {

	for i, container := range containers {
		if container.Name == name {
			return &containers[i]
		}
	}

	return nil
}

func createJSONPatch(originPod *corev1.Pod, mergePod *corev1.Pod) ([]jsonpatch.Operation, error) {
	origin, err := json.Marshal(originPod)
	if err != nil {
		return nil, err
	}

	merge, err := json.Marshal(mergePod)
	if err != nil {
		return nil, err
	}

	return jsonpatch.CreatePatch(origin, merge)
}

func runTemplate(originalPod *corev1.Pod, config *Config) (*corev1.Pod, error) {

	templatePod := &corev1.Pod{}
	copyOriginPod := originalPod.DeepCopy()

	// get templateData for dynamic Config
	var overlayErr error
	td := config.GetTemplateData()
	rd := render.NewRender()

	// add init template
	var initBuf bytes.Buffer
	if err := rd.AddTextTemplate("initContainer_tmpl", render.NewRenderData(func() (io.Writer, any) {

		return &initBuf, td
	}, func(writer io.Writer) error {

		initBuf := writer.(*bytes.Buffer)
		if templatePod, overlayErr = overlayPod(templatePod, initBuf.Bytes()); overlayErr != nil {
			return overlayErr
		}
		return nil
	}), tpls.InitContainerTemplate); err != nil {
		return nil, err
	}

	// add sidecar template
	var sidecarBuf bytes.Buffer
	if err := rd.AddTextTemplate("sidecar_tmpl", render.NewRenderData(func() (io.Writer, any) {

		return &sidecarBuf, td
	}, func(writer io.Writer) error {

		sidecarBuf := writer.(*bytes.Buffer)
		if templatePod, overlayErr = overlayPod(templatePod, sidecarBuf.Bytes()); overlayErr != nil {
			return overlayErr
		}
		return nil
	}), tpls.SidecarTemplate); err != nil {
		return nil, err
	}

	// running render template
	if renderErr := rd.RunRenderTemplate(); renderErr != nil {
		sugar.Error("injector render template error: ", renderErr)
		return nil, renderErr
	}

	var mergeErr error
	templatePod, mergeErr = mergeCustomTemplate(templatePod, originalPod, config)
	if mergeErr != nil {
		sugar.Error("injector merge custom template error: ", mergeErr)

		return nil, mergeErr
	}

	templateBytes, err := json.Marshal(templatePod)
	if err != nil {
		return nil, err
	}

	// 将模板pod合并到originalPod上
	retPod, err := overlayPod(copyOriginPod, templateBytes)
	if err != nil {
		sugar.Error("apply overlay template error: ", err)
		return nil, err
	}

	return retPod, nil
}

func mergeCustomTemplate(templatePod *corev1.Pod, originalPod *corev1.Pod, config *Config) (*corev1.Pod, error) {

	// 如果config和annotation都存在custom template, 那么以annotation为主
	merge := false
	var templateRaw []byte

	if config.ValueConfig.CustomTemplate != "" {
		merge = true
		templateRaw = []byte(config.ValueConfig.CustomTemplate)
	}

	for k, v := range originalPod.Annotations {
		if k == customTemplateAnnotation {
			merge = true
			templateRaw = []byte(v)
		}
	}

	if !merge {
		return templatePod, nil
	}

	return overlayPod(templatePod, templateRaw)
}

func potentialCheck(pod *corev1.Pod, req admission.Request) {
	if pod.Namespace == "" {
		pod.Namespace = req.Namespace
	}
}

// 这里使用K8s的StrategicMerge合并策略
// doc: https://kubernetes.io/zh-cn/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#notes-on-the-strategic-merge-patch
func overlayPod(target *corev1.Pod, overlay []byte) (*corev1.Pod, error) {

	ret := &corev1.Pod{}
	currentJSON, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	schema, err := strategicpatch.NewPatchMetaFromStruct(ret)
	if err != nil {
		return nil, err
	}

	originalMap, err := parseJSONMap(currentJSON, json.Unmarshal)
	if err != nil {
		return nil, err
	}
	patchMap, err := parseJSONMap(overlay, func(data []byte, v any) error {
		return yaml.Unmarshal(data, v)
	})

	result, err := strategicpatch.StrategicMergeMapPatchUsingLookupPatchMeta(originalMap, patchMap, schema)
	if err != nil {
		return nil, err
	}
	patch, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(patch, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func parseJSONMap(j []byte, unmarshal func(data []byte, v any) error) (map[string]any, error) {
	if j == nil {
		j = []byte("{}")
	}

	m := map[string]interface{}{}
	err := unmarshal(j, &m)
	if err != nil {
		return nil, mergepatch.ErrBadJSONDoc
	}
	return m, nil
}

func checkInject(pod *corev1.Pod) bool {

	// 在主机网络模式下, 注入proxy将会导致路由异常
	if pod.Spec.HostNetwork == true {
		return false
	}

	// 在这些namespace下, 我们不执行自动注入
	for _, ignore := range IgonreNamespace {
		if pod.ObjectMeta.Namespace == ignore {
			return false
		}
	}

	for an := range pod.Annotations {
		if an == sidecarIgnoreInjectAnnotation {
			return false
		}
	}

	return true
}

type SidecarInjectStatus struct {
	InitContainers []string `json:"initContainers,omitempty"`
	Containers     []string `json:"containers,omitempty"`
	Revision       string   `json:"revision,omitempty"`
}

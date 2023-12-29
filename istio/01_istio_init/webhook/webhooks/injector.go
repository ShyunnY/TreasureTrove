package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
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

func NewInjectorWebhook() *InjectorWebhook {
	injectWh := &InjectorWebhook{}
	injector := NewInjector()

	wh := &webhook.Admission{
		Handler: injector,
	}
	injectWh.Webhook = wh

	return injectWh
}

type Injector struct {
	Config *Config
	mux    sync.RWMutex

	namespace string
	watcher   Watcher
}

func NewInjector() *Injector {
	return &Injector{}
}

func (ij *Injector) Handle(ctx context.Context, req admission.Request) admission.Response {

	// 反序列化pod
	pod := &corev1.Pod{}
	if err := json.Unmarshal(req.Object.Raw, pod); err != nil {
		log.Println("fishnet injector webhook is unable to deserialize the pod, error: ", err.Error())

		return admission.Denied("unable deserialize pod")
	}

	log.Println("receiver pod")
	return ij.injectLogic(pod, req)
}

func (ij *Injector) injectLogic(originPod *corev1.Pod, req admission.Request) admission.Response {

	// 潜在字段处理
	potentialCheck(originPod, req)

	// 检查是否应该注入pod
	if !checkInject(originPod) {
		return webhook.Allowed("")
	}

	// 2.获取配置, 还未做, 目前先用静态配置
	// TODO: 实际上, 我们也许要通过K8s的ConfigMap获取sidecar模板, 这可以让我们进行动态获取配置.
	config := NewDefaultConfig()

	// 3. 运行模板
	mergePod, err := runTemplate(originPod, config)
	if err != nil {
		return admission.Allowed("")
	}

	if err := postProcessPod(mergePod, config); err != nil {
		return admission.Allowed("")
	}

	// 5. 构建patch
	// 使用增量pod与源pod构建patch
	patch, err := createJSONPatch(originPod, mergePod)
	if err != nil {

		return admission.Allowed("")
	}

	// TODO: 对容器进行重排序
	return admission.Patched("inject success", patch...)
}

func postProcessPod(mergePod *corev1.Pod, config *Config) error {

	if mergePod.Annotations == nil {
		mergePod.Annotations = map[string]string{}
	}

	if mergePod.Labels == nil {
		mergePod.Labels = map[string]string{}
	}

	// 设置环境变量
	if len(config.ValueConfig.ProxyEnv) > 0 {
		if container := findSidecarContainer(ProxyContainerName, mergePod.Spec.Containers); container == nil {
			return nil
		} else {
			setContainerEnv(container, config.ValueConfig.ProxyEnv)
		}
	}

	// 2. TODO: 设置Prometheus配置

	// 3. 设置Probe
	overwriteProbe(mergePod, config)

	// 4. 设置其余元数据
	applyMetadata(mergePod, config)

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

func applyMetadata(pod *corev1.Pod, config *Config) {

	for key, val := range config.ValueConfig.Annotations {
		pod.Annotations[key] = val
	}

	for key, val := range config.ValueConfig.Labels {
		pod.Labels[key] = val
	}

	if _, ok := pod.Labels[sidecarInjectAnnotation]; !ok {
		pod.Labels[sidecarInjectAnnotation] = "true"
	}

	//TODO set status metadata
}

// 将外部envs设置到给定的容器上
// 如果容器中env与外部env有同名env, 则外部的env将会覆盖容器中已存在的env
// example:
//
//		 container1.env: a,b,c,d
//		 newEnvs: b,f
//	  result: containers.env: a,b(newEnvs),c,d
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

	// TODO: 渲染data将从config中获取
	data := TemplateData{}

	// init模板渲染
	// TODO: 如果用户指定了initTemplate, 我们将保留(也许在spec.initContainer中声明部分/整体,也许在annotation中声明)
	var initBuf bytes.Buffer
	if err := config.InitTemplate.Execute(&initBuf, &data); err != nil {
		log.Println("exec template error: ", err.Error())
	}
	templatePod, _ = overlayPod(templatePod, initBuf.Bytes())

	// sidecar模板渲染
	// TODO: 如果用户指定了sidecarTemplate, 我们将保留(也许在spec.Containers中声明部分/整体,也许在annotation中声明)
	var sidecarBuf bytes.Buffer
	if err := config.SidecarTemplate.Execute(&sidecarBuf, &data); err != nil {
		log.Println("exec template error: ", err.Error())
	}
	templatePod, _ = overlayPod(templatePod, sidecarBuf.Bytes())

	// 合并成模板pod
	templateBytes, err := json.Marshal(templatePod)
	if err != nil {
		return nil, err
	}

	retPod, err := overlayPod(copyOriginPod, templateBytes)
	if err != nil {
		return nil, err
	}

	return retPod, nil
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

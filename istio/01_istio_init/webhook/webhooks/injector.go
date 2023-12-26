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

	potentialCheck(originPod, req)

	// 检查是否应该注入pod
	if !checkInject(originPod) {
		return webhook.Allowed("")
	}

	// TODO: 2.获取配置, 还未做, 目前先用静态配置
	// TODO: 实际上, 我们也许要通过K8s的ConfigMap获取sidecar模板, 这可以让我们进行动态获取配置.
	config := NewDefaultConfig()

	// 3. 运行模板
	mergePod, err := runTemplate(originPod, config)
	if err != nil {
		return admission.Allowed("")
	}

	// TODO 4. 后处理(添加元数据, 容器排序, 指标标签, 探针, 环境变量等)

	// 5. 构建patch
	// 使用增量pod与源pod构建patch
	patch, err := createJSONPatch(originPod, mergePod)
	if err != nil {

		return admission.Allowed("")
	}

	return admission.Patched("inject success", patch...)
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
	//TODO: (需要判断用户是否定义了已存在的initContainer, 我们是覆盖还是保留?)
	var initBuf bytes.Buffer
	if err := config.InitTemplate.Execute(&initBuf, &data); err != nil {
		log.Println("exec template error: ", err.Error())
	}
	templatePod, _ = overlayPod(templatePod, initBuf.Bytes())

	// sidecar模板渲染
	// TODO: (需要判断用户是否定义了已存在的sidecarContainer, 我们是覆盖还是保留?)
	var sidecarBuf bytes.Buffer
	if err := config.SidecarTemplate.Execute(&sidecarBuf, &data); err != nil {
		log.Println("exec template error: ", err.Error())
	}
	templatePod, _ = overlayPod(templatePod, sidecarBuf.Bytes())

	// TODO 3. 用户自定义Sidecar模板

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

	m := map[string]any{}
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
		if an == sideacrIgnoreInjectAnnotation {
			return false
		}
	}

	return true
}

package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type InjectorWebhook struct {
	*admission.Webhook
}

func NewInjectorWebhook() *InjectorWebhook {

	ijwh := &InjectorWebhook{}
	injector := NewInjector()

	wh := &webhook.Admission{
		Handler: injector,
	}

	ijwh.Webhook = wh

	return ijwh
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

	ij.injectLogic(pod, req)

	return admission.Allowed("")
}

func (ij *Injector) injectLogic(originPod *corev1.Pod, req admission.Request) admission.Response {

	// 0.潜在检查
	if originPod.Namespace == "" {
		originPod.Namespace = req.Namespace
	}

	// 1.检查是否应该注入pod
	if !checkInject(originPod) {
		return webhook.Allowed("")
	}

	// TODO: 2.获取配置, 还未做, 目前先用静态配置
	// TODO: 实际上, 我们也许要通过K8s的ConfigMap获取sidecar模板, 这可以让我们进行动态获取配置.
	config := NewConfig()

	// 3. 运行模板
	originalPodSpec, _ := json.Marshal(originPod)
	mergePodSpec := runTemplate(originPod, config)

	// 4. 添加额外元数据

	// 5. 构建patch

	return admission.Allowed("")
}

func runTemplate(pod *corev1.Pod, config *Config) (mergePod []byte) {

	// TODO: 我们通过buf追加进行合并模板注入
	// 1.1 创建一个空的Pod
	// 1.2 传入data和init模板返回的数据与空pod合并
	// 1.2 传入data和sidecar模板返回的数据与上一步pod合并
	// 1.3 如果用户有自定义模板则也与上一步pod进行合并

	templatePod := &corev1.Pod{}
	data := TemplateData{}

	// 1. init模板
	var initBuf bytes.Buffer
	if err := config.InitTemplate.Execute(&initBuf, &data); err != nil {
		log.Println("exec template error: ", err.Error())
	}

	// 2. sidecar模板
	// 1. init模板
	var sidecarBuf bytes.Buffer
	if err := config.InitTemplate.Execute(&sidecarBuf, &data); err != nil {
		log.Println("exec template error: ", err.Error())
	}

	// TODO 3. 用户自定义Sidecar模板

	return nil
}

func checkInject(pod *corev1.Pod) bool {

	if pod.Spec.HostNetwork == true {
		return false
	}

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

type TemplateData struct {
}

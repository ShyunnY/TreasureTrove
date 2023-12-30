package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
	"time"
	"webhook/kube"
)

var initPatch = `
spec:
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
      image: docker.io/istio/proxyv2:1.20.1
      imagePullPolicy: IfNotPresent
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
`

var testPod = &corev1.Pod{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:        "nginx",
		Labels:      map[string]string{"kubernetes.io/app": "nginx"},
		Annotations: map[string]string{},
	},
	Spec: corev1.PodSpec{
		NodeName: "demo-node",
		Containers: []corev1.Container{
			{
				Name:  "nginx",
				Image: "nginx:1.14.1",
				Ports: []corev1.ContainerPort{
					{
						Name:          "http",
						ContainerPort: 80,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			},
		},
	},
}

func TestCreatePatch(t *testing.T) {

	mergePod := testPod.DeepCopy()
	mergePod.ObjectMeta.Labels["fishnet.sidecar/inject"] = "true"

	originPod, _ := json.Marshal(testPod)
	merge, _ := json.Marshal(mergePod)

	patch, err := jsonpatch.CreatePatch(originPod, merge)
	if err != nil {
		t.Error(err.Error())
	}

	p, err := json.Marshal(patch)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("patch: %+v\n", string(p))
}

func TestCheckInject(t *testing.T) {

	var pods = []*corev1.Pod{
		func() *corev1.Pod {
			hostNetworkPod := testPod.DeepCopy()
			hostNetworkPod.Spec.HostNetwork = true
			return hostNetworkPod
		}(),
		func() *corev1.Pod {
			ignoreNsPod := testPod.DeepCopy()
			ignoreNsPod.Namespace = "kube-system"
			return ignoreNsPod
		}(),
		func() *corev1.Pod {
			ignoreAnPod := testPod.DeepCopy()
			ignoreAnPod.Annotations = map[string]string{
				sidecarIgnoreInjectAnnotation: "",
			}
			return ignoreAnPod
		}(),
	}

	assert.True(t, checkInject(testPod))
	for _, pod := range pods {
		assert.False(t, checkInject(pod))
	}
}

func TestOverlayPod(t *testing.T) {

	var basePatch = `
apiVersion: v1
kind: Pod
spec:
  containers:
    - image: nginx-1
      imagePullPolicy: Always
      name: nginx
`

	ret := testPod.DeepCopy()
	var err error
	var yamls = []struct {
		overlayYaml []byte
	}{
		{
			overlayYaml: []byte(basePatch),
		},
		{
			overlayYaml: []byte(initPatch),
		},
	}

	for _, tests := range yamls {
		ret, err = overlayPod(ret, tests.overlayYaml)
		assert.NoError(t, err)
		assert.NotNil(t, ret)
		fmt.Printf("ret: %+v\n", ret)
	}

}

func TestInjectLogic(t *testing.T) {

	meshPod := testPod.DeepCopy()
	meshPod.Annotations[sidecarOverwriteAnnotation] = "true"
	ij := Injector{}
	areq := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{Namespace: "mesh"},
	}

	ij.injectLogic(meshPod, areq)

}

func TestWebhookHandler(t *testing.T) {

	client, err := kube.NewClient(kube.ClientConfig{})
	assert.NoError(t, err)

	cliset, err := client.ClientSet()
	assert.NoError(t, err)

	meshPod := testPod.DeepCopy()
	raw, err := json.Marshal(meshPod)
	assert.NoError(t, err)

	areq := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: "default",
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
	}

	injector := NewInjector(cliset, corev1.NamespaceDefault, "inject_config")

	resp := injector.Handle(context.TODO(), areq)
	assert.NotEmpty(t, resp.Patches)

}

func TestInjectorConfigUpdate(t *testing.T) {

	client, err := kube.NewClient(kube.ClientConfig{})
	assert.NoError(t, err)

	cliset, err := client.ClientSet()
	assert.NoError(t, err)

	meshPod := testPod.DeepCopy()
	raw, err := json.Marshal(meshPod)
	assert.NoError(t, err)

	areq := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: "default",
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
	}

	injector := NewInjector(cliset, corev1.NamespaceDefault, "inject_config")

	resp := injector.Handle(context.TODO(), areq)

	assert.NotEmpty(t, resp.Patches)

	time.Sleep(time.Second * 10)

	resp = injector.Handle(context.TODO(), areq)
	assert.NotEmpty(t, resp.Patches)
}

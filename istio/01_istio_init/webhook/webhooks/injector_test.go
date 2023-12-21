package webhooks

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var testPod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "nginx",
		Labels: map[string]string{"kubernetes.io/app": "nginx"},
	},
	Spec: corev1.PodSpec{NodeName: "demo-node"},
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
				sideacrIgnoreInjectAnnotation: "",
			}
			return ignoreAnPod
		}(),
	}

	checkInject(testPod)

	assert.True(t, checkInject(testPod))
	for _, pod := range pods {
		assert.False(t, checkInject(pod))
	}
}

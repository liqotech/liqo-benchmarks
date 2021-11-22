package monitoring

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func preparePodsInformer(factory informers.SharedInformerFactory) {
	informer := factory.Core().V1().Pods().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, object interface{}) {
			handlePodUpdate(object.(*corev1.Pod))
		},
	})
}

func handlePodUpdate(pod *corev1.Pod) {
	klog.V(5).Infof("Received update for pod %q", namespacedName(pod))
	if t, ok := isPodReady(pod); ok {
		M().SetPodReady(pod.GetName(), t)
	}
}

// isPodReady returns true if a pod is ready; false otherwise.
func isPodReady(pod *corev1.Pod) (time.Time, bool) {
	conditions := pod.Status.Conditions
	for i := range conditions {
		if conditions[i].Type == corev1.PodReady {
			if conditions[i].Status == corev1.ConditionTrue {
				return conditions[i].LastTransitionTime.Time, true
			}
			return time.Time{}, false
		}
	}
	return time.Time{}, false
}

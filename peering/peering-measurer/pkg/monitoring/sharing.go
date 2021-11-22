package monitoring

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	liqoconst "github.com/liqotech/liqo/pkg/consts"
	discovery "github.com/liqotech/liqo/pkg/discovery"
)

func preparePodsInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(corev1.SchemeGroupVersion.WithResource("pods")).Informer()

	handle := func(object interface{}, handler func(*corev1.Pod)) {
		if unstruct, ok := object.(*unstructured.Unstructured); ok {
			var pod corev1.Pod
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &pod); err == nil {
				handler(&pod)
				return
			}
		}
		panic("Failed to convert the Pod returned by the informer")
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			handle(object, handleVirtualKubeletCreation)
		},
		UpdateFunc: func(_, object interface{}) {
			handle(object, handleVirtualKubeletUpdate)
		},
	})
}

func prepareNodesInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(corev1.SchemeGroupVersion.WithResource("nodes")).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, object interface{}) {
			if unstruct, ok := object.(*unstructured.Unstructured); ok {
				var node corev1.Node
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &node); err == nil {
					handleNodeUpdate(&node)
					return
				}
			}
			panic("Failed to convert the Node returned by the informer")
		},
	})
}

func handleVirtualKubeletCreation(pod *corev1.Pod) {
	klog.V(5).Infof("Received creation for pod %q", namespacedName(pod))
	if id, ok := pod.Labels[discovery.ClusterIDLabel]; ok {
		M().ClusterID(id).SetVirtualKubeletSetupStartTimestamp(pod.CreationTimestamp.Time)
	}
}

func handleVirtualKubeletUpdate(pod *corev1.Pod) {
	klog.V(5).Infof("Received update for pod %q", namespacedName(pod))
	if id, ok := pod.Labels[discovery.ClusterIDLabel]; ok {
		if pod.Status.Phase == corev1.PodRunning {
			M().ClusterID(id).SetVirtualKubeletSetupEndTimestamp(time.Now())
		}
	}
}

func handleNodeUpdate(node *corev1.Node) {
	klog.V(5).Infof("Received update for node %q", namespacedName(node))
	if id, ok := node.Labels[liqoconst.RemoteClusterID]; ok {
		if t, ok := isNodeReady(node); ok {
			M().ClusterID(id).SetNodeReady(t)
		}
	}
}

// IsPodReady returns true if a pod is ready; false otherwise.
func isNodeReady(node *corev1.Node) (time.Time, bool) {
	conditions := node.Status.Conditions
	for i := range conditions {
		if conditions[i].Type == corev1.NodeReady {
			if conditions[i].Status == corev1.ConditionTrue {
				return conditions[i].LastTransitionTime.Time, true
			}
			return time.Time{}, false
		}
	}
	return time.Time{}, false
}

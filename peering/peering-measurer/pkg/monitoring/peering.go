package monitoring

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
	discovery "github.com/liqotech/liqo/pkg/discovery"
)

func prepareForeignClusterInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(discoveryv1alpha1.ForeignClusterGroupVersionResource).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, object interface{}) {
			if unstruct, ok := object.(*unstructured.Unstructured); ok {
				var fc discoveryv1alpha1.ForeignCluster
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &fc); err == nil {
					handleForeignClusterUpdate(&fc)
					return
				}
			}
			panic("Failed to convert the ForeignCluster returned by the informer")
		},
	})
}

func prepareSecretsInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(corev1.SchemeGroupVersion.WithResource("secrets")).Informer()

	handle := func(object interface{}, handler func(*corev1.Secret)) {
		if unstruct, ok := object.(*unstructured.Unstructured); ok {
			var se corev1.Secret
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &se); err == nil {
				handler(&se)
				return
			}
		}
		panic("Failed to convert the Secret returned by the informer")
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			handle(object, handleSecretCreation)
		},
		UpdateFunc: func(_, object interface{}) {
			handle(object, handleSecretUpdate)
		},
	})
}

func handleForeignClusterUpdate(fc *discoveryv1alpha1.ForeignCluster) {
	klog.V(5).Infof("Received update for ForeignCluster %q", namespacedName(fc))
	id := fc.Spec.ClusterIdentity.ClusterID
	if id != "" {
		M().ClusterID(id).SetPeeringStartTimestamp(fc.GetCreationTimestamp().Time)
	}
}

func handleSecretCreation(se *corev1.Secret) {
	if se.Name != "liqo-remote-certificate" {
		return
	}

	klog.V(5).Infof("Received creation for Secret %q", namespacedName(se))
	id, ok := se.GetLabels()[discovery.ClusterIDLabel]
	if !ok {
		klog.Warning("Secret %q misses the remote ID label", namespacedName(se))
		return
	}

	M().ClusterID(id).SetAuthenticationIncomingEndTimestamp(se.GetCreationTimestamp().Time)
}

func handleSecretUpdate(se *corev1.Secret) {
	klog.V(5).Infof("Received update for Secret %q", namespacedName(se))

	if value, ok := se.GetLabels()[certificateAvailableLabel]; !ok || value != "true" {
		klog.V(5).Infof("Secret %q does not correspond to a valid identity", namespacedName(se))
		return
	}

	id, ok := se.GetLabels()[discovery.ClusterIDLabel]
	if !ok {
		klog.Warning("Secret %q misses the remote ID label", namespacedName(se))
		return
	}

	M().ClusterID(id).SetAuthenticationOutgoingEndTimestamp(time.Now())
}

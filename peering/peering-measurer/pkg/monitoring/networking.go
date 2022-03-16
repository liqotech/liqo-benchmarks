package monitoring

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	netv1alpha1 "github.com/liqotech/liqo/apis/net/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
)

func prepareNetworkConfigInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(netv1alpha1.NetworkConfigGroupVersionResource).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			if unstruct, ok := object.(*unstructured.Unstructured); ok {
				var nc netv1alpha1.NetworkConfig
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &nc); err == nil {
					handleNetworkConfigCreation(&nc)
					return
				}
			}
			panic("Failed to convert the NetworkConfig returned by the informer")
		},
	})
}

func prepareTunnelEndpointInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(netv1alpha1.TunnelEndpointGroupVersionResource).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, object interface{}) {
			if unstruct, ok := object.(*unstructured.Unstructured); ok {
				var te netv1alpha1.TunnelEndpoint
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &te); err == nil {
					handleTunnelEndpointProcessed(&te)
					return
				}
			}
			panic("Failed to convert the TunnelEndpoint returned by the informer")
		},
	})
}

func handleNetworkConfigCreation(nc *netv1alpha1.NetworkConfig) {
	klog.V(5).Infof("Received creation for NetworkConfig %q", namespacedName(nc))

	// Discard remote NetworkConfig
	if value, ok := nc.GetLabels()[consts.ReplicationRequestedLabel]; !ok || value != replicationLabelValueLocal {
		klog.V(5).Infof("Skipping remote NetworkConfig %q", namespacedName(nc))
		return
	}

	id := nc.Spec.RemoteCluster.ClusterID
	M().ClusterID(id).SetNetworkSetupStartTimestamp(nc.GetCreationTimestamp().Time)
}

func handleTunnelEndpointProcessed(te *netv1alpha1.TunnelEndpoint) {
	klog.V(5).Infof("Received update for TunnelEndpoint %q", namespacedName(te))

	if te.Status.Connection.Status == netv1alpha1.Connected {
		id := te.Spec.ClusterID
		M().ClusterID(id).SetNetworkSetupEndTimestamp(time.Now())
	}
}

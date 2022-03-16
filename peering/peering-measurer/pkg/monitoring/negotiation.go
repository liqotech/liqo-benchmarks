package monitoring

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
	sharingv1alpha1 "github.com/liqotech/liqo/apis/sharing/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
)

func prepareResourceRequestInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(discoveryv1alpha1.ResourceRequestGroupVersionResource).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			if unstruct, ok := object.(*unstructured.Unstructured); ok {
				var rr discoveryv1alpha1.ResourceRequest
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &rr); err == nil {
					handleResourceRequestCreation(&rr)
					return
				}
			}
			panic("Failed to convert the ResourceRequest returned by the informer")
		},
	})
}

func prepareResourceOfferInformer(factory dynamicinformer.DynamicSharedInformerFactory) {
	informer := factory.ForResource(sharingv1alpha1.ResourceOfferGroupVersionResource).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, object interface{}) {
			if unstruct, ok := object.(*unstructured.Unstructured); ok {
				var ro sharingv1alpha1.ResourceOffer
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, &ro); err == nil {
					handleResourceOfferAcceptance(&ro)
					return
				}
			}
			panic("Failed to convert the ResourceOffer returned by the informer")
		},
	})
}

func handleResourceRequestCreation(rr *discoveryv1alpha1.ResourceRequest) {
	klog.V(5).Infof("Received creation for ResourceRequest %q", namespacedName(rr))

	// Discard remote ResourceRequests
	if value, ok := rr.GetLabels()[consts.ReplicationRequestedLabel]; !ok || value != replicationLabelValueLocal {
		klog.V(5).Infof("Skipping remote ResourceRequest %q", namespacedName(rr))
		return
	}

	id, ok := rr.GetLabels()[consts.ReplicationDestinationLabel]
	if !ok {
		klog.Warning("ResourceRequest %q misses the remote ID label", namespacedName(rr))
		return
	}
	M().ClusterID(id).SetResourceNegotiationStartTimestamp(rr.GetCreationTimestamp().Time)
}

func handleResourceOfferAcceptance(ro *sharingv1alpha1.ResourceOffer) {
	klog.V(5).Infof("Received creation for ResourceOffer %q", namespacedName(ro))

	// Discard local ResourceOffers
	if value, ok := ro.GetLabels()[consts.ReplicationRequestedLabel]; !ok || value != replicationLabelValueRemote {
		klog.V(5).Infof("Skipping remote ResourceOffer %q", namespacedName(ro))
		return
	}

	if ro.Status.Phase == sharingv1alpha1.ResourceOfferAccepted {
		id := ro.Spec.ClusterId
		M().ClusterID(id).SetResourceNegotiationEndTimestamp(time.Now())
	}
}

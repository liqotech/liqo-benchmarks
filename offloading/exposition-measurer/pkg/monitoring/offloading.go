package monitoring

import (
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

func prepareEndpointSlicesInformer(factory informers.SharedInformerFactory) {
	informer := factory.Discovery().V1beta1().EndpointSlices().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			handleEndpointSlice(object.(*discoveryv1beta1.EndpointSlice))
		},
		UpdateFunc: func(_, object interface{}) {
			handleEndpointSlice(object.(*discoveryv1beta1.EndpointSlice))
		},
	})
}

func handleEndpointSlice(epslice *discoveryv1beta1.EndpointSlice) {
	klog.V(5).Infof("Received event for epslice %q", namespacedName(epslice))

	var ready uint
	for _, ep := range epslice.Endpoints {
		if pointer.BoolDeref(ep.Conditions.Ready, false) {
			ready++
		}
	}

	M().SetEndpointSliceReady(epslice.GetName(), ready)
}

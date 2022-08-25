// Package monitoring wraps the informer logic used to observe the peering process.
package monitoring

import (
	"context"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo-benchmarks/offloading/exposition-measurer/pkg/measurer"
)

const resyncPeriod = 0

var M func() *measurer.Measurer

func Start(ctx context.Context, client kubernetes.Interface, namespace string, expected uint, completion chan<- struct{}) {
	klog.V(1).Info("Configuring the measurer")
	m := measurer.NewMeasurer(expected, completion)
	M = func() *measurer.Measurer {
		return m
	}

	klog.V(1).Info("Configuring the informers")
	factory := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod, informers.WithNamespace(namespace))

	prepareEndpointSlicesInformer(factory)

	klog.V(1).Info("Starting the informer factory")
	factory.Start(ctx.Done())

	klog.V(1).Info("Waiting for cache sync")
	factory.WaitForCacheSync(ctx.Done())
	klog.V(1).Info("Cache correctly sync'ed")
}

func namespacedName(obj interface{}) string {
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	return key
}

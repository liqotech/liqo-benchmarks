// Package monitoring wraps the informer logic used to observe the peering process.
package monitoring

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo-benchmarks/peering/peering-measurer/pkg/measurer"
)

const resyncPeriod = 0

var M func() *measurer.GlobalMeasurer

// Redefined here, as part of an internal Liqo package.
const (
	replicationLabelValueLocal  = "true"
	replicationLabelValueRemote = "false"
	certificateAvailableLabel   = "discovery.liqo.io/certificate-available"
)

func Start(ctx context.Context, client dynamic.Interface, completion chan<- struct{}) {
	klog.V(1).Info("Configuring the measurer")
	m := measurer.NewMeasurer(completion)
	M = func() *measurer.GlobalMeasurer {
		return m
	}

	klog.V(1).Info("Configuring the informers")
	factory := dynamicinformer.NewDynamicSharedInformerFactory(client, resyncPeriod)

	prepareForeignClusterInformer(factory)
	prepareSecretsInformer(factory)
	prepareResourceRequestInformer(factory)
	prepareResourceOfferInformer(factory)
	prepareNetworkConfigInformer(factory)
	prepareTunnelEndpointInformer(factory)
	preparePodsInformer(factory)
	prepareNodesInformer(factory)

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

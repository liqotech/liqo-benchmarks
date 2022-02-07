package forge

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const Port = 80
const resyncPeriod = 0

func RetrieveServiceIP(ctx context.Context, client kubernetes.Interface, namespace string, create bool) (string, error) {
	if create {
		return CreateService(ctx, client, namespace)
	}

	return WatchService(ctx, client, namespace)
}

func CreateService(ctx context.Context, client kubernetes.Interface, namespace string) (string, error) {
	klog.V(2).Infof("Creating exposition service")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "exposition", Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    []corev1.ServicePort{{Port: Port}},
			Selector: map[string]string{"app.kubernetes.io/part-of": "benchmarks"},
		},
	}

	created, err := client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("Failed to create service %q: %v", klog.KObj(service), err)
		return "", err
	}

	klog.V(2).Info("Exposition service created correctly")
	return created.Spec.ClusterIP, nil
}

func WatchService(ctx context.Context, client kubernetes.Interface, namespace string) (string, error) {
	klog.V(2).Infof("Creating exposition service")

	ipch := make(chan string, 1)
	factory := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod, informers.WithNamespace(namespace))
	informer := factory.Core().V1().Services().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(object interface{}) {
			service := object.(*corev1.Service)
			klog.V(5).Infof("Received event for service %q", namespacedName(service))
			if service.Name == "exposition" {
				ipch <- service.Spec.ClusterIP
			}
		},
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	klog.V(1).Info("Starting the informer factory")
	factory.Start(ctx.Done())

	klog.V(1).Info("Waiting for cache sync")
	factory.WaitForCacheSync(ctx.Done())
	klog.V(1).Info("Cache correctly sync'ed")

	select {
	case ip := <-ipch:
		return ip, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func namespacedName(obj interface{}) string {
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	return key
}

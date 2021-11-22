// Package creation implements the logic required to create the benchmark objects.
package creation

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func Service(ctx context.Context, client kubernetes.Interface, namespace string) error {
	klog.V(2).Infof("Creating exposition service")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "exposition", Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    []corev1.ServicePort{{Port: 80}},
			Selector: map[string]string{"app.kubernetes.io/part-of": "benchmarks"},
		},
	}

	if _, err := client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		klog.Errorf("Failed to create service %q: %v", klog.KObj(service), err)
		return err
	}

	klog.V(2).Info("Exposition service created correctly")
	return nil
}

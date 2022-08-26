// Package creation implements the logic required to create the benchmark objects.
package creation

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	offloadingv1alpha1 "github.com/liqotech/liqo/apis/offloading/v1alpha1"
	"github.com/liqotech/liqo/pkg/consts"
)

func Namespace(ctx context.Context, cl kubernetes.Interface, name string) error {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}

	klog.V(4).Infof("Creating Namespace %q", klog.KObj(namespace))
	if _, err := cl.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Errorf("Failed to create Namespace %q: %v", klog.KObj(namespace), err)
		return err
	}

	klog.V(4).Infof("Namespace %q successfully created/updated", klog.KObj(namespace))
	return nil
}

func NamespaceOffloading(ctx context.Context, cl client.Client, namespace string) error {
	offloading := &offloadingv1alpha1.NamespaceOffloading{
		ObjectMeta: metav1.ObjectMeta{Name: consts.DefaultNamespaceOffloadingName, Namespace: namespace},
		Spec: offloadingv1alpha1.NamespaceOffloadingSpec{
			NamespaceMappingStrategy: offloadingv1alpha1.EnforceSameNameMappingStrategyType,
			PodOffloadingStrategy:    offloadingv1alpha1.LocalAndRemotePodOffloadingStrategyType,
			ClusterSelector:          corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{}},
		},
	}

	klog.V(4).Infof("Creating NamespaceOffloading %q", klog.KObj(offloading))
	if err := cl.Create(ctx, offloading); err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Errorf("Failed to create NamespaceOffloading %q: %v", klog.KObj(offloading), err)
		return err
	}

	klog.V(4).Infof("NamespaceOffloading %q successfully created", klog.KObj(offloading))
	return nil
}

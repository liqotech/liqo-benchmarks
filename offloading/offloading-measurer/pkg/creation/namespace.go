// Package creation implements the logic required to create the benchmark objects.
package creation

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	liqoconst "github.com/liqotech/liqo/pkg/consts"
)

func Namespace(ctx context.Context, client kubernetes.Interface, name string, liqoEnable bool) error {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}

	if liqoEnable {
		namespace.SetLabels(map[string]string{
			liqoconst.EnablingLiqoLabel: strconv.FormatBool(true),
		})
	}

	klog.V(4).Infof("Creating Namespace %q", klog.KObj(namespace))
	if _, err := client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil && !kerrors.IsAlreadyExists(err) {
		klog.Errorf("Failed to create Namespace %q: %v", klog.KObj(namespace), err)
		return err
	}

	klog.V(4).Infof("Namespace %q successfully created/updated", klog.KObj(namespace))
	return nil
}

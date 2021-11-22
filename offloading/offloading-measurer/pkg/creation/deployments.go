// Package creation implements the logic required to create the benchmark objects.
package creation

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

func Deployments(ctx context.Context, client kubernetes.Interface, namespace, affinity string, deploy, replicas uint) error {
	klog.V(2).Infof("Creating %v deployments with %v replicas each", deploy, replicas)

	for i := uint(0); i < deploy; i++ {
		name := fmt.Sprintf("deployment-%05d", i)
		if err := deployment(ctx, client, namespace, name, affinity, replicas); err != nil {
			klog.Errorf("Failed to create deployments: %w", err)
			return err
		}
	}

	klog.V(2).Info("All deployments correctly created")
	return nil
}

func deployment(ctx context.Context, client kubernetes.Interface, namespace, name, affinity string, replicas uint) error {
	labels := map[string]string{"app.kubernetes.io/name": name, "app.kubernetes.io/part-of": "benchmarks"}

	var nodeaffinity *corev1.Affinity
	if affinity != "" {
		affnty := strings.Split(affinity, "=")
		nodeaffinity = &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{
				{Key: affnty[0], Operator: corev1.NodeSelectorOpIn, Values: affnty[1:]},
			}}},
		}}}
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(int32(replicas)), Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: map[string]string{"multicluster.admiralty.io/elect": ""}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "pause", Image: "docker.io/rancher/pause:3.1"}},
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
						LabelSelector: metav1.SetAsLabelSelector(labels), TopologyKey: corev1.LabelHostname,
						MaxSkew: 3, WhenUnsatisfiable: corev1.ScheduleAnyway,
					}},
					Tolerations: []corev1.Toleration{{
						Key: "node-role.kubernetes.io/hollow", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule,
					}},
					Affinity: nodeaffinity,
				},
			},
		},
	}

	klog.V(4).Infof("Creating deployment %q with %v replicas", klog.KObj(deploy), replicas)
	if _, err := client.AppsV1().Deployments(namespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		klog.Errorf("Failed to create deployment %q: %v", klog.KObj(deploy), err)
		return err
	}

	klog.V(4).Infof("Deployment %q successfully created", klog.KObj(deploy))
	return nil
}

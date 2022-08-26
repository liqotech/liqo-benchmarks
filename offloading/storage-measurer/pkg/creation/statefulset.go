package creation

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

func StatefulSet(ctx context.Context, client kubernetes.Interface, namespace, affinity, storageClass, volumeSize string, replicas uint) error {
	labels := map[string]string{"app.kubernetes.io/name": "benchmark", "app.kubernetes.io/part-of": "benchmarks"}

	var nodeaffinity *corev1.Affinity
	if affinity != "" {
		affnty := strings.Split(affinity, "=")
		nodeaffinity = &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{
				{Key: affnty[0], Operator: corev1.NodeSelectorOpIn, Values: affnty[1:]},
			}}},
		}}}
	}

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "benchmark", Namespace: namespace, Labels: labels},
		Spec: appsv1.StatefulSetSpec{
			Replicas: pointer.Int32(int32(replicas)), PodManagementPolicy: appsv1.ParallelPodManagement, Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "pause", Image: "docker.io/rancher/pause:3.1"}},
					Affinity:   nodeaffinity,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "benchmark"},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources:        corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(volumeSize)}},
					StorageClassName: &storageClass,
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				},
			}},
		},
	}

	klog.V(4).Infof("Creating statefulset %q with %v replicas", klog.KObj(statefulset), replicas)
	if _, err := client.AppsV1().StatefulSets(namespace).Create(ctx, statefulset, metav1.CreateOptions{}); err != nil {
		klog.Errorf("Failed to create statefulset %q: %v", klog.KObj(statefulset), err)
		return err
	}

	klog.V(4).Infof("Statefulset %q successfully created", klog.KObj(statefulset))
	return nil
}

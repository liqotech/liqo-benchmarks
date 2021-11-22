package monitoring

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

func RetrieveTargetPodName(ctx context.Context, client kubernetes.Interface, selector string) (namespace, name string, err error) {
	pods, err := client.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("Failed to retrieve target pod name: %v", err)
		return "", "", err
	}

	if len(pods.Items) != 1 {
		klog.Errorf("Incorrect number of target pods: %v", len(pods.Items))
		return "", "", fmt.Errorf("incorrect number of target pods: %v", len(pods.Items))
	}

	return pods.Items[0].GetNamespace(), pods.Items[0].GetName(), nil
}

func RetrieveMetrics(ctx context.Context, client metricsv1beta1.PodMetricsInterface, name string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(10 * time.Second):
			if err := retrieveMetrics(ctx, client, name); err != nil {
				return err
			}
		}
	}
}

func retrieveMetrics(ctx context.Context, client metricsv1beta1.PodMetricsInterface, name string) error {
	metrics, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.Error("Failed to retrieve pod metrics: %v", err)
		return err
	}

	var cpu, memory int64
	for _, container := range metrics.Containers {
		cpu += container.Usage.Cpu().ScaledValue(resource.Micro)
		memory += container.Usage.Memory().Value()
	}

	M().Metrics(metrics.Timestamp.Time.Add(-metrics.Window.Duration), metrics.Timestamp.Time, cpu, memory)
	return nil
}

package metrics

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func Nodes(ctx context.Context, client kubernetes.Interface, selector string, expected uint) []string {
	klog.V(2).Infof("Retrieving nodes with selector %v (expected: %v)", selector, expected)

	inner := func() []string {
		nodes, err := nodes(ctx, client, selector)
		if err != nil {
			klog.Errorf("Failed to retrieve nodes: %v", err)
		} else {
			klog.V(2).Infof("Found %v nodes, expected: %v", len(nodes), expected)
			if len(nodes) >= int(expected) {
				return nodes
			}
		}

		klog.V(2).Info("Sleeping 10 seconds before retrying...")
		return nil
	}

	if nodes := inner(); nodes != nil {
		return nodes
	}

	for {
		select {
		case <-ctx.Done():
			klog.Info("Context canceled, aborting")
			return nil
		case <-time.After(10 * time.Second):
			if nodes := inner(); nodes != nil {
				return nodes
			}
		}
	}
}

func nodes(ctx context.Context, client kubernetes.Interface, selector string) ([]string, error) {
	klog.V(4).Infof("Retrieving nodes with selector %v", selector)
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("Failed to retrieve nodes: %v", err)
		return nil, err
	}

	names := make([]string, 0)
	for i := range nodes.Items {
		node := nodes.Items[i]
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				names = append(names, node.GetName())
			}
		}
	}

	klog.V(4).Infof("Successfully retrieved %v nodes, %v ready", len(nodes.Items), len(names))
	return names, nil
}

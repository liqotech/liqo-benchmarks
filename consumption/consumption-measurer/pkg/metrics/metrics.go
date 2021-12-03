// Package metrics implements the logic to collect and process consumption metrics.
package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
)

const endpoint = "unix:///run/containerd/containerd.sock"

func NewClient(ctx context.Context) (pb.RuntimeServiceClient, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	klog.V(2).Infof("Establishing a connection to %v", endpoint)
	conn, err := grpc.DialContext(ctx, endpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		klog.Errorf("Failed to establish a connection to %v: %v", endpoint, err)
		return nil, err
	}
	klog.V(2).Infof("Connection successfully established to %v", endpoint)

	return pb.NewRuntimeServiceClient(conn), nil
}

func Retrieve(ctx context.Context, client pb.RuntimeServiceClient) error {
	klog.V(4).Infof("Retrieving stats")

	request := &pb.ListContainerStatsRequest{Filter: &pb.ContainerStatsFilter{}}
	klog.V(5).Infof("Request: %v", request)
	response, err := client.ListContainerStats(ctx, request)
	klog.V(5).Infof("Response: %v", response)
	if err != nil {
		klog.Errorf("Failed to retrieve stats: %v", err)
		return err
	}

	Decode(response)
	return nil
}

func Decode(response *pb.ListContainerStatsResponse) {
	klog.V(4).Infof("Successfully retrieved stats for %v containers", len(response.GetStats()))
	for _, stat := range response.GetStats() {
		namespace := stat.Attributes.Labels["io.kubernetes.pod.namespace"]
		pod := stat.Attributes.Labels["io.kubernetes.pod.name"]
		container := stat.Attributes.Labels["io.kubernetes.container.name"]
		klog.V(4).Infof("Decoding stat for container %v (namespace=%v, pod=%v, container=%v)", stat.Attributes.Id, namespace, pod, container)

		if !strings.HasPrefix(namespace, "liqo") || strings.HasPrefix(pod, "svclb") {
			continue
		}

		if stat.GetCpu() == nil || stat.GetMemory() == nil {
			klog.V(4).Infof("Skipping stat for container %v (namespace=%v, pod=%v, container=%v) as incomplete",
				stat.Attributes.Id, namespace, pod, container)
			continue
		}

		fmt.Printf("%v,%v,%v,%v\n", "container_cpu_usage_nanoseconds_total", pod+container,
			stat.GetCpu().GetTimestamp(), stat.GetCpu().GetUsageCoreNanoSeconds().GetValue())
		fmt.Printf("%v,%v,%v,%v\n", "container_memory_working_set_bytes", pod+container,
			stat.GetMemory().GetTimestamp(), stat.GetMemory().GetWorkingSetBytes().GetValue())
	}
}

// Package metrics implements the logic to collect and process consumption metrics.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func RetrieveAll(ctx context.Context, client kubernetes.Interface, nodes []string) error {
	klog.V(4).Infof("Retrieving metrics from %v nodes", len(nodes))
	for _, node := range nodes {
		if err := Retrieve(ctx, client, node); err != nil {
			return err
		}
	}
	klog.V(4).Infof("All metrics successfully retrieved")
	return nil
}

func Retrieve(ctx context.Context, client kubernetes.Interface, node string) error {
	klog.V(4).Infof("Retrieving metrics from node %v", node)
	response, err := client.CoreV1().RESTClient().Get().Resource("nodes").
		Name(node).SubResource("proxy", "metrics", "cadvisor").DoRaw(ctx)
	if err != nil {
		klog.Errorf("Failed to retrieve metrics for node %v: %v", node, err)
		return err
	}

	klog.V(4).Infof("Metrics successfully retrieved from node %v", node)
	Decode(string(response))
	return nil
}

func Decode(data string) {
	decoder := expfmt.SampleDecoder{
		Dec:  expfmt.NewDecoder(strings.NewReader(data), expfmt.FmtText),
		Opts: &expfmt.DecodeOptions{},
	}

	for {
		var v model.Vector
		if err := decoder.Decode(&v); err != nil {
			if errors.Is(err, io.EOF) {
				// Expected loop termination condition.
				return
			}
			klog.Warningf("Failed decoding entry (skipping): %v", err)
			continue
		}

		for _, metric := range v {
			if !strings.HasPrefix(string(metric.Metric["namespace"]), "liqo") ||
				!(strings.HasPrefix(pod(metric), "liqo") || strings.HasPrefix(pod(metric), "virtual-kubelet")) {
				continue
			}

			name := string(metric.Metric[model.MetricNameLabel])
			switch name {
			case "container_cpu_usage_seconds_total":
				if metric.Metric["image"] != "" {
					continue
				}
				fmt.Printf("%v,%v,%v,%v\n", name, pod(metric), timestamp(metric), value(metric, 6))

			case "container_memory_working_set_bytes":
				if metric.Metric["image"] != "" {
					continue
				}
				fmt.Printf("%v,%v,%v,%v\n", name, pod(metric), timestamp(metric), value(metric, 0))

			case "container_network_receive_bytes_total":
				if !strings.HasPrefix(pod(metric), "virtual-kubelet") {
					continue
				}
				fmt.Printf("%v,%v,%v,%v\n", name, pod(metric), timestamp(metric), value(metric, 0))

			case "container_network_transmit_bytes_total":
				if !strings.HasPrefix(pod(metric), "virtual-kubelet") {
					continue
				}
				fmt.Printf("%v,%v,%v,%v\n", name, pod(metric), timestamp(metric), value(metric, 0))
			}
		}
	}
}

func pod(metric *model.Sample) string {
	return string(metric.Metric["pod"])
}

func timestamp(metric *model.Sample) string {
	return strconv.FormatInt(metric.Timestamp.Time().Unix(), 10)
}

func value(metric *model.Sample, precision int) string {
	return strconv.FormatFloat(float64(metric.Value), 'f', precision, 64)
}

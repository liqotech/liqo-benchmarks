package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

	"github.com/liqotech/liqo-benchmarks/peering/offloading-measurer/pkg/creation"
	"github.com/liqotech/liqo-benchmarks/peering/offloading-measurer/pkg/monitoring"
)

func main() {
	// Configure the flags.
	namespace := flag.String("namespace", "offloading-benchmark", "The name of the namespace where the benchmark is executed")
	affinity := flag.String("affinity", "", "The node affinity label")
	metricsTarget := flag.String("metrics-target", "", "The label selector of the target pod to collect the metrics (skipped if empty)")
	pods := flag.Uint("pods", 1, "The number of replicas per deployment")
	deployments := flag.Uint("deployments", 1, "The number of deployments to be created")
	liqoEnable := flag.Bool("enable-liqo-offloading", false, "Whether to label the namespace to enable liqo offloading")

	klog.InitFlags(nil)
	flag.Parse()

	// Initialize the client
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	client, metrics := prepareClients()

	// Start collecting the pod metrics
	if *metricsTarget != "" {
		namespace, name, err := monitoring.RetrieveTargetPodName(ctx, client, *metricsTarget)
		if err != nil {
			os.Exit(1)
		}

		go func() {
			if err := monitoring.RetrieveMetrics(ctx, metrics.PodMetricses(namespace), name); err != nil {
				os.Exit(1)
			}
		}()
	}

	// Create the namespace
	if err := creation.Namespace(ctx, client, *namespace, *liqoEnable); err != nil {
		os.Exit(1)
	}

	for i := 5; i > 0; i-- {
		klog.V(1).Infof("Waiting for namespace offloading initialization to (hopefully) complete (%v)", i)
		time.Sleep(time.Second)
	}

	// Configure the informers
	expected := (*pods) * (*deployments)
	completion := make(chan struct{})
	monitoring.Start(ctx, client, *namespace, expected, completion)

	// Create the Deployments
	monitoring.M().SetOffloadingStartTimestamp(time.Now())
	if err := creation.Deployments(ctx, client, *namespace, *affinity, *deployments, *pods); err != nil {
		os.Exit(1)
	}

	// Wait for pods readiness
	klog.V(1).Info("Waiting for pods to become ready")
	select {
	case <-ctx.Done():
		break
	case <-completion:
		klog.V(1).Info("All pods correctly ready")
		cancel()
	}

	// Print the output
	fmt.Println()
	monitoring.M().ToCSV(os.Stdout)
	fmt.Println()
	monitoring.M().ToTable(os.Stdout)
	fmt.Println()

	klog.V(1).Info("Everything completed. Bye!")
}

func prepareClients() (kubernetes.Interface, metricsv1beta1.MetricsV1beta1Interface) {
	klog.V(4).Infof("Loading kubernetes client")
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Unable to create client config: %s", err)
		os.Exit(1)
	}

	config.QPS = 10000
	config.Burst = 10000
	client := kubernetes.NewForConfigOrDie(config)
	metrics := metricsv1beta1.NewForConfigOrDie(config)
	klog.V(4).Infof("Loaded kubernetes clients")
	return client, metrics
}

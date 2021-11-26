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

	"github.com/liqotech/liqo-benchmarks/consumption/consumption-measurer/pkg/metrics"
)

func main() {
	// Configure the flags.
	interval := flag.Duration("interval", 5*time.Second, "The scraping interval")
	selector := flag.String("selector", "node.kubernetes.io/instance-type=k3s", "The selector to identify the target nodes")
	expected := flag.Uint("expected", 1, "The number of expected nodes")
	klog.InitFlags(nil)
	flag.Parse()

	// Initialize the client
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	client := prepareClient()

	// Retrieve the nodes to scrape
	klog.V(1).Info("Retrieving the nodes to scrape")
	nodes := metrics.Nodes(ctx, client, *selector, *expected)

	klog.V(1).Info("Starting scraping metrics...")
	ticker := time.NewTicker(*interval)

	fmt.Printf("metric,pod,timestamp,value\n")
outer:
	for {
		select {
		case <-ticker.C:
			if err := metrics.RetrieveAll(ctx, client, nodes); err != nil {
				os.Exit(1)
			}
		case <-ctx.Done():
			ticker.Stop()
			cancel()
			break outer
		}
	}

	klog.V(1).Info("Everything completed. Bye!")
}

func prepareClient() kubernetes.Interface {
	klog.V(4).Infof("Loading kubernetes client")
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Unable to create client config: %s", err)
		os.Exit(1)
	}

	// Make sure the measurer is not throttled.
	config.QPS = 10000
	config.Burst = 10000
	client := kubernetes.NewForConfigOrDie(config)
	klog.V(4).Infof("Loaded kubernetes client")
	return client
}

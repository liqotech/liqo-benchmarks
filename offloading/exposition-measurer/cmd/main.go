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

	"github.com/liqotech/liqo-benchmarks/offloading/exposition-measurer/pkg/creation"
	"github.com/liqotech/liqo-benchmarks/offloading/exposition-measurer/pkg/monitoring"
)

func main() {
	// Configure the flags.
	namespace := flag.String("namespace", "offloading-benchmark", "The name of the namespace where the benchmark is executed")
	svccreate := flag.Bool("create-service", true, "Whether to create the service or not")
	endpoints := flag.Uint("endpoints", 1, "The number of expected endpoints (i.e. pods)")

	klog.InitFlags(nil)
	flag.Parse()

	// Initialize the client
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	client := prepareClient()

	// Configure the informers
	completion := make(chan struct{})
	monitoring.Start(ctx, client, *namespace, *endpoints, completion)

	if *svccreate {
		// Create the service
		monitoring.M().SetExpositionStartTimestamp(time.Now())
		if err := creation.Service(ctx, client, *namespace); err != nil {
			os.Exit(1)
		}
	}

	// Wait for endpoints readiness
	klog.V(1).Info("Waiting for endpoints to become ready")
	select {
	case <-ctx.Done():
		break
	case <-completion:
		klog.V(1).Info("All endpoints correctly ready")
		cancel()
	}

	// Print the output
	fmt.Println()
	monitoring.M().Output(os.Stdout)
	fmt.Println()

	klog.V(1).Info("Everything completed. Bye!")
}

func prepareClient() kubernetes.Interface {
	klog.V(4).Infof("Loading kubernetes client")
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Unable to create client config: %s", err)
		os.Exit(1)
	}

	config.QPS = 10000
	config.Burst = 10000
	client := kubernetes.NewForConfigOrDie(config)
	klog.V(4).Infof("Loaded kubernetes clients")
	return client
}

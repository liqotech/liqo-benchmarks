package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo-benchmarks/peering/peering-measurer/pkg/creation"
	"github.com/liqotech/liqo-benchmarks/peering/peering-measurer/pkg/monitoring"
	"github.com/liqotech/liqo-benchmarks/peering/peering-measurer/pkg/service"
)

const ChannelBufferSize = 3

func main() {
	// Configure the flags.
	serviceName := flag.String("service-name", "", "The name of the service used to retrieve the remote endpoints")
	expectedEndpoints := flag.Uint64("expected", 1, "The number of remote endpoints to retrieve before starting")
	extraWait := flag.Duration("extra-wait", 0*time.Second, "The amount of time waited before the benchmark is started")
	klog.InitFlags(nil)
	flag.Parse()

	if *serviceName == "" {
		klog.Error("The service name must be specified.")
		os.Exit(1)
	}

	// Initialize the client
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	client := prepareClient()

	// Configure the informers
	completed := 0
	completion := make(chan struct{}, ChannelBufferSize)
	monitoring.Start(ctx, client, completion)

	// Retrieve the target IP addresses
	ips := service.RetrieveTargetIPs(ctx, *serviceName, *expectedEndpoints)

	klog.V(2).Infof("Waiting additional %v for the testbed to be completely ready", extraWait)
	waitForOrExit(ctx, *extraWait)

	// Create the ForeignClusters
	monitoring.M().SetPeeringStartTimestamp(time.Now())
	if err := creation.ForeignClusters(ctx, client, ips); err != nil {
		os.Exit(1)
	}

	// Wait for node readiness
	klog.V(1).Info("Waiting for nodes to become ready")
outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case <-completion:
			completed++
			if completed == len(ips) {
				klog.V(1).Info("All nodes correctly ready")
				cancel()
				break outer
			}
		}
	}

	// Print the output
	fmt.Println()
	utilruntime.Must(monitoring.M().ToCSV(os.Stdout))
	fmt.Println()
	monitoring.M().ToTable(os.Stdout)
	fmt.Println()

	klog.V(1).Info("Everything completed. Bye!")
}

func prepareClient() dynamic.Interface {
	klog.V(4).Infof("Loading dynamic client")
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Unable to create client config: %s", err)
		os.Exit(1)
	}

	// Make sure the measurer is not throttled.
	config.QPS = 10000
	config.Burst = 10000
	client := dynamic.NewForConfigOrDie(config)
	klog.V(4).Infof("Loaded dynamic client")
	return client
}

func waitForOrExit(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
		klog.Info("Context canceled, exiting...")
		os.Exit(0)
	case <-time.After(d):
		break
	}
}

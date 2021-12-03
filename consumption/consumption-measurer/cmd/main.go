package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	"github.com/liqotech/liqo-benchmarks/consumption/consumption-measurer/pkg/metrics"
)

func main() {
	// Configure the flags.
	interval := flag.Duration("interval", 1*time.Second, "The scraping interval")
	klog.InitFlags(nil)
	flag.Parse()

	// Initialize the client
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	client, err := metrics.NewClient(ctx)
	if err != nil {
		os.Exit(1)
	}

	klog.V(1).Info("Starting scraping metrics...")
	ticker := time.NewTicker(*interval)

	fmt.Printf("metric,pod,timestamp,value\n")
outer:
	for {
		select {
		case <-ticker.C:
			if err := metrics.Retrieve(ctx, client); err != nil {
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

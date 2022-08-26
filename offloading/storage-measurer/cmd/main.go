package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/liqotech/liqo-benchmarks/offloading/storage-measurer/pkg/creation"
	"github.com/liqotech/liqo-benchmarks/offloading/storage-measurer/pkg/monitoring"
	offloadingv1alpha1 "github.com/liqotech/liqo/apis/offloading/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = offloadingv1alpha1.AddToScheme(scheme)
}

func main() {
	// Configure the flags.
	namespace := flag.String("namespace", "storage-benchmark", "The name of the namespace where the benchmark is executed")
	affinity := flag.String("affinity", "", "The node affinity label")
	replicas := flag.Uint("replicas", 1, "The number of replicas to create")
	storageClass := flag.String("storage-class", "", "The storage class for volumes creation")
	volumeSize := flag.String("volume-size", "10M", "The size of the volume to be created for each pod")
	liqoEnable := flag.Bool("enable-liqo-offloading", false, "Whether enable liqo offloading for the namespace")

	klog.InitFlags(nil)
	flag.Parse()

	// Initialize the client
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	clientgo, clientctrl := prepareClients()

	// Create the namespace
	if err := creation.Namespace(ctx, clientgo, *namespace); err != nil {
		os.Exit(1)
	}

	if *liqoEnable {
		if err := creation.NamespaceOffloading(ctx, clientctrl, *namespace); err != nil {
			os.Exit(1)
		}

		for i := 5; i > 0; i-- {
			klog.V(1).Infof("Waiting for namespace offloading initialization to (hopefully) complete (%v)", i)
			time.Sleep(time.Second)
		}
	}

	// Configure the informers
	completion := make(chan struct{})
	monitoring.Start(ctx, clientgo, *namespace, *replicas, completion)

	// Create the Deployments
	monitoring.M().SetOffloadingStartTimestamp(time.Now())
	if err := creation.StatefulSet(ctx, clientgo, *namespace, *affinity, *storageClass, *volumeSize, *replicas); err != nil {
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
	monitoring.M().ToSummary(os.Stdout)
	fmt.Println()
	utilruntime.Must(monitoring.M().ToCSV(os.Stdout))
	fmt.Println()
	monitoring.M().ToTable(os.Stdout)
	fmt.Println()

	klog.V(1).Info("Everything completed. Bye!")
}

func prepareClients() (kubernetes.Interface, client.Client) {
	klog.V(4).Infof("Loading kubernetes client")
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Unable to create client config: %s", err)
		os.Exit(1)
	}

	config.QPS = 10000
	config.Burst = 10000
	clientgo := kubernetes.NewForConfigOrDie(config)
	clientctrl, err := client.New(config, client.Options{Scheme: scheme})
	utilruntime.Must(err)

	klog.V(4).Infof("Loaded kubernetes clients")
	return clientgo, clientctrl
}

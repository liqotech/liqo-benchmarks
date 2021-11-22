// Package creation implements the logic required to create the benchmark objects.
package creation

import (
	"context"
	"fmt"
	"net"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
)

func RetrieveTargetIPs(serviceName string, expected uint64) []net.IP {
	klog.V(2).Infof("Retrieving ForeignCluster IP addresses from service %v", serviceName)

	for {
		ips, err := retrieveTargetIPs(serviceName)
		if err != nil {
			klog.Errorf("Failed to retrieve IP addresses: %v", err)
		} else {
			klog.V(2).Infof("Found %v IPs, expected: %v", len(ips), expected)
			if len(ips) >= int(expected) {
				return ips
			}
		}

		klog.V(2).Info("Sleeping 10 seconds before retrying...")
		time.Sleep(10 * time.Second)
	}
}

func retrieveTargetIPs(serviceName string) ([]net.IP, error) {
	ips, err := net.LookupIP(serviceName)
	if err != nil {
		return nil, err
	}

	// Filter out IPv6 addresses.
	ipv4ips := make([]net.IP, 0)
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4ips = append(ipv4ips, ip)
		}
	}

	return ipv4ips, nil
}

func ForeignClusters(ctx context.Context, client dynamic.Interface, ips []net.IP) error {
	// Create the ForeignClusters.
	klog.V(2).Infof("Creating %v ForeignClusters", len(ips))
	for i, ip := range ips {
		name := fmt.Sprintf("foreign-cluster-%03d", i)
		url := fmt.Sprintf("https://%v", ip.String())
		if err := foreignCluster(ctx, client, name, url); err != nil {
			klog.Errorf("Failed to create ForeignClusters: %v", err)
			return err
		}
	}

	klog.V(2).Info("All ForeignClusters correctly created")
	return nil
}

func foreignCluster(ctx context.Context, client dynamic.Interface, name, url string) error {
	gvr := discoveryv1alpha1.ForeignClusterGroupVersionResource

	klog.V(4).Infof("Creating ForeignCluster %q with target %q", name, url)
	fc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(forgeForeignCluster(name, url))
	if err != nil {
		klog.Errorf("Failed to create ForeignCluster %q: %v", name, err)
		return err
	}

	if _, err := client.Resource(gvr).Create(ctx, &unstructured.Unstructured{Object: fc}, metav1.CreateOptions{}); err != nil {
		klog.Errorf("Failed to create ForeignCluster %q: %v", name, err)
		return err
	}

	klog.V(4).Infof("ForeignCluster %v successfully created", name)
	return nil
}

func forgeForeignCluster(name, url string) *discoveryv1alpha1.ForeignCluster {
	return &discoveryv1alpha1.ForeignCluster{
		TypeMeta:   metav1.TypeMeta{Kind: "ForeignCluster", APIVersion: discoveryv1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: discoveryv1alpha1.ForeignClusterSpec{
			IncomingPeeringEnabled: discoveryv1alpha1.PeeringEnabledNo,
			OutgoingPeeringEnabled: discoveryv1alpha1.PeeringEnabledYes,
			ForeignAuthURL:         url,
		},
	}
}

package service

import (
	"context"
	"net"
	"time"

	"k8s.io/klog/v2"
)

func RetrieveTargetIPs(ctx context.Context, serviceName string, expected uint64) []net.IP {
	klog.V(2).Infof("Retrieving IP addresses from service %v", serviceName)

	inner := func() []net.IP {
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
		return nil
	}

	if ips := inner(); ips != nil {
		return ips
	}

	for {
		select {
		case <-ctx.Done():
			klog.Info("Context canceled, aborting")
			return nil
		case <-time.After(10 * time.Second):
			if ips := inner(); ips != nil {
				return ips
			}
		}
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

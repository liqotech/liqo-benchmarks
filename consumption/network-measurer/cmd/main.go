package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo-benchmarks/consumption/network-measurer/pkg/service"
)

func main() {
	// Configure the flags.
	iface := flag.String("interface", "eth0", "The interface packets are captured from")
	targetservice := flag.String("target-service", "", "The name of the service used to retrieve the source/destination ips")
	targetport := flag.Uint("target-port", 0, "The filtering source/destination port")
	reversed := flag.Bool("reverse", false, "Reverse host and port filters")
	expectedEndpoints := flag.Uint64("expected", 1, "The number of remote endpoints to retrieve before starting")
	klog.InitFlags(nil)
	flag.Parse()

	if *targetservice == "" || *targetport == 0 {
		klog.Error("Mandatory parameters not set")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Retrieve the target IP addresses
	ips := service.RetrieveTargetIPs(ctx, *targetservice, *expectedEndpoints)
	select {
	case <-ctx.Done():
		klog.Info("Context canceled, exiting...")
		os.Exit(0)
	default:
		break
	}

	var filters []string
	for _, ip := range ips {
		filters = append(filters, fmt.Sprintf("# host %s", ip.String()))
	}
	filter := fmt.Sprintf("tcp _ port %d and (%s)", *targetport, strings.Join(filters, " or "))

	handleTransmit, err := pcap.OpenLive(*iface, 96, false, pcap.BlockForever)
	if err != nil {
		klog.Errorf("Failed to capture from %v: %v", *iface, err)
		os.Exit(1)
	}

	handleReceive, err := pcap.OpenLive(*iface, 96, false, pcap.BlockForever)
	if err != nil {
		klog.Errorf("Failed to capture from %v: %v", *iface, err)
		os.Exit(1)
	}

	hostTransmit, hostReceive := "dst", "src"
	if *reversed {
		hostTransmit, hostReceive = "src", "dst"
	}

	filterTransmit := strings.ReplaceAll(strings.ReplaceAll(filter, "_", "dst"), "#", hostTransmit)
	if err := handleTransmit.SetBPFFilter(filterTransmit); err != nil {
		klog.Errorf("Failed to configure filter %v: %v", filterTransmit, err)
		os.Exit(1)
	}

	filterReceive := strings.ReplaceAll(strings.ReplaceAll(filter, "_", "src"), "#", hostReceive)
	if err := handleReceive.SetBPFFilter(filterReceive); err != nil {
		klog.Errorf("Failed to configure filter %v: %v", filterReceive, err)
		os.Exit(1)
	}

	var transmit, receive uint64

	klog.V(1).Infof("Capturing from %q with filters %q and %q", *iface, filterTransmit, filterReceive)
	packetSourceTransmit := gopacket.NewPacketSource(handleTransmit, handleTransmit.LinkType())
	packetSourceReceive := gopacket.NewPacketSource(handleReceive, handleReceive.LinkType())

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			select {
			case packet := <-packetSourceTransmit.Packets():
				atomic.AddUint64(&transmit, uint64(packet.Metadata().Length))
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case packet := <-packetSourceReceive.Packets():
				atomic.AddUint64(&receive, uint64(packet.Metadata().Length))
			case <-ctx.Done():
				return
			}
		}
	}()

	fmt.Printf("metric,pod,timestamp,value\n")
outer:
	for {
		select {
		case now := <-time.After(1 * time.Second):
			tran := atomic.LoadUint64(&transmit)
			recv := atomic.LoadUint64(&receive)
			fmt.Printf("liqo_network_transmit_bytes_total,liqo,%s,%v\n", timestamp(now), tran)
			fmt.Printf("liqo_network_receive_bytes_total,liqo,%s,%v\n", timestamp(now), recv)
		case <-ctx.Done():
			break outer
		}
	}

	wg.Wait()
	klog.V(1).Info("Everything completed. Bye!")
}

func timestamp(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo-benchmarks/offloading/exposition-syn-measurer/pkg/forge"
)

func main() {
	// Configure the flags.
	iface := flag.String("interface", "eth0", "The interface packets are captured from")
	namespace := flag.String("namespace", "exposition-benchmark", "The name of the namespace where the benchmark is executed")
	svccreate := flag.Bool("create-service", true, "Whether to create the service or not")
	interval := flag.Duration("interval", 1*time.Millisecond, "The interval between subsequent SYN packets")
	klog.InitFlags(nil)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client := prepareClient()

	ip, err := forge.RetrieveServiceIP(ctx, client, *namespace, *svccreate)
	if err != nil {
		klog.Errorf("Failed to identify target IP: %v", err)
		os.Exit(1)
	}

	dstip := net.ParseIP(ip)
	if dstip == nil {
		klog.Errorf("Failed to parse target IP: %q", ip)
		os.Exit(1)
	}

	fmt.Printf("Retrieved: %s\n", timestamp(time.Now()))

	var lc net.ListenConfig
	conn, err := lc.ListenPacket(ctx, "ip4:tcp", "0.0.0.0")
	if err != nil {
		klog.Error("Failed to open connection: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	filter := fmt.Sprintf("tcp src port %d", forge.Port)
	handleReceive, err := pcap.OpenLive(*iface, 96, false, pcap.BlockForever)
	if err != nil {
		klog.Errorf("Failed to capture from %v: %v", *iface, err)
		os.Exit(1)
	}

	if err := handleReceive.SetBPFFilter(filter); err != nil {
		klog.Errorf("Failed to configure filter %v: %v", filter, err)
		os.Exit(1)
	}

	klog.V(1).Infof("Capturing from %v with filter %v", *iface, filter)
	packetSourceReceive := gopacket.NewPacketSource(handleReceive, handleReceive.LinkType())

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case packet := <-packetSourceReceive.Packets():
				if packet.Layer(layers.LayerTypeTCP) == nil {
					klog.Warningf("Invalid packet: missing TCP header")
					break
				}

				tcp, ok := packet.TransportLayer().(*layers.TCP)
				if !ok {
					klog.Warningf("Invalid packet: missing TCP header")
					break
				}

				if tcp.RST {
					klog.V(1).Infof("Received RST packet (port: %d)", tcp.DstPort)
					break
				}

				if tcp.SYN && tcp.ACK {
					klog.V(1).Infof("Received SYN-ACK packet (port: %d)", tcp.DstPort)
					fmt.Printf("Connected: %s\n", timestamp(packet.Metadata().Timestamp))
					cancel()
					break
				}

				klog.V(1).Infof("Received unknown packet (port: %d)", tcp.DstPort)
			case <-ctx.Done():
				return
			}
		}
	}()

	srcaddr, err := forge.RetrieveOutboundIP(dstip)
	if err != nil {
		klog.Errorf("Failed retrieving source IP address: %v", err)
		os.Exit(1)
	}
	srcport := uint(10000)
outer:
	for {
		select {
		case <-time.After(*interval):
			srcport++
			if srcport == 65000 {
				srcport = 10000
			}

			// Generate the syn packet
			packet, err := forge.SynPacket(srcaddr, dstip, srcport, forge.Port)
			if err != nil {
				klog.Warningf("Failed to generate packet: %v", err)
			}

			if _, err := conn.WriteTo(packet, &net.IPAddr{IP: dstip}); err != nil {
				klog.Warningf("Failed sending SYN packet: %v", err)
			}

			klog.V(2).Infof("Sent SYN packet to %v:%d (src: %v:%d)", dstip, forge.Port, srcaddr, srcport)
		case <-ctx.Done():
			break outer
		}
	}

	wg.Wait()
	klog.V(1).Info("Everything completed. Bye!")
}

func timestamp(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
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

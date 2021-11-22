// Package measurer wraps the logic used to perform the measurements.
package measurer

import (
	"encoding/csv"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"k8s.io/klog/v2"
)

type GlobalMeasurer struct {
	sync.Mutex

	measurers map[string]*FCMeasurer

	start      time.Time
	completion chan<- struct{}
}

type FCMeasurer struct {
	clusterID  string
	completion chan<- struct{}

	peeringProcessStart time.Time

	authenticationIncomingEnd time.Time
	authenticationOutgoingEnd time.Time

	resourceNegotiationStart time.Time
	resourceNegotiationEnd   time.Time

	networkSetupStart time.Time
	networkSetupEnd   time.Time

	virtualKubeletSetupStart time.Time
	virtualKubeletSetupEnd   time.Time

	nodeReady time.Time
}

func NewMeasurer(completion chan<- struct{}) *GlobalMeasurer {
	return &GlobalMeasurer{
		completion: completion,
		measurers:  make(map[string]*FCMeasurer),
	}
}

func NewFCMeasurer(clusterID string, start time.Time, completion chan<- struct{}) *FCMeasurer {
	return &FCMeasurer{
		clusterID:  clusterID,
		completion: completion,

		peeringProcessStart: start,
	}
}

func (m *GlobalMeasurer) ClusterID(clusterID string) *FCMeasurer {
	m.Lock()
	defer m.Unlock()

	if inner, ok := m.measurers[clusterID]; ok {
		return inner
	}

	inner := NewFCMeasurer(clusterID, m.start, m.completion)
	m.measurers[clusterID] = inner
	return inner
}

func (m *GlobalMeasurer) SetPeeringStartTimestamp(t time.Time) {
	m.start = t
}

func (m *GlobalMeasurer) ToCSV(w io.Writer) error {
	writer := csv.NewWriter(w)

	// Write the header line
	header := []string{
		"cluster-id", "peering-process-start", "incoming-authentication-end", "outgoing-authentication-end",
		"resource-negotiation-start", "resource-negotiation-end", "network-setup-start", "network-setup-end",
		"virtual-kubelet-setup-start", "virtual-kubelet-setup-end", "node-ready"}
	if err := writer.Write(header); err != nil {
		klog.Error("Failed to write the CSV output: ", err)
		return err
	}

	// Write one line for each measurer
	for _, measurer := range m.measurers {
		if err := writer.Write(measurer.ToSliceRaw()); err != nil {
			klog.Error("Failed to write the CSV output: ", err)
			return err
		}
	}
	writer.Flush()
	return nil
}

func (m *GlobalMeasurer) ToTable(w io.Writer) {
	table := tablewriter.NewWriter(w)

	// Write the header line
	table.SetBorder(false)
	table.SetHeader([]string{
		"cluster-id", "total", "authentication", "resource negotiation", "network setup", "virtual kubelet setup", "node setup",
	})

	// Write one line for each measurer
	for _, measurer := range m.measurers {
		table.Append(measurer.ToSliceParsed())
	}
	table.Render()
}

func (m *FCMeasurer) ToSliceRaw() []string {
	return []string{
		m.clusterID, ustr(m.peeringProcessStart), ustr(m.authenticationIncomingEnd), ustr(m.authenticationOutgoingEnd),
		ustr(m.resourceNegotiationStart), ustr(m.resourceNegotiationEnd), ustr(m.networkSetupStart), ustr(m.networkSetupEnd),
		ustr(m.virtualKubeletSetupStart), ustr(m.virtualKubeletSetupEnd), ustr(m.nodeReady),
	}
}

func (m *FCMeasurer) ToSliceParsed() []string {
	return []string{
		m.clusterID, dstr(m.Total()), dstr(m.Authentication()), dstr(m.ResourceNegotiation()),
		dstr(m.NetworkSetup()), dstr(m.VirtualKubeletSetup()), dstr(m.NodeSetup()),
	}
}

func (m *FCMeasurer) SetPeeringStartTimestamp(t time.Time) {
	if !checkInterval(m.peeringProcessStart, t) {
		klog.Warningf("[%v] Peering process start timestamps do not seem to match %v, %v", m.clusterID, pstr(m.peeringProcessStart), pstr(t))
	}
}

func (m *FCMeasurer) SetAuthenticationOutgoingEndTimestamp(t time.Time) {
	if m.authenticationOutgoingEnd.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Outgoing authentication end timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.authenticationOutgoingEnd = now
		klog.V(4).Infof("[%v] Outgoing authentication ended at %v (%v)", m.clusterID, pstr(now), pstr(t))
		klog.V(2).Infof("[%v] Authentication completed in %v", m.clusterID, dstr(m.Authentication()))
	}
}

func (m *FCMeasurer) SetAuthenticationIncomingEndTimestamp(t time.Time) {
	if m.authenticationIncomingEnd.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Incoming authentication end timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.authenticationIncomingEnd = now
		klog.V(4).Infof("[%v] Incoming authentication ended at %v (%v)", m.clusterID, pstr(now), pstr(t))
	}
}

func (m *FCMeasurer) SetResourceNegotiationStartTimestamp(t time.Time) {
	if m.resourceNegotiationStart.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Resource negotiation start timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.resourceNegotiationStart = now
		klog.V(4).Infof("[%v] Resource negotiation started at %v (%v)", m.clusterID, pstr(now), pstr(t))
	}
}

func (m *FCMeasurer) SetResourceNegotiationEndTimestamp(t time.Time) {
	if m.resourceNegotiationEnd.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Resource negotiation end timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.resourceNegotiationEnd = now
		klog.V(4).Infof("[%v] Resource negotiation ended at %v (%v)", m.clusterID, pstr(now), pstr(t))
		klog.V(2).Infof("[%v] Resource negotiation completed in %v", m.clusterID, dstr(m.ResourceNegotiation()))
	}
}

func (m *FCMeasurer) SetNetworkSetupStartTimestamp(t time.Time) {
	if m.networkSetupStart.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Network setup start timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.networkSetupStart = now
		klog.V(4).Infof("[%v] Resource setup started at %v (%v)", m.clusterID, pstr(now), pstr(t))
	}
}

func (m *FCMeasurer) SetNetworkSetupEndTimestamp(t time.Time) {
	if m.networkSetupEnd.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Network setup end timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.networkSetupEnd = now
		klog.V(4).Infof("[%v] Network setup ended at %v (%v)", m.clusterID, pstr(now), pstr(t))
		klog.V(2).Infof("[%v] Network setup completed in %v", m.clusterID, dstr(m.NetworkSetup()))
	}
}

func (m *FCMeasurer) SetVirtualKubeletSetupStartTimestamp(t time.Time) {
	if m.virtualKubeletSetupStart.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Virtual kubelet setup start timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.virtualKubeletSetupStart = now
		klog.V(4).Infof("[%v] Virtual kubelet setup started at %v (%v)", m.clusterID, pstr(now), pstr(t))
	}
}

func (m *FCMeasurer) SetVirtualKubeletSetupEndTimestamp(t time.Time) {
	if m.virtualKubeletSetupEnd.IsZero() {
		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Virtual kubelet setup end timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.virtualKubeletSetupEnd = now
		klog.V(4).Infof("[%v] Virtual kubelet setup ended at %v (%v)", m.clusterID, pstr(now), pstr(t))
		klog.V(2).Infof("[%v] Virtual kubelet setup completed in %v", m.clusterID, dstr(m.VirtualKubeletSetup()))
	}
}

func (m *FCMeasurer) SetNodeReady(t time.Time) {
	if m.nodeReady.IsZero() {
		// Make sure also the virtual-kubelet setup end timestamp is set,
		// since that event might be received slightly later than node readiness.
		m.SetVirtualKubeletSetupEndTimestamp(t)

		now := time.Now()
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Node ready timestamps do not seem to match %v, %v", m.clusterID, pstr(now), pstr(t))
		}

		m.nodeReady = now
		klog.V(4).Infof("[%v] Node ready at %v (%v)", m.clusterID, pstr(now), pstr(t))
		klog.V(2).Infof("[%v] Node ready in %v", m.clusterID, dstr(m.Total()))
		m.completion <- struct{}{}
	}
}

func (m *FCMeasurer) Total() time.Duration {
	return m.nodeReady.Sub(m.peeringProcessStart)
}

func (m *FCMeasurer) Authentication() time.Duration {
	return m.authenticationOutgoingEnd.Sub(m.peeringProcessStart)
}

func (m *FCMeasurer) ResourceNegotiation() time.Duration {
	return m.resourceNegotiationEnd.Sub(m.resourceNegotiationStart)
}

func (m *FCMeasurer) NetworkSetup() time.Duration {
	return m.networkSetupEnd.Sub(m.networkSetupStart)
}

func (m *FCMeasurer) VirtualKubeletSetup() time.Duration {
	return m.virtualKubeletSetupEnd.Sub(m.virtualKubeletSetupStart)
}

func (m *FCMeasurer) NodeSetup() time.Duration {
	return m.nodeReady.Sub(m.virtualKubeletSetupEnd)
}

func pstr(t time.Time) string {
	return t.UTC().Format("15:04:05.000")
}

func ustr(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}

func dstr(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}

func checkInterval(a, b time.Time) bool {
	delta := a.Sub(b)
	return delta >= -500*time.Millisecond && delta < 1500*time.Millisecond
}

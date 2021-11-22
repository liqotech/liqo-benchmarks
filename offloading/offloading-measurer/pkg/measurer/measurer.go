// Package measurer wraps the logic used to perform the measurements.
package measurer

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

type Metric struct {
	start, end time.Time
	cpu, ram   int64
}

type Measurer struct {
	pods    sync.Map // signature: map[string]time.Time
	metrics []Metric

	start    time.Time
	ready    uint
	expected uint

	done chan<- struct{}
}

func NewMeasurer(expected uint, done chan<- struct{}) *Measurer {
	return &Measurer{
		expected: expected,
		done:     done,
	}
}

func (m *Measurer) SetOffloadingStartTimestamp(t time.Time) {
	m.start = t
}

func (m *Measurer) SetPodReady(pod string, t time.Time) {
	now := time.Now()
	if _, found := m.pods.LoadOrStore(pod, now); !found {
		if !checkInterval(now, t) {
			klog.Warningf("[%v] Pod ready timestamps do not seem to match %v, %v", pod, pstr(now), pstr(t))
		}

		m.ready++
		if m.ready*100%m.expected == 0 {
			klog.V(2).Infof("%3d%% pods ready (%v second)", m.ready*100/m.expected, dstr(now.Sub(m.start)))
		}

		if m.ready == m.expected {
			close(m.done)
		}
	}
}

func (m *Measurer) Metrics(start, end time.Time, cpu, ram int64) {
	if len(m.metrics) > 0 && m.metrics[len(m.metrics)-1].start == start {
		return
	}

	m.metrics = append(m.metrics, Metric{
		start: start, end: end,
		cpu: cpu, ram: ram,
	})
}

func (m *Measurer) ToCSV(w io.Writer) {
	output := m.ReadyTimesAsUnixNano()

	fmt.Fprintf(w, "Start: %s\n", unstr(m.start))
	fmt.Fprintf(w, "End  : %s\n", output[len(output)-1])

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Percentiles:")

	writer := csv.NewWriter(w)
	for i := 0; i < len(output); i += 10 {
		utilruntime.Must(writer.Write(output[i : i+10]))
	}
	writer.Flush()

	if len(m.metrics) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Resource consumption:")

		writer = csv.NewWriter(w)
		utilruntime.Must(writer.Write([]string{"start", "end", "cpu", "ram"}))
		utilruntime.Must(writer.WriteAll(m.MetricsAsString()))
		writer.Flush()
	}
}

func (m *Measurer) ToTable(w io.Writer) {
	table := tablewriter.NewWriter(w)

	// Write the header line
	table.SetBorder(false)
	table.SetHeader([]string{
		"10%", "20%", "30%", "40%", "50%", "60%", "70%", "80%", "90%", "100%",
	})

	// Write the results line
	table.Append(m.ReadyTimesAsDuration())
	table.Render()
}

func (m *Measurer) ReadyTimes() []time.Time {
	var all []time.Time
	m.pods.Range(func(_, value interface{}) bool {
		all = append(all, value.(time.Time))
		return true
	})

	// Sort the readiness times in descending order
	sort.Slice(all, func(i, j int) bool {
		return all[i].UnixNano() > all[j].UnixNano()
	})

	// Take a subsample of one element per percent point.
	sampled := make([]time.Time, 100)
	for i := 0; i < 100; i++ {
		idx := int(math.Floor(float64(i) * float64(len(all)) / 100.))
		sampled[100-i-1] = all[idx]
	}
	return sampled
}

func (m *Measurer) ReadyTimesAsUnixNano() []string {
	var output []string
	for _, time := range m.ReadyTimes() {
		output = append(output, unstr(time))
	}
	return output
}

func (m *Measurer) ReadyTimesAsDuration() []string {
	var output []string
	rt := m.ReadyTimes()
	for i := 9; i < 100; i += 10 {
		output = append(output, dstr(rt[i].Sub(m.start)))
	}
	return output
}

func (m *Measurer) MetricsAsString() [][]string {
	output := make([][]string, len(m.metrics))

	for i, metric := range m.metrics {
		output[i] = []string{
			ustr(metric.start), ustr(metric.end),
			strconv.FormatInt(metric.cpu, 10),
			strconv.FormatInt(metric.ram, 10),
		}
	}

	return output
}

func pstr(t time.Time) string {
	return t.UTC().Format("15:04:05.000")
}

func unstr(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}

func ustr(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func dstr(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 1, 64)
}

func checkInterval(a, b time.Time) bool {
	delta := a.Sub(b)
	return delta >= -500*time.Millisecond && delta < 1500*time.Millisecond
}

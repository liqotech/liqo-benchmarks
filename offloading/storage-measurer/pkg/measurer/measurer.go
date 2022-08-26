// Package measurer wraps the logic used to perform the measurements.
package measurer

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"k8s.io/klog/v2"
)

type Measurer struct {
	pvcs sync.Map // signature: map[string]time.Time
	pods sync.Map // signature: map[string]time.Time

	start    time.Time
	end      time.Time
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
			m.end = now
			close(m.done)
		}
	}
}

func (m *Measurer) SetPVCBound(pvc string) {
	m.pvcs.LoadOrStore(strings.TrimPrefix(pvc, "benchmark-"), time.Now())
}

func (m *Measurer) ToSummary(w io.Writer) {
	fmt.Fprintf(w, "Start: %s\n", unstr(m.start))
	fmt.Fprintf(w, "End  : %s\n", unstr(m.end))
}

func (m *Measurer) ToCSV(w io.Writer) error {
	writer := csv.NewWriter(w)

	// Write the header line
	header := []string{"name", "pvc-bound", "pod-ready"}
	if err := writer.Write(header); err != nil {
		klog.Error("Failed to write the CSV output: ", err)
		return err
	}

	// Write one line for each measurer
	for _, pod := range m.PodNames() {
		if err := writer.Write(m.ToSliceRaw(pod)); err != nil {
			klog.Error("Failed to write the CSV output: ", err)
			return err
		}
	}

	writer.Flush()
	return nil
}

func (m *Measurer) ToTable(w io.Writer) {
	table := tablewriter.NewWriter(w)

	// Write the header line
	table.SetBorder(false)
	table.SetHeader([]string{"name", "pvc-bound", "pod-ready"})

	// Write one line for each measurer
	for _, pod := range m.PodNames() {
		table.Append(m.ToSliceParsed(pod))
	}

	table.Render()
}

func (m *Measurer) PodNames() []string {
	names := make([]string, 0)
	m.pods.Range(func(key, _ interface{}) bool {
		names = append(names, key.(string))
		return true
	})

	sort.StringSlice.Sort(names)
	return names
}

func (m *Measurer) ToSliceRaw(name string) []string {
	pvc, found := m.pvcs.Load(name)
	pod, _ := m.pods.Load(name)

	if !found {
		return []string{name, unstr(time.Time{}), unstr(pod.(time.Time))}
	}
	return []string{name, unstr(pvc.(time.Time)), unstr(pod.(time.Time))}
}

func (m *Measurer) ToSliceParsed(name string) []string {
	pvc, found := m.pvcs.Load(name)
	pod, _ := m.pods.Load(name)

	if !found {
		return []string{name, dstr(0), dstr(pod.(time.Time).Sub(m.start))}
	}
	return []string{name, dstr(pvc.(time.Time).Sub(m.start)), dstr(pod.(time.Time).Sub(m.start))}
}

func pstr(t time.Time) string {
	return t.UTC().Format("15:04:05.000")
}

func unstr(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}

func dstr(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 1, 64)
}

func checkInterval(a, b time.Time) bool {
	delta := a.Sub(b)
	return delta >= -500*time.Millisecond && delta < 1500*time.Millisecond
}

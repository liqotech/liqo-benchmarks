// Package measurer wraps the logic used to perform the measurements.
package measurer

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"k8s.io/klog/v2"
)

type Measurer struct {
	epslices map[string]uint

	start    time.Time
	end      time.Time
	ready    uint
	expected uint

	done chan<- struct{}
}

func NewMeasurer(expected uint, done chan<- struct{}) *Measurer {
	return &Measurer{
		epslices: make(map[string]uint),
		expected: expected,
		done:     done,
	}
}

func (m *Measurer) SetExpositionStartTimestamp(t time.Time) {
	m.start = t
}

func (m *Measurer) SetEndpointSliceReady(name string, ready uint) {
	previous := m.epslices[name]
	m.epslices[name] = ready

	m.ready += ready - previous
	if m.ready*100%m.expected == 0 {
		klog.V(2).Infof("%3d%% epslices ready", m.ready*100/m.expected)
	}

	if m.ready == m.expected {
		m.end = time.Now()
		close(m.done)
	}
}

func (m *Measurer) Output(w io.Writer) {
	if !m.start.IsZero() {
		fmt.Fprintf(w, "Start: %s - %s\n", unstr(m.start), pstr(m.start))
	}
	fmt.Fprintf(w, "End  : %s - %s\n", unstr(m.end), pstr(m.end))
	if !m.start.IsZero() {
		fmt.Fprintf(w, "Total: %s seconds\n", dstr(m.end.Sub(m.start)))
	}
}

func pstr(t time.Time) string {
	return t.UTC().Format("15:04:05.000")
}

func unstr(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}

func dstr(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}

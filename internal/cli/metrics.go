package cli

import (
	"fmt"
	"io"
	"openapi-kt/core"
	"openapi-kt/internal/output"
	"time"
)

// Execution metrics are nondeterministic CLI metadata, never generator options
// or artifacts. Phase names describe actual boundaries, not imagined core steps.
type executionMetrics struct {
	ConfigBytes  *int64           `json:"config_bytes,omitempty"`
	TotalNS      int64            `json:"total_ns"`
	PhaseNS      map[string]int64 `json:"phase_ns"`
	CorePhaseNS  map[string]int64 `json:"core_phase_ns,omitempty"`
	PeakRSSBytes *int64           `json:"process_peak_rss_bytes,omitempty"`
	InputBytes   int64            `json:"input_bytes"`
	PlannedFiles int64            `json:"planned_files"`
	PlannedBytes int64            `json:"planned_bytes"`
}
type metricCollector struct {
	value executionMetrics
	now   func() time.Time
	start time.Time
}

func newMetricCollector(enabled bool, now func() time.Time) *metricCollector {
	if !enabled {
		return nil
	}
	return &metricCollector{value: executionMetrics{PhaseNS: map[string]int64{}}, now: now, start: now()}
}
func (c *metricCollector) phase(name string) func() {
	if c == nil {
		return func() {}
	}
	start := c.now()
	return func() { c.value.PhaseNS[name] += c.now().Sub(start).Nanoseconds() }
}
func (c *metricCollector) coreObserver() core.GenerationPhaseObserver {
	if c == nil {
		return nil
	}
	return func(phase core.GenerationPhase) func() {
		start := c.now()
		return func() {
			if c.value.CorePhaseNS == nil {
				c.value.CorePhaseNS = map[string]int64{}
			}
			c.value.CorePhaseNS[string(phase)] += c.now().Sub(start).Nanoseconds()
		}
	}
}
func (c *metricCollector) input(bytes int) {
	if c != nil {
		c.value.InputBytes += int64(bytes)
	}
}
func (c *metricCollector) planned(a core.GenerationArtifacts) {
	if c != nil {
		c.value.PlannedFiles = int64(len(a.Files))
		for _, f := range a.Files {
			c.value.PlannedBytes += int64(len(f.Content))
		}
	}
}
func (c *metricCollector) configBytes(bytes int) {
	if c != nil {
		n := int64(bytes)
		c.value.ConfigBytes = &n
	}
}

// Only a complete, unambiguous generated envelope has planned counts. Runtime
// duplicates are counted once per root/path; output may still refuse the plan.
func (c *metricCollector) plannedConfig(updates map[string]output.OwnerUpdate) {
	if c == nil {
		return
	}
	paths := map[string]string{}
	for root, u := range updates {
		for _, a := range u.Generated {
			for _, f := range a.Files {
				key := root + "\x00" + f.Path
				if old, ok := paths[key]; ok && old != f.Content {
					return
				}
				paths[key] = f.Content
			}
		}
	}
	c.value.PlannedFiles = int64(len(paths))
	for _, content := range paths {
		c.value.PlannedBytes += int64(len(content))
	}
}
func (c *metricCollector) finish() *executionMetrics {
	if c == nil {
		return nil
	}
	c.value.TotalNS = c.now().Sub(c.start).Nanoseconds()
	if bytes, ok := processPeakRSSBytes(); ok {
		c.value.PeakRSSBytes = &bytes
	}
	return &c.value
}
func printMetrics(m *executionMetrics, stderr io.Writer) {
	if m == nil {
		return
	}
	_, _ = fmt.Fprintf(stderr, "metrics: total_ns=%d input_bytes=%d planned_files=%d planned_bytes=%d", m.TotalNS, m.InputBytes, m.PlannedFiles, m.PlannedBytes)
	if m.ConfigBytes != nil {
		_, _ = fmt.Fprintf(stderr, " config_bytes=%d", *m.ConfigBytes)
	}
	for _, stage := range []string{"config", "name_lock_read", "read", "generate", "warning_policy", "check", "commit"} {
		if ns, ok := m.PhaseNS[stage]; ok {
			_, _ = fmt.Fprintf(stderr, " %s_ns=%d", stage, ns)
		}
	}
	for _, phase := range []core.GenerationPhase{core.PhaseOptions, core.PhaseParse, core.PhaseAnalyze, core.PhaseTransform, core.PhaseEmit} {
		if ns, ok := m.CorePhaseNS[string(phase)]; ok {
			_, _ = fmt.Fprintf(stderr, " core_%s_ns=%d", phase, ns)
		}
	}
	if m.PeakRSSBytes != nil {
		_, _ = fmt.Fprintf(stderr, " process_peak_rss_bytes=%d", *m.PeakRSSBytes)
	}
	_, _ = fmt.Fprintln(stderr)
}

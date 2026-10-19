package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type coreMetricReport struct {
	Metrics struct {
		PhaseNS     map[string]int64 `json:"phase_ns"`
		CorePhaseNS map[string]int64 `json:"core_phase_ns"`
		PeakRSS     *int64           `json:"process_peak_rss_bytes"`
	} `json:"metrics"`
}

func assertCoreMetrics(t *testing.T, out invocation, exit int, want ...string) coreMetricReport {
	t.Helper()
	result(t, out, exit)
	var r coreMetricReport
	if err := json.Unmarshal(out.Stdout, &r); err != nil {
		t.Fatal(err)
	}
	var keys []string
	var sum int64
	for p, ns := range r.Metrics.CorePhaseNS {
		if ns < 0 {
			t.Fatal(p, ns)
		}
		keys = append(keys, p)
		sum += ns
	}
	sort.Strings(keys)
	sort.Strings(want)
	if !reflect.DeepEqual(keys, want) || sum > r.Metrics.PhaseNS["generate"] {
		t.Fatal(r.Metrics, keys, want)
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if r.Metrics.PeakRSS == nil || *r.Metrics.PeakRSS <= 0 {
			t.Fatal("missing real process RSS", r)
		}
	} else if r.Metrics.PeakRSS != nil {
		t.Fatal("unsupported host RSS", r)
	}
	return r
}

func TestCoreMetricsNativeInputFailuresAndNoopPreservation(t *testing.T) {
	p := newProject(t)
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	assertCoreMetrics(t, p.generate(t, "--metrics", "--check"), 0, "options", "parse", "analyze", "transform", "emit")
	assertCoreMetrics(t, p.generate(t, "--metrics"), 0, "options", "parse", "analyze", "transform", "emit")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("metrics changed output bytes/mtimes")
	}
	put(t, p.Input, []byte(`{"openapi":`))
	assertCoreMetrics(t, p.generate(t, "--metrics"), 1, "options", "parse")
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bad":{"type":"object","minProperties":1}}}}`))
	assertCoreMetrics(t, p.generate(t, "--metrics"), 1, "options", "parse", "analyze")
	p.Input = filepath.Join(p.Root, "missing.json")
	assertCoreMetrics(t, p.generate(t, "--metrics"), 4)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("failed measured generation changed output")
	}
}

func TestCoreMetricsNativeStdinAndText(t *testing.T) {
	p := newProject(t)
	assertCoreMetrics(t, invokeInput(t, p.Root, bytes.NewReader(read(t, p.Input)), "validate", "--input=-", "--metrics", "--diagnostic-format=json"), 0, "options", "parse", "analyze", "transform", "emit")
	out := invoke(t, repo, "validate", "--input", p.Input, "--metrics")
	if out.Code != 0 || !strings.HasPrefix(string(out.Stdout), "ok:") {
		t.Fatal(out)
	}
	for _, field := range []string{"core_options_ns=", "core_parse_ns=", "core_analyze_ns=", "core_transform_ns=", "core_emit_ns="} {
		if !strings.Contains(string(out.Stderr), field) {
			t.Fatal("missing text field", field, out)
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if !strings.Contains(string(out.Stderr), "process_peak_rss_bytes=") {
			t.Fatal(out)
		}
	}
}

func TestCoreMetricsNativeConfigurationAccumulationAndPartialFailure(t *testing.T) {
	dir, cfg := configFiles(t)
	args := []string{"generate", "--config", cfg, "--metrics", "--diagnostic-format=json"}
	assertCoreMetrics(t, invoke(t, repo, args...), 0, "options", "parse", "analyze", "transform", "emit")
	root := filepath.Join(dir, "generated")
	before := snapshot(t, root)
	assertCoreMetrics(t, invoke(t, repo, append(args, "--check")...), 0, "options", "parse", "analyze", "transform", "emit")
	put(t, filepath.Join(dir, "Beta.json"), []byte(`{"openapi":`))
	// Earlier successfully emitted spec remains in collected timings, but failure
	// returns no complete generated envelope and commits neither selected spec.
	// A short phase may measure zero on a coarse host clock. assertCoreMetrics
	// verifies entered keys and nonnegative durations, including the earlier emit.
	assertCoreMetrics(t, invoke(t, repo, args...), 1, "options", "parse", "analyze", "transform", "emit")
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("partial measured config failure changed output")
	}
}

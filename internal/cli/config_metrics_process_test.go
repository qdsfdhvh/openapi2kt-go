package cli_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConfigMetricsAdoptionAndDistinctRoots(t *testing.T) {
	_, cfg, root := legacyConfigRoot(t)
	before := snapshot(t, root)
	out := invoke(t, repo, "generate", "--config", cfg, "--spec", "alpha", "--adopt-legacy", "--check", "--metrics", "--diagnostic-format=json")
	result(t, out, 3)
	m := configMetrics(t, out)
	if m.PlannedFiles != 1 || m.PlannedBytes <= 0 {
		t.Fatal("metadata-only planned counts are not written counts", m)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("adoption metrics check wrote")
	}
	_, cfg = multiRootConfig(t)
	out = invoke(t, repo, "validate", "--config", cfg, "--metrics", "--diagnostic-format=json")
	r := result(t, out, 0)
	m = configMetrics(t, out)
	if m.PlannedFiles != int64(len(r.Files)) {
		t.Fatal("distinct roots dedup incorrectly", m, r)
	}
}
func TestConfigMetricsDefaultControl(t *testing.T) {
	cfg, root := runtimeConfiguration(t, true)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	before := snapshot(t, root)
	out := invoke(t, repo, "generate", "--config", cfg, "--check", "--diagnostic-format=json")
	result(t, out, 0)
	var raw map[string]json.RawMessage
	if e := json.Unmarshal(out.Stdout, &raw); e != nil {
		t.Fatal(e)
	}
	if raw["metrics"] != nil || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("defaultmetrics/outputchanged")
	}
}
func TestConfigMetricsNativeAggregation(t *testing.T) {
	cfg, root := runtimeConfiguration(t, true)
	result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
	before := snapshot(t, root)
	out := invoke(t, repo, "generate", "--config", cfg, "--metrics", "--check", "--diagnostic-format=json")
	result(t, out, 0)
	var raw struct {
		Metrics struct {
			ConfigBytes  *int64           `json:"config_bytes"`
			InputBytes   int64            `json:"input_bytes"`
			PlannedFiles int64            `json:"planned_files"`
			PlannedBytes int64            `json:"planned_bytes"`
			PhaseNS      map[string]int64 `json:"phase_ns"`
		} `json:"metrics"`
		Files []string `json:"files"`
	}
	if e := json.Unmarshal(out.Stdout, &raw); e != nil {
		t.Fatal(e)
	}
	if len(out.Stderr) != 0 || raw.Metrics.ConfigBytes == nil || *raw.Metrics.ConfigBytes <= 0 || raw.Metrics.InputBytes <= 0 || raw.Metrics.PlannedFiles != int64(len(raw.Files)) || raw.Metrics.PlannedBytes <= 0 {
		t.Fatal(raw)
	}
	for _, p := range []string{"config", "read", "generate", "check"} {
		if n, ok := raw.Metrics.PhaseNS[p]; !ok || n < 0 {
			t.Fatal(p, raw)
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("configmetricscheck wrote")
	}
}

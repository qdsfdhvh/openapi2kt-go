package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"openapi-kt/core"
)

type measuredReport struct {
	resultReport
	Metrics *struct {
		TotalNS      int64            `json:"total_ns"`
		PhaseNS      map[string]int64 `json:"phase_ns"`
		InputBytes   int64            `json:"input_bytes"`
		PlannedFiles int64            `json:"planned_files"`
		PlannedBytes int64            `json:"planned_bytes"`
	} `json:"metrics"`
}

func measured(t *testing.T, out invocation, exit int, phases ...string) measuredReport {
	t.Helper()
	result(t, out, exit)
	var r measuredReport
	if e := json.Unmarshal(out.Stdout, &r); e != nil {
		t.Fatal(e)
	}
	if r.Metrics == nil {
		t.Fatal("missing metrics", string(out.Stdout))
	}
	var keys []string
	var sum int64
	for k, v := range r.Metrics.PhaseNS {
		keys = append(keys, k)
		if v < 0 {
			t.Fatal(v)
		}
		sum += v
	}
	sort.Strings(keys)
	sort.Strings(phases)
	if !reflect.DeepEqual(keys, phases) || r.Metrics.TotalNS < sum || r.Metrics.InputBytes < 0 || r.Metrics.PlannedFiles < 0 || r.Metrics.PlannedBytes < 0 {
		t.Fatal(r.Metrics, keys, phases)
	}
	return r
}
func TestCLIMetricsStructuralControl(t *testing.T) {
	p := newProject(t)
	r := result(t, p.generate(t), 0)
	if len(r.Files) == 0 {
		t.Fatal(r)
	}
	var raw map[string]any
	out := p.generate(t, "--check")
	result(t, out, 0)
	if err := json.Unmarshal(out.Stdout, &raw); err != nil {
		t.Fatal(err)
	}
	if _, exists := raw["metrics"]; exists {
		t.Fatal(raw)
	}
}
func TestCLIMetricsNative(t *testing.T) {
	p := newProject(t)
	base := result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "Notes.kt"), []byte("synthetic handwritten"))
	before := snapshot(t, p.Output)
	r := measured(t, p.generate(t, "--metrics"), 0, "read", "generate", "commit")
	if !reflect.DeepEqual(base, r.resultReport) || r.Metrics.InputBytes != int64(len(read(t, p.Input))) || r.Metrics.PlannedFiles != int64(len(base.Files)) {
		t.Fatal(r)
	}
	var bytes int64
	for _, f := range base.Files {
		bytes += int64(len(read(t, filepath.Join(p.Output, filepath.FromSlash(f)))))
	}
	if r.Metrics.PlannedBytes != bytes {
		t.Fatal(r.Metrics)
	}
	measured(t, p.generate(t, "--metrics", "--check"), 0, "read", "generate", "check")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("metrics changed content/manifest/owned-handwritten mtimes")
	}
	measured(t, invoke(t, "", "validate", "--input", p.Input, "--metrics", "--diagnostic-format=json"), 0, "read", "generate")
	text := invoke(t, "", "generate", "--input", p.Input, "--output", p.Output, "--package-name", "org.example.catalog", "--metrics")
	if text.Code != 0 || !strings.HasPrefix(string(text.Stdout), "ok:") || !strings.Contains(string(text.Stderr), "metrics: total_ns=") {
		t.Fatal(text)
	}
}
func TestCLIMetricsFailuresAndDenial(t *testing.T) {
	p := warningProject(t)
	r := measured(t, p.generate(t, "--metrics", "--deny-warnings", "--check"), 1, "read", "generate", "warning_policy")
	if r.Status != "error" || len(r.Files) != 0 || r.Metrics.PlannedFiles == 0 || r.Metrics.PlannedBytes == 0 || len(r.Warnings) != 2 {
		t.Fatal(r)
	}
	if _, e := os.Stat(p.Output); !os.IsNotExist(e) {
		t.Fatal("policy metrics created output", e)
	}
	put(t, p.Input, []byte(`{"openapi":`))
	r = measured(t, p.generate(t, "--metrics"), 1, "read", "generate")
	if r.Metrics.PlannedFiles != 0 || r.Metrics.PlannedBytes != 0 {
		t.Fatal(r)
	}
	p.Input = filepath.Join(p.Root, "missing.json")
	r = measured(t, p.generate(t, "--metrics"), 4, "read")
	if r.Metrics.InputBytes != 0 {
		t.Fatal(r)
	}
	p.Input = p.Root // Opening succeeds, but reading a directory fails.
	r = measured(t, p.generate(t, "--metrics"), 4, "read")
	if r.Metrics.InputBytes != 0 {
		t.Fatal(r)
	}
	p = newProject(t)
	put(t, p.Input, []byte(strings.Repeat("x", core.MaxDocumentBytes+16)))
	r = measured(t, p.generate(t, "--metrics"), 1, "read", "generate")
	if r.Metrics.InputBytes != int64(core.MaxDocumentBytes+1) || r.Metrics.PlannedFiles != 0 {
		t.Fatal("limited input count", r)
	}
	p = warningProject(t)
	p.Output = filepath.Join(p.Root, "blocked")
	put(t, p.Output, []byte("synthetic preserved output"))
	r = measured(t, p.generate(t, "--metrics"), 4, "read", "generate", "commit")
	if r.Metrics.PlannedFiles == 0 || string(read(t, p.Output)) != "synthetic preserved output" {
		t.Fatal(r)
	}
	p = newProject(t)
	r = measured(t, p.generate(t, "--metrics", "--check"), 3, "read", "generate", "check")
	if r.Status != "different" {
		t.Fatal(r)
	}
	if _, e := os.Stat(p.Output); !os.IsNotExist(e) {
		t.Fatal("metrics check wrote", e)
	}
}

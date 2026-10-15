package cli

import (
	"bytes"
	"fmt"
	"openapi-kt/core"
	"openapi-kt/internal/output"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func measuredConfiguration(t *testing.T) (arguments, []byte, []byte) {
	t.Helper()
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	data, e := os.ReadFile("../../tests/fixtures/http/shared-runtime.json")
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(input, data, 0600); e != nil {
		t.Fatal(e)
	}
	config := []byte(fmt.Sprintf("config_version=1\n[defaults]\ninput='%s'\noutput_dir='generated'\nvalidation=true\nmodel_name_suffix='Dto'\nruntime_package='org.example.shared.runtime'\n[[specs]]\nname='alpha'\npackage_name='org.example.runtimealpha'\n[[specs]]\nname='beta'\npackage_name='org.example.runtimebeta'\n", filepath.ToSlash(input)))
	cfg := filepath.Join(root, "openapi-kt.toml")
	if e := os.WriteFile(cfg, config, 0600); e != nil {
		t.Fatal(e)
	}
	return arguments{command: "validate", config: cfg}, config, data
}
func TestConfigMetricsExactClockAndSourceUnion(t *testing.T) {
	args, cfg, data := measuredConfiguration(t)
	calls := 0
	clock := func() time.Time { calls++; return time.Unix(0, int64(calls*10)) }
	c, base := executeWithClock(args, clock)
	if c != 0 || calls != 0 || base.Metrics != nil {
		t.Fatal(c, calls, base)
	}
	args.metrics = true
	c, r := executeWithClock(args, clock)
	m := r.Metrics
	if c != 0 || calls != 32 || m.TotalNS != 310 || !reflect.DeepEqual(m.PhaseNS, map[string]int64{"config": 10, "read": 20, "generate": 220}) || !reflect.DeepEqual(m.CorePhaseNS, map[string]int64{"options": 20, "parse": 20, "analyze": 20, "transform": 20, "emit": 20}) || m.ConfigBytes == nil || *m.ConfigBytes != int64(len(cfg)) || m.InputBytes != int64(len(data)*2) {
		t.Fatal(c, calls, m)
	}
	expected := map[string]string{}
	for _, pkg := range []string{"org.example.runtimealpha", "org.example.runtimebeta"} {
		art, e := core.Generate(core.GenerationRequest{Document: data, Format: core.JSON, Options: core.GenerationOptions{PackageName: pkg, RuntimePackage: "org.example.shared.runtime", ModelNameSuffix: "Dto", Validation: true}})
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range art.Files {
			expected[f.Path] = f.Content
		}
	}
	var size int64
	for _, b := range expected {
		size += int64(len(b))
	}
	if m.PlannedFiles != int64(len(expected)) || m.PlannedBytes != size {
		t.Fatal(m, len(expected), size)
	}
	r.Metrics = nil
	if !reflect.DeepEqual(base, r) {
		t.Fatal("metrics changed config report")
	}
	var text bytes.Buffer
	printMetrics(m, &text)
	if !strings.Contains(text.String(), fmt.Sprintf("config_bytes=%d config_ns=10 read_ns=20 generate_ns=220 core_options_ns=20", len(cfg))) {
		t.Fatal(text.String())
	}
	args.command = "generate"
	args.check = true
	calls = 0
	c, r = executeWithClock(args, clock)
	if c != 3 || calls != 34 || r.Metrics.TotalNS != 330 || r.Metrics.PhaseNS["check"] != 10 {
		t.Fatal(c, calls, r.Metrics)
	}
	if _, e := os.Stat(filepath.Join(filepath.Dir(args.config), "generated")); !os.IsNotExist(e) {
		t.Fatal("check created root")
	}
}
func TestConfigMetricsSourceFailureAndDisabledClock(t *testing.T) {
	args, cfg, _ := measuredConfiguration(t)
	for _, invalid := range []bool{false, true} {
		if invalid {
			if e := os.WriteFile(args.config, []byte("bad"), 0600); e != nil {
				t.Fatal(e)
			}
		}
		_, r := executeWithClock(args, func() time.Time { t.Fatal("default configuration read clock"); return time.Time{} })
		if r.Metrics != nil {
			t.Fatal(r)
		}
	}
	if e := os.WriteFile(args.config, cfg, 0600); e != nil {
		t.Fatal(e)
	}
	args.metrics = true
	args.spec = "missing"
	calls := 0
	c, r := executeWithClock(args, func() time.Time { calls++; return time.Unix(0, int64(calls*10)) })
	if c != 1 || calls != 4 || r.Metrics.TotalNS != 30 || r.Metrics.InputBytes != 0 || r.Metrics.PlannedFiles != 0 || !reflect.DeepEqual(r.Metrics.PhaseNS, map[string]int64{"config": 10}) {
		t.Fatal(c, calls, r.Metrics)
	}
	args.spec = ""
	args.config = filepath.Join(t.TempDir(), "missing.toml")
	_, r = executeWithClock(args, func() time.Time { return time.Time{} })
	if r.Metrics.ConfigBytes == nil || *r.Metrics.ConfigBytes != 0 {
		t.Fatal(r.Metrics)
	}
}
func TestConfigMetricPlannedCandidatesDedupConflictAndRoots(t *testing.T) {
	a := core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "A.kt", Content: "字"}}}
	for _, kind := range []string{"dedup", "differentRoot", "conflict"} {
		t.Run(kind, func(t *testing.T) {
			updates := map[string]output.OwnerUpdate{"a": {Generated: map[string]core.GenerationArtifacts{"one": a, "two": a}}}
			wantFiles, wantBytes := int64(1), int64(3)
			switch kind {
			case "differentRoot":
				updates["b"] = output.OwnerUpdate{Generated: map[string]core.GenerationArtifacts{"three": a}}
				wantFiles, wantBytes = 2, 6
			case "conflict":
				updates["a"] = output.OwnerUpdate{Generated: map[string]core.GenerationArtifacts{"one": a, "two": {Files: []core.GeneratedFile{{Path: "A.kt", Content: "different"}}}}}
				wantFiles, wantBytes = 0, 0
			}
			m := newMetricCollector(true, func() time.Time { return time.Time{} })
			m.plannedConfig(updates)
			if m.value.PlannedFiles != wantFiles || m.value.PlannedBytes != wantBytes {
				t.Fatal(m.value)
			}
		})
	}
}

package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type configMetricView struct {
	TotalNS      int64            `json:"total_ns"`
	ConfigBytes  *int64           `json:"config_bytes"`
	InputBytes   int64            `json:"input_bytes"`
	PlannedFiles int64            `json:"planned_files"`
	PlannedBytes int64            `json:"planned_bytes"`
	PhaseNS      map[string]int64 `json:"phase_ns"`
}

func configMetrics(t *testing.T, out invocation) configMetricView {
	t.Helper()
	if len(out.Stderr) != 0 {
		t.Fatal("JSON telemetry leaked stderr", out)
	}
	var raw struct {
		Metrics *configMetricView `json:"metrics"`
	}
	decoder := json.NewDecoder(bytes.NewReader(out.Stdout))
	if e := decoder.Decode(&raw); e != nil || raw.Metrics == nil {
		t.Fatal(raw, e)
	}
	var extra any
	if e := decoder.Decode(&extra); e != io.EOF {
		t.Fatal("not exactly one JSON report", e)
	}
	if raw.Metrics.TotalNS < 0 {
		t.Fatal(raw)
	}
	for _, n := range raw.Metrics.PhaseNS {
		if n < 0 {
			t.Fatal(raw)
		}
	}
	return *raw.Metrics
}
func TestConfigMetricsNativeFailureCountsAndPreservation(t *testing.T) {
	for _, kind := range []string{"secondInput", "secondSchema", "malformed", "oversize", "runtimeRoots", "deny", "modified", "collision"} {
		t.Run(kind, func(t *testing.T) {
			dir, cfg := configFiles(t)
			root := filepath.Join(dir, "generated")
			result(t, invoke(t, repo, "generate", "--config", cfg, "--diagnostic-format=json"), 0)
			config, e := os.ReadFile(cfg)
			if e != nil {
				t.Fatal(e)
			}
			a, e := os.ReadFile(filepath.Join(dir, "Alpha.json"))
			if e != nil {
				t.Fatal(e)
			}
			b, e := os.ReadFile(filepath.Join(dir, "Beta.json"))
			if e != nil {
				t.Fatal(e)
			}
			exit, code := 1, "E_CONFIG_SYNTAX"
			wantInput := int64(0)
			planned := false
			extra := []string{}
			switch kind {
			case "secondInput":
				if e := os.Remove(filepath.Join(dir, "Beta.json")); e != nil {
					t.Fatal(e)
				}
				wantInput = int64(len(a))
				exit, code = 4, "E_INPUT_IO"
			case "secondSchema":
				b = []byte("{bad")
				if e := os.WriteFile(filepath.Join(dir, "Beta.json"), b, 0600); e != nil {
					t.Fatal(e)
				}
				wantInput = int64(len(a) + len(b))
				code = "E_JSON_INVALID"
			case "malformed":
				config = []byte("bad")
			case "oversize":
				config = bytes.Repeat([]byte(" "), 1024*1024+10)
				code = "E_CONFIG_LIMIT"
			case "runtimeRoots":
				config = append(config, []byte("runtime_package='org.example.config.models'\noutput_dir='other'\n")...)
				code = "E_CONFIG_RUNTIME_ROOTS"
			case "deny":
				a, e = os.ReadFile(filepath.Join(repo, "tests/fixtures/cli/warning-policy.json"))
				if e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(filepath.Join(dir, "Alpha.json"), a, 0600); e != nil {
					t.Fatal(e)
				}
				wantInput = int64(len(a) + len(b))
				extra = []string{"--deny-warnings"}
				code = "E_WARNING_DENIED"
				planned = true
			case "modified":
				if e := os.WriteFile(filepath.Join(root, "org/example/config/models/Alpha.kt"), []byte("modified"), 0600); e != nil {
					t.Fatal(e)
				}
				wantInput = int64(len(a) + len(b))
				exit, code = 4, "E_OUTPUT_MODIFIED"
				planned = true
			case "collision":
				b = bytes.ReplaceAll(a, []byte(`"id"`), []byte(`"other"`))
				if e := os.WriteFile(filepath.Join(dir, "Beta.json"), b, 0600); e != nil {
					t.Fatal(e)
				}
				wantInput = int64(len(a) + len(b))
				exit, code = 4, "E_OUTPUT_COLLISION"
			}
			if e := os.WriteFile(cfg, config, 0600); e != nil {
				t.Fatal(e)
			}
			before := snapshot(t, root)
			out := invoke(t, repo, append([]string{"generate", "--config", cfg, "--metrics", "--diagnostic-format=json"}, extra...)...)
			if kind == "deny" {
				r := result(t, out, exit)
				if len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != code || len(r.Warnings) == 0 || len(r.Files) != 0 {
					t.Fatal(r)
				}
			} else {
				fails(t, out, exit, code)
			}
			m := configMetrics(t, out)
			wantConfig := len(config)
			if wantConfig > 1024*1024+1 {
				wantConfig = 1024*1024 + 1
			}
			if m.ConfigBytes == nil || *m.ConfigBytes != int64(wantConfig) || m.InputBytes != wantInput || (m.PlannedFiles > 0) != planned || (m.PlannedBytes > 0) != planned {
				t.Fatal(kind, m, wantInput, wantConfig)
			}
			if _, ok := m.PhaseNS["config"]; !ok {
				t.Fatal(m)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("failed metrics changed prior output")
			}
		})
	}
}
func TestConfigMetricsNativeTextPartialValidateAndEmpty(t *testing.T) {
	dir, cfg := configFiles(t)
	root := filepath.Join(dir, "generated")
	normal := invoke(t, repo, "generate", "--config", cfg)
	if normal.Code != 0 {
		t.Fatal(normal)
	}
	before := snapshot(t, root)
	out := invoke(t, repo, "generate", "--config", cfg, "--metrics")
	if out.Code != 0 || !bytes.Equal(normal.Stdout, out.Stdout) || !bytes.Contains(out.Stderr, []byte(" config_bytes=")) || !bytes.Contains(out.Stderr, []byte(" config_ns=")) || !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal(out)
	}
	if e := os.Remove(filepath.Join(dir, "Beta.json")); e != nil {
		t.Fatal(e)
	}
	a, e := os.ReadFile(filepath.Join(dir, "Alpha.json"))
	if e != nil {
		t.Fatal(e)
	}
	out = invoke(t, repo, "validate", "--config", cfg, "--spec", "alpha", "--metrics", "--diagnostic-format=json")
	result(t, out, 0)
	m := configMetrics(t, out)
	if m.InputBytes != int64(len(a)) || m.PlannedFiles != 1 || m.PhaseNS["commit"] != 0 || m.PhaseNS["check"] != 0 {
		t.Fatal(m)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("validate changed output")
	}
	if e := os.WriteFile(cfg, []byte("config_version=1\n[defaults]\noutput_dir='generated'\n"), 0600); e != nil {
		t.Fatal(e)
	}
	out = invoke(t, repo, "generate", "--config", cfg, "--reconcile", "--check", "--metrics", "--diagnostic-format=json")
	result(t, out, 3)
	m = configMetrics(t, out)
	if m.InputBytes != 0 || m.PlannedFiles != 0 || len(m.PhaseNS) != 2 {
		t.Fatal("empty envelope invented read/generate", m)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("empty check changed output")
	}
	missing := invoke(t, repo, "generate", "--config", filepath.Join(dir, "missing.toml"), "--metrics", "--diagnostic-format=json")
	fails(t, missing, 4, "E_CONFIG_IO")
	m = configMetrics(t, missing)
	if m.ConfigBytes == nil || *m.ConfigBytes != 0 || m.InputBytes != 0 || len(m.PhaseNS) != 1 {
		t.Fatal(m)
	}
	errText := invoke(t, repo, "generate", "--config", filepath.Join(dir, "missing.toml"), "--metrics")
	if errText.Code != 4 || len(errText.Stdout) != 0 || !strings.Contains(string(errText.Stderr), "metrics: ") {
		t.Fatal(errText)
	}
}

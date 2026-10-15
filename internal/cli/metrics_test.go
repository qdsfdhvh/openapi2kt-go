package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCLIMetricsClockAndDeterministicArtifacts(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	data := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"text":{"type":"string"}}}}}}`)
	if err := os.WriteFile(input, data, 0600); err != nil {
		t.Fatal(err)
	}
	args := arguments{command: "validate", input: input, packageName: "org.example.metrics"}
	called := 0
	clock := func() time.Time { called++; return time.Unix(0, int64(called*10)) }
	baseCode, base := executeWithClock(args, clock)
	if baseCode != 0 || called != 0 || base.Metrics != nil {
		t.Fatal(baseCode, called, base)
	}
	args.metrics = true
	code, r := executeWithClock(args, clock)
	if code != 0 || called != 16 || r.Metrics.TotalNS != 150 || !reflect.DeepEqual(r.Metrics.PhaseNS, map[string]int64{"read": 10, "generate": 110}) || !reflect.DeepEqual(r.Metrics.CorePhaseNS, map[string]int64{"options": 10, "parse": 10, "analyze": 10, "transform": 10, "emit": 10}) || r.Metrics.InputBytes != int64(len(data)) || r.Metrics.PlannedFiles != 1 {
		t.Fatal(code, called, r)
	}
	measured := r.Metrics
	r.Metrics = nil
	if !reflect.DeepEqual(base, r) {
		t.Fatal("metrics altered report", base, r)
	}
	var out bytes.Buffer
	printMetrics(measured, &out)
	text := out.String()
	if !strings.Contains(text, " read_ns=10 generate_ns=110 core_options_ns=10 core_parse_ns=10 core_analyze_ns=10 core_transform_ns=10 core_emit_ns=10") || strings.Contains(text, input) {
		t.Fatal(text)
	}
}
func TestCLIMetricsArguments(t *testing.T) {
	for _, args := range [][]string{{"validate", "--input=x", "--metrics"}, {"generate", "--input=x", "--output=o", "--package-name=p", "--metrics", "--check"}} {
		p, d, ok := parse(args)
		if !ok || d != nil || !p.metrics {
			t.Fatal(args, p, d, ok)
		}
	}
	for _, args := range [][]string{{"validate", "--input=x", "--metrics=true"}, {"validate", "--input=x", "--metrics", "--metrics"}, {"validate", "--input=x", "--metrics", "false"}, {"download", "--url=https://example.test", "--output=x", "--metrics"}} {
		if _, _, ok := parse(args); ok {
			t.Fatal(args)
		}
	}
	for _, cmd := range []string{"generate", "validate"} {
		if !strings.Contains(help(cmd), "--metrics") {
			t.Fatal(help(cmd))
		}
	}
	if strings.Contains(help("download"), "--metrics") {
		t.Fatal("downloadmetrics")
	}
	for _, args := range [][]string{{"validate", "--help", "--metrics"}, {"validate", "--metrics=true", "--diagnostic-format=json"}} {
		var out, err bytes.Buffer
		Run(args, &out, &err)
		if strings.Contains(out.String(), `"metrics"`) {
			t.Fatal(out.String())
		}
	}
}

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

func TestNameLockMetricsEnteredPhaseAndKotlinOnlyCounts(t *testing.T) {
	base := t.TempDir()
	input, root := filepath.Join(base, "input.json"), filepath.Join(base, "out")
	data := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"text":{"type":"string"}}}}}}`)
	if e := os.WriteFile(input, data, 0600); e != nil {
		t.Fatal(e)
	}
	args := arguments{command: "validate", input: input, output: root, packageName: "org.example.lockmetric", hoistSharedShapes: true, lockHoistedNames: true, metrics: true}
	calls := 0
	clock := func() time.Time { calls++; return time.Unix(0, int64(calls*10)) }
	code, r := executeWithClock(args, clock)
	if code != 0 || calls != 18 || r.Metrics.TotalNS != 170 || r.Metrics.InputBytes != int64(len(data)) || r.Metrics.PlannedFiles != 1 || !reflect.DeepEqual(r.Metrics.PhaseNS, map[string]int64{"name_lock_read": 10, "read": 10, "generate": 110}) {
		t.Fatal(code, calls, r.Metrics)
	}
	var out bytes.Buffer
	printMetrics(r.Metrics, &out)
	if !strings.Contains(out.String(), " name_lock_read_ns=10 read_ns=10 generate_ns=110") {
		t.Fatal(out.String())
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("metricvalidate wrote")
	}
	args.command = "generate"
	args.metrics = false
	calls = 0
	if code, r := executeWithClock(args, clock); code != 0 || calls != 0 || r.Metrics != nil {
		t.Fatal(code, calls, r)
	}
	if e := os.WriteFile(filepath.Join(root, ".openapi-kt-names.json"), []byte("synthetic edited metadata"), 0600); e != nil {
		t.Fatal(e)
	}
	args.metrics = true
	calls = 0
	code, r = executeWithClock(args, clock)
	if code != 4 || calls != 4 || r.Metrics.TotalNS != 30 || r.Metrics.InputBytes != 0 || r.Metrics.PlannedFiles != 0 || !reflect.DeepEqual(r.Metrics.PhaseNS, map[string]int64{"name_lock_read": 10}) {
		t.Fatal(code, calls, r.Metrics)
	}
}

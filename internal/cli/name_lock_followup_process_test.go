package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNameLockNativeMetricsKotlinCountsAndValidateOverrides(t *testing.T) {
	for _, configured := range []bool{false, true} {
		base := t.TempDir()
		input := lockInput(t, base)
		root := filepath.Join(base, "generated")
		args := lockArgs(input, root)
		reads := int64(1)
		if configured {
			cfg := lockConfig(t, base, []string{"alpha", "beta", "gamma"})
			args = []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			reads = 3
		}
		result(t, invoke(t, base, args...), 0)
		before := snapshot(t, root)
		out := invoke(t, base, append(args, "--metrics", "--check")...)
		r := result(t, out, 0)
		m := configMetrics(t, out)
		var size int64
		for _, p := range r.Files {
			if !strings.HasSuffix(p, ".kt") {
				t.Fatal("metadata in report", p)
			}
			size += int64(len(read(t, filepath.Join(root, p))))
		}
		if m.PlannedFiles != int64(len(r.Files)) || m.PlannedBytes != size || m.InputBytes != int64(len(read(t, input)))*reads {
			t.Fatal("metadata entered Kotlin/input counts", m, size)
		}
		if ns, ok := m.PhaseNS["name_lock_read"]; !ok || ns < 0 {
			t.Fatal("missing native state-read phase", m)
		}
		text := append([]string(nil), args...)
		for i, a := range text {
			if a == "--diagnostic-format=json" {
				text[i] = "--diagnostic-format=text"
			}
		}
		out = invoke(t, base, append(text, "--metrics", "--check")...)
		if out.Code != 0 || !strings.Contains(string(out.Stderr), "name_lock_read_ns=") {
			t.Fatal("native text metrics", out)
		}
		if configured {
			v := append([]string(nil), args...)
			v[0] = "validate"
			v = append(v, "--spec=alpha", "--output", root, "--model-name-suffix=Dto")
			result(t, invoke(t, base, v...), 0)
			fails(t, invoke(t, base, append(v, "--lock-hoisted-names=false")...), 2, "E_USAGE")
		}
		if !reflect.DeepEqual(before, snapshot(t, root)) {
			t.Fatal("metric check/validate override mutated state")
		}
	}
}

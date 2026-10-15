package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestFullTOMLResourceBudgets(t *testing.T) {
	// Exercise each independent budget below the input byte limit, including
	// the dependency's initial parser nesting guard on a single expression.
	for _, tc := range []struct{ name, text, code string }{
		{"deep arrays", "config_version=1\ndefaults.exclude_apis=" + strings.Repeat("[", MaxDepth+1) + "'x'" + strings.Repeat("]", MaxDepth+1), "E_CONFIG_LIMIT"},
		{"deep inline tables", "config_version=1\ndefaults=" + strings.Repeat("{x=", MaxDepth+1) + "'x'" + strings.Repeat("}", MaxDepth+1), "E_CONFIG_LIMIT"},
		{"deep dotted key", "config_version=1\n" + strings.Repeat("x.", MaxDepth) + "x='x'", "E_CONFIG_LIMIT"},
		{"library nesting guard", "config_version=1\ndefaults.exclude_apis=" + strings.Repeat("[", 10001) + "'x'" + strings.Repeat("]", 10001), "E_CONFIG_LIMIT"},
		{"wide array", "config_version=1\ndefaults.exclude_apis=[" + strings.Repeat("'x',", MaxNodes) + "]", "E_CONFIG_LIMIT"},
		{"within structural depth", "config_version=1\ndefaults.exclude_apis=" + strings.Repeat("[", MaxDepth-2) + "'x'" + strings.Repeat("]", MaxDepth-2), "E_CONFIG_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.text) > MaxBytes {
				t.Fatal("test masks structural budget with byte limit")
			}
			_, err := Parse([]byte(tc.text))
			if err == nil || err.Code != tc.code {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
	for _, n := range []int{8188, 8192} {
		var text strings.Builder
		text.WriteString("config_version=1\n")
		for i := 1; i < n; i++ {
			fmt.Fprintf(&text, "x%d=1\n", i)
		}
		_, err := Parse([]byte(text.String()))
		want := "E_CONFIG_UNKNOWN_FIELD"
		if n == 8192 {
			want = "E_CONFIG_LIMIT"
		}
		if err == nil || err.Code != want {
			t.Fatalf("work budget with %d keys: want %s, got %v", n, want, err)
		}
	}
	// Nested array entries consume tracker slots even when they have no
	// keys. Later root lookups must be rejected before scanning those slots.
	var anonymousWork strings.Builder
	anonymousWork.WriteString("config_version=1\nx=[" + strings.Repeat("[],", 50000) + "]\n")
	for i := range 4000 {
		fmt.Fprintf(&anonymousWork, "x%d=1\n", i)
	}
	if anonymousWork.Len() > MaxBytes {
		t.Fatal("test masks work budget with byte limit")
	}
	if _, err := Parse([]byte(anonymousWork.String())); err == nil || err.Code != "E_CONFIG_LIMIT" {
		t.Fatal("anonymous tracker entries did not exhaust work", err)
	}
	// A wide legal string array near the node budget still has its old type.
	text := "config_version=1\ndefaults.exclude_apis=[" + strings.Repeat("'x',", MaxNodes-7) + "]"
	cfg, err := Parse([]byte(text))
	if err != nil || len(cfg.Defaults["exclude_apis"].([]string)) != MaxNodes-7 {
		t.Fatal("legal array near node limit", err)
	}
	if _, err := Parse([]byte(text + strings.Repeat(" ", MaxBytes-len(text)))); err != nil {
		t.Fatal("exact byte limit", err)
	}
	for _, inline := range []bool{false, true} {
		for _, n := range []int{MaxSpecs, MaxSpecs + 1} {
			var b strings.Builder
			b.WriteString("config_version=1\n")
			if inline {
				b.WriteString("specs=[")
			}
			for i := range n {
				if inline {
					fmt.Fprintf(&b, "{name='s%d'},", i)
				} else {
					fmt.Fprintf(&b, "[[specs]]\nname='s%d'\n", i)
				}
			}
			if inline {
				b.WriteString("]")
			}
			cfg, err := Parse([]byte(b.String()))
			if n == MaxSpecs && (err != nil || len(cfg.Specs) != n) || n > MaxSpecs && (err == nil || err.Code != "E_CONFIG_LIMIT") {
				t.Fatalf("spec budget inline=%v n=%d: %v", inline, n, err)
			}
		}
	}
}

func TestTOMLStringsPreserveDecodedPaths(t *testing.T) {
	for _, tc := range []struct{ value, want string }{
		{`'C:\synthetic\spec.json'`, `C:\synthetic\spec.json`},
		{`"C:\\synthetic\\spec.json"`, `C:\synthetic\spec.json`},
		{"'''\ndir/line\nbreak.json'''", "dir/line\nbreak.json"},
		{`"dir/\x61lpha.json"`, "dir/alpha.json"},
		{`"dir/\U0001f600.json"`, "dir/😀.json"},
		{`"dir/quote\".json"`, "dir/quote\".json"},
	} {
		cfg, err := Parse([]byte("config_version=1\ndefaults.input=" + tc.value))
		if err != nil || cfg.Defaults["input"] != tc.want {
			t.Fatalf("decoded path want %q, got %#v, diagnostic %v", tc.want, cfg, err)
		}
	}
}

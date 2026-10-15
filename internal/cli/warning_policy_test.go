package cli

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestWarningPolicyArguments(t *testing.T) {
	for _, args := range [][]string{
		{"validate", "--input=x", "--deny-warnings"},
		{"validate", "--input=x", "--allow-warning=W_FUTURE", "--deny-warnings", "--allow-warning", "W_A1"},
		{"generate", "--input=x", "--output=o", "--package-name=p", "--deny-warnings", "--check"},
	} {
		p, d, ok := parse(args)
		if !ok || d != nil || !p.denyWarnings {
			t.Fatal(args, p, d, ok)
		}
	}
	for _, args := range [][]string{
		{"validate", "--input=x", "--deny-warnings=true"},
		{"validate", "--input=x", "--deny-warnings", "--deny-warnings"},
		{"validate", "--input=x", "--allow-warning=W_A"},
		{"validate", "--input=x", "--deny-warnings", "--allow-warning"},
		{"download", "--url=https://example.test", "--output=x", "--deny-warnings"},
	} {
		if _, _, ok := parse(args); ok {
			t.Fatal(args)
		}
	}
	for _, code := range []string{"", "W_", "w_A", "E_A", "W_a", "W_A,W_B", "W_*", "W_雪", "W_A\nB", "W_A-B", "W_A B"} {
		if _, _, ok := parse([]string{"validate", "--input=x", "--deny-warnings", "--allow-warning=" + code}); ok {
			t.Fatal(code)
		}
	}
	for _, command := range []string{"generate", "validate"} {
		if h := help(command); !strings.Contains(h, "--deny-warnings") || !strings.Contains(h, "--allow-warning") {
			t.Fatal(h)
		}
	}
	if strings.Contains(help("download"), "--deny-warnings") {
		t.Fatal("download policy")
	}
}
func TestWarningPolicyKeepsSourceAndOrder(t *testing.T) {
	root, child := "", "/components/schemas/a~0~1\n"
	w := core.Diagnostics{{Code: "W_A", Message: "safe", Pointer: &root}, {Code: "W_B", Message: "safe", Pointer: &child}, {Code: "W_A", Message: "safe", Pointer: &child}}
	before := append(core.Diagnostics(nil), w...)
	d := deniedWarnings(w, []string{"W_B", "W_B", "W_FUTURE"})
	if len(d) != 2 || *d[0].Pointer != root || *d[1].Pointer != child || !reflect.DeepEqual(before, w) {
		t.Fatal(d, w)
	}
	if len(deniedWarnings(w, []string{"W_A", "W_B"})) != 0 || len(deniedWarnings(nil, nil)) != 0 {
		t.Fatal("allowed or empty rejected")
	}
}

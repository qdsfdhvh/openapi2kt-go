package cli_test

import (
	"reflect"
	"strings"
	"testing"
)

func TestNativeEnumValidationNativeWireTypeAndFailurePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"code":{"type":"integer","enum":[1,2]}}}}}}`))
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	found := false
	for _, file := range result(t, p.generate(t, "--validation", "--check"), 0).Files {
		if strings.Contains(string(read(t, p.Output+"/"+file)), "does not belong to enum.") {
			found = true
		}
	}
	if !found || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("native checked enum missing or check wrote")
	}
	p.change(t, func(doc map[string]any) {
		schemas(doc)["Record"].(map[string]any)["properties"].(map[string]any)["code"].(map[string]any)["enum"] = []any{1, 1}
	})
	for _, check := range []bool{false, true} {
		flags := []string{"--validation"}
		if check {
			flags = append(flags, "--check")
		}
		fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid enum overwrote output")
		}
	}
}

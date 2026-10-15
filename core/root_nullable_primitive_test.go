package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func rootNullablePrimitiveInput(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/root-nullable-primitives.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestRootNullablePrimitiveCatalogOrder(t *testing.T) {
	input := rootNullablePrimitiveInput(t)
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	var components map[string]json.RawMessage
	if err := json.Unmarshal(doc["components"], &components); err != nil {
		t.Fatal(err)
	}
	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(components["schemas"], &catalog); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	parts := make([]string, 0, len(names))
	for _, name := range names {
		key, _ := json.Marshal(name)
		parts = append(parts, string(key)+":"+string(catalog[name]))
	}
	components["schemas"] = json.RawMessage("{" + strings.Join(parts, ",") + "}")
	encoded, err := json.Marshal(components)
	if err != nil {
		t.Fatal(err)
	}
	doc["components"] = encoded
	reversed, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		a, b := request(input, "Dto"), request(reversed, "Dto")
		a.Options.Validation, b.Options.Validation = flag, flag
		old, err := core.GenerateWithDiagnostics(a)
		if err != nil {
			t.Fatal(err)
		}
		got, err := core.GenerateWithDiagnostics(b)
		if err != nil || !reflect.DeepEqual(old, got) {
			t.Fatal("catalog reorder", err)
		}
	}
}

func TestRootNullablePrimitiveWholeControl(t *testing.T) {
	b, err := testutil.NullableBuiltinAliasControl(rootNullablePrimitiveInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = flag
		if _, err := core.GenerateWithDiagnostics(req); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRootNullablePrimitiveDesiredSuccess(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(rootNullablePrimitiveInput(t), suffix)
			req.Options.Validation = flag
			got, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(err)
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
			if len(got.Warnings) != 0 {
				t.Fatal(got.Warnings)
			}
			for _, n := range []string{"String", "Boolean", "Int", "Long", "Float", "Double"} {
				if s := source(t, got, "Z"+n+suffix); !strings.Contains(s, "= kotlin."+n+"?") {
					t.Fatal(s)
				}
				if s := source(t, got, "ZBase"+n+suffix); !strings.Contains(s, "= kotlin."+n+"\n") {
					t.Fatal(s)
				}
			}
			for _, n := range []string{"AFalse", "CFalseInherited"} {
				if s := source(t, got, n+suffix); !strings.Contains(s, "= kotlin.String\n") {
					t.Fatal(s)
				}
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, got, n+suffix); !strings.Contains(s, "= kotlin.String?") {
					t.Fatal(s)
				}
			}
			if s := source(t, got, "Holder"+suffix); !strings.Contains(s, "val narrow: kotlin.String,") || !strings.Contains(s, "kotlin.collections.Map<kotlin.String, kotlin.Long?>") {
				t.Fatal(s)
			}
			if s := source(t, got, "SubmitInput"+suffix); !strings.Contains(s, "val requiredNull: kotlin.Long?,") || !strings.Contains(s, "Patch<kotlin.Long>") {
				t.Fatal(s)
			}
		}
	}
}

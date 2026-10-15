package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedAggregationWide(n int, nullable bool) []byte {
	var d struct {
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	_ = json.Unmarshal(aggregationWide(n, nullable), &d)
	var wide struct {
		AllOf []json.RawMessage `json:"allOf"`
	}
	_ = json.Unmarshal(d.Components.Schemas["Wide"], &wide)
	return schemas(`{"Lower":` + string(wide.AllOf[0]) + `,"Upper":` + string(wide.AllOf[1]) + `,"Wide":{"allOf":[{"$ref":"#/components/schemas/Lower"},{"type":"object","allOf":[{"$ref":"#/components/schemas/Upper"}]}]}}`)
}

func TestReferencedNestedAllOfBoundsRetainSlotSourceAndExpansionBudgets(t *testing.T) {
	for _, tc := range []struct {
		n        int
		nullable bool
		pass     bool
	}{{120, false, true}, {121, false, false}, {240, true, true}, {241, true, false}} {
		r := request(referencedAggregationWide(tc.n, tc.nullable), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != tc.pass || e != nil && (e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal(tc, e)
		}
		if e == nil && strings.Count(source(t, got, "WideDto"), "private fun openapiKtValidatePrimitive") != tc.n {
			t.Fatal("Dropped real aggregate slots", tc)
		}
	}
	for _, depth := range []int{46, 47} {
		lower := `{"type":"object","properties":{"value":{"type":"integer","minimum":1}}}`
		for range depth {
			lower = `{"type":"object","allOf":[` + lower + `]}`
		}
		r := request(schemas(`{"Record":{"allOf":[`+lower+`,{"properties":{"value":{"type":"integer","maximum":2}}}]}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (depth == 46) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(depth, e)
		}
	}
	for _, n := range []int{100, 5500} {
		members := make([]string, n)
		for i := range members {
			members[i] = `{"$ref":"#/components/schemas/Lower"}`
		}
		r := request(schemas(`{"Lower":{"properties":{"value":{"type":"integer","minimum":1,"maximum":2}}},"Record":{"allOf":[`+strings.Join(members, ",")+`]}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 100) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, e)
		}
		if e == nil && strings.Count(source(t, got, "RecordDto"), "below minimum") != 1 {
			t.Fatal("Repeated source produced unbounded helper")
		}
	}
	for _, depth := range []int{16, 17} {
		value := `{"type":"object","allOf":[{"properties":{"value":{"type":"integer","minimum":1}}},{"type":"object","allOf":[{"properties":{"value":{"type":"integer","maximum":2}}}]}]}`
		for range depth {
			value = `{"type":"array","minItems":1,"items":` + value + `}`
		}
		r := request(fieldDoc(value), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (depth == 16) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(fmt.Sprint("Checked edge ", depth), e)
		}
	}
}

package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedCountWide(n int, boxed bool) []byte {
	var doc map[string]json.RawMessage
	_ = json.Unmarshal(referencedAggregationWide(n, boxed), &doc)
	var components map[string]json.RawMessage
	_ = json.Unmarshal(doc["components"], &components)
	var catalog map[string]json.RawMessage
	_ = json.Unmarshal(components["schemas"], &catalog)
	// Retain exact number/source field bytes while adding only object counts.
	for _, name := range []string{"Lower", "Upper"} {
		value := string(catalog[name])
		key := `"minProperties":0,`
		if name == "Upper" {
			key = `"maxProperties":240,`
		}
		catalog[name] = json.RawMessage("{" + key + value[1:])
	}
	components["schemas"], _ = json.Marshal(catalog)
	doc["components"], _ = json.Marshal(components)
	result, _ := json.Marshal(doc)
	return result
}

func TestReferencedNestedAllOfCountsRetainOriginalResourceBudgets(t *testing.T) {
	for _, tc := range []struct {
		n, slots int
		boxed    bool
		pass     bool
	}{{120, 240, false, true}, {121, 242, false, false}, {240, 240, true, true}, {241, 241, true, false}} {
		r := request(referencedCountWide(tc.n, tc.boxed), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != tc.pass || e != nil && (e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(tc, e)
		}
		if e == nil {
			text := source(t, got, "WideDto")
			if strings.Count(text, "private fun openapiKtValidatePrimitive") != tc.n || strings.Count(text, "element.size") != 2 || strings.Count(text, "generatedSerializer()") != 1 {
				t.Fatal("Count wrapper changed slots or lost primitive checks", tc.slots)
			}
		}
	}
	for _, depth := range []int{46, 47} {
		lower := `{"type":"object","minProperties":0,"properties":{"value":{"type":"integer","minimum":1}}}`
		for range depth {
			lower = `{"type":"object","allOf":[` + lower + `]}`
		}
		r := request(schemas(`{"Record":{"allOf":[`+lower+`,{"maxProperties":2,"properties":{"value":{"type":"integer","maximum":2}}}]}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (depth == 46) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(depth, e)
		}
	}
	for _, hops := range []int{46, 47} {
		terminal := `{"type":"object","minProperties":1,"properties":{"value":{"type":"integer"}}}`
		consumer := fmt.Sprintf(`{"allOf":[{"$ref":"#/components/schemas/A%02d"},{"properties":{"value":{"type":"integer"}},"maxProperties":2}]}`, hops-1)
		r := request(rootSpecialChain(hops, terminal, consumer), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (hops == 46) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Actual named alias count source height", hops, e)
		}
		if e == nil && !strings.Contains(source(t, got, "HolderDto"), "element.size < 1") {
			t.Fatal("Named alias source lost whole-object check")
		}
	}
	for _, depth := range []int{16, 17} {
		value := `{"type":"object","allOf":[{"properties":{"value":{"type":"integer"}},"minProperties":1},{"type":"object","allOf":[{"properties":{"value":{"type":"integer"}},"maxProperties":1}]}]}`
		for range depth {
			value = `{"type":"array","minItems":1,"items":` + value + `}`
		}
		r := request(fieldDoc(value), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (depth == 16) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Actual checked collection chain", depth, e)
		}
	}
}

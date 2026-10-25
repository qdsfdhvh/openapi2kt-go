package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleAllOfGraduatedOriginalInput(t *testing.T) {
	for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
		input := schemas(`{"Item":{"allOf":[{"properties":{"value":{"type":"string"}}}]},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}]` + tail + `}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("original unsupported allOf input did not graduate", err)
		}
	}
}

func TestTupleAllOfCumulativeSourcesAndRepeatedTarget(t *testing.T) {
	fields := map[string]any{}
	for i := range 50 {
		fields["p"+strings.Repeat("x", i)] = map[string]any{"type": "string"}
	}
	for _, tuple := range []bool{false, true} {
		catalog := map[string]any{"Base": map[string]any{"properties": fields}}
		for i := range 100 {
			name := "Target" + strings.Repeat("x", i)
			catalog[name] = map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/Base"}, map[string]any{"properties": map[string]any{"extra": map[string]any{"type": "integer"}}}}}
			array := map[string]any{"type": "array"}
			ref := map[string]any{"$ref": "#/components/schemas/" + name}
			if tuple {
				array["prefixItems"] = []any{ref}
				array["items"] = map[string]any{"type": "integer"}
			} else {
				array["items"] = ref
			}
			catalog["Array"+strings.Repeat("x", i)] = array
		}
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if err != nil {
			t.Fatal(err)
		}
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if !tuple && err != nil {
			t.Fatal("new tuple budget changed accepted ordinary allOf leaf role", err)
		}
		if tuple && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("overlapping tuple source snapshots escaped aggregate cap", err)
		}
	}
	member := `{"$ref":"#/components/schemas/Item"}`
	input := schemas(`{"Item":{"allOf":[{"properties":{"value":{"type":"string"}}}]},"Probe":{"type":"array","prefixItems":[` + strings.TrimSuffix(strings.Repeat(member+",", 1000), ",") + `]}}`)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("repeated original target multiplied snapshots", err)
	}
}

func TestTupleAllOfReferences(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/tuple-allof-references.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.PackageName = "org.example.tupleallof"
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ name, want string }{{"RowsDto", "ItemDto"}, {"MaybeRowsDto", "ItemDto?"}, {"InternalRowsDto", "OwnerDto.Node"}, {"NodeRowsDto", "NodeDto"}, {"MixedRowsDto", "kotlinx.serialization.json.JsonElement"}, {"ScalarTailDto", "kotlin.Int"}} {
				text := source(t, got, tc.name)
				if !strings.Contains(text, tc.want) {
					t.Fatal(tc, text)
				}
			}
			counts := map[string]int{}
			for _, warning := range got.Warnings {
				counts[warning.Code]++
			}
			if counts["W_TUPLE_APPROXIMATION"] != 7 || counts["W_ALL_OF_APPROXIMATION"] != 4 || len(got.Warnings) != 11 {
				t.Fatal(got.Warnings)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("tuple allOf nondeterminism", err)
			}
		}
	}
}

func TestTupleAllOfNoConstructorSources(t *testing.T) {
	for _, target := range []string{`{"allOf":[{}]}`, `{"allOf":[{"properties":{"pin":{"const":"only"}}}]}`, `{"allOf":[{"additionalProperties":{"type":"string"}}]}`, `{"allOf":[{"oneOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"integer"}}}]}]}`} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			rq := request(schemas(`{"Item":`+target+`,"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}]`+tail+`}}`), "Dto")
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("tuple allOf manufactured constructor", target, err)
			}
		}
	}
}

func TestTupleAllOfOwnDefaultAndUniqueRolesRemainSeparate(t *testing.T) {
	for _, tc := range []struct{ probe, pointer string }{
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"default":[{"value":"valid"}]}`, "/components/schemas/Probe/default"},
		{`{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"default":[{"value":"valid"}]}}}`, "/components/schemas/Probe/properties/rows/prefixItems"},
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"uniqueItems":true}`, "/components/schemas/Probe/uniqueItems"},
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}`, "/components/schemas/Probe/uniqueItems"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Item":{"allOf":[{"properties":{"value":{"type":"string"}},"required":["value"]}]},"Probe":`+tc.probe+`}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if checked && tc.probe == `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}` {
				if err != nil {
					t.Fatal("actual allOf DTO explicit-tail List equality", err)
				}
				found := false
				for _, w := range got.Warnings {
					found = found || w.Code == "W_TUPLE_APPROXIMATION"
				}
				if !found {
					t.Fatal("lost original tuple approximation")
				}
				continue
			}
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("tuple introduced an independent default/nominal uniqueness role", tc, checked, err)
			}
		}
	}
}

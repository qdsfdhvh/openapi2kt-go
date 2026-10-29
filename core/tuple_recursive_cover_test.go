package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const tupleCoverAliases = `,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Narrow":{"$ref":"#/components/schemas/Maybe","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`

func TestTupleRecursiveCoverSources(t *testing.T) {
	for _, graph := range []string{`"Value":` + coverChainRoot("Value", "Value"), `"Value":` + coverChainRoot("ToOther", "Other") + `,"ToOther":{"$ref":"#/components/schemas/Other"},"Other":` + coverChainRoot("Value", "Value")} {
		for _, target := range []string{"Value", "Maybe", "Narrow", "Bridge", "Hops/properties/value"} {
			for _, tail := range []string{"", `,"items":true`, `,"items":false`, `,"items":{"type":"integer"}`} {
				input := schemas(`{` + graph + tupleCoverAliases + `,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}]` + tail + `}}`)
				for _, checked := range []bool{false, true} {
					rq := request(input, "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(target, tail, checked, err)
					}
					want := "kotlinx.serialization.json.JsonElement"
					if tail == `,"items":{"type":"integer"}` {
						want = "kotlin.Int"
					} else if tail != `,"items":true` && (target == "Maybe" || target == "Bridge" || strings.Contains(target, "/")) {
						want += "?"
					}
					if !strings.Contains(source(t, got, "RowsDto"), "List<"+want+">") {
						t.Fatal(target, tail, source(t, got, "RowsDto"))
					}
					if again, failure := core.GenerateWithDiagnostics(rq); !reflect.DeepEqual(got, again) || failure != nil {
						t.Fatal("recursive tuple nondeterminism", failure)
					}
				}
			}
		}
	}
}

func TestTupleRecursiveCoverOwnDefaults(t *testing.T) {
	for _, target := range []string{"Value", "Maybe", "Bridge"} {
		input := schemas(`{"Value":` + coverChainRoot("Value", "Value") + tupleCoverAliases + `,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}],"default":[null,{"description":{"$ref":"data","allOf":[1]}},[true,"ready"]]},"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}],"default":[1,null]},"required":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}],"default":[]}},"required":["required"]}}`)
		for _, role := range []string{"root", "alias", "field", "all"} {
			var doc map[string]any
			if err := json.Unmarshal(input, &doc); err != nil {
				t.Fatal(err)
			}
			catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
			if role != "all" {
				if role != "root" {
					delete(catalog["Rows"].(map[string]any), "default")
				}
				if role != "alias" {
					delete(catalog["Alias"].(map[string]any), "default")
				}
				if role != "field" {
					for _, field := range catalog["Holder"].(map[string]any)["properties"].(map[string]any) {
						delete(field.(map[string]any), "default")
					}
				}
			}
			candidate, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := core.GenerateWithDiagnostics(request(candidate, "Dto")); err != nil {
				for _, d := range err.(core.Diagnostics) {
					if d.Pointer != nil {
						t.Log(*d.Pointer)
					}
				}
				t.Fatal(target, role, err)
			}
		}
	}
	for _, bad := range []string{`,"default":null`, `,"nullable":"true"`, `,"minimum":1`} {
		input := schemas(`{"Value":` + coverChainRoot("Value", "Value") + `,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"` + bad + `}],"items":true}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("discarded source admitted unsupported annotation", bad, err)
		}
	}
}

func TestTupleRecursiveCoverErasedDepth(t *testing.T) {
	for _, graph := range []string{`"Value":` + coverChainRoot("Value", "Value"), `"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + coverChainRoot("Value", "Value")} {
		for count := 39; count <= 48; count++ {
			props := []string{}
			for i := 0; i < count; i++ {
				target := "Value"
				if i > 0 {
					target = fmt.Sprintf("Hops/properties/h%d", i-1)
				}
				props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
			}
			catalog := graph + `,"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
			target := fmt.Sprintf("Hops/properties/h%d", count-1)
			var ordinary error
			for _, role := range []string{"ordinary", "uniform", "raw", "scalar-tail"} {
				array := `"type":"array","items":{"$ref":"#/components/schemas/` + target + `"}`
				if role != "ordinary" {
					array = `"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}]`
					if role == "raw" {
						array += `,"items":true`
					} else if role == "scalar-tail" {
						array += `,"items":{"type":"integer"}`
					}
				}
				got, err := core.GenerateWithDiagnostics(request(schemas(`{`+catalog+`,"Rows":{`+array+`},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}}`), "Dto"))
				if role == "ordinary" {
					ordinary = err
				} else if (ordinary == nil) != (err == nil) {
					t.Fatal("erased recursive graph depth differs", count, role, ordinary, err)
				}
				if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(count, role, err)
				}
			}
		}
	}
}

func TestTupleRecursiveCoverSharedSourceBudget(t *testing.T) {
	for _, count := range []int{1, 500} {
		for _, positional := range []bool{false, true} {
			parts := []string{}
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("Value%03d", i)
				parts = append(parts, `"`+name+`":`+coverChainRoot(name, name))
				array := `"type":"array","items":{"$ref":"#/components/schemas/` + name + `"}`
				if positional {
					array = `"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + name + `"}],"items":true`
				}
				parts = append(parts, fmt.Sprintf(`"Rows%03d":{%s}`, i, array))
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
			if count == 500 && positional {
				if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("distinct original cover graphs escaped shared tuple budget", err)
				}
			} else if err != nil {
				t.Fatal("original ordinary/small tuple control rejected", count, positional, err)
			}
		}
	}
}

func TestTupleRecursiveCoverDefaultBoundsAndSuppression(t *testing.T) {
	for _, annotation := range []string{`7`, `null`, `[` + strings.Repeat(`0,`, 16384) + `0]`, `["` + strings.Repeat("x", 32769) + `"]`} {
		input := schemas(`{"Value":` + coverChainRoot("Value", "Value") + `,"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":` + annotation + `}},"required":["rows"]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("required suppression hid invalid/over-budget outer annotation", len(annotation), err)
		}
	}
	input := schemas(`{"Value":` + coverChainRoot("Value", "Value") + `,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"nullable":true,"default":null}}`)
	if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
		t.Fatal("outer nullable null factory", err)
	}
}

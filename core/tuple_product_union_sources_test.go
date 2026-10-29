package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleProductUnionSources(t *testing.T) {
	product := borrowedProductUnionSource()
	for _, reference := range []bool{false, true} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			for _, checked := range []bool{false, true} {
				member := product
				if reference {
					member = `{"$ref":"#/components/schemas/Alias"}`
				}
				rq := request(schemas(`{"Product":`+product+`,"Alias":{"$ref":"#/components/schemas/Product"},"Rows":{"type":"array","prefixItems":[`+member+`]`+tail+`}}`), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(reference, tail, checked, err)
				}
				text := source(t, got, "RowsDto")
				if tail == "" && strings.Contains(text, "List<kotlinx.serialization.json.JsonElement>") || tail != "" && !strings.Contains(text, "List<kotlin.Int>") {
					t.Fatal("lost actual product union codec", text)
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("product tuple nondeterminism", err)
				}
			}
		}
	}
}

func TestTupleProductUnionOwnDefaults(t *testing.T) {
	product := borrowedProductUnionSource()
	for _, reference := range []bool{false, true} {
		for _, root := range []bool{false, true} {
			for _, value := range []string{`[{"kind":"id","id":"ready","common":"shared"}]`, `[{"kind":"id"}]`, `[{"kind":"unknown","id":"ready"}]`} {
				member := product
				if reference {
					member = `{"$ref":"#/components/schemas/Product"}`
				}
				tuple := `{"type":"array","prefixItems":[` + member + `],"default":` + value + `}`
				owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
				if !root {
					owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
				}
				got, err := core.GenerateWithDiagnostics(request(schemas(`{"Product":`+product+`,`+owner+`}`), "Dto"))
				if (err == nil) != (value == `[{"kind":"id","id":"ready","common":"shared"}]`) || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal(reference, root, value, err)
				}
			}
		}
	}
}

func TestTupleProductUnionNullableAndSourceIdentity(t *testing.T) {
	product := strings.TrimSuffix(borrowedProductUnionSource(), "}") + `,"nullable":true}`
	for _, member := range []string{product, `{"$ref":"#/components/schemas/Product"}`, `{"$ref":"#/components/schemas/Nullable"}`, `{"$ref":"#/components/schemas/Nullable","nullable":false}`} {
		input := schemas(`{"Product":` + product + `,"Nullable":{"$ref":"#/components/schemas/Product","nullable":true},"Rows":{"type":"array","prefixItems":[` + member + `],"default":[null]}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("inline nullable or named Null carrier", member, err)
		}
	}
	for _, prefix := range []string{`[` + product + `,` + product + `]`, `[{"$ref":"#/components/schemas/Product"},{"$ref":"#/components/schemas/Nullable"}]`} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Product":`+product+`,"Nullable":{"$ref":"#/components/schemas/Product","nullable":true},"Rows":{"type":"array","prefixItems":`+prefix+`}}`), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "RowsDto"), "List<kotlinx.serialization.json.JsonElement>") {
			t.Fatal("distinct original source/null views must remain mixed", err)
		}
	}
	input := schemas(`{"Owner":{"properties":{"choice":` + borrowedProductUnionSource() + `}},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Owner/properties/choice"}],"default":[{"kind":"name","name":"ready"}]}}`)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("existing internal actual product classifier", *err.(core.Diagnostics)[0].Pointer, err)
	}
}

func TestTupleProductUnionCartesianAndDepthLimits(t *testing.T) {
	for _, size := range []int{15, 16} {
		var left, right []string
		for i := 0; i < size; i++ {
			left = append(left, fmt.Sprintf(`{"properties":{"kind":{"const":"k%d"},"value":{"type":"integer"}}}`, i))
		}
		for i := 0; i < 16; i++ {
			right = append(right, fmt.Sprintf(`{"properties":{"mode":{"const":"m%d"}}}`, i))
		}
		product := `{"allOf":[{"oneOf":[` + strings.Join(left, ",") + `]},{"anyOf":[` + strings.Join(right, ",") + `]}]}`
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":{"type":"array","prefixItems":[`+product+`]`+tail+`}}`), "Dto"))
			if size == 15 && err != nil || size == 16 && (err == nil || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
				t.Fatal("Cartesian branch cap", size, tail, err)
			}
		}
	}
	for _, referenced := range []bool{false, true} {
		boundary := 0
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			last := 0
			for depth := 39; depth <= 49; depth++ {
				member := borrowedProductUnionSource()
				if referenced {
					member = `{"$ref":"#/components/schemas/Product"}`
				}
				leaf := `{"type":"array","prefixItems":[` + member + `]` + tail + `}`
				for i := 1; i < depth; i++ {
					leaf = `{"type":"array","items":` + leaf + `}`
				}
				input := schemas(`{"Product":` + borrowedProductUnionSource() + `,"Rows":` + leaf + `,"Consumer":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}}`)
				_, err := core.Generate(request(input, "Dto"))
				if err == nil {
					last = depth
				} else if err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
					t.Fatal("original product depth", referenced, depth, tail, err)
				}
			}
			if last < 39 || last > 48 {
				t.Fatal("missing source boundary", last)
			}
			if boundary == 0 {
				boundary = last
			} else if last != boundary {
				t.Fatal("erased composition source work escaped cached height", referenced, tail, boundary, last)
			}
		}
	}
}

func TestTupleProductUnionConjoinedDefaultsAndDiscardedSources(t *testing.T) {
	product := `{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"text"},"name":{"type":"string"}},"required":["name"]},{"properties":{"kind":{"const":"count"},"n":{"type":"integer"}},"required":["n"]}]},{"anyOf":[{"properties":{"mode":{"const":"fast"},"fast":{"type":"boolean"}},"required":["fast"]},{"properties":{"mode":{"const":"slow"},"slow":{"type":"boolean"}},"required":["slow"]}]}]}`
	for _, value := range []string{`[{"kind":"text","name":"ready","mode":"fast","fast":true}]`, `[{"kind":"text","name":"ready","mode":"slow","slow":false}]`, `[{"kind":"text","name":"ready","mode":"future","fast":true}]`, `[{"kind":"text","name":"ready","mode":"slow","fast":true}]`} {
		for _, reference := range []bool{false, true} {
			member := product
			if reference {
				member = `{"$ref":"#/components/schemas/Product"}`
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Product":`+product+`,"Rows":{"type":"array","prefixItems":[`+member+`],"default":`+value+`}}`), "Dto"))
			valid := !strings.Contains(value, `"future"`) && !strings.Contains(value, `"slow","fast"`)
			if (err == nil) != valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("conjoined default selection", reference, value, err)
			}
		}
	}
	for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
		bad := `{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"one"},"nested":{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"future"}}}}},{"properties":{"kind":{"const":"two"},"n":{"type":"integer"}}}]},{"properties":{"common":{"type":"string"}}}]}`
		input := schemas(`{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[` + bad + `]` + tail + `}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Rows/prefixItems/0/allOf/0/oneOf/0/properties/nested/properties/state/default")
		cycle := schemas(`{"Rows":{"type":"array","prefixItems":[{"allOf":[{"$ref":"#/components/schemas/Loop"}]}]` + tail + `},"Loop":{"allOf":[{"$ref":"#/components/schemas/Loop"},{"oneOf":[{"properties":{"kind":{"const":"one"},"n":{"type":"integer"}}}]}]}}`)
		got, err := core.GenerateWithDiagnostics(request(cycle, "Dto"))
		if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_CYCLE" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("projected composition cycle was hidden", tail, err)
		}
	}
}

func TestTupleProductUnionOverlappingSourceBudget(t *testing.T) {
	fields := map[string]any{}
	for i := 0; i < 50; i++ {
		fields[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "string"}
	}
	for _, count := range []int{1, 100} {
		for _, tuple := range []bool{false, true} {
			catalog := map[string]any{"Base": map[string]any{"properties": fields}}
			for i := 0; i < count; i++ {
				product := map[string]any{}
				if err := json.Unmarshal([]byte(borrowedProductUnionSource()), &product); err != nil {
					t.Fatal(err)
				}
				product["allOf"].([]any)[1] = map[string]any{"$ref": "#/components/schemas/Base"}
				array := map[string]any{"type": "array", "items": product}
				if tuple {
					array["prefixItems"], array["items"] = []any{product}, true
				}
				catalog[fmt.Sprintf("Rows%03d", i)] = array
			}
			input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if tuple && count == 100 {
				if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("overlapping original product source escaped cumulative budget", err)
				}
			} else if err != nil {
				t.Fatal("source-only or small tuple budget control", count, tuple, err)
			}
		}
	}
}

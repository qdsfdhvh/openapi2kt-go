package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

const composedTerminal = `{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"type":"string","const":"b"},"name":{"type":"string"}},"required":["name"]}]}]}`

func TestComposedUnionNoDiscardedScratchAmbiguousOwner(t *testing.T) {
	for _, tc := range []struct{ body, path string }{
		{`"Owner":{"allOf":[{"properties":{"choice":` + composedTerminal + `}},{"properties":{"choice":{"type":"string"}}}]}`, "Owner/allOf/0/properties/choice"},
		{`"Owner":{"properties":{"id":{"type":"string"}},"additionalProperties":` + composedTerminal + `}`, "Owner/additionalProperties"},
		{`"Owner":{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"choice":` + composedTerminal + `}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]},{"oneOf":[{"properties":{"mode":{"type":"string","const":"x"}}},{"properties":{"mode":{"type":"string","const":"y"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/choice"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			aliasFailure(t, schemas(`{`+tc.body+`,"Alias":{"$ref":"#/components/schemas/`+tc.path+`"}}`), "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
			late := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + tc.body + `}},"paths":{"/product":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/` + tc.path + `"}}}}}}}}}`)
			aliasFailure(t, late, "E_REF_UNSUPPORTED", "/paths/~1product/get/responses/200/content/application~1json/schema")
		})
	}
}

func TestComposedUnionAliasesRetainShapeExclusionsAndNoDrop(t *testing.T) {
	base := `"Product":` + composedTerminal + `,"Maybe":{"$ref":"#/components/schemas/Product","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Product"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
	for _, tc := range []struct{ consumer, code, pointer string }{
		{`{"allOf":[{"$ref":"#/components/schemas/Maybe"},{"properties":{"extra":{"type":"string"}}}]}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Consumer/allOf/0"},
		{`{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}`, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/allOf/0"},
		{`{"oneOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}`, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/oneOf/0"},
	} {
		aliasFailure(t, schemas(`{`+base+`,"Consumer":`+tc.consumer+`}`), tc.code, tc.pointer)
	}
	invalid := strings.Replace(base, `"id":{"type":"string"}`, `"id":{"type":"string","format":"unsupported"}`, 1)
	aliasFailure(t, schemas(`{`+invalid+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Product/allOf/0/oneOf/0/properties/id/format")
	for _, checked := range []bool{false, true} {
		req := request(schemas(`{"Product":{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"bad":{"type":"string","minLength":1}}},{"properties":{"kind":{"type":"string","const":"b"}}}]},{"properties":{"bad":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Product","nullable":true}}`), "Dto")
		req.Options.Validation = checked
		r, e := core.GenerateWithDiagnostics(req)
		if e == nil || len(r.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" {
			t.Fatal("discarded bounds escaped", checked, e)
		}
	}
}

func TestComposedUnionTerminalSourceCacheHeight(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, namedAlias := range []bool{false, true} {
			for _, typed := range []bool{false, true} {
				limit := 47
				if namedAlias {
					limit--
				}
				if typed {
					limit -= 2
				}
				for _, n := range []int{limit, limit + 1} {
					fields := []string{}
					for i := range n {
						target := "Product"
						if namedAlias {
							target = "Maybe"
						}
						if i > 0 {
							target = fmt.Sprint("Hops/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					body := `"Product":` + composedTerminal + `,"Maybe":{"$ref":"#/components/schemas/Product","nullable":true},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
					if typed {
						body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["value"]}`
					}
					req := request(schemas(`{`+body+`}`), "Dto")
					req.Options.Validation = checked
					r, e := core.GenerateWithDiagnostics(req)
					if n == limit {
						if e != nil {
							t.Fatal(checked, namedAlias, typed, n, e)
						}
					} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
						t.Fatal("cached source height lost", checked, namedAlias, typed, n, e)
					}
				}
			}
		}
	}
}

func TestComposedUnionTerminalProductAndSlotCaps(t *testing.T) {
	choice := func(wire string, n int) map[string]any {
		rows := []any{}
		for i := range n {
			rows = append(rows, map[string]any{"properties": map[string]any{wire: map[string]any{"type": "string", "const": fmt.Sprint(wire, i)}}})
		}
		return map[string]any{"oneOf": rows}
	}
	for _, checked := range []bool{false, true} {
		for _, n := range []int{15, 16} {
			input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Product": map[string]any{"allOf": []any{choice("kind", n), choice("mode", 16)}}, "Alias": map[string]any{"$ref": "#/components/schemas/Product", "nullable": true}}}})
			req := request(input, "Dto")
			req.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(req)
			if n == 15 {
				if e != nil || strings.Count(source(t, r, "ProductDto"), " && ") != 240 {
					t.Fatal(n, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, e)
			}
		}
		for _, n := range []int{239, 240} {
			props := map[string]any{}
			required := []string{}
			for i := range n {
				key := fmt.Sprint("p", i)
				props[key] = map[string]any{"type": "boolean"}
				required = append(required, key)
			}
			input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Product": map[string]any{"allOf": []any{map[string]any{"properties": props, "required": required}, choice("kind", 2)}}, "Alias": map[string]any{"$ref": "#/components/schemas/Product", "nullable": true}}}})
			req := request(input, "Dto")
			req.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(req)
			if n == 239 {
				if e != nil {
					t.Fatal(n, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, e)
			}
		}
	}
}

func TestComposedUnionFirstUseFromUnionField(t *testing.T) {
	input := schemas(`{"Product":` + composedTerminal + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Product"}}},"AConsumer":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["value"]},{"properties":{"kind":{"type":"string","const":"b"},"text":{"type":"string"}}}]}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "AConsumerDto"), "value: org.example.models.models.ProductDto") {
		t.Fatal("first field use lost canonical product")
	}
}

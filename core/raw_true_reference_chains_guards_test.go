package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func rawTrueChainCatalog(terminal string) string {
	return `"Value":` + terminal + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
}

func TestRawTrueReferenceChainsSourceDepthAndOneSlotBudgets(t *testing.T) {
	for _, terminal := range []string{`{"type":"object","nullable":true}`, `true`} {
		for _, cached := range []bool{false, true} {
			for _, count := range []int{45, 46, 47} {
				props := []string{}
				for i := 0; i < count; i++ {
					target := "Value"
					if i > 0 {
						target = fmt.Sprintf("Hops/properties/h%d", i-1)
					}
					props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
				}
				body := `"Value":` + terminal + `,"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
				use := fmt.Sprintf("Hops/properties/h%d", count-1)
				if cached {
					body += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
					use = "Bridge"
				}
				body += `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/` + use + `","nullable":false}},"required":["value"]}`
				got, err := core.GenerateWithDiagnostics(request(schemas(`{`+body+`}`), "Dto"))
				pass := count <= 46 && (!cached || count <= 45)
				if pass {
					if err != nil || !strings.Contains(source(t, got, "HolderDto"), "value: kotlinx.serialization.json.JsonElement,") {
						t.Fatal(terminal, cached, count, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("JSON narrowing reset source height", terminal, cached, count, err)
				}
			}
		}
		for _, nullable := range []bool{false, true} {
			for _, n := range []int{240, 241} {
				props, req, params := []string{}, []string{}, []string{}
				for i := 0; i < n; i++ {
					props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, nullable))
					req = append(req, fmt.Sprintf(`"p%d"`, i))
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, nullable))
				}
				catalog := rawTrueChainCatalog(terminal)
				model := schemas(`{` + catalog + `,"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(req, ",") + `]}}`)
				http := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + catalog + `}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`)
				for _, tc := range []struct {
					input []byte
					code  string
				}{{model, "E_MODEL_LIMIT"}, {http, "E_OPERATION_LIMIT"}} {
					got, err := core.GenerateWithDiagnostics(request(tc.input, "Dto"))
					if n == 240 {
						if err != nil {
							t.Fatal(terminal, nullable, n, tc.code, err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != tc.code || len(got.Artifacts.Files) != 0 {
						t.Fatal("JSON source native slot cap changed", terminal, nullable, n, tc.code, err)
					}
				}
			}
		}
	}
}

func TestRawTrueReferenceChainsCompleteGrammarAndSeparateRoles(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ terminal, pointer string }{
			{`{"type":"object","minProperties":1}`, "/components/schemas/Value/minProperties"},
			{`{"type":"object","maxProperties":1}`, "/components/schemas/Value/maxProperties"},
			{`{"type":"object","required":["missing"]}`, "/components/schemas/Value"},
			{`{"allOf":[{"type":"object","minProperties":1}]}`, "/components/schemas/Value/allOf/0/minProperties"},
		} {
			r := request(schemas(`{`+rawTrueChainCatalog(tc.terminal)+`}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("raw source grammar masked/dropped", checked, tc, err)
			}
		}
		for _, terminal := range []string{`{"properties":{"kind":{"type":"string","const":"only"}}}`, `{"allOf":[{"properties":{"kind":{"type":"string","const":"only"}}}]}`, `{"additionalProperties":{"type":"string"}}`} {
			r := request(schemas(`{`+rawTrueChainCatalog(terminal)+`}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if terminal == `{"additionalProperties":{"type":"string"}}` {
				// Preserve the original dictionary input; separate named collection
				// authority now exposes typed Map rather than a raw JSON role.
				if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlin.collections.Map<kotlin.String, kotlin.String>\n") {
					t.Fatal("original named-map graduation lost backing", err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" {
				t.Fatal("const-only/dictionary granted raw role", checked, terminal, err)
			}
		}
		for _, leaf := range []string{`{"type":"object"}`, `true`} {
			r := request(schemas(`{"Owner":{"properties":{"leaf":`+leaf+`}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/leaf"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
				t.Fatal("borrowed raw/true leaf admitted", leaf, err)
			}
		}
		r := request(schemas(`{"Hops":{"properties":{"a":{"$ref":"#/components/schemas/Hops/properties/b"},"b":{"$ref":"#/components/schemas/Hops/properties/a"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/a"}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("transparent JSON source cycle accepted", err)
		}
		r = request(schemas(`{"Value":true,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":null}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
		r.Options.Validation = checked
		got, err = core.GenerateWithDiagnostics(r)
		if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Hops/properties/value/default" {
			t.Fatal("ref sibling dropped", err)
		}
	}
}

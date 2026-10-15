package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func coverChainRoot(array, object string) string {
	return `{"oneOf":[{"type":"null"},{"type":"string"},{"type":"boolean"},{"type":"number"},{"type":"array","items":{"$ref":"#/components/schemas/` + array + `"}},{"type":"object","additionalProperties":{"$ref":"#/components/schemas/` + object + `"}}]}`
}
func TestRecursiveCoverReferenceChainsIncomingGraphAndCachedCollectionDepth(t *testing.T) {
	for _, tc := range []struct {
		catalog string
		height  int
	}{
		{`"Value":` + coverChainRoot("Value", "Value"), 2},
		{`"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + coverChainRoot("Value", "Value"), 4},
		{`"Value":` + coverChainRoot("ToOther", "Other") + `,"ToOther":{"$ref":"#/components/schemas/Other"},"Other":` + coverChainRoot("ToValue", "Value") + `,"ToValue":{"$ref":"#/components/schemas/Value"}`, 6},
		{`"Value":` + coverChainRoot("Other", "End") + `,"Other":` + coverChainRoot("Value", "End") + `,"End":` + coverChainRoot("End", "End"), 6},
	} {
		for _, cached := range []bool{false, true} {
			limit := 46 - tc.height
			if cached {
				limit--
			}
			for _, count := range []int{limit, limit + 1} {
				props := []string{}
				for i := 0; i < count; i++ {
					target := "Value"
					if i > 0 {
						target = fmt.Sprintf("Hops/properties/h%d", i-1)
					}
					props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
				}
				body := tc.catalog + `,"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
				use := fmt.Sprintf("Hops/properties/h%d", count-1)
				if cached {
					body += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
					use = "Bridge"
				}
				body += `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/` + use + `","nullable":false}},"required":["value"]}`
				got, err := core.GenerateWithDiagnostics(request(schemas(`{`+body+`}`), "Dto"))
				if count == limit {
					if err != nil || !strings.Contains(source(t, got, "HolderDto"), "value: kotlinx.serialization.json.JsonElement,") {
						t.Fatal(tc.height, cached, count, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("incomingfalse aliases erased SCC suffix", tc.height, cached, count, err)
				}
			}
		}
		// Four suffix steps: field/ref to Rows plus Rows item/ref to Hops/value.
		// Graph height and one internal+final ref add separately, no scalar max.
		for _, n := range []int{44 - tc.height, 45 - tc.height} {
			leaf := `{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}`
			for i := 0; i < n; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			body := tc.catalog + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Rows":` + leaf + `,"AConsumer":{"properties":{"value":{"$ref":"#/components/schemas/Rows","nullable":false}}}`
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+body+`}`), "Dto"))
			if n == 44-tc.height {
				if err != nil {
					t.Fatal("cachedRows graph depth48", tc.height, n, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("cachedRows graph depth49 accepted", tc.height, n, err)
			}
		}
	}
}

func TestRecursiveCoverReferenceChainsOneSlotLimitsAndGrammar(t *testing.T) {
	base := `"Value":` + coverChainRoot("Value", "Value") + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
	for _, nullable := range []bool{false, true} {
		for _, n := range []int{240, 241} {
			props, req, params := []string{}, []string{}, []string{}
			for i := 0; i < n; i++ {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, nullable))
				req = append(req, fmt.Sprintf(`"p%d"`, i))
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, nullable))
			}
			for _, tc := range []struct {
				input []byte
				code  string
			}{{schemas(`{` + base + `,"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(req, ",") + `]}}`), "E_MODEL_LIMIT"}, {[]byte(`{"openapi":"3.1.0","components":{"schemas":{` + base + `}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`), "E_OPERATION_LIMIT"}} {
				got, err := core.GenerateWithDiagnostics(request(tc.input, "Dto"))
				if n == 240 {
					if err != nil {
						t.Fatal(nullable, n, tc.code, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != tc.code || len(got.Artifacts.Files) != 0 {
					t.Fatal("cover single-slot limits changed", nullable, n, tc.code, err)
				}
			}
		}
	}
	for _, checked := range []bool{false, true} {
		for _, terminal := range []string{strings.Replace(coverChainRoot("Value", "Value"), `{"type":"string"}`, `{"type":"string","minLength":1}`, 1), strings.Replace(coverChainRoot("Value", "Value"), `"#/components/schemas/Value"`, `"#/components/schemas/Missing"`, 1), strings.Replace(coverChainRoot("Value", "Value"), `{"type":"boolean"}`, `{"type":"string"}`, 1)} {
			input := schemas(`{"Value":` + terminal + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("badcompletegraph dropped", checked, terminal, err)
			}
		}
		input := schemas(`{` + base + `,"Borrow":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"id":{"type":"string"}}}]}}`)
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
			t.Fatal("cover lent nominal shape", err)
		}
	}
}

package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBuiltinReferenceChainsSourceAndSlotBudgets(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, n := range []int{46, 47} {
			fields := []string{}
			for i := 0; i < n; i++ {
				target := "Value"
				if i > 0 {
					target = fmt.Sprintf("Hops/properties/h%d", i-1)
				}
				fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
			}
			body := `"Value":{"type":["integer","null"],"format":"int64"},"Hops":{"properties":{` + strings.Join(fields, ",") + `}}`
			use := fmt.Sprintf("Hops/properties/h%d", n-1)
			if cached {
				body += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
				use = "Bridge"
			}
			body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/` + use + `","nullable":false}},"required":["value"]}`
			for _, checked := range []bool{false, true} {
				r := request(schemas(`{`+body+`}`), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				pass := n == 46 && !cached
				if pass {
					if err != nil || !strings.Contains(source(t, got, "ConsumerDto"), "value: kotlin.Long,") {
						t.Fatal(cached, n, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("cached false lost source hops", cached, n, err)
				}
			}
		}
	}
	for _, scalar := range []string{`{"type":"integer","format":"int64"}`, `{"type":"number","format":"double"}`} {
		for _, boxed := range []bool{false, true} {
			limit := 120
			if boxed {
				limit = 240
			}
			for _, n := range []int{limit, limit + 1} {
				props, required, parameters := []string{}, []string{}, []string{}
				for i := 0; i < n; i++ {
					props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, boxed))
					required = append(required, fmt.Sprintf(`"p%d"`, i))
					parameters = append(parameters, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, boxed))
				}
				components := `{"Value":` + scalar + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
				model := schemas(components + `,"Consumer":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
				http := []byte(`{"openapi":"3.1.0","components":{"schemas":` + components + `}},"paths":{"/value":{"get":{"parameters":[` + strings.Join(parameters, ",") + `],"responses":{"204":{}}}}}}`)
				for _, tc := range []struct {
					input []byte
					code  string
				}{{model, "E_MODEL_LIMIT"}, {http, "E_OPERATION_LIMIT"}} {
					got, err := core.GenerateWithDiagnostics(request(tc.input, "Dto"))
					if n == limit {
						if err != nil {
							t.Fatal(scalar, boxed, n, tc.code, err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != tc.code || len(got.Artifacts.Files) != 0 {
						t.Fatal("native/boxed layout changed", scalar, boxed, n, tc.code, err)
					}
				}
			}
		}
	}
}

func TestBuiltinReferenceChainsCheckedDepthAndOriginalSourceDiagnostics(t *testing.T) {
	for _, edges := range []int{16, 17} {
		field := `{"$ref":"#/components/schemas/Hops/properties/value"}`
		for range edges {
			field = `{"type":"array","items":` + field + `}`
		}
		r := request(schemas(`{"Value":{"type":"string","minLength":2},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Consumer":{"properties":{"value":`+field+`},"required":["value"]}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 {
			if err != nil || !strings.Contains(source(t, got, "ConsumerDto"), "length >= 2") {
				t.Fatal(edges, err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Value" || len(got.Artifacts.Files) != 0 {
			t.Fatal("named checks lost original source or edge limit", err)
		}
	}
	for _, tc := range []struct{ body, code, pointer string }{
		{`{"Value":{"type":"integer","x-native-type":"bigint"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "bigint-graduated", ""},      // Original named bigint input graduates only in its own codec/source-role increment.
		{`{"Value":{"type":"integer","x-native-type":"kotlin.Long"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "caller-graduated", ""}, // Original caller-native source graduates independently; no magic field codec.
		{`{"Source":{"properties":{"leaf":{"type":"string"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"}}`, "", ""},                                                                                            // Original borrowed leaf input graduates in its own source-role increment.
		{`{"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`{"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"}}}}`, "E_REF_CYCLE", "/components/schemas/Bridge/$ref"},
		{`{"Value":{"type":"string"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value","minLength":1}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/value/minLength"},
		{`{"Value":{"type":"string"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Missing"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "E_REF_MISSING", "/components/schemas/Hops/properties/value/$ref"},
		{`{"Value":{"type":"string"},"Hops":{"properties":{"value":{"$ref":"other.json#/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "E_REF_UNSUPPORTED", "/components/schemas/Hops/properties/value/$ref"},
		{`{"Value":{"type":"string","default":"x"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Value/default"},
	} {
		if tc.code == "caller-graduated" {
			got, err := core.GenerateWithDiagnostics(request(schemas(tc.body), "Dto"))
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlin.Long\n") || strings.Contains(source(t, got, "HopsDto"), "LongAsStringSerializer") {
				t.Fatal("original caller Long acquired magic profile", err)
			}
		} else if tc.code == "bigint-graduated" {
			got, err := core.GenerateWithDiagnostics(request(schemas(tc.body), "Dto"))
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "public typealias BridgeDto = kotlin.Long\n") || !strings.Contains(source(t, got, "HopsDto"), "LongAsStringSerializer::class") {
				t.Fatal("original bigint root source/field codec changed", err)
			}
		} else if tc.code == "" {
			got, err := core.GenerateWithDiagnostics(request(schemas(tc.body), "Dto"))
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "public typealias BridgeDto = kotlin.String\n") {
				t.Fatal("original borrowed root leaf backing changed", err)
			}
		} else {
			aliasFailure(t, schemas(tc.body), tc.code, tc.pointer)
		}
	}
}

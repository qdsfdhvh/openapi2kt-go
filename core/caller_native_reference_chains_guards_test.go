package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestCallerNativeReferenceChainsSourceAndKnownLayoutBudgets(t *testing.T) {
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
			body := `"Value":{"x-native-type":"org.example.scalarvalues.Money","nullable":true},"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
			use := fmt.Sprintf("Hops/properties/h%d", count-1)
			if cached {
				body += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
				use = "Bridge"
			}
			body += `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/` + use + `","nullable":false}},"required":["value"]}`
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+body+`}`), "Dto"))
			pass := count <= 46 && (!cached || count <= 45)
			if pass {
				if err != nil || !strings.Contains(source(t, got, "HolderDto"), "org.example.scalarvalues.Money,") {
					t.Fatal(cached, count, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("provider alias became source barrier", cached, count, err)
			}
		}
	}
	for _, native := range []string{"kotlin.Long", "kotlin.Double", "org.example.scalarvalues.Money"} {
		for _, nullable := range []bool{false, true} {
			limit := 240
			if !nullable && (native == "kotlin.Long" || native == "kotlin.Double") {
				limit = 120
			}
			for _, n := range []int{limit, limit + 1} {
				props, req, params := []string{}, []string{}, []string{}
				for i := 0; i < n; i++ {
					props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, nullable))
					req = append(req, fmt.Sprintf(`"p%d"`, i))
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, nullable))
				}
				catalog := `{"Value":{"x-native-type":"` + native + `"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
				model := schemas(catalog + `,"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(req, ",") + `]}}`)
				http := []byte(`{"openapi":"3.1.0","components":{"schemas":` + catalog + `}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`)
				for _, tc := range []struct {
					input []byte
					code  string
				}{{model, "E_MODEL_LIMIT"}, {http, "E_OPERATION_LIMIT"}} {
					got, err := core.GenerateWithDiagnostics(request(tc.input, "Dto"))
					if n == limit {
						if err != nil {
							t.Fatal(native, nullable, n, tc.code, err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != tc.code || len(got.Artifacts.Files) != 0 {
						t.Fatal("known native layout cap changed", native, nullable, n, tc.code, err)
					}
				}
			}
		}
	}
}

func TestCallerNativeReferenceChainsOriginalGrammarAndBorrowedRole(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ terminal, pointer string }{
			{`{"x-native-type":"org.example.scalarvalues.Money","properties":{"value":{"type":"string"}}}`, "/components/schemas/Value/x-native-type"},
			{`{"x-native-type":"org.example.scalarvalues.Money","default":"x"}`, "/components/schemas/Value/default"},
			{`{"x-native-type":"kotlin.String;bad"}`, "/components/schemas/Value/x-native-type"},
		} {
			// An excluded structured source can be rejected at the early chain
			// admission seam. A no-chain control establishes its real keyword
			// diagnostics; an early E_REF_UNSUPPORTED is not that evidence.
			control := request(schemas(`{"Value":`+tc.terminal+`}`), "Dto")
			control.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(control)
			if err == nil || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer || len(got.Artifacts.Files) != 0 {
				t.Fatal("original provider terminal grammar lost", checked, tc, err)
			}
			r := request(schemas(`{"Value":`+tc.terminal+`,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
			r.Options.Validation = checked
			got, err = core.GenerateWithDiagnostics(r)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("invalid provider chain accepted", checked, tc, err)
			}
			if !strings.Contains(tc.terminal, "properties") && (err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer) {
				t.Fatal("admitted scalar terminal diagnostic masked", checked, tc, err)
			}
		}
		r := request(schemas(`{"Source":{"properties":{"leaf":{"x-native-type":"org.example.scalarvalues.Money"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed caller native leaf added", err)
		}
	}
}

package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestBigintReferenceChainsSourceAndNativeSlotBudgets(t *testing.T) {
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
			body := `"Value":{"x-native-type":"bigint","nullable":true},"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
			use := fmt.Sprintf("Hops/properties/h%d", count-1)
			if cached {
				body += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
				use = "Bridge"
			}
			body += `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/` + use + `","nullable":false}},"required":["value"]}`
			for _, checked := range []bool{false, true} {
				r := request(schemas(`{`+body+`}`), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				pass := count <= 46 && (!cached || count <= 45)
				if pass {
					if err != nil || !strings.Contains(source(t, got, "HolderDto"), "kotlin.Long,") {
						t.Fatal(cached, count, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("cached false lost full source work", cached, count, err)
				}
			}
		}
	}
	for _, boxed := range []bool{false, true} {
		limit := 120
		if boxed {
			limit = 240
		}
		for _, n := range []int{limit, limit + 1} {
			props, req, params := []string{}, []string{}, []string{}
			for i := 0; i < n; i++ {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, boxed))
				req = append(req, fmt.Sprintf(`"p%d"`, i))
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, boxed))
			}
			catalog := `{"Value":{"x-native-type":"bigint"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
			model := schemas(catalog + `,"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(req, ",") + `]}}`)
			http := []byte(`{"openapi":"3.1.0","components":{"schemas":` + catalog + `}},"paths":{"/value":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`)
			for _, tc := range []struct {
				input []byte
				code  string
			}{{model, "E_MODEL_LIMIT"}, {http, "E_OPERATION_LIMIT"}} {
				got, err := core.GenerateWithDiagnostics(request(tc.input, "Dto"))
				if n == limit {
					if err != nil {
						t.Fatal(boxed, n, tc.code, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != tc.code || len(got.Artifacts.Files) != 0 {
					t.Fatal("bigint gained nominal layout", boxed, n, tc.code, err)
				}
			}
		}
	}
}

func TestBigintReferenceChainsOriginalGrammarAndHelperOwnership(t *testing.T) {
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"Value":{"x-native-type":"bigint","minimum":0},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		keyword := "minimum"
		if checked {
			keyword = "x-native-type"
		}
		if err == nil || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Value/"+keyword || len(got.Artifacts.Files) != 0 {
			t.Fatal("terminal noDrop masked", checked, err)
		}
		for _, terminal := range []string{`{"type":"string","format":"binary"}`, `{"x-native-type":"kotlin.Long"}`, `{"type":"object"}`} {
			r := request(schemas(`{"Value":`+terminal+`,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if terminal == `{"type":"string","format":"binary"}` {
				// Keep the original input and explicitly graduate only the new
				// byte/binary source profile, never the bigint/caller classifier.
				if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlin.ByteArray\n") || strings.Contains(source(t, got, "HopsDto"), "Serializable(with") {
					t.Fatal("original bytes chain source changed", err)
				}
			} else if terminal == `{"x-native-type":"kotlin.Long"}` {
				// Preserve original caller Long input; its own new profile keeps
				// numeric field wire, independently of the bigint magic codec.
				if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlin.Long\n") || strings.Contains(source(t, got, "HopsDto"), "LongAsStringSerializer") {
					t.Fatal("caller Long acquired bigint codec", err)
				}
			} else if terminal == `{"type":"object"}` {
				// Keep the original input as an explicit actual raw Model positive.
				if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlinx.serialization.json.JsonElement\n") {
					t.Fatal("original raw input carrier lost", err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
				t.Fatal("bigint spelling widened source role", terminal, err)
			}
		}
		r = request(schemas(`{"Source":{"properties":{"leaf":{"x-native-type":"bigint"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"}}`), "Dto")
		r.Options.Validation = checked
		got, err = core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed special leaf added", err)
		}
	}
	input := schemas(`{"Value":{"x-native-type":"bigint","nullable":true},"Alias":{"$ref":"#/components/schemas/Value","nullable":false}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got.Artifacts.Files {
		if strings.Contains(f.Path, "LongAsStringSerializer") {
			t.Fatal("alias-only source owns field helper")
		}
	}
	// Caller Long already has direct field support but must keep numeric wire,
	// even alongside actual new bigint chain consumers.
	input = schemas(`{"Value":{"x-native-type":"bigint"},"Caller":{"x-native-type":"kotlin.Long"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"big":{"$ref":"#/components/schemas/Hops/properties/value"},"caller":{"$ref":"#/components/schemas/Caller"}},"required":["big","caller"]}}`)
	got, err = core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || strings.Count(source(t, got, "HolderDto"), "LongAsStringSerializer::class") != 1 {
		t.Fatal("caller Long acquired magic codec", err)
	}
}

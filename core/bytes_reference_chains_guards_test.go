package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestBytesReferenceChainsSourceAndSlotBudgets(t *testing.T) {
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
			body := `"Value":{"type":"string","format":"byte","nullable":true},"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
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
					if err != nil || !strings.Contains(source(t, got, "HolderDto"), "kotlin.ByteArray,") {
						t.Fatal(cached, count, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("bytes cached false source reset", cached, count, err)
				}
			}
		}
	}
	for _, nullable := range []bool{false, true} {
		for _, stream := range []bool{false, true} {
			limit := 240
			if stream {
				limit--
			}
			for _, n := range []int{limit, limit + 1} {
				params := []string{}
				for i := 0; i < n; i++ {
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, nullable))
				}
				catalog := `{"Value":{"type":"string","format":"binary"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`
				response := `{"204":{}}`
				if stream {
					response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
				}
				input := []byte(`{"openapi":"3.1.0","components":{"schemas":` + catalog + `},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
				got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
				if n == limit {
					if err != nil {
						t.Fatal(nullable, stream, n, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(got.Artifacts.Files) != 0 {
					t.Fatal("bytes method layout changed", nullable, stream, n, err)
				}
			}
		}
		for _, n := range []int{240, 241} {
			props, req := []string{}, []string{}
			for i := 0; i < n; i++ {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, nullable))
				req = append(req, fmt.Sprintf(`"p%d"`, i))
			}
			input := schemas(`{"Value":{"type":"string","format":"byte"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(req, ",") + `]}}`)
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if n == 240 {
				if err != nil {
					t.Fatal(nullable, n, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
				t.Fatal("bytes constructor slot cap changed", nullable, n, err)
			}
		}
	}
}

func TestBytesReferenceChainsOriginalGrammarAndSeparateSourceRoles(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ terminal, pointer string }{
			{`{"type":"string","format":"byte","default":"x"}`, "/components/schemas/Value/default"},
			{`{"type":"string","format":"binary","minLength":2}`, "/components/schemas/Value/minLength"},
		} {
			r := request(schemas(`{"Value":`+tc.terminal+`,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer || len(got.Artifacts.Files) != 0 {
				t.Fatal("whole byte grammar masked", checked, tc, err)
			}
		}
		for _, terminal := range []string{`{"x-native-type":"kotlin.ByteArray"}`, `{"type":"object"}`} {
			r := request(schemas(`{"Value":`+terminal+`,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if terminal == `{"x-native-type":"kotlin.ByteArray"}` {
				// Preserve original native input; independently admitted caller
				// profile keeps provider ByteArray rather than byte-format authority.
				if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlin.ByteArray\n") || strings.Contains(source(t, got, "HopsDto"), "Serializable(with") {
					t.Fatal("caller ByteArray backing lost", err)
				}
			} else if terminal == `{"type":"object"}` {
				// Keep the original input; separate raw Model authority projects JSON.
				if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= kotlinx.serialization.json.JsonElement\n") {
					t.Fatal("raw chain source lost", err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
				t.Fatal("bytes spelling authorized other source", terminal, err)
			}
		}
		r := request(schemas(`{"Source":{"properties":{"leaf":{"type":"string","format":"byte"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed byte special leaf expanded", err)
		}
	}
}

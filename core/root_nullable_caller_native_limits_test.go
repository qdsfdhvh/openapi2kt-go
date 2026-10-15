package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestRootNullableCallerNativeAliasMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, narrow := range []bool{false, true} {
			limit := 240
			if narrow {
				limit = 120
			}
			if stream {
				limit--
			}
			for _, n := range []int{limit, limit + 1} {
				var params []string
				for i := range n {
					extra := ""
					if narrow {
						extra = `,"nullable":false`
					}
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/A"%s}}`, i, extra))
				}
				response := `{"204":{}}`
				if stream {
					response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
				}
				b := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"x-native-type":"kotlin.Long"}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
				r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
				if n == limit {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, narrow, stream, r, e)
				}
			}
		}
	}
}
func TestRootNullableCallerNativeAliasProfiles(t *testing.T) {
	for _, name := range []string{"org.example.Money", "org.example.Amount", "kotlin.Long", "kotlin.ByteArray", "kotlin.time.Instant", "kotlinx.serialization.json.JsonElement", "kotlinx.serialization.json.JsonObject"} {
		for _, kind := range []string{"", "string", "integer", "number", "boolean"} {
			ty := ""
			if kind != "" {
				ty = `"type":"` + kind + `",`
			}
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{`+ty+`"x-native-type":"`+name+`"}}`), "Dto")
			for _, flag := range []bool{false, true} {
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if e != nil {
					t.Fatal(name, kind, e)
				}
				if s := source(t, r, "ADto"); !strings.Contains(s, "= "+name+"?") {
					t.Fatal(s)
				}
			}
		}
	}
	// Intrinsic native roots, native enum/const and schemas requiring a physical
	// shape remain separate contracts; explicit false is not a shape provider.
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"x-native-type":"org.example.Money"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
}

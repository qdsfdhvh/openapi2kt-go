package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestIntrinsicNullableCallerNativeMethodSlots(t *testing.T) {
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
				b := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"x-native-type":"kotlin.Long","nullable":true}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
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
func TestIntrinsicNullableCallerNativeModelSlots(t *testing.T) {
	for _, n := range []int{240, 241} {
		var fields, required []string
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Z"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		b := schemas(`{"Z":{"x-native-type":"org.example.scalarvalues.Money","nullable":true},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestIntrinsicNullableCallerNativeProfiles(t *testing.T) {
	for _, name := range []string{"org.example.Money", "org.example.Amount", "kotlin.Long", "kotlin.ByteArray", "kotlin.time.Instant", "kotlinx.serialization.json.JsonElement", "kotlinx.serialization.json.JsonObject"} {
		for _, kind := range []string{"", "string", "integer", "number", "boolean"} {
			ty := ""
			if kind != "" {
				ty = `"type":"` + kind + `",`
			}
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{`+ty+`"nullable":true,"x-native-type":"`+name+`"}}`), "Dto")
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
}

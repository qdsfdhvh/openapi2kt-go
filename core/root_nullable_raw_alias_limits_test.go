package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestRootNullableRawAliasSlots(t *testing.T) {
	for _, n := range []int{240, 241} {
		var fields, required []string
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"object"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
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
func TestRootNullableRawAliasCollectionHeight(t *testing.T) {
	for _, n := range []int{44, 45} {
		list := `{"$ref":"#/components/schemas/A"}`
		for range n {
			list = `{"type":"array","items":` + list + `}`
		}
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"object"},"ListRoot":` + list + `,"Holder":{"properties":{"v":{"$ref":"#/components/schemas/ListRoot"}}}}`)
		// Whole-envelope non-null control differs only in alias root intent.
		control := []byte(strings.Replace(string(b), `"nullable":true`, `"nullable":false`, 1))
		controlResult, controlError := core.GenerateWithDiagnostics(request(control, "Dto"))
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if (e == nil) != (controlError == nil) {
			t.Fatal("nullability must not add a source hop", n, controlResult, controlError, r, e)
		}
		if n == 44 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}
func TestRootNullableRawAliasMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		limit := 240
		if stream {
			limit--
		}
		for _, n := range []int{limit, limit + 1} {
			var params []string
			for i := range n {
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/A"}}`, i))
			}
			responses := `{"204":{}}`
			if stream {
				responses = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
			}
			b := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"object"}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + responses + `}}}}`)
			r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
			if n == limit {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
	}
}

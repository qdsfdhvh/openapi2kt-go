package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestExplicitNullLiteralsKeepSourceAndSerializedOperationBudgets(t *testing.T) {
	for _, literal := range []string{`{"type":"null","const":null}`, `{"type":"null","enum":[null]}`, `{"const":null}`, `{"enum":[null]}`} {
		for _, checked := range []bool{false, true} {
			for _, hops := range []int{46, 47} {
				r := request(rootSpecialChain(hops, literal, fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d"}}}`, hops-1)), "Dto")
				r.Options.Validation = checked
				got, e := core.GenerateWithDiagnostics(r)
				if (e == nil) != (hops == 46) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(literal, checked, hops, e)
				}
			}
			for _, n := range []int{240, 241} {
				fields, required, params := []string{}, []string{}, []string{}
				for i := range n {
					fields = append(fields, fmt.Sprintf(`"p%d":%s`, i, literal))
					required = append(required, fmt.Sprintf(`"p%d"`, i))
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":%s}`, i, literal))
				}
				for _, operation := range []bool{false, true} {
					input := schemas(`{"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
					code := "E_MODEL_LIMIT"
					if operation {
						input = []byte(`{"openapi":"3.1.0","paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`)
						code = "E_OPERATION_LIMIT"
					}
					r := request(input, "Dto")
					r.Options.Validation = checked
					got, e := core.GenerateWithDiagnostics(r)
					if (e == nil) != (n == 240) || e != nil && (e.(core.Diagnostics)[0].Code != code || !reflect.DeepEqual(got, core.GenerationReport{})) {
						t.Fatal(literal, checked, n, operation, e)
					}
				}
			}
		}
	}
	for _, n := range []int{240, 241} {
		values := strings.TrimSuffix(strings.Repeat("null,", n), ",")
		_, e := core.GenerateWithDiagnostics(request(fieldDoc(`{"type":"null","enum":[`+values+`]}`), "Dto"))
		code, tail := "E_SCHEMA_INVALID", "enum/1"
		if n == 241 {
			code, tail = "E_ENUM_LIMIT", "enum"
		}
		if e == nil || e.(core.Diagnostics)[0].Code != code || !strings.HasSuffix(*e.(core.Diagnostics)[0].Pointer, "/"+tail) {
			t.Fatal("Raw original vocabulary limit before uniqueness", n, e)
		}
	}
	for _, n := range []int{16, 17} {
		value := `{"type":"null","enum":[null]}`
		for range n {
			value = `{"type":"array","minItems":1,"items":` + value + `}`
		}
		r := request(fieldDoc(value), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 16) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Actual checked collection budget", n, e)
		}
	}
}

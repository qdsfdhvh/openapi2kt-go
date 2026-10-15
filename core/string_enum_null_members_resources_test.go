package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumNullMembersRetainSourceAndSlotBudgets(t *testing.T) {
	for _, n := range []int{46, 47} {
		r := request(rootSpecialChain(n, `{"type":["string","null"],"enum":[null,"ready","done"]}`, fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)), "Dto")
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 46) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, e)
		}
	}
	for _, n := range []int{240, 241} {
		fields, required, params := []string{}, []string{}, []string{}
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/State"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
			params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/State"}}`, i))
		}
		for _, operation := range []bool{false, true} {
			catalog := `"State":{"type":["string","null"],"enum":["ready",null,"done"]}`
			paths := ""
			if operation {
				paths = `,"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}`
			} else {
				catalog += `,"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
			}
			r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{`+catalog+`}}`+paths+`}`), "Dto")
			got, e := core.GenerateWithDiagnostics(r)
			code := "E_MODEL_LIMIT"
			if operation {
				code = "E_OPERATION_LIMIT"
			}
			if (e == nil) != (n == 240) || e != nil && (e.(core.Diagnostics)[0].Code != code || len(got.Artifacts.Files) != 0) {
				t.Fatal(n, operation, e)
			}
		}
	}
}

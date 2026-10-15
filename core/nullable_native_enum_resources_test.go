package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableNativeEnumsRetainBoxedAndNarrowedSlotLimits(t *testing.T) {
	for _, narrow := range []bool{false, true} {
		for _, operation := range []bool{false, true} {
			limit := 240
			if narrow {
				limit = 120
			}
			for _, n := range []int{limit, limit + 1} {
				props, required, params := []string{}, []string{}, []string{}
				for i := range n {
					ref := `{"$ref":"#/components/schemas/Code"}`
					if narrow {
						ref = `{"$ref":"#/components/schemas/Code","nullable":false}`
					}
					props = append(props, fmt.Sprintf(`"p%d":%s`, i, ref))
					required = append(required, fmt.Sprintf(`"p%d"`, i))
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":%s}`, i, ref))
				}
				catalog := `"Code":{"type":["integer","null"],"format":"int64","enum":[1,2],"minimum":2}`
				paths := ""
				if operation {
					paths = `,"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}`
				} else {
					catalog += `,"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
				}
				r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{`+catalog+`}}`+paths+`}`), "Dto")
				r.Options.Validation = true
				got, e := core.GenerateWithDiagnostics(r)
				if (e == nil) != (n == limit) {
					t.Fatal(narrow, operation, n, e)
				}
				if e != nil {
					code := "E_MODEL_LIMIT"
					if operation {
						code = "E_OPERATION_LIMIT"
					}
					if e.(core.Diagnostics)[0].Code != code || len(got.Artifacts.Files) != 0 {
						t.Fatal(e)
					}
				}
			}
		}
	}
}

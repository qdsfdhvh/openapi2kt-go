package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestValueClassInternalChainsHTTPLayoutAndRawValue(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		limit := 120
		if nullable {
			limit = 240
		}
		for _, n := range []int{limit, limit + 1} {
			params := []string{}
			for i := range n {
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%v}}`, i, nullable))
			}
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"integer","format":"int64","x-kotlin-value-class":true},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"}}},"paths":{"/v":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`)
			req := request(input, "Dto")
			r, e := core.GenerateWithDiagnostics(req)
			if n == limit {
				if e != nil {
					t.Fatal(nullable, n, e)
				}
				if !strings.Contains(source(t, r, "DefaultApi"), "rawValue") {
					t.Fatal("canonical value-class query did not use rawValue")
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal("HTTP parameter layout drift", nullable, n, e)
			}
		}
	}
}

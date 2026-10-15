package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestValueClassInternalChainSlotLayout(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		limit := 120
		if nullable {
			limit = 240
		}
		for _, n := range []int{limit, limit + 1} {
			props := []string{}
			required := []string{}
			for i := range n {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%v}`, i, nullable))
				required = append(required, fmt.Sprintf(`"p%d"`, i))
			}
			input := schemas(`{"Value":{"type":"integer","format":"int64","x-kotlin-value-class":true,"nullable":true},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"},"Wide":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
			req := request(input, "Dto")
			r, e := core.GenerateWithDiagnostics(req)
			if n == limit {
				if e != nil {
					t.Fatal(nullable, n, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal("canonical wrapper JVM slot layout lost", nullable, n, e)
			}
		}
	}
}

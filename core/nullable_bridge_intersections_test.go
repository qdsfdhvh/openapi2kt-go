package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestNullableNamedBridgeIntersectionNearestIntent(t *testing.T) {
	for _, target := range []bool{false, true} {
		for _, alias := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			for _, member := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				input := schemas(fmt.Sprintf(`{"Value":{"nullable":%t,"properties":{"id":{"type":"string"}}},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/hop"%s},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"%s},{"nullable":true,"properties":{"extra":{"type":"boolean"}}}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Consumer"}},"required":["value"]}}`, target, alias, member))
				want := target
				if alias != "" {
					want = strings.Contains(alias, "true")
				}
				if member != "" {
					want = strings.Contains(member, "true")
				}
				rq := request(input, "Dto")
				r, e := core.GenerateWithDiagnostics(rq)
				if e != nil {
					t.Fatal(target, alias, member, e)
				}
				if strings.Contains(source(t, r, "HolderDto"), "ConsumerDto? = null") != want {
					t.Fatal("original bridge null intent lost", target, alias, member, want, source(t, r, "HolderDto"))
				}
			}
		}
	}
}

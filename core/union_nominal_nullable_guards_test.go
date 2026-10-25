package core_test

import (
	"strings"
	"testing"
)

func TestUnionNominalNullableAliasesDoNotAuthorizeShapeBorrowing(t *testing.T) {
	base := `"Choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"id":{"type":"string"}}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]},"Maybe":{"$ref":"#/components/schemas/Choice","nullable":true},"False":{"$ref":"#/components/schemas/Maybe","nullable":false}`
	for _, tc := range []struct{ body, code, pointer string }{
		{`"Consumer":{"oneOf":[{"$ref":"#/components/schemas/Maybe"},{"properties":{"id":{"type":"string"}}}]}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Consumer/oneOf/0"},
		{`"Consumer":{"allOf":[{"$ref":"#/components/schemas/Maybe"},{"properties":{"id":{"type":"string"}}}]}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Consumer/allOf/0"},
		{`"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Choice"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"id":{"type":"string"}}}]}`, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/allOf/0"},
	} {
		t.Run(tc.pointer, func(t *testing.T) {
			if tc.body == `"Consumer":{"allOf":[{"$ref":"#/components/schemas/Maybe"},{"properties":{"id":{"type":"string"}}}]}` {
				accepted := report(t, schemas(`{`+base+`,`+tc.body+`}`), "Dto")
				if !strings.Contains(source(t, accepted, "ConsumerDto"), "public sealed class ConsumerDto") {
					t.Fatal("nullable union operand lost physical variants")
				}
			} else {
				aliasFailure(t, schemas(`{`+base+`,`+tc.body+`}`), tc.code, tc.pointer)
			}
		})
	}
	aliasFailure(t, schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}}}`), "E_REF_CYCLE", "/components/schemas/Alias/$ref")
	invalid := strings.Replace(base, `"id":{"type":"string"}`, `"id":{"type":"string","format":"unsupported"}`, 1)
	aliasFailure(t, schemas(`{`+invalid+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Choice/oneOf/0/properties/id/format")
}

func TestUnionNominalFirstLateUseRetainsLexicalOwner(t *testing.T) {
	base := `"Owner":{"properties":{"choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"id":{"type":"string"}}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]}}},"Borrow":{"allOf":[{"$ref":"#/components/schemas/Owner"},{"properties":{"extra":{"type":"string"}}}]}`
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + base + `}},"paths":{"/union":{"get":{"operationId":"readUnion","responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Owner/properties/choice"}}}}}}}}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "DefaultApi"), "org.example.models.models.OwnerDto.Choice") {
		t.Fatal("late HTTP use guessed borrowed union owner")
	}
}

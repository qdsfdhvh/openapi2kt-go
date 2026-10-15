package core_test

import (
	"testing"

	"openapi-kt/core"
)

func TestCollectionAliasConsumerSourceReferenceHeight(t *testing.T) {
	for _, tc := range []struct {
		name, definitions, leaf string
		boundary                int
	}{
		{"catalog-scalar", `"Leaf":{"type":"string"},"Link":{"$ref":"#/components/schemas/Leaf"},`, `{"$ref":"#/components/schemas/Link"}`, 44},
		{"borrowed-scalar", `"Carrier":{"properties":{"leaf":{"type":"string"},"link":{"$ref":"#/components/schemas/Carrier/properties/leaf"}}},`, `{"$ref":"#/components/schemas/Carrier/properties/link"}`, 44},
		{"borrowed-collection", `"Carrier":{"properties":{"leaf":{"type":"array","items":{"type":"string"}},"link":{"$ref":"#/components/schemas/Carrier/properties/leaf"}}},`, `{"$ref":"#/components/schemas/Carrier/properties/link"}`, 43},
		{"nullable-union", "", `{"anyOf":[{"type":"string"},{"type":"null"}]}`, 45},
		{"primitive-union", "", `{"oneOf":[{"type":"integer"},{"type":"number"}]}`, 45},
		{"nested-nullable-union", "", `{"anyOf":[{"anyOf":[{"type":"string"},{"type":"null"}]},{"type":"null"}]}`, 44},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []int{tc.boundary, tc.boundary + 1} {
				alias := tc.leaf
				for range n {
					alias = `{"type":"array","items":` + alias + `}`
				}
				doc := schemas(`{` + tc.definitions + `"ZAlias":` + alias + `,"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/ZAlias"}}}}`)
				if n == tc.boundary {
					req := request(doc, "Dto")
					req.Options.Validation = true
					if _, err := core.GenerateWithDiagnostics(req); err != nil {
						t.Fatal(err)
					}
				} else {
					aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v")
				}
			}
		})
	}
}

package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestActualNamedHierarchyLeafNestedDTOAllOf(t *testing.T) {
	for _, ref := range []string{"Leaf/properties/payload", "Bridge"} {
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"properties":{"payload":{"properties":{"id":{"type":"string"}},"required":["id"]}}},"Other":{"properties":{"other":{"type":"boolean"}}},"Bridge":{"$ref":"#/components/schemas/Leaf/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
				t.Fatal("actualnamedleaf nestedDTO allOf rejected", ref, e)
			}
			if strings.Contains(source(t, r, "ConsumerDto"), "SerialName(\"kind\")") {
				t.Fatal("ownerhierarchy tag leaked into nestedsource intersection")
			}
		}
	}
}

func TestHierarchyNestedIntersectionKeepsRootCommonInlineAndComposedLeafSeparate(t *testing.T) {
	for _, tc := range []struct{ body, path string }{
		{`"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"properties":{"id":{"type":"string"}}},"Other":{"properties":{"other":{"type":"boolean"}}}`, "Leaf"},
		{`"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"properties":{"payload":{"properties":{"id":{"type":"string"}}}}},{"properties":{"extra":{"type":"string"}}}]},"Other":{"properties":{"other":{"type":"boolean"}}}`, "Leaf/allOf/0/properties/payload"},
		{`"Root":{"oneOf":[{"properties":{"kind":{"const":"a"},"payload":{"properties":{"id":{"type":"string"}}}}},{"properties":{"kind":{"const":"b"},"other":{"type":"boolean"}}}],"discriminator":{"propertyName":"kind"}}`, "Root/oneOf/0/properties/payload"},
		{`"Root":{"properties":{"payload":{"properties":{"id":{"type":"string"}}}},"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"properties":{"id":{"type":"string"}}},"Other":{"properties":{"other":{"type":"boolean"}}}`, "Root/properties/payload"},
	} {
		for _, bridge := range []bool{false, true} {
			body := tc.body
			ref := tc.path
			if bridge {
				body += `,"Bridge":{"$ref":"#/components/schemas/` + ref + `"}`
				ref = "Bridge"
			}
			input := schemas(`{` + body + `,"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				if tc.path == "Leaf" || tc.path == "Leaf/allOf/0/properties/payload" {
					r, e := core.GenerateWithDiagnostics(rq)
					if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") {
						t.Fatal("preexisting namedleaf root-shape behavior changed", bridge, e)
					}
					continue
				}
				if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
					t.Fatal("hierarchyroot/common/inline/composedleaf acquired shape permission", tc.path, bridge, e)
				}
			}
		}
	}
}

func TestHierarchyNestedIntersectionKeepsIndependentConstructorSlots(t *testing.T) {
	for _, width := range []int{239, 240} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Owner"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Owner":{"properties":{"payload":{"properties":{` + strings.Join(fields, ",") + `}}}},"Other":{"properties":{"other":{"type":"boolean"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 239 && e != nil {
			t.Fatal("nested239+own1 inherited a foreign tag slot", e)
		}
		if width == 240 && (e == nil || len(r.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("hierarchy source bypassed combined240", e)
		}
	}
}

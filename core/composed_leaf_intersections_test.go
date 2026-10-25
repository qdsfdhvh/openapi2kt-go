package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestActualNamedComposedHierarchyLeafNestedDTOAllOf(t *testing.T) {
	for _, ref := range []string{"Leaf/allOf/0/properties/payload", "Bridge"} {
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"properties":{"payload":{"properties":{"id":{"type":"string"}},"required":["id"]}}},{"properties":{"own":{"type":"string"}}}]},"Other":{"properties":{"other":{"type":"boolean"}}},"Bridge":{"$ref":"#/components/schemas/Leaf/allOf/0/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
				t.Fatal("actualnamedcomposedleaf nestedDTO allOf rejected", ref, e)
			}
			if strings.Contains(source(t, r, "ConsumerDto"), "SerialName(\"kind\")") {
				t.Fatal("parentkind leaked")
			}
		}
	}
}

func TestComposedLeafIntersectionCollectionOwnedSources(t *testing.T) {
	for _, slot := range []struct{ schema, path string }{
		{`{"type":"array","items":{"properties":{"id":{"type":"string"}},"required":["id"]}}`, "items"},
		{`{"additionalProperties":{"properties":{"id":{"type":"string"}},"required":["id"]}}`, "additionalProperties"},
	} {
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"properties":{"payload":` + slot.schema + `}},{"properties":{"own":{"type":"string"}}}]},"Other":{"properties":{"other":{"type":"boolean"}}},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Leaf/allOf/0/properties/payload/` + slot.path + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
			t.Fatal("composedleaf original collection owner", slot.path, e)
		}
	}
}

func TestComposedLeafIntersectionDiscardedTargetAndRootAnnotations(t *testing.T) {
	for _, target := range []string{
		`{"properties":{"id":{"type":"string"}},"default":{}}`,
		`{"properties":{"id":{"type":"string"}},"minProperties":1}`,
		`{"properties":{"kind":{"const":"a"}}}`,
		`{"allOf":[{"properties":{"id":{"type":"string"}}}]}`,
	} {
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"properties":{"payload":` + target + `}},{"properties":{"own":{"type":"string"}}}]},"Other":{"properties":{"other":{"type":"boolean"}}},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Leaf/allOf/0/properties/payload"},{"properties":{"extra":{"type":"string"}}}]}}`)
		if target == `{"allOf":[{"properties":{"id":{"type":"string"}}}]}` {
			r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") {
				t.Fatal("graduatedoriginal composedleaf target", e)
			}
			continue
		}
		if r, e := core.GenerateWithDiagnostics(request(input, "Dto")); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("composedleaf source acquired unrelated target authority", target, e)
		}
	}
	input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"properties":{"payload":{"properties":{"id":{"type":"string"}}}}},{"properties":{"payload":{"type":"string"}}}]},"Other":{"properties":{"other":{"type":"boolean"}}},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Leaf/allOf/0/properties/payload"},{"properties":{"extra":{"type":"string"}}}]}}`)
	if r, e := core.GenerateWithDiagnostics(request(input, "Dto")); e == nil || len(r.Artifacts.Files) != 0 {
		t.Fatal("discarded original target acquired owner", e)
	}
}

func TestComposedLeafIntersectionKeepsIndependentConstructorSlots(t *testing.T) {
	for _, width := range []int{239, 240} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Owner"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Owner":{"allOf":[{"properties":{"payload":{"properties":{` + strings.Join(fields, ",") + `}}}}]},"Other":{"properties":{"other":{"type":"boolean"}}},"Bridge":{"$ref":"#/components/schemas/Owner/allOf/0/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 239 && e != nil {
			t.Fatal("nested239+own1 inherited a foreign tag slot", e)
		}
		if width == 240 && (e == nil || len(r.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("hierarchy source bypassed combined240", e)
		}
	}
}

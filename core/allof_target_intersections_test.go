package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestOriginalNestedAllOfTargetAsIntersectionOperand(t *testing.T) {
	for _, ref := range []string{"Owner/properties/payload", "Bridge"} {
		input := schemas(`{"Owner":{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"count":{"type":"integer"}}}]}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
				t.Fatal("originalunique nestedallOf operand", ref, e)
			}
		}
	}
}

func TestAllOfTargetIntersectionKeepsCombinedSlotBounds(t *testing.T) {
	for _, width := range []int{239, 240} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		input := schemas(`{"Owner":{"properties":{"payload":{"allOf":[{"properties":{` + strings.Join(fields, ",") + `}}]}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 239 && e != nil {
			t.Fatal("aggregate240 rejected", e)
		}
		if width == 240 && (e == nil || len(r.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("internal borrowedfields bypassed240", e)
		}
	}
}

func TestAllOfTargetIntersectionNamedSourcesAndNearestNullable(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, member := range []bool{false, true} {
			input := schemas(fmt.Sprintf(`{"Base":{"nullable":true,"properties":{"id":{"type":"string","default":"base"}}},"Alias":{"$ref":"#/components/schemas/Base"},"Owner":{"properties":{"payload":{"nullable":%t,"allOf":[{"$ref":"#/components/schemas/Alias"},{"nullable":true,"properties":{"count":{"type":"integer"}}}]}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/payload","nullable":%t},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"nullable":true,"properties":{"extra":{"type":"string"}}}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Consumer"}},"required":["value"]}}`, nullable, member))
			r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
			if e != nil {
				t.Fatal(nullable, member, e)
			}
			want := "public val value: org.example.models.models.ConsumerDto,"
			if member {
				want = "public val value: org.example.models.models.ConsumerDto? = null,"
			}
			if !strings.Contains(source(t, r, "HolderDto"), want) {
				t.Fatal("nearestbridge nullable view", nullable, member, source(t, r, "HolderDto"))
			}
			if !strings.Contains(source(t, r, "ConsumerDto"), `id: kotlin.String? = "base"`) {
				t.Fatal("originalnamedfield default erased", source(t, r, "ConsumerDto"))
			}
		}
	}
}

func TestAllOfTargetIntersectionOwnerAndTargetRolesRemainSeparate(t *testing.T) {
	for _, pair := range []struct{ owner, path string }{
		{`{"allOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}}]}`, "Owner/allOf/0/properties/payload"},
		{`{"oneOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}},{"properties":{"other":{"type":"string"}}}]}`, "Owner/oneOf/0/properties/payload"},
		{`{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}],"default":{}}}}`, "Owner/properties/payload"},
		{`{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}],"minProperties":1}}}`, "Owner/properties/payload"},
		{`{"properties":{"payload":{"allOf":[{"oneOf":[{"properties":{"left":{"type":"string"}}},{"properties":{"right":{"type":"boolean"}}}]}]}}}`, "Owner/properties/payload"},
	} {
		input := schemas(`{"Owner":` + pair.owner + `,"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + pair.path + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
		if strings.Contains(pair.path, "/allOf/") || strings.Contains(pair.path, "/oneOf/") {
			r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") {
				t.Fatal("graduatedoriginalowner", pair.path, e)
			}
			continue
		}
		if r, e := core.GenerateWithDiagnostics(request(input, "Dto")); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("composedtarget acquired unrelated owner/target permission", pair.path, e)
		}
	}
}

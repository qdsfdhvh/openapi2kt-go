package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestInternalOrdinaryDTOAllOfOriginalFields(t *testing.T) {
	for _, ref := range []string{"Owner/properties/node", "Bridge"} {
		input := schemas(`{"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}},"required":["id"]}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/node"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"boolean"}}}]}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil {
				t.Fatal("actual ordinary internal DTO intersection rejected", ref, e)
			}
			if !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
				t.Fatal("originalinternalfields missing")
			}
		}
	}
}

func TestInternalDTOIntersectionRolesAndCombinedWidth(t *testing.T) {
	for _, node := range []string{
		`{}`,
		`{"properties":{"pin":{"const":"fixed"}}}`,
		`{"allOf":[{"properties":{"id":{"type":"string"}}}]}`,
		`{"properties":{"id":{"type":"string"}},"default":{"id":"own"}}`,
		`{"properties":{"id":{"type":"string"}},"minProperties":1}`,
		`{"additionalProperties":{"type":"string"}}`,
	} {
		for _, ref := range []string{"Owner/properties/node", "Bridge"} {
			input := schemas(`{"Owner":{"properties":{"node":` + node + `}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/node"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				if node == `{"allOf":[{"properties":{"id":{"type":"string"}}}]}` {
					r, e := core.GenerateWithDiagnostics(rq)
					if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") {
						t.Fatal("originalnestedallOf operand", node, ref, e)
					}
					continue
				}
				if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
					t.Fatal("internal intersection gained excluded role", node, ref, e)
				}
			}
		}
	}
	for _, width := range []int{239, 240} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		input := schemas(`{"Owner":{"properties":{"payload":{"properties":{` + strings.Join(fields, ",") + `}}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 239 && e != nil {
			t.Fatal("aggregate240 rejected", e)
		}
		if width == 240 && (e == nil || len(r.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("internal borrowedfields bypassed240", e)
		}
	}
}

func TestInternalDTOIntersectionOwnerRoleAndEscapedSource(t *testing.T) {
	for _, owner := range []string{
		`{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}]}`,
		`{"oneOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"other":{"type":"string"}}}]}`,
		`{"properties":{"id":{"type":"string"}},"additionalProperties":{"properties":{"id":{"type":"string"}}}}`,
	} {
		path := "Owner/allOf/0/properties/node"
		if strings.Contains(owner, `"oneOf"`) {
			path = "Owner/oneOf/0/properties/node"
		}
		if strings.Contains(owner, `"additionalProperties"`) {
			path = "Owner/additionalProperties"
		}
		input := schemas(`{"Owner":` + owner + `,"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + path + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
		if !strings.Contains(owner, `"additionalProperties"`) {
			r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") {
				t.Fatal("original composednestedsource lost fields", path, e)
			}
			continue
		}
		if r, e := core.GenerateWithDiagnostics(request(input, "Dto")); e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("scratch/composed source authorized ordinary lexical role", path, e)
		}
	}
	input := schemas(`{"Owner/~":{"properties":{"payload/~":{"properties":{"id":{"type":"string"}},"required":["id"]}}},"Bridge":{"$ref":"#/components/schemas/Owner~1~0/properties/payload~1~0"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}`)
	r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
	if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
		t.Fatal("escaped original lexical source lost", e)
	}
}

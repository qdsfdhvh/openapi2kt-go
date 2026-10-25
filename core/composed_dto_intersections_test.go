package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestActualComposedOwnedOrdinaryDTOAllOf(t *testing.T) {
	for _, tc := range []struct{ owner, path string }{
		{`{"allOf":[{"properties":{"payload":{"properties":{"id":{"type":"string"}},"required":["id"]}}},{"properties":{"outer":{"type":"string"}}}]}`, "Owner/allOf/0/properties/payload"},
		{`{"oneOf":[{"properties":{"kind":{"const":"a"},"payload":{"properties":{"id":{"type":"string"}},"required":["id"]}}},{"properties":{"kind":{"const":"b"},"other":{"type":"string"}}}]}`, "Owner/oneOf/0/properties/payload"},
	} {
		for _, ref := range []string{tc.path, "Bridge"} {
			input := schemas(`{"Owner":` + tc.owner + `,"Bridge":{"$ref":"#/components/schemas/` + tc.path + `"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"boolean"}}}]}}`)
			rq := request(input, "Dto")
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
				t.Fatal("uniqueactual composed-owned ordinaryDTO allOf rejected", tc.path, ref, e)
			}
		}
	}
}

func TestComposedDTOIntersectionNoVariantRootHierarchyOrAmbiguousCopies(t *testing.T) {
	for _, tc := range []struct{ body, path string }{
		{`"Owner":{"oneOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}}},{"properties":{"kind":{"const":"b"},"other":{"type":"boolean"}}}]}`, "Owner/oneOf/0"},
		{`"Owner":{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"},"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"kind":{"const":"b"}}}]},{"oneOf":[{"properties":{"side":{"const":"x"}}},{"properties":{"side":{"const":"y"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/node"},
		{`"Owner":{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"node":{"type":"string"}}}]}`, "Owner/allOf/0/properties/node"},
		{`"Owner":{"allOf":[{"properties":{"id":{"type":"string"}},"additionalProperties":{"properties":{"id":{"type":"string"}}}},{"properties":{"extra":{"type":"string"}}}]}`, "Owner/allOf/0/additionalProperties"},
		{`"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},"Other":{"properties":{"other":{"type":"boolean"}}}`, "Leaf/properties/node"},
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
				if tc.path == "Leaf/properties/node" {
					r, e := core.GenerateWithDiagnostics(rq)
					if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") || strings.Contains(source(t, r, "ConsumerDto"), `SerialName("kind")`) {
						t.Fatal("originalnamedleaf nestedsource lost fields or borrowed parenttag", e)
					}
					continue
				}
				if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
					t.Fatal("excluded/ambiguous source borrowed an arbitrary owner", tc.path, bridge, e)
				}
			}
		}
	}
}

package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestOriginalAllOfTargetAcrossUniqueComposedAndHierarchyOwners(t *testing.T) {
	for _, tc := range []struct{ body, path string }{
		{`"Owner":{"allOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"count":{"type":"integer"}}}]}}}]}`, "Owner/allOf/0/properties/payload"},
		{`"Owner":{"oneOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"count":{"type":"integer"}}}]}}},{"properties":{"other":{"type":"boolean"}}}]}`, "Owner/oneOf/0/properties/payload"},
		{`"Root":{"oneOf":[{"$ref":"#/components/schemas/Owner"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Other":{"properties":{"other":{"type":"boolean"}}},"Owner":{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"count":{"type":"integer"}}}]}}}`, "Owner/properties/payload"},
		{`"Root":{"oneOf":[{"$ref":"#/components/schemas/Owner"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Other":{"properties":{"other":{"type":"boolean"}}},"Owner":{"allOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"count":{"type":"integer"}}}]}}}]}`, "Owner/allOf/0/properties/payload"},
	} {
		for _, ref := range []string{tc.path, "Bridge"} {
			input := schemas(`{` + tc.body + `,"Bridge":{"$ref":"#/components/schemas/` + tc.path + `"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				r, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "val id: kotlin.String,") {
					t.Fatal("actualoriginal composedtarget owner", tc.path, ref, e)
				}
				if strings.Contains(source(t, r, "ConsumerDto"), "SerialName(\"kind\")") {
					t.Fatal("parenttag leaked")
				}
			}
		}
	}
}

func TestComposedTargetOwnerAmbiguousDiscardedAndHierarchyCommonRefusals(t *testing.T) {
	for _, tc := range []struct{ body, path string }{
		{`"Owner":{"allOf":[{"oneOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}},{"properties":{"other":{"type":"string"}}}]},{"oneOf":[{"properties":{"left":{"type":"string"}}},{"properties":{"right":{"type":"string"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/payload"},
		{`"Owner":{"allOf":[{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}},{"properties":{"payload":{"type":"string"}}}]}`, "Owner/allOf/0/properties/payload"},
		{`"Root":{"properties":{"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}]}},"oneOf":[{"$ref":"#/components/schemas/Owner"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Owner":{"properties":{"id":{"type":"string"}}},"Other":{"properties":{"other":{"type":"boolean"}}}`, "Root/properties/payload"},
		{`"Root":{"oneOf":[{"properties":{"kind":{"const":"a"},"payload":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}},{"properties":{"kind":{"const":"b"},"other":{"type":"boolean"}}}],"discriminator":{"propertyName":"kind"}}`, "Root/oneOf/0/properties/payload"},
	} {
		for _, bridge := range []bool{false, true} {
			body, ref := tc.body, tc.path
			if bridge {
				body += `,"Bridge":{"$ref":"#/components/schemas/` + ref + `"}`
				ref = "Bridge"
			}
			input := schemas(`{` + body + `,"Consumer":{"allOf":[{"$ref":"#/components/schemas/` + ref + `"},{"properties":{"extra":{"type":"string"}}}]}}`)
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				if r, e := core.GenerateWithDiagnostics(rq); e == nil || len(r.Artifacts.Files) != 0 {
					t.Fatal("composedtarget accepted ambiguous/discarded/common/inlineowner", tc.path, bridge, e)
				}
			}
		}
	}
}

package core_test

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestPinnedStructuredConstActualUnionAndHierarchyDataBodies(t *testing.T) {
	cases := []struct{ name, body, owner, classifier string }{
		{"union", `{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":{"literal":true}}}},{"properties":{"tag":{"const":"two"},"payload":{"const":{"literal":false}}}}]}}`, "ChoiceDto", "kotlinx.serialization.json.JsonObject"},
		{"hierarchy", `{"Base":{"type":"object","properties":{"tag":{"type":"string"}}},"Alpha":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"tag":{"const":"one"},"payload":{"const":[1,{"literal":true}]}}}]},"Beta":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"tag":{"const":"two"},"payload":{"const":[2]}}}]},"Root":{"oneOf":[{"$ref":"#/components/schemas/Alpha"},{"$ref":"#/components/schemas/Beta"}],"discriminator":{"propertyName":"tag","mapping":{"one":"#/components/schemas/Alpha","two":"#/components/schemas/Beta"}}}}`, "AlphaDto", "kotlinx.serialization.json.JsonArray"},
	}
	for _, tc := range cases {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				r := request(schemas(tc.body), "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				got, e := core.GenerateWithDiagnostics(r)
				if e != nil {
					t.Fatal("pinned non-discriminator structured data body should use existing kind/default projection", tc.name, style, checked, e)
				}
				code := source(t, got, tc.owner)
				if !strings.Contains(code, tc.classifier) || !strings.Contains(code, "payload") {
					t.Fatal("structured body type/default missing", tc.name, code)
				}
				if again, e := core.GenerateWithDiagnostics(r); e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("pinned structured bodies not deterministic", e)
				}
			}
		}
	}
}

// Historical negative inputs remain exact, now with API and dispatch assertions.
func assertPinnedStructuredConstGraduation(t *testing.T, got core.GenerationReport, err error) {
	t.Helper()
	if err != nil {
		t.Fatal("Original pinned data-body input did not graduate", err)
	}
	code := source(t, got, "ChoiceDto")
	if !strings.Contains(code, "public val payload: kotlinx.serialization.json.JsonObject =") || strings.Contains(code, `obj["payload"]?.jsonPrimitive`) || strings.Contains(code, "public data object One") {
		t.Fatal("Original pinned data source lost its stateful JSON body", code)
	}
}

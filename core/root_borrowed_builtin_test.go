package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func rootBorrowedBuiltinInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/root-borrowed-builtin.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRootBorrowedBuiltinCanonicalBacking(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(rootBorrowedBuiltinInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ name, scalar string }{{"Text", "String"}, {"Flag", "Boolean"}, {"Count", "Int"}, {"Wide", "Long"}, {"Ratio", "Float"}, {"Measure", "Double"}} {
				if !strings.Contains(source(t, got, tc.name+"BridgeDto"), "public typealias "+tc.name+"BridgeDto = kotlin."+tc.scalar+"?\n") {
					t.Fatal(tc, source(t, got, tc.name+"BridgeDto"))
				}
			}
			if !strings.Contains(source(t, got, "TextBridgeFalseDto"), "= kotlin.String\n") || !strings.Contains(source(t, got, "AgainBridgeDto"), "= kotlin.String?\n") {
				t.Fatal("borrowed scalar path nullable backing changed")
			}
			if !strings.Contains(source(t, got, "PatchBuiltinInputDto"), "Patch<kotlin.Long>") || !strings.Contains(source(t, got, "DefaultApi"), "wide: kotlin.Long") {
				t.Fatal("borrowed root consumers did not use shared scalar/Patch seam")
			}
		}
	}
}

func TestRootBorrowedBuiltinFiniteSourceRoles(t *testing.T) {
	for _, leaf := range []struct{ source, reference string }{
		{`{"properties":{"leaf":{"type":"string"}}}`, "Source/properties/leaf"},
		{`{"type":"array","items":{"type":"string"}}`, "Source/items"},
		{`{"type":"object","additionalProperties":{"type":"string"}}`, "Source/additionalProperties"},
		{`{"allOf":[{"properties":{"leaf":{"type":"string"}}}]}`, "Source/allOf/0/properties/leaf"},
		{`{"properties":{"escaped/key~":{"type":"string"}}}`, "Source/properties/escaped~1key~0"},
	} {
		body := `{"Source":` + leaf.source + `,"ABridge":{"$ref":"#/components/schemas/` + leaf.reference + `"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/ABridge","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}`
		for _, checked := range []bool{false, true} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "public typealias BridgeDto = kotlin.String\n") {
				t.Fatal(leaf, checked, err)
			}
		}
	}
}

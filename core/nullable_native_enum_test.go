package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableNativeEnumRootsAndAliasesKeepTypedChecksAndNearestIntent(t *testing.T) {
	for _, terminal := range []string{`{"type":["integer","null"],"enum":[1,2],"minimum":2}`, `{"type":"integer","enum":[1,2],"minimum":2}`} {
		own := ""
		if !strings.Contains(terminal, `"null"`) {
			own = `,"nullable":true`
		}
		doc := schemas(`{"Code":` + terminal + `,"ATrue":{"$ref":"#/components/schemas/Code"` + own + `},"BFalse":{"$ref":"#/components/schemas/ATrue","nullable":false},"Holder":{"properties":{"maybe":{"$ref":"#/components/schemas/ATrue"},"value":{"$ref":"#/components/schemas/BFalse"}},"required":["maybe","value"]},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/ATrue"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/ATrue"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Rows":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		r := request(doc, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(e)
		}
		for _, name := range []string{"ATrueDto", "BridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlin.Int?") {
				t.Fatal(name)
			}
		}
		h := source(t, got, "HolderDto")
		if !strings.Contains(h, "public val maybe: kotlin.Int?") || strings.Contains(h, "public val value: kotlin.Int?") || !strings.Contains(h, "value >= 2") {
			t.Fatal(h)
		}
		again, e := core.GenerateWithDiagnostics(r)
		if e != nil || !reflect.DeepEqual(got, again) {
			t.Fatal(e)
		}
	}
}

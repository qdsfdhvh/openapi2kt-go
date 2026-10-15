package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumNullMembersKeepNullableScalarsDefaultsAndOriginalChecks(t *testing.T) {
	for _, vocabulary := range []string{`[null,1,2,3,4]`, `[1,null,2,3,4]`, `[1,2,3,4,null]`} {
		input := schemas(`{"Code":{"type":["integer","null"],"enum":` + vocabulary + `,"minimum":2,"multipleOf":2},"Alias":{"$ref":"#/components/schemas/Code"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Rows":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"maybe":{"$ref":"#/components/schemas/Rows"},"value":{"$ref":"#/components/schemas/False"},"defaulted":{"type":["integer","null"],"enum":` + vocabulary + `,"default":2,"minimum":2},"flag":{"type":"boolean","nullable":true,"enum":[false,null,true],"default":true}}}}`)
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(vocabulary, e)
		}
		for _, name := range []string{"CodeDto", "AliasDto", "BridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlin.Int?") {
				t.Fatal(name)
			}
		}
		h := source(t, got, "HolderDto")
		for _, marker := range []string{"defaulted: kotlin.Int? = 2", "flag: kotlin.Boolean? = true", "value == 1 || value == 2 || value == 3 || value == 4", "value >= 2", "value % 2 == 0"} {
			if !strings.Contains(h, marker) {
				t.Fatal(marker, h)
			}
		}
		if strings.Contains(h, "value == null") {
			t.Fatal("Null became a non-null membership literal", h)
		}
		again, e := core.GenerateWithDiagnostics(r)
		if e != nil || !reflect.DeepEqual(got, again) {
			t.Fatal(e)
		}
		r.Options.Validation = false
		if _, e := core.GenerateWithDiagnostics(r); e == nil {
			t.Fatal("Default-off accepted newly supported null vocabulary")
		}
	}
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Each input must first be accepted by the existing reference planner without
// annotations. Defaults grant no reference or enum vocabulary admission.
func referencedStringEnumCollections(own bool) []byte {
	annotation := func(value string) string {
		if own {
			return `,"default":` + value
		}
		return ""
	}
	return schemas(`{
 "Holder":{"properties":{
 "named":{"$ref":"#/components/schemas/Statuses"` + annotation(`["on","off"]`) + `},
 "namedMap":{"$ref":"#/components/schemas/StatusMaps"` + annotation(`{"a/~":"off","$ref":"on"}`) + `},
 "internal":{"$ref":"#/components/schemas/Hops/properties/narrow"` + annotation(`[["on",null,"off"],null,[]]`) + `},
 "borrowed":{"$ref":"#/components/schemas/Sources/properties/modes"` + annotation(`["raw"]`) + `}
 }},
 "Status":{"type":"string","enum":["on","off"]},
 "Statuses":{"type":"array","items":{"$ref":"#/components/schemas/Status"}},
 "StatusMaps":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Status"}},
 "NullableRows":{"type":["array","null"],"items":{"type":["string","null"],"enum":["on","off"]}},
 "Rows":{"type":"array","items":{"$ref":"#/components/schemas/NullableRows"}},
 "Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Rows","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},
 "Mode":{"enum":["raw","wire"]},
 "Sources":{"properties":{"modes":{"type":"array","items":{"$ref":"#/components/schemas/Mode"}}}}
 }`)
}

func TestReferencedStringEnumCollectionFieldDefaultsBaseline(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedStringEnumCollections(false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("no-default reference control", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{
				"val named: org.example.models.models.StatusesDto? = null",
				"val namedMap: org.example.models.models.StatusMapsDto? = null",
				"val internal: kotlin.collections.List<org.example.models.models.NullableRowsDto?>? = null",
				"val borrowed: kotlin.collections.List<org.example.models.models.ModeDto>? = null",
			} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			if !strings.Contains(source(t, got, "StatusesDto"), "typealias StatusesDto = kotlin.collections.List<org.example.models.models.StatusDto>") {
				t.Fatal("public alias drift")
			}
			t.Logf("accepted actual no-default checked=%v style=%s", checked, style)
		}
	}
}

func TestReferencedStringEnumCollectionFieldDefaultsOwnAnnotation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedStringEnumCollections(true), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("own collection annotation", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{
				"val named: org.example.models.models.StatusesDto? = `openapi-kt default tree",
				`"[\"on\",\"off\"]"`,
				`"{\"\$ref\":\"on\",\"a/~\":\"off\"}"`,
				`"[[\"on\",null,\"off\"],null,[]]"`,
				`"[\"raw\"]"`,
			} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
		}
	}
}

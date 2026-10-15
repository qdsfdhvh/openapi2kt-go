package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Each input must first be accepted by the existing reference planner without
// annotations. Defaults grant no reference or native vocabulary admission.
func referencedNativeEnumCollections(own bool) []byte {
	annotation := func(value string) string {
		if own {
			return `,"default":` + value
		}
		return ""
	}
	return schemas(`{
 "Holder":{"properties":{
 "ints":{"$ref":"#/components/schemas/Ints"` + annotation(`[1,3,2]`) + `},
 "doubleMaps":{"$ref":"#/components/schemas/DoubleMaps"` + annotation(`{"a/~":1.25,"$ref":-0.0}`) + `},
 "internal":{"$ref":"#/components/schemas/Hops/properties/narrow"` + annotation(`[[1],null,[]]`) + `},
 "borrowed":{"$ref":"#/components/schemas/Bridge"` + annotation(`[2]`) + `}
 }},
 "IntCode":{"type":"integer","format":"int32","enum":[1,2,3]},
 "Ints":{"type":"array","items":{"$ref":"#/components/schemas/IntCode"}},
 "DoubleMaps":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/DoubleCode"}},
 "DoubleCode":{"type":"number","format":"double","enum":[1.25,-0.0]},
 "NullableRows":{"type":["array","null"],"items":{"$ref":"#/components/schemas/IntCode"}},
 "Rows":{"type":"array","items":{"$ref":"#/components/schemas/NullableRows"}},
 "Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Rows","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},
 "Bridge":{"$ref":"#/components/schemas/Ints"}
 }`)
}

func TestReferencedNativeEnumCollectionFieldDefaultsBaseline(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedNativeEnumCollections(false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("no-default reference control", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{
				"val ints: org.example.models.models.IntsDto? = null",
				"val doubleMaps: org.example.models.models.DoubleMapsDto? = null",
				"val internal: kotlin.collections.List<org.example.models.models.NullableRowsDto?>? = null",
				"val borrowed: org.example.models.models.IntsDto? = null",
			} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			if !strings.Contains(source(t, got, "IntsDto"), "typealias IntsDto = kotlin.collections.List<kotlin.Int>") {
				t.Fatal("public scalar alias drift")
			}
			t.Logf("accepted actual no-default checked=%v style=%s", checked, style)
		}
	}
}

func TestReferencedNativeEnumCollectionFieldDefaultsOwnAnnotation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedNativeEnumCollections(true), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("own collection annotation", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{
				"val ints: org.example.models.models.IntsDto? = `openapi-kt default tree",
				`"[1,3,2]"`,
				`"{\"\$ref\":-0,\"a/~\":1.25}"`,
				`"[[1],null,[]]"`,
				`"[2]"`,
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

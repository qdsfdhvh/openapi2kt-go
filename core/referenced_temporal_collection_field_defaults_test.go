package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Each input must first be accepted by the existing reference planner without
// annotations. Defaults grant no reference or temporal source admission.
func referencedTemporalCollections(own bool) []byte {
	annotation := func(value string) string {
		if own {
			return `,"default":` + value
		}
		return ""
	}
	return schemas(`{
 "Holder":{"properties":{
 "namedDateList":{"$ref":"#/components/schemas/Days"` + annotation(`["2024-02-29"]`) + `},
 "namedInstantMap":{"$ref":"#/components/schemas/Stamps"` + annotation(`{"a/~":"2024-02-29T12:34:56.123456789+02:30"}`) + `},
 "internal":{"$ref":"#/components/schemas/Hops/properties/narrow"` + annotation(`[["2024-02-29"]]`) + `},
 "borrowed":{"$ref":"#/components/schemas/Sources/properties/maps"` + annotation(`[{"a":"2024-02-29T00:00:00Z"}]`) + `}
 }},
 "Days":{"type":"array","items":{"type":"string","format":"date"}},
 "Stamps":{"additionalProperties":{"type":"string","format":"date-time"}},
 "Nested":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Days"}},
 "Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Nested","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},
 "Stamp":{"type":"string","format":"date-time"},
 "Sources":{"properties":{"maps":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Stamp"}}}}}
 }`)
}

func TestReferencedTemporalCollectionFieldDefaultsBaseline(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedTemporalCollections(false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("no-default reference control", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{"val namedDateList: org.example.models.models.DaysDto? = null", "val namedInstantMap: org.example.models.models.StampsDto? = null", "val internal: kotlin.collections.List<org.example.models.models.DaysDto>? = null", "val borrowed: kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.time.Instant>>? = null"} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			if !strings.Contains(source(t, got, "DaysDto"), "typealias DaysDto = kotlin.collections.List<kotlinx.datetime.LocalDate>") {
				t.Fatal("public alias drift")
			}
			t.Logf("accepted actual no-default checked=%v style=%s", checked, style)
		}
	}
}

func TestReferencedTemporalCollectionFieldDefaultsOwnAnnotation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedTemporalCollections(true), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("own collection annotation", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{"val namedDateList: org.example.models.models.DaysDto? = `openapi-kt default tree", "serializer<org.example.models.models.DaysDto>()", "serializer<org.example.models.models.StampsDto>()", "Scalar Instant", "2024-02-29T12:34:56.123456789+02:30"} {
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

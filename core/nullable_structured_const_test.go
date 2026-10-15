package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableStructuredConstOriginalRootCounterexamplesGraduate(t *testing.T) {
	for _, input := range []string{
		`{"Value":{"const":{},"nullable":true}}`,
		`{"Value":{"type":["array","null"],"const":[]}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "ValueDto"), "?\n") {
			t.Fatal("Original nullable root input lost its classifier/null view", input, err)
		}
	}
}
func TestNullableStructuredConstRootsKeepCanonicalKindsAndNearestFalseViews(t *testing.T) {
	input := schemas(`{"Object":{"type":["object","null"],"const":{"n":1}},"Array":{"type":"array","nullable":true,"const":[null]},"Alias":{"$ref":"#/components/schemas/Object"},"Narrow":{"$ref":"#/components/schemas/Alias","nullable":false},"Reopen":{"$ref":"#/components/schemas/Narrow","nullable":true},"ArrayAlias":{"$ref":"#/components/schemas/Array"},"Holder":{"properties":{"fixed":{"$ref":"#/components/schemas/Alias"},"fixedArray":{"$ref":"#/components/schemas/ArrayAlias"},"nullableItems":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"narrowItems":{"type":"array","items":{"$ref":"#/components/schemas/Alias","nullable":false}},"literalItems":{"additionalProperties":{"const":{},"nullable":true}}}}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		for name, carrier := range map[string]string{"ObjectDto": "JsonObject?", "AliasDto": "JsonObject?", "NarrowDto": "JsonObject", "ReopenDto": "JsonObject?", "ArrayDto": "JsonArray?", "ArrayAliasDto": "JsonArray?"} {
			if !strings.Contains(source(t, got, name), "typealias "+name+" = kotlinx.serialization.json."+carrier+"\n") {
				t.Fatal(name, source(t, got, name))
			}
		}
		holder := source(t, got, "HolderDto")
		for _, marker := range []string{"public val fixed: kotlinx.serialization.json.JsonObject =", "public val fixedArray: kotlinx.serialization.json.JsonArray =", "kotlin.collections.List<kotlinx.serialization.json.JsonObject?>", "kotlin.collections.List<kotlinx.serialization.json.JsonObject>", "kotlin.collections.Map<kotlin.String, kotlinx.serialization.json.JsonObject?>"} {
			if !strings.Contains(holder, marker) {
				t.Fatal("Nullable root changed body defaults or nearest value intent", marker, holder)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Nullable structured sources drifted", err)
		}
	}
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStructuredConstValueRootsAliasesAndTypedCollections(t *testing.T) {
	input := schemas(`{"Object":{"type":"object","const":{"$ref":"data","n":1e2}},"Array":{"type":"array","const":[null,true]},"Inferred":{"const":{"value":1}},"Alias":{"$ref":"#/components/schemas/Object"},"Chain":{"$ref":"#/components/schemas/Alias"},"ArrayAlias":{"$ref":"#/components/schemas/Array"},"Holder":{"properties":{"kotlinx":{"type":"string"},"fixed":{"$ref":"#/components/schemas/Chain"},"fixedArray":{"$ref":"#/components/schemas/ArrayAlias"},"objects":{"type":"array","items":{"$ref":"#/components/schemas/Object"}},"arrays":{"additionalProperties":{"$ref":"#/components/schemas/ArrayAlias"}},"inlineObjects":{"type":"array","items":{"const":{"inline":true}}},"inlineArrays":{"additionalProperties":{"type":"array","const":[1,2]}}}}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		for name, marker := range map[string]string{"ObjectDto": "typealias ObjectDto = kotlinx.serialization.json.JsonObject", "ArrayDto": "typealias ArrayDto = kotlinx.serialization.json.JsonArray", "ChainDto": "typealias ChainDto = kotlinx.serialization.json.JsonObject"} {
			if !strings.Contains(source(t, got, name), marker) {
				t.Fatal(name, marker, source(t, got, name))
			}
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{"public val fixed: kotlinx.serialization.json.JsonObject =", "public val fixedArray: kotlinx.serialization.json.JsonArray =", "kotlin.collections.List<kotlinx.serialization.json.JsonObject>", "kotlin.collections.Map<kotlin.String, kotlinx.serialization.json.JsonArray>"} {
			if !strings.Contains(text, marker) {
				t.Fatal("Structured const lost body/reference/value role", marker, text)
			}
		}
		for _, warning := range got.Warnings {
			if warning.Code != "W_CONST_STRUCTURED_APPROXIMATION" || warning.Pointer == nil || !strings.HasSuffix(*warning.Pointer, "/const") {
				t.Fatal("Structured root/leaf uses scalar/raw/collection warning", warning)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Structured const values rerun drifted", err)
		}
	}
}

func TestStructuredConstOriginalRootCounterexamplesGraduate(t *testing.T) {
	// Original field-slice negative inputs remain verbatim as explicit value
	// graduations; unsupported nullable roots remain separate negative coverage.
	for _, original := range []string{
		`{"Value":{"const":{}}}`,
		`{"Value":{"type":"array","const":[]}}`,
		`{"Holder":{"properties":{"values":{"type":"array","items":{"const":{}}}}}}`,
		`{"Holder":{"properties":{"values":{"additionalProperties":{"const":[]}}}}}`,
	} {
		if _, err := core.Generate(request(schemas(original), "Dto")); err != nil {
			t.Fatal(original, err)
		}
	}
}

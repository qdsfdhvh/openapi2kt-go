package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedDefaultDocument(value string) []byte {
	return schemas(`{"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":` + value + `},"again":{"$ref":"#/components/schemas/Alias","default":` + value + `}}},"Alias":{"$ref":"#/components/schemas/Rows"},"Rows":{"type":"array","items":{"additionalProperties":{"type":"integer","format":"int64","nullable":true}}}}`)
}

func TestReferencedCollectionFieldDefaultsTypedAliases(t *testing.T) {
	for _, checked := range []bool{false, true} {
		r := request(referencedDefaultDocument(`[{"z":null,"a":9223372036854775807}]`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(checked, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{"public val value: org.example.models.models.RowsDto? =", "kotlinx.serialization.serializer<org.example.models.models.RowsDto>()"} {
			if !strings.Contains(text, marker) {
				t.Fatal(marker, text)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("determinism", err)
		}
	}
}

func TestReferencedCollectionFieldDefaultsFixture(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/referenced-collection-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, marker := range []string{"public val rows: kotlin.collections.List<org.example.models.models.NullableLongsDto?>? =", "public val listMaps: org.example.models.models.ListMapsDto? =", "public val mapMaps: org.example.models.models.MapMapsDto? =", "public val required: org.example.models.models.RequiredDto,", "kotlinx.serialization.serializer<org.example.models.models.MapMapsDto>()"} {
				if !strings.Contains(text, marker) {
					t.Fatal(marker, text)
				}
			}
			patch := source(t, got, "PatchInputDto")
			if !strings.Contains(patch, "Patch.Absent") || strings.Contains(patch, "default tree") {
				t.Fatal("Patch initializer", patch)
			}
		}
	}
}

func TestReferencedCollectionFieldDefaultsOriginalPointer(t *testing.T) {
	got, err := core.GenerateWithDiagnostics(request(referencedDefaultDocument(`[{"a/~":9223372036854775808}]`), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/value/default/0/a~1~0" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal(err)
	}
}

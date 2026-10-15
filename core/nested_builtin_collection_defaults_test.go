package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedBuiltinCollectionDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/nested-builtin-collection-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{"data_class", "plain_class"} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, marker := range []string{
				"public val required: kotlin.collections.List<kotlin.collections.List<kotlin.Long>>,",
				"public val rows: kotlin.collections.List<kotlin.collections.List<kotlin.Long?>?>? =",
				"public val listMaps: kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.String?>?>? =",
				"public val mapLists: kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.Boolean?>?>? =",
				"public val mapMaps: kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlin.Float?>>? =",
				"kotlinx.serialization.serializer<kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlin.Float?>>>()",
			} {
				if !strings.Contains(text, marker) {
					t.Fatal("Lost nested typed API", marker, text)
				}
			}
			required := source(t, got, "RequiredNullableDto")
			if !strings.Contains(required, "public val value: kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.Int>>?,") || strings.Contains(required, " = null") {
				t.Fatal("required acquired initializer", required)
			}
			patch := source(t, got, "PatchInputDto")
			if !strings.Contains(patch, "Patch<kotlin.collections.List<kotlin.collections.List<kotlin.Long>>>") || !strings.Contains(patch, "Patch.Absent") || !strings.Contains(patch, "public val required: kotlin.collections.List<kotlin.collections.List<kotlin.Long>>?,") {
				t.Fatal(patch)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic nested defaults", err)
			}
		}
	}
	// This exact formerly rejected input graduates under the separate nested
	// builtin profile; other default roles below keep their original guards.
	input = schemas(`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"type":"string"}},"default":[]}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "HolderDto"), "kotlin.collections.List<kotlin.collections.List<kotlin.String>>? =") {
		t.Fatal("Original nested empty-list admission", err)
	}
}

func TestNestedBuiltinCollectionPureModelAlreadySupported(t *testing.T) {
	for _, field := range []string{
		`{"type":"array","items":{"type":"array","items":{"type":"integer"}}}`,
		`{"type":"array","items":{"additionalProperties":{"type":"string"}}}`,
		`{"additionalProperties":{"type":"array","items":{"type":"boolean"}}}`,
		`{"additionalProperties":{"additionalProperties":{"type":"number","format":"float"}}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("pure tree unsupported", field, err)
			}
			if strings.Contains(source(t, got, "RecordDto"), "JsonElement") {
				t.Fatal("pure tree flattened", field)
			}
		}
	}
}

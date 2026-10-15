package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedStringEnumCollectionDefaultsFixture(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/nested-string-enum-collection-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			text := source(t, got, "HolderDto")
			for _, marker := range []string{
				"public val required: kotlin.collections.List<kotlin.collections.List<RequiredInnerInner>>",
				"kotlin.collections.List<kotlin.collections.List<RowsInnerInner?>?>? =",
				"kotlin.collections.List<kotlin.collections.Map<kotlin.String, ListMapInnerValue?>?>? =",
				"kotlin.collections.Map<kotlin.String, kotlin.collections.List<MapListValueInner?>?>? =",
				"kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, MapMapValueValue?>>? =",
				"org.example.models.models.HolderDto.RowsInnerInner",
				"public sealed class MixedInnerValueInnerValue",
			} {
				if !strings.Contains(text, marker) {
					t.Fatal(marker, text)
				}
			}
			patch := source(t, got, "PatchInputDto")
			if !strings.Contains(patch, "Patch<kotlin.collections.List<kotlin.collections.List<RowsInnerInner>>>") || !strings.Contains(patch, "Patch.Absent") {
				t.Fatal(patch)
			}
		}
	}
}

func TestNestedStringEnumCollectionDefaultsTypedTrees(t *testing.T) {
	// Preserve the complete original negative input, including its required
	// fieldDoc role, as a VERBATIM graduation before testing optional defaults.
	for _, checked := range []bool{false, true} {
		r := request(fieldDoc(`{"type":"array","items":{"type":"array","items":{"type":"string","enum":["one"]}},"default":[]}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "RecordDto"), "`openapi-kt default enum") {
			t.Fatal("original required negative graduation", err)
		}
	}
	for _, tc := range []struct{ field, marker string }{
		{`{"type":"array","items":{"type":"array","items":{"type":"string","enum":["one","two"]}},"default":[["one"]]}`, "List<kotlin.collections.List<ValueInnerInner>>"},
		{`{"type":"array","items":{"additionalProperties":{"enum":["one","two"]}},"default":[{"a":"one"}]}`, "List<kotlin.collections.Map<kotlin.String, ValueInnerValue>>"},
		{`{"additionalProperties":{"type":"array","items":{"enum":["one","two"]}},"default":{"a":["one"]}}`, "Map<kotlin.String, kotlin.collections.List<ValueValueInner>>"},
		{`{"additionalProperties":{"additionalProperties":{"type":"string","enum":["one","two"]}},"default":{"a":{"b":"one"}}}`, "Map<kotlin.String, kotlin.collections.Map<kotlin.String, ValueValueValue>>"},
		// Historical negative preserved VERBATIM; only this String leaf graduates.
		{`{"type":"array","items":{"type":"array","items":{"type":"string","enum":["one"]}},"default":[]}`, "List<kotlin.collections.List<ValueInnerInner>>"},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				r := request(schemas(`{"Record":{"properties":{"value":`+tc.field+`}}}`), "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc.field, style, checked, err)
				}
				text := source(t, got, "RecordDto")
				if !strings.Contains(text, tc.marker) || !strings.Contains(text, "`openapi-kt default enum") || !strings.Contains(text, "kotlinx.serialization.serializer<kotlin.collections.") || !strings.Contains(text, "org.example.models.models.RecordDto.Value") {
					t.Fatal(text)
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", err)
				}
			}
		}
	}
}

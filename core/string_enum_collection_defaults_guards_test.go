package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumCollectionDefaultsFormerGuardsAndOriginalAllOf(t *testing.T) {
	// Preserve the historical rejected schemas verbatim as explicit admissions.
	for _, field := range []string{
		`{"type":"array","items":{"enum":["x"]},"default":[]}`,
		`{"type":"object","additionalProperties":{"enum":["x"]},"default":{}}`,
		`{"type":"array","items":{"type":"string","enum":["one"]},"default":[]}`,
		`{"type":"object","additionalProperties":{"type":"string","enum":["one"]},"default":{}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "RecordDto"), "public sealed class") {
				t.Fatal(field, checked, err)
			}
		}
	}
	field := `{"additionalProperties":{"enum":["one","two"]},"default":{"a":"one"}}`
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"Holder":{"allOf":[{"properties":{"value":`+field+`}}],"properties":{"other":{"type":"string"}}}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "HolderDto"), `org.example.models.models.HolderDto.ValueValue`) {
			t.Fatal("Lost allOf default lexical identity", err)
		}
		// Later overrides still own their declaration. The probe cannot leak peers.
		r.Document = schemas(`{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}`)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "sealed class") {
			t.Fatal("Discarded probe leaked enum", err)
		}
		r.Document = schemas(`{"Holder":{"allOf":[{"properties":{"value":` + strings.Replace(field, `"a":"one"`, `"a":"future"`, 1) + `}},{"properties":{"value":{"type":"string"}}}]}}`)
		got, err = core.GenerateWithDiagnostics(r)
		if err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/allOf/0/properties/value/default/a" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Lost discarded original default", err)
		}
		r.Document = schemas(`{"Holder":{"allOf":[{"properties":{"value":` + strings.Replace(field, `"default":`, `"minProperties":1,"default":`, 1) + `}},{"properties":{"value":{"type":"string"}}}]}}`)
		if _, err = core.GenerateWithDiagnostics(r); err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/allOf/0/properties/value/minProperties" {
			t.Fatal("Discarded count dropped", err)
		}
	}
}

func TestStringEnumCollectionDefaultsRetainSourceAuthority(t *testing.T) {
	// A separate reference to the original collection leaf must bind the actual
	// vocabulary declaration, including its source/null member, not a probe copy.
	input := schemas(`{"Holder":{"properties":{"values":{"type":"array","items":{"enum":[null,"one"],"nullable":true},"default":[null,"one"]}}},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Holder/properties/values/items"}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "UseDto"), "org.example.models.models.HolderDto.ValuesInner?") {
			t.Fatal("Original enum owner lost", err)
		}
	}
}

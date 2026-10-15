package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestEnumDefaultsOriginalValueAndVocabularyGuards(t *testing.T) {
	for _, field := range []string{
		`{"type":"string","enum":["a"],"default":"b"}`,
		`{"type":"string","enum":["a"],"default":true}`,
		`{"type":"integer","enum":[0,1],"default":"0"}`,
		`{"type":"boolean","enum":[false],"default":"false"}`,
		`{"type":"string","enum":["a","a"],"default":"a"}`,
		`{"type":["string","null"],"enum":["a",null],"default":null}`,
		`{"type":"integer","enum":[1,2],"default":3}`,
		`{"type":"string","enum":["a"],"format":"date","default":"a"}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(field), "Dto"))
		if field == `{"type":["string","null"],"enum":["a",null],"default":null}` {
			// Preserve the original explicit nullable input as a graduated case.
			if err != nil || !strings.Contains(source(t, got, "RecordDto"), "public data object A") || !strings.Contains(source(t, got, "RecordDto"), "Unrecognized") {
				t.Fatal("Nullable String null vocabulary/default graduation failed", err)
			}
			continue
		}
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("invalid enum default dropped", field, got, err)
		}
	}
	failure(t, schemas(`{"Record":{"allOf":[{"properties":{"value":{"enum":["a"],"default":"b"}}},{"properties":{"value":{"type":"string"}}}]}}`), "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/0/properties/value/default")
}

func TestEnumDefaultsContextAndVariantOwnedTypes(t *testing.T) {
	input := httpDoc(`{"operationId":"enumDefaults","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"patch":{"type":["string","null"],"enum":["a","b"],"default":"a"},"required":{"enum":["a","b"],"default":"a"}},"required":["required"]}}}},"responses":{"204":{}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "EnumDefaultsInputDto")
	if !strings.Contains(text, "Patch<Patch> = org.example.models.models.Patch.Absent") || !strings.Contains(text, "val required: Required,") {
		t.Fatal("enum default changed Patch/required presence", text)
	}
	choice := schemas(`{"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"state":{"enum":["a","b"],"default":"a"}}},{"properties":{"kind":{"const":"b"},"state":{"enum":["a","b"],"default":"b"}}}]}}`)
	got, err = core.GenerateWithDiagnostics(request(choice, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text = source(t, got, "ChoiceDto")
	if strings.Contains(text, "abstract val state") || strings.Count(text, "public sealed class State") != 2 || !strings.Contains(text, "ChoiceDto A State`(\"a\")") || !strings.Contains(text, "ChoiceDto B State`(\"b\")") {
		t.Fatal("lexical enum default fabricated common type", text)
	}
}

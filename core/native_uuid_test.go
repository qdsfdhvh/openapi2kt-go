package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeUuidMapping(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		r := request(schemas(`{"Id":{"type":"string","format":"uuid"},"Holder":{"properties":{"id":{"type":"string","format":"uuid"},"nullable":{"type":["string","null"],"format":"uuid"},"plain":{"type":"string"},"native":{"type":"string","format":"uuid","x-native-type":"kotlin.String"},"items":{"type":"array","items":{"type":"string","format":"uuid"}}}}}`), "Dto")
		r.Options.ModelStyle = style
		report, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, report, "IdDto"), "= kotlin.uuid.Uuid\n") {
			t.Fatal("UUID must map to Kotlin Uuid")
		}
		text := source(t, report, "HolderDto")
		for _, want := range []string{"val id: kotlin.uuid.Uuid?", "val nullable: kotlin.uuid.Uuid?", "val plain: kotlin.String?", "val native: kotlin.String?", "List<kotlin.uuid.Uuid>"} {
			if !strings.Contains(text, want) {
				t.Fatal(want, text)
			}
		}
	}
}

// Defaults and consts must be decodable by Kotlin's actual Uuid serializer.
// Preserve malformed String controls from the previous unvalidated mapping.
func TestNativeUuidLiteralGrammar(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	for _, text := range []string{id, strings.ToUpper(id), strings.ReplaceAll(id, "-", ""), "00000000-0000-0000-0000-000000000000", "ffffffff-ffff-ffff-ffff-ffffffffffff"} {
		for _, annotation := range []string{"default", "const"} {
			r := request(fieldDoc(`{"type":"string","format":"uuid","`+annotation+`":"`+text+`"}`), "Dto")
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(text, annotation, err)
			}
		}
	}
	for _, text := range []string{"", "not-a-uuid", "quote\"\\\né", "{" + id + "}", "urn:uuid:" + id, " " + id, id + " ", "550e8400e29b-41d4-a716-446655440000", "550e8400-e29b_41d4-a716-446655440000", "550e8400-e29b-41d4-a716-44665544000g"} {
		encoded, _ := json.Marshal(text)
		for _, annotation := range []string{"default", "const"} {
			for _, shape := range []struct{ field, suffix string }{
				{`{"type":"string","format":"uuid","` + annotation + `":` + string(encoded) + `}`, "/" + annotation},
			} {
				failure(t, fieldDoc(shape.field), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value"+shape.suffix)
			}
		}
	}
}

func TestNativeUuidDefaultContainers(t *testing.T) {
	for _, shape := range []struct{ field, suffix string }{
		{`{"type":"array","items":{"type":"string","format":"uuid"},"default":["bad"]}`, "/default/0"},
		{`{"additionalProperties":{"type":"string","format":"uuid"},"default":{"key":"bad"}}`, "/default/key"},
		{`{"type":"array","items":{"type":"array","items":{"type":"string","format":"uuid"}},"default":[["bad"]]}`, "/default/0/0"},
	} {
		failure(t, fieldDoc(shape.field), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value"+shape.suffix)
	}
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveMapDefaultsCheckedValuesAndAllOf(t *testing.T) {
	for _, field := range []string{
		`{"additionalProperties":{"type":"string","minLength":1,"maxLength":2},"minProperties":1,"default":{"😀":"😀"}}`,
		`{"type":"object","nullable":true,"additionalProperties":{"type":"integer","minimum":1},"minProperties":1,"default":null}`,
		`{"type":["object","null"],"properties":{},"required":[],"additionalProperties":{"type":["boolean","null"]},"default":{"x":null,"y":true}}`,
		`{"type":"object","additionalProperties":{"type":"number","format":"float","minimum":1},"default":{"x":1.000000001}}`,
	} {
		r := request(fieldDoc(field), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(field, err)
		}
	}
	input := schemas(`{"Base":{"properties":{"value":{"type":"object","additionalProperties":{"type":"integer"},"default":{"old":1}}}},"Holder":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"value":{"type":"object","additionalProperties":{"type":"integer"},"default":{"new":2}}}}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "HolderDto")
	if !strings.Contains(text, `\"new\":2`) || strings.Contains(text, `\"old\":1`) {
		t.Fatal("allOf default winner changed", text)
	}
}

func TestPrimitiveMapDefaultsTypedPresenceAndIdentity(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"kotlin":{"type":"string"},"required":{"type":"object","additionalProperties":{"type":"integer"},"default":{"count":1}},"optional":{"type":"object","additionalProperties":{"type":"integer","format":"int64"},"default":{"min":-9223372036854775808}},"nullable":{"type":"object","nullable":true,"additionalProperties":{"type":"boolean"},"default":null},"values":{"type":"object","additionalProperties":{"type":"number","format":"float","nullable":true},"default":{"a":1.25,"b":null}},"empty":{"type":"object","additionalProperties":{"type":"string"},"default":{}}},"required":["required"]}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		report, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, report, "HolderDto")
		for _, marker := range []string{"public val required: kotlin.collections.Map<kotlin.String, kotlin.Int>,", "public val optional: kotlin.collections.Map<kotlin.String, kotlin.Long>? =", "public val nullable: kotlin.collections.Map<kotlin.String, kotlin.Boolean>? = null", "kotlin.collections.Map<kotlin.String, kotlin.Float?>? =", "public val empty: kotlin.collections.Map<kotlin.String, kotlin.String>? ="} {
			if !strings.Contains(text, marker) {
				t.Fatal(marker, text)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(report, again) {
			t.Fatal("nondeterministic default", err)
		}
	}
	generate := func(value string) core.GenerationReport {
		t.Helper()
		report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"value":{"type":"object","additionalProperties":{"type":"number","format":"float"},"default":`+value+`}}}}`), "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		return report
	}
	if !reflect.DeepEqual(generate(`{"b":1.250,"a":1.000000001}`), generate(`{"a":1,"b":1.25}`)) {
		t.Fatal("key order/mapped float changed identity")
	}
	if reflect.DeepEqual(generate(`{"a":1}`), generate(`{"a":2}`)) {
		t.Fatal("different default collapsed")
	}
}

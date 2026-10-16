package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const closedUnionDefaultTarget = `{"oneOf":[{"properties":{"value":{"type":"string"},"meta":{"properties":{"flag":{"type":"boolean"}}},"age":{"type":"integer","default":7}},"required":["value"]},{"properties":{"value":{"type":"integer"}},"required":["value"]}]}`

func closedUnionDefaultDocument(target, field string) []byte {
	return schemas(`{"Choice":` + target + `,"Holder":{"properties":{"v":` + field + `}}}`)
}

func TestClosedUnionFieldDefaults(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for _, value := range []string{`{"type":"Variant1","value":"x"}`, `{"value":2,"type":"Variant2"}`} {
				r := request(closedUnionDefaultDocument(closedUnionDefaultTarget, `{"$ref":"#/components/schemas/Choice","default":`+value+`}`), "Dto")
				r.Options.ModelStyle, r.Options.Validation = style, checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(style, checked, value, err)
				}
				if !strings.Contains(source(t, got, "HolderDto"), "public val v: org.example.models.models.ChoiceDto? = `openapi-kt model default org example models models ChoiceDto`(") {
					t.Fatal("default lost actual union type")
				}
				r.Document = closedUnionDefaultDocument(closedUnionDefaultTarget, `{"$ref":"#/components/schemas/Choice"}`)
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || source(t, baseline, "ChoiceDto") != source(t, got, "ChoiceDto") || !reflect.DeepEqual(baseline.Warnings, got.Warnings) {
					t.Fatal("default changed closed union or warnings", err)
				}
			}
		}
	}
}

func TestClosedUnionDefaultActualSerialNames(t *testing.T) {
	// The actual owner's reserved name changes Variant1 to Variant11. A
	// scratch root named Validation would choose a different serial name.
	input := schemas(`{"Variant1":` + closedUnionDefaultTarget + `,"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Variant1","default":{"type":"Variant11","value":"x"}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, ""))
	if err != nil || !strings.Contains(source(t, got, "Variant1"), `@kotlinx.serialization.SerialName("Variant11")`) {
		t.Fatal("actual root collision changed serial name", err)
	}
	// Referenced variants retain the component suffix in their serial name.
	input = schemas(`{"Left":{"properties":{"value":{"type":"string"}},"required":["value"]},"Right":{"properties":{"value":{"type":"integer"}},"required":["value"]},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Left"},{"$ref":"#/components/schemas/Right"}]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Choice","default":{"type":"LeftDto","value":"x"}}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
		t.Fatal("actual referenced subtype suffix", err)
	}
	carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"type":"LeftDto"`, `"type":"Left"`, 1)), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/type")
}

func TestClosedUnionDefaultDiagnostics(t *testing.T) {
	for _, tc := range []struct{ value, tail string }{
		{`{}`, ""},
		{`{"value":"x"}`, ""},
		{`{"type":"future","value":"x"}`, "/type"},
		{`{"type":null,"value":"x"}`, "/type"},
		{`{"type":1,"value":"x"}`, "/type"},
		{`{"type":[],"value":"x"}`, "/type"},
		{`{"type":"Variant1","value":false}`, "/value"},
		{`{"type":"Variant2","value":"x"}`, "/value"},
		{`{"type":"Variant1","value":"x","z/~":false}`, "/z~1~0"},
		{`{"z/~":0,"value":false,"type":"Variant1"}`, "/z~1~0"},
		{`{"value":false,"z/~":0,"type":"Variant1"}`, "/value"},
		{`{"type":"Variant1","value":"x","meta":{"flag":"wrong"}}`, "/meta/flag"},
		{`{"type":"Variant1"}`, ""},
		{`[]`, ""},
		{`null`, ""},
	} {
		for _, checked := range []bool{false, true} {
			input := closedUnionDefaultDocument(closedUnionDefaultTarget, `{"$ref":"#/components/schemas/Choice","default":`+tc.value+`}`)
			carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
	}
	native := `{"oneOf":[{"properties":{"value":{"type":"string"},"code":{"type":"integer","enum":[1,2]}},"required":["value"]},{"properties":{"value":{"type":"integer"},"code":{"type":"integer","enum":[1,2]}},"required":["value"]}]}`
	for _, checked := range []bool{false, true} {
		carrierDefaultFailure(t, closedUnionDefaultDocument(native, `{"$ref":"#/components/schemas/Choice","default":{"type":"V1","value":"x","code":3}}`), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/code")
	}
	oversized := `{"$ref":"#/components/schemas/Choice","default":{"type":"Variant1","value":"` + strings.Repeat("a", 32767) + `"}}`
	carrierDefaultFailure(t, closedUnionDefaultDocument(closedUnionDefaultTarget, oversized), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/value")
}

func TestClosedUnionDefaultNullAndCanonicalBudget(t *testing.T) {
	nullable := strings.TrimSuffix(closedUnionDefaultTarget, "}") + `,"nullable":true}`
	for _, value := range []string{`null`, `{"type":"Variant1","value":"x"}`} {
		if _, err := core.GenerateWithDiagnostics(request(closedUnionDefaultDocument(nullable, `{"$ref":"#/components/schemas/Choice","default":`+value+`}`), "Dto")); err != nil {
			t.Fatal("nullable root closed union", value, err)
		}
	}
	// The existing root-union Null carrier contract remains the authority;
	// consumer optionality/nullable alone does not invent a root Null variant.
	carrierDefaultFailure(t, closedUnionDefaultDocument(closedUnionDefaultTarget, `{"$ref":"#/components/schemas/Choice","nullable":true,"default":null}`), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	floatTarget := `{"oneOf":[{"properties":{"value":{"type":"number"},"pad":{"type":"string"}},"required":["value"]},{"properties":{"value":{"type":"integer"}},"required":["value"]}]}`
	prefix, suffix := `{"type":"Variant1","value":1e10,"pad":"`, `"}`
	value := prefix + strings.Repeat("x", 32768-len(prefix)-len(suffix)) + suffix
	// Canonical Double spelling adds an exponent sign after original preflight.
	carrierDefaultFailure(t, closedUnionDefaultDocument(floatTarget, `{"$ref":"#/components/schemas/Choice","default":`+value+`}`), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default")
}

func TestClosedUnionDefaultPresenceAndExclusions(t *testing.T) {
	for _, checked := range []bool{false, true} {
		field := `{"$ref":"#/components/schemas/Choice","nullable":true,"default":{"type":"Variant1","value":"x"}}`
		input := closedUnionDefaultDocument(closedUnionDefaultTarget, field)
		required := []byte(strings.Replace(string(input), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1))
		r := request(required, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") {
			t.Fatal("required closed union acquired initializer", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(required), `"value":"x"`, `"value":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/value")
		large := `"value":"` + strings.Repeat("x", 32767) + `"`
		carrierDefaultFailure(t, []byte(strings.Replace(string(required), `"value":"x"`, large, 1)), checked, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/value")
		requestDoc := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{"Choice":` + closedUnionDefaultTarget + `}}}`
		r.Document = []byte(requestDoc)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("closed union request Patch", err)
		}
		found := false
		for _, file := range got.Artifacts.Files {
			found = found || strings.Contains(file.Content, ".Patch<org.example.models.models.ChoiceDto>")
		}
		if !found {
			t.Fatal("closed reference lost Patch")
		}
		carrierDefaultFailure(t, []byte(strings.Replace(requestDoc, `"value":"x"`, `"value":false`, 1)), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/value")
		carrierDefaultFailure(t, []byte(strings.Replace(requestDoc, `"value":"x"`, large, 1)), checked, "E_MODEL_LIMIT", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/value")
		discarded := schemas(`{"Choice":` + closedUnionDefaultTarget + `,"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		r.Document = discarded
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("discarded closed annotation", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"value":"x"`, `"value":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/value")
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"value":"x"`, large, 1)), checked, "E_MODEL_LIMIT", "/components/schemas/Holder/allOf/0/properties/v/default/value")
	}
	// The synthetic class discriminator must not overwrite a business property.
	collision := `{"oneOf":[{"properties":{"value":{"type":"string"},"type":{"type":"string"}}},{"properties":{"value":{"type":"integer"}}}]}`
	carrierDefaultFailure(t, closedUnionDefaultDocument(collision, `{"$ref":"#/components/schemas/Choice","default":{"type":"Variant2","value":2}}`), false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/v/default")
	// Root/alias defaults do not gain permission from the consuming field.
	alias := schemas(`{"Choice":` + closedUnionDefaultTarget + `,"Alias":{"$ref":"#/components/schemas/Choice"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":{"type":"Variant1","value":"x"}}}}}`)
	if report, err := core.GenerateWithDiagnostics(request(alias, "Dto")); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("closed alias default role expanded", err)
	}
	owned := schemas(`{"Owner":{"properties":{"node":` + closedUnionDefaultTarget + `}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/node","default":{"type":"Variant1","value":"x"}}}}}`)
	baseline := []byte(strings.Replace(string(owned), `,"default":{"type":"Variant1","value":"x"}`, "", 1))
	if _, err := core.GenerateWithDiagnostics(request(baseline, "Dto")); err != nil {
		t.Fatal("internal-owned no-default type must remain accepted", err)
	}
	carrierDefaultFailure(t, owned, false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/v/default")
	rootDefault := strings.TrimSuffix(closedUnionDefaultTarget, "}") + `,"default":{"type":"Variant1","value":"x"}}`
	carrierDefaultFailure(t, closedUnionDefaultDocument(rootDefault, `{"$ref":"#/components/schemas/Choice","default":{"type":"Variant1","value":"x"}}`), false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Choice/default")
}

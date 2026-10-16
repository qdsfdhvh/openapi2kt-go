package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const ordinaryAllOfDefault = `{"allOf":[{"properties":{"label":{"type":"string"},"state":{"enum":["on","off"]}},"required":["label"]},{"properties":{"count":{"type":"integer","format":"int64"},"meta":{"properties":{"flag":{"type":"boolean"}}}},"required":["count"]}],"properties":{"rows":{"type":"array","items":{"type":"integer"}}},"default":{"rows":[1,2],"meta":{"flag":true},"state":"on","count":7,"label":"x"}}`

func TestAllOfObjectFieldDefaults(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(carrierDefaultDocument(ordinaryAllOfDefault, false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("ordinary allOf default", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			if !strings.Contains(text, "public val v: V? = `openapi-kt model default org example models models HolderDto V`(") || !strings.Contains(text, `\"count\":7,\"label\":\"x\",\"meta\":{\"flag\":true},\"rows\":[1,2],\"state\":\"on\"`) {
				t.Fatal("actual type/canonical helper", text)
			}
			without := strings.Replace(ordinaryAllOfDefault, `,"default":{"rows":[1,2],"meta":{"flag":true},"state":"on","count":7,"label":"x"}`, "", 1)
			r.Document = carrierDefaultDocument(without, false)
			baseline, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
				t.Fatal("default changed shared source warnings", err, got.Warnings, baseline.Warnings)
			}
			r.Document = carrierDefaultDocument(ordinaryAllOfDefault, true)
			required, err := core.GenerateWithDiagnostics(r)
			if err != nil || strings.Contains(source(t, required, "HolderDto"), " = `openapi-kt model default") {
				t.Fatal("required constructor default", err)
			}
		}
	}
}

func TestAllOfObjectDefaultMemberOwnDefaults(t *testing.T) {
	field := `{"allOf":[{"properties":{"child":{"properties":{"flag":{"type":"boolean"}},"default":{"flag":true}}}},{"properties":{"n":{"type":"integer"}}}],"default":{"n":1}}`
	for _, checked := range []bool{false, true} {
		r := request(carrierDefaultDocument(field, false), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("omitted object member own-default", err)
		}
		bad := strings.Replace(field, `"default":{"flag":true}`, `"default":{"flag":"wrong"}`, 1)
		carrierDefaultFailure(t, carrierDefaultDocument(bad, false), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/allOf/0/properties/child/default/flag")
	}
}

func TestAllOfObjectDefaultOriginalMemberDiagnostics(t *testing.T) {
	for _, tc := range []struct{ old, replacement, tail string }{
		{`"label":"x"`, `"label":false`, "/label"},
		{`"count":7`, `"count":null`, "/count"},
		{`"state":"on"`, `"state":"unknown"`, "/state"},
		{`"meta":{"flag":true}`, `"meta":{"flag":"yes"}`, "/meta/flag"},
		{`"rows":[1,2]`, `"rows":[1,"x"]`, "/rows/1"},
		{`"rows":[1,2]`, `"z/~":1`, "/z~1~0"},
		{`,"label":"x"`, "", ""},
	} {
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, carrierDefaultDocument(strings.Replace(ordinaryAllOfDefault, tc.old, tc.replacement, 1), false), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
	}
	// Optional presence alone cannot authorize a null annotation.
	field := `{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"boolean"}}}],"default":null}`
	carrierDefaultFailure(t, carrierDefaultDocument(field, false), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	for _, checked := range []bool{false, true} {
		r := request(carrierDefaultDocument(strings.Replace(field, `"default":null`, `"nullable":true,"default":null`, 1), false), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("nullable allOf object default", err)
		}
	}
}

func TestAllOfObjectDefaultMemberReferencesAndBounds(t *testing.T) {
	input := schemas(`{"Base":{"properties":{"label":{"type":"string"}},"required":["label"]},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"n":{"type":"integer","minimum":1}}}],"default":{"label":"ok","n":2}}}}}`)
	r := request(input, "Dto")
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal("original allOf member reference", err)
	}
	carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"n":2`, `"n":0`, 1)), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/n")
	carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"label":"ok"`, `"label":false`, 1)), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/label")
}

func TestAllOfObjectDefaultBeforePresenceAndOverride(t *testing.T) {
	nullable := strings.Replace(ordinaryAllOfDefault, `"allOf":`, `"nullable":true,"allOf":`, 1)
	for _, checked := range []bool{false, true} {
		r := request(carrierRequestDocument(nullable, false), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("nullable request allOf default", err)
		}
		found := false
		for _, file := range got.Artifacts.Files {
			found = found || strings.Contains(file.Content, "public val v:") && strings.Contains(file.Content, ".Patch<V> =")
		}
		if !found {
			t.Fatal("request field lost Patch presence")
		}
		bad := strings.Replace(nullable, `"label":"x"`, `"label":false`, 1)
		carrierDefaultFailure(t, carrierRequestDocument(bad, false), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/label")
		carrierDefaultFailure(t, carrierDefaultDocument(bad, true), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/label")
		input := schemas(`{"Holder":{"allOf":[{"properties":{"v":` + ordinaryAllOfDefault + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		r.Document = input
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") {
			t.Fatal("discarded allOf object default leaked", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"label":"x"`, `"label":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/label")
	}
}

func TestAllOfObjectDefaultExclusionsAndBudget(t *testing.T) {
	for _, field := range []string{
		`{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}],"default":{"a":"x"}}`,
		`{"allOf":[{"properties":{"a":{"type":"string"}},"additionalProperties":true},{"properties":{"b":{"type":"boolean"}}}],"default":{"a":"x"}}`,
		`{"allOf":[{"properties":{"a":{"const":"x"}}}],"default":{"a":"x"}}`,
		`{"allOf":[{},{}],"default":{}}`,
	} {
		carrierDefaultFailure(t, carrierDefaultDocument(field, false), true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/v/default")
	}
	large := strings.Replace(ordinaryAllOfDefault, `"label":"x"`, `"label":"`+strings.Repeat("a", 32767)+`"`, 1)
	carrierDefaultFailure(t, carrierDefaultDocument(large, false), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/label")
}

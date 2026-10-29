package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRootDictionaryAllOfOriginalDefaultRefusalNowOwnsFactory(t *testing.T) {
	input := schemas(`{"A":{"allOf":[{"$ref":"#/components/schemas/Z"}],"default":{}},"Z":{"type":"object","additionalProperties":{"type":"string"}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatalf("original default case: checked=%t: %v", checked, err)
		}
		if text := source(t, got, "A"); !strings.Contains(text, "fun `A default`()") {
			t.Fatalf("original default case lost factory:\n%s", text)
		}
	}
}

func TestRootDictionaryAllOfOwnDefaults(t *testing.T) {
	input := schemas(`{
		"AOuter":{"allOf":[{"description":"view"},{"$ref":"#/components/schemas/ZMaybe","nullable":false},{}],"default":{"outer":"one"}},
		"Alias":{"$ref":"#/components/schemas/AOuter","default":{"alias":"two"}},
		"Holder":{"properties":{"optional":{"$ref":"#/components/schemas/AOuter"},"required":{"$ref":"#/components/schemas/Alias"}},"required":["required"]},
		"ZMap":{"additionalProperties":{"type":"string"},"default":{"target":"three"}},
		"ZMaybe":{"allOf":[{"$ref":"#/components/schemas/ZMap"}],"nullable":true,"default":null}
	}`)
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(input, "Dto")
			rq.Options.Validation, rq.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatalf("checked=%t style=%s: %v", checked, style, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatalf("generation is not deterministic: %v", err)
			}
			for _, name := range []string{"AOuterDto", "AliasDto", "ZMapDto", "ZMaybeDto"} {
				if text := source(t, got, name); !strings.Contains(text, "fun `"+name+" default`()") {
					t.Fatalf("%s lost its own factory:\n%s", name, text)
				}
			}
			text := source(t, got, "HolderDto")
			optional, required := false, false
			for _, line := range strings.Split(text, "\n") {
				if strings.Contains(line, "optional:") && strings.Contains(line, "= null") {
					optional = true
				}
				if strings.Contains(line, "required:") && !strings.Contains(line, "=") {
					required = true
				}
			}
			if !optional || !required {
				t.Fatalf("wrapper defaults changed field presence:\n%s", text)
			}
		}
	}
}

func TestRootDictionaryAllOfOwnDefaultsRejectInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		body, code, pointer string
	}{
		{`"default":{"bad":false}`, "E_SCHEMA_INVALID", "/components/schemas/A/default/bad"},
		{`"default":null`, "E_SCHEMA_INVALID", "/components/schemas/A/default"},
	} {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"A":{"allOf":[{},{"$ref":"#/components/schemas/ZMap"}],` + tc.body + `},"ZMap":{"additionalProperties":{"type":"string"}}}`)
			carrierDefaultFailure(t, input, checked, tc.code, tc.pointer)
		}
	}
}

func TestRootDictionaryAllOfOwnDefaultSourcesAndBounds(t *testing.T) {
	for _, checked := range []bool{false, true} {
		input := schemas(`{
			"A/~":{"allOf":[{"description":"view"},{"$ref":"#/components/schemas/Z"}],"default":{"a/b":false}},
			"Z":{"additionalProperties":{"type":"string"},"default":{"source":"ok"}}
		}`)
		carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/A~1~0/default/a~1b")

		input = schemas(`{
			"A":{"allOf":[{"$ref":"#/components/schemas/Z"}],"default":{"ok":"yes"}},
			"Z":{"additionalProperties":{"type":"string"},"default":{"bad":false}}
		}`)
		carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Z/default/bad")
	}
	for _, tc := range []struct {
		target, value, pointer string
	}{
		{`"minProperties":1,"additionalProperties":{"type":"string"}`, `{}`, "/components/schemas/A/default"},
		{`"additionalProperties":{"type":"string","minLength":2}`, `{"key":"x"}`, "/components/schemas/A/default/key"},
	} {
		input := schemas(`{"A":{"allOf":[{"$ref":"#/components/schemas/Z"}],"default":` + tc.value + `},"Z":{` + tc.target + `}}`)
		carrierDefaultFailure(t, input, true, "E_SCHEMA_INVALID", tc.pointer)
	}
}

func TestRootDictionaryAllOfRawDefaults(t *testing.T) {
	input := schemas(`{
		"Open":{"additionalProperties":true,"default":{"target":true}},
		"Object":{"allOf":[{"$ref":"#/components/schemas/Open"}],"default":{"wrapper":null}},
		"Nullable":{"allOf":[{"$ref":"#/components/schemas/Open"}],"nullable":true,"default":null}
	}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatalf("raw defaults: checked=%t: %v", checked, err)
		}
		for _, name := range []string{"Open", "Object", "Nullable"} {
			if text := source(t, got, name); !strings.Contains(text, "fun `"+name+" default`()") {
				t.Fatalf("%s lost its own factory:\n%s", name, text)
			}
		}
	}
}

func TestRootDictionaryAllOfDefaultsKeepUnsupportedNullableCounts(t *testing.T) {
	input := schemas(`{
		"A":{"allOf":[{"$ref":"#/components/schemas/Z"}],"default":null},
		"Z":{"nullable":true,"additionalProperties":{"type":"string"},"minProperties":1}
	}`)
	carrierDefaultFailure(t, input, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/A/default")
}

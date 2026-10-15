package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestDictionaryDeclarationsAndReferences(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.Root(), "..", "model-semantics", "dictionaries.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "Dto"} {
		r := report(t, input, suffix)
		checks := map[string][]string{
			"Counts":                     {"public typealias Counts" + suffix + " = kotlin.collections.Map<kotlin.String, kotlin.Long>"},
			"Open":                       {"= kotlinx.serialization.json.JsonObject"},
			"Objects":                    {"kotlin.collections.Map<kotlin.String, Objects" + suffix + "Value>"},
			"Objects" + suffix + "Value": {"public data class Objects" + suffix + "Value", "val id: kotlin.String"},
			"Holder":                     {"kotlin.collections.Map<kotlin.String, kotlin.Int?>", "kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.Boolean>>", "org.example.models.models.Counts" + suffix, "public data class InlineValue", "public sealed class StatusesValue"},
		}
		for name, fragments := range checks {
			target := name + suffix
			if name == "Objects"+suffix+"Value" {
				target = name
			}
			s := source(t, r, target)
			for _, fragment := range fragments {
				if !strings.Contains(s, fragment) {
					t.Fatalf("%s missing %s\n%s", target, fragment, s)
				}
			}
		}
		if !reflect.DeepEqual(r, report(t, input, suffix)) {
			t.Fatal("non-deterministic dictionary report")
		}
	}
}

func TestDictionaryDiagnostics(t *testing.T) {
	for _, tc := range []struct{ schema, code, pointer string }{
		{`{"type":"object","additionalProperties":null}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/additionalProperties"},
		{`{"type":"object","additionalProperties":12}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/additionalProperties"},
		{`{"type":"object","additionalProperties":{"type":"string","format":"unsupported-format"}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/additionalProperties/format"},
		{`{"type":"object","additionalProperties":{"type":"string","description":false}}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/additionalProperties/description"},
		{`{"type":"object","properties":[],"additionalProperties":true}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/properties"},
		{`{"type":"object","required":["missing"],"additionalProperties":true}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/required"},
		{`{"type":"object","properties":{"id":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Missing"}}`, "E_REF_MISSING", "/components/schemas/Record/properties/value/additionalProperties"},
		{`{"type":"object","additionalProperties":{"type":"array"}}`, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/additionalProperties"},
	} {
		failure(t, fieldDoc(tc.schema), tc.code, tc.pointer)
	}
	failure(t, schemas(`{"Bag":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Bag"}}}`), "E_REF_CYCLE", "/components/schemas/Bag")
	failure(t, schemas(`{"A":{"type":"object","additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/B"}}},"B":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/A"}}}`), "E_REF_CYCLE", "/components/schemas/A")
	failure(t, schemas(`{"Bag":{"type":"object","additionalProperties":{"type":"object","properties":{"x":{"type":"string"}}}},"BagValue":{"type":"string"}}`), "E_NAME_COLLISION", "/components/schemas/Bag/additionalProperties")
	failure(t, schemas(`{"Bag":{"allOf":[{"type":"object","additionalProperties":{"type":"string"}},{"type":"object"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Bag/allOf/0/additionalProperties")
	// Preserve the original common-dictionary rejection input as an explicit
	// graduation: compatible JsonObject fields now share the real backing type.
	common := report(t, fieldDoc(`{"oneOf":[{"type":"object","properties":{"shared":{"type":"object","additionalProperties":true},"a":{"type":"string"}}},{"type":"object","properties":{"shared":{"type":"object","additionalProperties":true},"b":{"type":"string"}}}]}`), "")
	if !strings.Contains(source(t, common, "Record"), "public abstract val shared: kotlinx.serialization.json.JsonObject?") {
		t.Fatal("original common dictionary input lost shared backing", common)
	}
}

func TestDictionaryFixedPropertiesDoNotCollectExtras(t *testing.T) {
	for _, ap := range []string{`true`, `false`, `{"type":"integer"}`} {
		r := report(t, schemas(`{"Fixed":{"type":"object","required":["id"],"properties":{"id":{"type":"string"}},"additionalProperties":`+ap+`}}`), "")
		s := source(t, r, "Fixed")
		if strings.Contains(s, "Map<") || !strings.Contains(s, "val id: kotlin.String") {
			t.Fatal(s)
		}
		if len(r.Warnings) != 1 || r.Warnings[0].Code != "W_ADDITIONAL_PROPERTIES_APPROXIMATION" || *r.Warnings[0].Pointer != "/components/schemas/Fixed/additionalProperties" {
			t.Fatal(r.Warnings)
		}
	}
	r := report(t, schemas(`{"Closed":{"type":"object","additionalProperties":false},"Free":{"type":"object"},"Open":{"additionalProperties":{}},"Nullable":{"type":["object","null"],"additionalProperties":{"type":"boolean"}}}`), "")
	for _, name := range []string{"Closed", "Open"} {
		if !strings.Contains(source(t, r, name), "= kotlinx.serialization.json.JsonObject") {
			t.Fatal(name)
		}
	}
	if !strings.Contains(source(t, r, "Free"), "= kotlinx.serialization.json.JsonElement") {
		t.Fatal("changed legacy empty object")
	}
	if !strings.Contains(source(t, r, "Nullable"), "kotlin.Boolean>?") {
		t.Fatal("nullable root lost")
	}
}

func TestDictionaryFixedExtrasCompositionSources(t *testing.T) {
	input := schemas(`{"Base":{"type":"object","properties":{"id":{"type":"string"}},"additionalProperties":{"type":"integer"}},"Record":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"code":{"type":"boolean"}}}]}}`)
	r := report(t, input, "")
	if len(r.Warnings) != 3 || r.Warnings[0].Code != "W_ADDITIONAL_PROPERTIES_APPROXIMATION" || *r.Warnings[1].Pointer != "/components/schemas/Base/additionalProperties" || r.Warnings[2].Code != "W_ALL_OF_APPROXIMATION" {
		t.Fatal(r.Warnings)
	}
	failure(t, schemas(`{"Record":{"allOf":[{"type":"object","properties":{"id":{"type":"string"}},"additionalProperties":{"type":"string","format":"unsupported-format"}},{"properties":{"id":{"type":"integer"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/additionalProperties/format")
	// Exploratory union shape walks must not duplicate extras warnings.
	r = report(t, fieldDoc(`{"oneOf":[{"type":"object","properties":{"tag":{"type":"string","const":"left"},"left":{"type":"string"}},"additionalProperties":false},{"type":"object","properties":{"tag":{"type":"string","const":"right"},"right":{"type":"integer"}},"additionalProperties":false}]}`), "")
	count := 0
	for _, warning := range r.Warnings {
		if warning.Code == "W_ADDITIONAL_PROPERTIES_APPROXIMATION" {
			count++
		}
	}
	if count != 2 {
		t.Fatal("repeated shape discovery duplicated warnings", r.Warnings)
	}
}

func TestDictionaryAliasGraphAndValueNamespaces(t *testing.T) {
	// Sorting visits children first; cached heights must still reject the late ancestor.
	for _, n := range []int{24, 25} {
		components := map[string]any{"A00": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}}
		for i := 1; i < n; i++ {
			components[fmt.Sprintf("A%02d", i)] = map[string]any{"type": "object", "additionalProperties": map[string]any{"$ref": fmt.Sprintf("#/components/schemas/A%02d", i-1)}}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": components}})
		_, err := core.Generate(request(input, ""))
		if (err == nil) != (n == 24) {
			t.Fatal(n, err)
		}
		if err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
	}
	failure(t, schemas(`{"Bag":{"type":"object","additionalProperties":{"oneOf":[{"type":"string"},{"type":"boolean"}]}},"StringBagValue":{"type":"object"}}`), "E_NAME_COLLISION", "/components/schemas/Bag/additionalProperties/oneOf")
	failure(t, fieldDoc(`{"type":"object","properties":{"id":{"type":"string"}},"additionalProperties":{"type":"object","properties":{"bad":{"type":"string","format":"unsupported-format"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/additionalProperties/properties/bad/format")
	// Validation applies to unconsumed components and ignored extras, not example payloads.
	r := report(t, schemas(`{"Bag":{"type":"object","additionalProperties":{"type":"boolean","examples":[{"$ref":"https://example.test/not-a-reference"}]}},"Fixed":{"type":"object","properties":{"id":{"type":"string"}},"additionalProperties":{"type":"integer","enum":[1]}}}`), "")
	if len(r.Warnings) != 2 || r.Warnings[0].Code != "W_ENUM_NATIVE_APPROXIMATION" || r.Warnings[1].Code != "W_ADDITIONAL_PROPERTIES_APPROXIMATION" {
		t.Fatal(r.Warnings)
	}
}

func TestDictionaryDepthSlotsAndNominalRecursion(t *testing.T) {
	for _, n := range []int{47, 48} {
		value := `{"type":"string"}`
		for range n {
			value = `{"type":"object","additionalProperties":` + value + `}`
		}
		_, err := core.Generate(request(fieldDoc(value), ""))
		if (err == nil) != (n == 47) {
			t.Fatal(n, err)
		}
		if err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(err)
		}
	}
	for _, n := range []int{240, 241} {
		fields := map[string]any{}
		required := []string{}
		for i := range n {
			key := fmt.Sprintf("field%d", i)
			fields[key] = map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "integer", "format": "int64"}}
			required = append(required, key)
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": map[string]any{"type": "object", "properties": fields, "required": required}}}})
		_, err := core.Generate(request(input, ""))
		if (err == nil) != (n == 240) {
			t.Fatal(n, err)
		}
	}
	// A class declaration breaks transparent alias recursion and must remain supported.
	report(t, schemas(`{"Tree":{"type":"object","additionalProperties":{"type":"object","properties":{"children":{"$ref":"#/components/schemas/Tree"}}}}}`), "")
	report(t, schemas(`{"Bag":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Node"}},"Node":{"type":"object","properties":{"children":{"$ref":"#/components/schemas/Bag"}}}}`), "")
}

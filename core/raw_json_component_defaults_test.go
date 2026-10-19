package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// A raw JSON carrier component root owns an opt-in factory on the same grammar
// as the field role: an open dictionary yields JsonObject, an
// unconstrained/empty object yields JsonElement.
func TestRawJSONComponentDefaults(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, tc := range []struct{ name, schemas, file, marker string }{
				{"open dict", `{"R":{"type":"object","additionalProperties":true,"default":{"a":1}}}`, "RDto", `("{\"a\":1}")`},
				{"empty object", `{"R":{"type":"object","default":{"a":1}}}`, "RDto", `("{\"a\":1}")`},
				{"array value", `{"R":{"type":"object","default":[1,2]}}`, "RDto", `("[1,2]")`},
				{"scalar value", `{"R":{"type":"object","default":5}}`, "RDto", `("5")`},
				{"string value", `{"R":{"type":"object","default":"s"}}`, "RDto", `("\"s\"")`},
			} {
				t.Run(tc.name, func(t *testing.T) {
					r := request(schemas(tc.schemas), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(checked, style, err)
					}
					text := source(t, got, tc.file)
					if !strings.Contains(text, "public fun `RDto default`()") || !strings.Contains(text, tc.marker) {
						t.Fatal(checked, style, "raw carrier factory lost", tc.marker, text)
					}
				})
			}
		}
	}
}

// A raw carrier component root accepts only annotations its field role accepts,
// and no other role widens.
func TestRawJSONComponentDefaultGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ name, schemas, code, pointer string }{
			{"open dict array", `{"R":{"type":"object","additionalProperties":true,"default":[1]}}`, "E_SCHEMA_INVALID", "/components/schemas/R/default"},
			{"null", `{"R":{"type":"object","additionalProperties":true,"default":null}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/R/default"},
			// Typed dictionaries and closed objects keep their own roles.
			{"closed with extras", `{"R":{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":true,"default":{"a":"x"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/R/default"},
			// A typed dictionary is the collection role, not a raw carrier.
			{"typed dict valid", `{"R":{"type":"object","additionalProperties":{"type":"integer"},"default":{"k":1}}}`, "", ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				r := request(schemas(tc.schemas), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if tc.code == "" {
					if err != nil || len(got.Artifacts.Files) == 0 {
						t.Fatalf("typed dictionary role lost: %v", err)
					}
					return
				}
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatalf("%s: want %s at %s, got %v", tc.name, tc.code, tc.pointer, err)
				}
			})
		}
	}
}

// The raw carrier default is not inherited by a consuming field, and a schema
// without the annotation keeps its exact previous output.
func TestRawJSONComponentDefaultNonInheritance(t *testing.T) {
	const document = `{"R":{"type":"object","additionalProperties":true,"default":{"a":1}},"Holder":{"type":"object","properties":{"v":{"$ref":"#/components/schemas/R"}}}}`
	for _, checked := range []bool{false, true} {
		r := request(schemas(document), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(checked, err)
		}
		holder := source(t, got, "HolderDto")
		if !strings.Contains(holder, "public val v:") || !strings.Contains(holder, "? = null,") || strings.Contains(holder, `"a"`) {
			t.Fatal(checked, "consumer inherited the carrier default", holder)
		}
	}
	plain := report(t, schemas(`{"R":{"type":"object","additionalProperties":true}}`), "Dto")
	if text := source(t, plain, "RDto"); strings.Contains(text, "default`") {
		t.Fatal("annotation-less carrier gained a factory", text)
	}
}

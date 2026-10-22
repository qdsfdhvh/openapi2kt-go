package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// An explicitly nullable component root accepts a **non-null** annotation: the
// declaration keeps its nullable public type and the factory returns an ordinary
// instance of it. Explicit null factories have their own positive controls.
func TestNullableComponentDefaults(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, tc := range []struct{ name, schemas, alias, marker string }{
				{"scalar", `{"N":{"type":["string","null"],"default":"x"}}`, "kotlin.String?", `("\"x\"")`},
				{"temporal", `{"N":{"type":["string","null"],"format":"date","default":"2024-02-29"}}`, "kotlinx.datetime.LocalDate?", `("\"2024-02-29\"")`},
				{"bytes", `{"N":{"type":["string","null"],"format":"byte","default":[1,2]}}`, "kotlin.ByteArray?", `("[1,2]")`},
				{"list", `{"N":{"type":["array","null"],"items":{"type":"string"},"default":["a"]}}`, "kotlin.collections.List<kotlin.String>?", `("[\"a\"]")`},
				{"map", `{"N":{"type":["object","null"],"additionalProperties":{"type":"integer"},"default":{"k":1}}}`, "kotlin.collections.Map<kotlin.String, kotlin.Int>?", `("{\"k\":1}")`},
			} {
				t.Run(tc.name, func(t *testing.T) {
					r := request(schemas(tc.schemas), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(checked, style, err)
					}
					text := source(t, got, "NDto")
					if !strings.Contains(text, "public fun `NDto default`()") || !strings.Contains(text, tc.marker) {
						t.Fatal(checked, style, "nullable factory lost", tc.marker, text)
					}
					// The declaration keeps its nullable public type.
					if !strings.Contains(text, "public typealias NDto = "+tc.alias) {
						t.Fatal(checked, style, "nullable alias changed", text)
					}
				})
			}
			// A nullable ordinary object and a nullable allOf constructor keep
			// their class and return the nullable type.
			for _, tc := range []struct{ name, schemas string }{
				{"object", `{"N":{"type":"object","nullable":true,"required":["z"],"properties":{"z":{"type":"string"}},"default":{"z":"v"}}}`},
				{"allOf", `{"N":{"allOf":[{"properties":{"a":{"type":"string"}},"required":["a"]}],"nullable":true,"default":{"a":"v"}}}`},
			} {
				t.Run("nullable "+tc.name, func(t *testing.T) {
					r := request(schemas(tc.schemas), "Dto")
					r.Options.Validation, r.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(checked, style, err)
					}
					if text := source(t, got, "NDto"); !strings.Contains(text, "public fun default(): NDto? =") {
						t.Fatal(checked, style, "nullable object factory", text)
					}
				})
			}
			// A nullable String enum keeps its sealed class and gets a non-null factory.
			r := request(schemas(`{"N":{"type":["string","null"],"enum":["a","b"],"default":"a"}}`), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			if text := source(t, got, "NDto"); !strings.Contains(text, "public fun `NDto default`(): NDto =") || !strings.Contains(text, `("\"a\"")`) {
				t.Fatal(checked, style, "nullable enum factory", text)
			}
		}
	}
}

// Source roles outside the nullable factory contract keep their rejections.
func TestNullableComponentDefaultGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ name, schemas, code, pointer string }{
			{"wrong kind", `{"N":{"type":["string","null"],"default":5}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default"},
			{"enum non-member", `{"N":{"type":["string","null"],"enum":["a"],"default":"z"}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default"},
			{"list element", `{"N":{"type":["array","null"],"items":{"type":"integer"},"default":["a"]}}`, "E_SCHEMA_INVALID", "/components/schemas/N/default/0"},
			{"counted nullable", `{"N":{"type":"object","nullable":true,"minProperties":0,"properties":{"z":{"type":"string"}},"default":null}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/N/" + map[bool]string{false: "minProperties", true: "default"}[checked]},
		} {
			t.Run(tc.name, func(t *testing.T) {
				r := request(schemas(tc.schemas), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if checked && tc.name == "counted nullable" {
					if err != nil || !strings.Contains(source(t, got, "NDto"), "public fun default(): NDto? = null") {
						t.Fatal("counted nullable null factory lost", err)
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

// Keep the exact original rejected inputs as explicit positive regressions.
func TestNullableComponentNullOriginalRejectionsNowAccepted(t *testing.T) {
	for _, input := range []string{
		`{"N":{"type":["string","null"],"default":null}}`,
		`{"N":{"type":["array","null"],"items":{"type":"string"},"default":null}}`,
		`{"N":{"type":["object","null"],"additionalProperties":{"type":"string"},"default":null}}`,
		`{"N":{"type":["string","null"],"enum":["a"],"default":null}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "NDto"), "public fun `NDto default`()") {
				t.Fatal(input, checked, err)
			}
		}
	}
}

// The nullable default is not inherited by a consuming field, and a component
// without the annotation keeps its exact previous output.
func TestNullableComponentDefaultNonInheritance(t *testing.T) {
	const document = `{"N":{"type":["string","null"],"default":"x"},"Holder":{"type":"object","properties":{"v":{"$ref":"#/components/schemas/N"}}}}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(schemas(document), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			holder := source(t, got, "HolderDto")
			if !strings.Contains(holder, "public val v:") || !strings.Contains(holder, "? = null,") || strings.Contains(holder, `"x"`) {
				t.Fatal(checked, style, "consumer inherited the component default", holder)
			}
		}
	}
	// Unannotated nullable components emit no factory at all.
	plain := report(t, schemas(`{"N":{"type":["string","null"]}}`), "Dto")
	if text := source(t, plain, "NDto"); strings.Contains(text, "default`") {
		t.Fatal("annotation-less nullable component gained a factory", text)
	}
}

package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Paths are written from the outer field to its direct native enum terminal.
// Default data keys deliberately look like schema keywords/JSON Pointers.
func nestedNativeEnumDefaultTree(path, leaf, value, scalar string) (field, typ, pointer string) {
	typ = "kotlin." + scalar
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == 'L' {
			leaf = `{"type":"array","items":` + leaf + `}`
			value = `[` + value + `]`
			typ = "kotlin.collections.List<" + typ + ">"
		} else {
			leaf = `{"additionalProperties":` + leaf + `}`
			value = `{"a/~":` + value + `}`
			typ = "kotlin.collections.Map<kotlin.String, " + typ + ">"
		}
	}
	for _, edge := range path {
		if edge == 'L' {
			pointer += "/items"
		} else {
			pointer += "/additionalProperties"
		}
	}
	return strings.TrimSuffix(leaf, "}") + `,"default":` + value + `}`, typ, pointer
}

func nestedNativeEnumDefaultRequest(field string, checked bool, style core.ModelStyle) core.GenerationRequest {
	r := request(schemas(`{"Holder":{"properties":{"value":`+field+`}}}`), "Dto")
	r.Options.Validation, r.Options.ModelStyle = checked, style
	return r
}

func nestedNativeEnumDefaultFailure(t testing.TB, r core.GenerationRequest, code, pointer string) {
	t.Helper()
	got, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != code || pointer != "" && (ds[0].Pointer == nil || *ds[0].Pointer != pointer) || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatalf("want empty report and %s at %s, got %#v / %v", code, pointer, got, err)
	}
}

func TestNestedNativeEnumCollectionDefaultsFiveNativeTypesAndFinitePaths(t *testing.T) {
	for _, leaf := range []struct{ name, schema, value string }{
		{"Int", `{"type":"integer","format":"int32","enum":[-2147483648,2]}`, `-2147483648`},
		{"Long", `{"type":"integer","format":"int64","enum":[-9223372036854775808,9223372036854775807]}`, `9223372036854775807`},
		{"Float", `{"type":"number","format":"float","enum":[0.5,16777217]}`, `16777217`},
		{"Double", `{"type":"number","format":"double","enum":[0.5,9007199254740993]}`, `9007199254740993`},
		{"Boolean", `{"type":"boolean","enum":[true,false]}`, `true`},
	} {
		for _, path := range []string{"LL", "LM", "ML", "MM", "LMLM"} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					t.Run(fmt.Sprintf("%s/%s/%t/%s", leaf.name, path, checked, style), func(t *testing.T) {
						field, typ, pointer := nestedNativeEnumDefaultTree(path, leaf.schema, leaf.value, leaf.name)
						r := nestedNativeEnumDefaultRequest(field, checked, style)
						got, err := core.GenerateWithDiagnostics(r)
						if err != nil {
							t.Fatal(err)
						}
						text := source(t, got, "HolderDto")
						for _, marker := range []string{"public val value: " + typ + "? =", "serializer<" + typ + ">", "`openapi-kt default tree "} {
							if !strings.Contains(text, marker) {
								t.Fatal("lost native generic/default binding", marker, text)
							}
						}
						if strings.Contains(text, "sealed class") || strings.Contains(text, "Unrecognized") || strings.Contains(text, "LongAsStringSerializer") || strings.Contains(text, "does not belong to enum.") != checked {
							t.Fatal("native scalar/check projection changed", text)
						}
						if checked {
							if len(got.Warnings) != 0 {
								t.Fatal("checked source still approximate", got.Warnings)
							}
						} else if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_ENUM_NATIVE_APPROXIMATION" || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/Holder/properties/value"+pointer+"/enum" {
							t.Fatal("duplicated or relocated original leaf warning", got.Warnings)
						}
						again, err := core.GenerateWithDiagnostics(r)
						if err != nil || !reflect.DeepEqual(got, again) {
							t.Fatal("nondeterministic report", err)
						}
						artifacts, err := core.Generate(r)
						if err != nil || !reflect.DeepEqual(artifacts, got.Artifacts) {
							t.Fatal("Core entry point drift", err)
						}
						pure := field[:strings.LastIndex(field, `,"default":`)] + `}`
						control, err := core.GenerateWithDiagnostics(nestedNativeEnumDefaultRequest(pure, checked, style))
						if err != nil || !reflect.DeepEqual(got.Warnings, control.Warnings) {
							t.Fatal("default changed original pure-schema warning policy", err, got.Warnings, control.Warnings)
						}
					})
				}
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsVerbatimGraduations(t *testing.T) {
	// Exact former fields in the named negative tables. Both tables used
	// request(fieldDoc(field), "Dto"), Validation=false/true, default Data style.
	// Keep the first field's original whitespace; these are not substitutes.
	for _, field := range []string{
		// nested_builtin_collection_defaults_guards_test.go, OriginalKindsNullabilityAndNoDrop.
		`{"type":"array","items":{"type":"array","items":{"type":"integer","enum":[1] }},"default":[]}`,
		// nested_string_enum_collection_defaults_guards_test.go, OriginalGrammarMembershipAndNoDrop.
		`{"type":"array","items":{"type":"array","items":{"type":"integer","enum":[1]}},"default":[]}`,
		`{"type":"array","items":{"type":"array","items":{"type":"boolean","enum":[true]}},"default":[]}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "RecordDto"), "public val value: kotlin.collections.List<kotlin.collections.List<kotlin.") {
				t.Fatal("verbatim former negative did not graduate", field, checked, err)
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsNullableLayersAndEmptyTrees(t *testing.T) {
	for _, field := range []string{
		`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":["integer","null"],"enum":[1,2]}},"default":[null,[null,1],[]]}`,
		`{"type":["object","null"],"additionalProperties":{"type":"object","additionalProperties":{"type":"boolean","nullable":true,"enum":[true]}},"default":null}`,
		`{"additionalProperties":{"type":"array","nullable":true,"items":{"additionalProperties":{"type":"array","items":{"type":"number","format":"double","enum":[1]}}}},"default":{"$ref":[{"type":[1],"/~":[]}],"nil":null,"empty":[]}}`,
		// Original native duplicate grammar does not conflate integer and
		// floating variants; exact default membership accepts either value.
		`{"type":"array","items":{"type":"array","items":{"type":"number","enum":[1,1.0]}},"default":[[1e0]]}`,
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				if _, err := core.GenerateWithDiagnostics(nestedNativeEnumDefaultRequest(field, checked, style)); err != nil {
					t.Fatal(field, checked, style, err)
				}
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsOrdinaryAllOfAndNestedDTOConsumers(t *testing.T) {
	field := nestedDefaultChain(2, `{"type":"integer","enum":[1,2]}`, `[[1]]`)
	for _, checked := range []bool{false, true} {
		for _, input := range []string{
			`{"Holder":{"allOf":[{"properties":{"value":` + field + `}}],"properties":{"label":{"type":"string"}}}}`,
			`{"Holder":{"properties":{"child":{"properties":{"value":` + field + `}},"rows":{"type":"array","items":{"properties":{"value":` + field + `}}},"map":{"additionalProperties":{"properties":{"value":` + field + `}}}}}}`,
		} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt default tree List List Scalar Int`(\"[[1]]\")") {
				t.Fatal("ordinary DTO consumer lost admitted own annotation", checked, err)
			}
		}
	}
}

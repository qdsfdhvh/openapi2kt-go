package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionFieldDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/union-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{"data_class", "plain_class"} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			holder := source(t, got, "HolderDto")
			for _, marker := range []string{
				"public val str: Str? = `openapi-kt model default org example models models HolderDto Str`(\"\\\"x\\\"\")",
				"public val i: I? = `openapi-kt model default org example models models HolderDto I`(\"3\")",
				"public val n: N? = `openapi-kt model default org example models models HolderDto N`(\"1.5\")",
				"public val b: B? = `openapi-kt model default org example models models HolderDto B`(\"true\")",
				"public val maybe: Maybe? = null,",
				"public val requiredUnion: RequiredUnion,",
			} {
				if !strings.Contains(holder, marker) {
					t.Fatal("Lost union default API", marker, holder)
				}
			}
			for _, wrapper := range []string{"sealed class Str", "class StringStr(", "class IntStr(", "class NumberStr(", "class BooleanStr("} {
				if !strings.Contains(holder, wrapper) {
					t.Fatal("Lost primitive union declaration", wrapper)
				}
			}
			if strings.Contains(holder, "requiredUnion: RequiredUnion =") {
				t.Fatal("required union default acquired an initializer", holder)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic union defaults", err, style, checked)
			}
		}
	}
	// The exact formerly rejected inline primitive union default graduates.
	input = schemas(`{"Holder":{"properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}],"default":"x"}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt model default org example models models HolderDto V`(") {
		t.Fatal("Original union default admission", got, err)
	}
}

func TestUnionFieldDefaultsGuards(t *testing.T) {
	base := `{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}],`
	for name, tc := range map[string]struct {
		field string
	}{
		"string format":         {`{"oneOf":[{"type":"string","format":"date"},{"type":"integer"}],"default":"2024-01-02"}`},
		"object branch":         {`{"oneOf":[{"type":"object","properties":{"a":{"type":"string"}}},{"type":"integer"}],"default":{"a":"x"}}`},
		"nullable branch":       {`{"oneOf":[{"type":"string","nullable":true},{"type":"integer"}],"default":"x"}`},
		"wrong kind":            {base + `"default":[]}`},
		"unmatched number kind": {`{"oneOf":[{"type":"string"},{"type":"boolean"}],"default":1}`},
		"null without intent":   {`{"oneOf":[{"type":"string"},{"type":"integer"}],"default":null}`},
	} {
		t.Run(name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				input := schemas(`{"Name":{"type":"string"},"Holder":{"properties":{"v":` + tc.field + `}}}`)
				r := request(input, "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatalf("excluded union default leaked (checked=%v): %v", checked, err)
				}
			}
		})
	}
	// A same-kind fold stays with its own scalar role.
	for _, input := range []string{
		`{"Holder":{"properties":{"v":{"oneOf":[{"type":"string"},{"type":"string"}],"default":"x"}}}}`,
	} {
		r := request(schemas(input), "Dto")
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("adjacent role regressed", input, err)
		}
	}
}

func TestUnionFieldDefaultsReferencedBranches(t *testing.T) {
	for _, tc := range []struct {
		name, schemas string
		ok            bool
	}{
		{"referenced enum member", `{"Status":{"type":"string","enum":["s1","s2"]},"Holder":{"properties":{"v":{"oneOf":[{"$ref":"#/components/schemas/Status"},{"type":"boolean"}],"default":"s1"}}}}`, true},
		{"referenced enum nonmember", `{"Status":{"type":"string","enum":["s1","s2"]},"Holder":{"properties":{"v":{"oneOf":[{"$ref":"#/components/schemas/Status"},{"type":"boolean"}],"default":"zz"}}}}`, false},
		{"referenced native member", `{"Code":{"type":"integer","format":"int32","enum":[1,2]},"Holder":{"properties":{"v":{"oneOf":[{"$ref":"#/components/schemas/Code"},{"type":"string"}],"default":2}}}}`, true},
		{"referenced native nonmember", `{"Code":{"type":"integer","format":"int32","enum":[1,2]},"Holder":{"properties":{"v":{"oneOf":[{"$ref":"#/components/schemas/Code"},{"type":"string"}],"default":9}}}}`, false},
		{"reference chain", `{"Status":{"type":"string","enum":["s1"]},"Alias":{"$ref":"#/components/schemas/Status"},"Holder":{"properties":{"v":{"oneOf":[{"$ref":"#/components/schemas/Alias"},{"type":"boolean"}],"default":"s1"}}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := core.GenerateWithDiagnostics(request(schemas(tc.schemas), "Dto"))
			if tc.ok && err != nil {
				t.Fatal(err)
			}
			if !tc.ok && (err == nil || len(got.Artifacts.Files) != 0) {
				t.Fatal("referenced union branch admitted", err)
			}
		})
	}
}

func TestUnionCollectionElementDefaults(t *testing.T) {
	for name, tc := range map[string]struct{ field string }{
		"list":            {`{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}]},"default":["x",1]}`},
		"map":             {`{"type":"object","additionalProperties":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}]},"default":{"k":"x"}}`},
		"nested property": {`{"type":"object","properties":{"u":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}]}},"default":{"u":"x"}}`},
		"pinned":          {`{"type":"array","items":{"oneOf":[{"type":"string","enum":["a","b"]},{"type":"integer"}]},"default":["a"]}`},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"v":`+tc.field+`}}}`), "Dto"))
			if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt model default") {
				t.Fatal(got, err)
			}
		})
	}
	// A token that matches no element union branch keeps the element Pointer.
	bad := schemas(`{"Holder":{"properties":{"v":{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"boolean"}]},"default":[1]}}}}`)
	r := request(bad, "Dto")
	r.Options.Validation = true
	report, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/v/default/0" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatalf("element union Pointer: %v", err)
	}
}

func TestUnionFieldDefaultsPins(t *testing.T) {
	// Enum and const pins are enforced against the matched branch vocabulary.
	for _, tc := range []struct {
		name, field string
		ok          bool
	}{
		{"string enum member", `{"oneOf":[{"type":"string","enum":["a","b"]},{"type":"integer","enum":[1,2]}],"default":"a"}`, true},
		{"native enum member", `{"oneOf":[{"type":"string"},{"type":"integer","format":"int32","enum":[1,2]}],"default":2}`, true},
		{"string enum nonmember", `{"oneOf":[{"type":"string","enum":["a","b"]},{"type":"integer"}],"default":"z"}`, false},
		{"native enum nonmember", `{"oneOf":[{"type":"string"},{"type":"integer","enum":[1,2]}],"default":9}`, false},
		{"const equal", `{"oneOf":[{"type":"string","const":"a"},{"type":"integer"}],"default":"a"}`, true},
		{"const differs", `{"oneOf":[{"type":"string","const":"a"},{"type":"integer"}],"default":"b"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := schemas(`{"Holder":{"properties":{"v":` + tc.field + `}}}`)
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if tc.ok && err != nil {
				t.Fatal(err)
			}
			if !tc.ok && (err == nil || len(got.Artifacts.Files) != 0) {
				t.Fatal("pinned non-member admitted", err)
			}
		})
	}
}

func TestUnionFieldDefaultsNullBranch(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"null"}],"default":null}}}}`)
	got := report(t, input, "Dto")
	text := source(t, got, "HolderDto")
	if !strings.Contains(text, "public val v: V? = null,") || !strings.Contains(text, "public sealed class V") {
		t.Fatal(text)
	}
}

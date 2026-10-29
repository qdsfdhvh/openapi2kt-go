package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedEnumCollectionIndependentFactories(t *testing.T) {
	for _, leaf := range []struct{ schema, first, second string }{
		{`{"enum":["one","two","x"],"default":"one"}`, `"one"`, `"two"`},
		{`{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"default":"one"}`, `"one"`, `"two"`},
		{`{"type":"integer","enum":[1,2],"default":1}`, `1`, `2`},
		{`{"type":"integer","format":"int64","enum":[1,2],"default":1}`, `1`, `2`},
		{`{"type":"number","format":"float","enum":[1,2],"default":1}`, `1`, `2`},
		{`{"type":"number","format":"double","enum":[1,2],"default":1}`, `1`, `2`},
		{`{"type":"boolean","enum":[true,false],"default":true}`, `true`, `false`},
	} {
		for _, tree := range []struct{ shape, first, second string }{
			{`{"type":"array","items":{"$ref":"#/components/schemas/WordAlias"}}`, `[FIRST]`, `[SECOND]`},
			{`{"additionalProperties":{"$ref":"#/components/schemas/WordAlias"}}`, `{"a":FIRST}`, `{"b":SECOND}`},
			{`{"type":"array","items":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias"}}}}`, `[{"a":[FIRST]}]`, `[{"b":[SECOND]}]`},
		} {
			first := strings.ReplaceAll(tree.first, "FIRST", leaf.first)
			second := strings.ReplaceAll(tree.second, "SECOND", leaf.second)
			shape := strings.TrimSuffix(tree.shape, "}") + `,"default":` + first + `}`
			body := `{"Word":LEAF,"WordAlias":{"$ref":"#/components/schemas/Word","default":SECOND},"Tree":SHAPE,"Alias":{"$ref":"#/components/schemas/Tree","default":OTHER},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":OWN},"unannotated":{"$ref":"#/components/schemas/Tree"}},"default":{"value":OTHER}}}`
			body = strings.NewReplacer("LEAF", leaf.schema, "SECOND", leaf.second, "SHAPE", shape, "OTHER", second, "OWN", first).Replace(body)
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					rq := request(schemas(body), "Dto")
					rq.Options.Validation, rq.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(rq)
					if !checked && strings.Contains(leaf.schema, "minLength") {
						if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
							t.Fatal("checked source bypassed validation opt-in", err)
						}
						continue
					}
					if err != nil {
						t.Fatal("actual enum reference collection factory rejected", leaf.schema, tree.shape, checked, style, err)
					}
					if !strings.Contains(source(t, got, "TreeDto"), "TreeDto default") || !strings.Contains(source(t, got, "AliasDto"), "AliasDto default") {
						t.Fatal("independent root/alias collection factories missing")
					}
					if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("enum reference factories nondeterministic", err)
					}
				}
			}
		}
	}
}

// Keep the exact formerly rejected inputs when independent component-owned
// annotations become valid; consuming annotations still validate separately.
func independentEnumFactoryGraduation(t *testing.T, rq core.GenerationRequest) {
	t.Helper()
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("independent original enum factory source rejected", err)
	}
	if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("independent enum factory source nondeterminism", err)
	}
}

func TestIndependentEnumComponentDefaultFormerRefusals(t *testing.T) {
	for _, body := range []string{
		`{"Code":{"type":"integer","format":"int32","enum":[1],"default":1},"Codes":{"type":"array","items":{"$ref":"#/components/schemas/Code"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[]}}}}`,
		`{"Status":{"type":"string","enum":["on"],"default":"on"},"Statuses":{"type":"array","items":{"$ref":"#/components/schemas/Status"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Statuses","default":[]}}}}`,
		`{"Code":{"type":"integer","enum":[1]},"Alias":{"$ref":"#/components/schemas/Code","default":1},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":1}}}}`,
		`{"Status":{"type":"string","enum":["on"]},"Alias":{"$ref":"#/components/schemas/Status","default":"on"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":"on"}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			independentEnumFactoryGraduation(t, rq)
		}
	}
}

func TestReferencedEnumCollectionFactoryDefaultGuards(t *testing.T) {
	body := `{"Word":{"enum":["one","two","x"],"minLength":2,"default":"one"},"WordAlias":{"$ref":"#/components/schemas/Word","default":"two"},"Tree":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias"}},"default":{"root":["one"]}},"Alias":{"$ref":"#/components/schemas/Tree","default":{"a/~":["two"]}},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":{"field":["one"]}},"unannotated":{"$ref":"#/components/schemas/Tree"}},"required":["value"],"default":{"value":{"object":["two"]}}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "UseDto"), "unannotated: org.example.models.models.TreeDto? = null") {
		t.Fatal("unannotated consumer inherited a collection factory value", source(t, got, "UseDto"))
	}
	for _, tc := range []struct{ from, to, pointer string }{
		{`"default":"one"`, `"default":"future"`, "/components/schemas/Word/default"},
		{`"default":"two"`, `"default":"x"`, "/components/schemas/WordAlias/default"},
		{`"root":["one"]`, `"root":["future"]`, "/components/schemas/Tree/default/root/0"},
		{`"a/~":["two"]`, `"a/~":["x"]`, "/components/schemas/Alias/default/a~1~0/0"},
		{`"field":["one"]`, `"field":["future"]`, "/components/schemas/Use/properties/value/default/field/0"},
		{`"object":["two"]`, `"object":["x"]`, "/components/schemas/Use/default/value/object/0"},
		{`"nullable":true`, `"nullable":false`, "/components/schemas/Maybe/default"},
	} {
		enumBoundsDiagnostic(t, schemas(strings.Replace(body, tc.from, tc.to, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
	for _, value := range []string{`{}`, `null`} {
		bad := strings.Replace(body, `"default":{"root":["one"]}`, `"nullable":true,"default":`+value, 1)
		bad = strings.Replace(bad, `"minLength":2`, `"minLength":-1`, 1)
		enumBoundsDiagnostic(t, schemas(bad), true, "E_SCHEMA_INVALID", "/components/schemas/Word/minLength")
	}
	for _, bad := range []string{
		`{"Word":{"enum":["one"]},"Tree":{"type":"array","items":{"$ref":"#/components/schemas/Word","default":"one"},"default":["one"]}}`,
		`{"Word":{"enum":["one"]},"Tree":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["one"]},"default":[["one"]]}}`,
		`{"Word":{"enum":["one"]},"Tree":{"type":"array","items":{"$ref":"#/components/schemas/Tree"},"default":[]}}`,
	} {
		rq.Document = schemas(bad)
		if report, failure := core.GenerateWithDiagnostics(rq); failure == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("factory admitted child annotation or recursive source", failure)
		}
	}
	for _, size := range []int{32750, 32766} {
		word := strings.Repeat("a", size)
		rq.Document = schemas(`{"Word":{"enum":["` + word + `"]},"Tree":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["` + word + `"]}}`)
		report, failure := core.GenerateWithDiagnostics(rq)
		if size == 32750 && failure != nil {
			t.Fatal("reference collection factory rejected valid default below32KiB", failure)
		}
		if size == 32766 && (failure == nil || failure.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("reference collection factory escaped32KiB budget", failure)
		}
	}
}

func TestReferencedEnumCollectionFactoriesWithoutConsumers(t *testing.T) {
	for _, checked := range []bool{false, true} {
		body := `{"Word":{"enum":["one","two"],"default":"one"},"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["one"]},"Outer":{"additionalProperties":{"$ref":"#/components/schemas/Inner"},"default":{"$ref":["two"],"items":["one"]}},"Alias":{"$ref":"#/components/schemas/Outer","default":{"default":["two"]}},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null}}`
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("actual factories require an unrelated DTO consumer", checked, err)
		}
		for _, name := range []string{"InnerDto", "OuterDto", "AliasDto", "MaybeDto"} {
			if !strings.Contains(source(t, got, name), name+" default") {
				t.Fatal("missing independent factory", name)
			}
		}
		nativeEnumDefaultFailure(t, schemas(strings.Replace(body, `"default":["two"]`, `"default":["future"]`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Alias/default/default/0")
	}
}

func TestUncheckedNativeEnumReferenceRootFactory(t *testing.T) {
	body := `{"Code":{"type":"integer","enum":[1,2]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"default":[1]}}`
	rq := request(schemas(strings.Replace(body, `,"default":[1]`, "", 1)), "Dto")
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("original unchecked native reference collection is not admitted", err)
	}
	rq.Document = schemas(body)
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("independent factory rejected the original admitted native collection", err)
	}
}

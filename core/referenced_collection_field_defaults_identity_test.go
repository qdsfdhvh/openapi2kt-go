package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedDefaultOperations(value string) []byte {
	field := `{"$ref":"#/components/schemas/Rows","default":` + value + `}`
	doc := nestedBuiltinDefaultOperations(field)
	return []byte(strings.TrimSuffix(string(doc), "}") + `,"components":{"schemas":{"Rows":{"additionalProperties":{"additionalProperties":{"type":"number","format":"float","nullable":true}}}}}}`)
}

func TestReferencedCollectionFieldDefaultsCanonicalIdentityAndOptimization(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, mode := range []string{"dedup", "merge", "combined"} {
				r := request(referencedDefaultOperations(`{"b":{"b":null},"a":{"b":1.25,"a":1}}`), "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				r.Options.DeduplicateModels = mode != "merge"
				r.Options.MergeIdenticalModels = mode != "dedup"
				r.Options.HoistSharedShapes = mode == "combined"
				r.Options.LockHoistedNames = mode == "combined"
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(mode, err)
				}
				canonical := "AlphaResponseDto"
				alias := "BetaResponseDto"
				if mode == "combined" {
					canonical = "ResponseDto"
					alias = "AlphaResponseDto"
				}
				if !strings.Contains(source(t, got, alias), "typealias "+alias) || !strings.Contains(source(t, got, canonical), "serializer<org.example.models.models.RowsDto>()") {
					t.Fatal("no actual alias/typed default", mode, source(t, got, canonical))
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				r.Document = referencedDefaultOperations(`{"a":{"a":1.000000001,"b":1.250},"b":{"b":null}}`)
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("key order / mapped float identity", mode, err)
				}
				r.Document = referencedDefaultOperations(`{"a":{"a":2,"b":1.25},"b":{"b":null}}`)
				changed, err := core.GenerateWithDiagnostics(r)
				if mode == "combined" {
					if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
						t.Fatal("changed default reused lock", err)
					}
				} else if err != nil || reflect.DeepEqual(got, changed) {
					t.Fatal("default identity lost", err)
				}
			}
		}
	}
	// Distinct defaults cannot merge, dedup or hoist a pair of actual classes.
	doc := string(referencedDefaultOperations(`{"a":{"b":1}}`))
	at := strings.Index(doc, `"/b":`)
	doc = doc[:at] + strings.Replace(doc[at:], `"default":{"a":{"b":1}}`, `"default":{"a":{"b":2}}`, 1)
	for _, mode := range []string{"dedup", "merge", "combined"} {
		r := request([]byte(doc), "Dto")
		r.Options.DeduplicateModels = mode != "merge"
		r.Options.MergeIdenticalModels = mode != "dedup"
		r.Options.HoistSharedShapes = mode == "combined"
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"AlphaResponseDto", "BetaResponseDto"} {
			if strings.Contains(source(t, got, name), "typealias "+name) {
				t.Fatal("different defaults collapsed", mode, name)
			}
		}
	}
}

func TestReferencedCollectionFieldDefaultsRequiredAndPatchHaveNoInitializerIdentity(t *testing.T) {
	doc := referencedDefaultDocument(`[]`)
	doc = []byte(strings.Replace(string(doc), `"again":{"$ref":"#/components/schemas/Alias","default":[]}}}`, `"again":{"$ref":"#/components/schemas/Alias","default":[]}},"required":["value","again"]}`, 1))
	first, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := core.GenerateWithDiagnostics(request([]byte(strings.ReplaceAll(string(doc), `"default":[]`, `"default":[{"a":1}]`)), "Dto"))
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("required annotation invented runtime identity", err)
	}
}

func FuzzReferencedCollectionFieldDefaults(f *testing.F) {
	for _, value := range []string{`[]`, `[{"a":1,"b":null}]`, `[{"a/~":false}]`, `[{"a":1.0}]`, `null`, `[null]`, `{}`} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 || !json.Valid([]byte(value)) {
			return
		}
		for _, checked := range []bool{false, true} {
			r := request(referencedDefaultDocument(value), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				if !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("partial error output")
				}
				continue
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("non-deterministic reference default", err)
			}
		}
	})
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func scalarDefaultOperations(value string) []byte {
	field := `{"$ref":"#/components/schemas/Value","default":` + value + `}`
	doc := nestedBuiltinDefaultOperations(field)
	return []byte(strings.TrimSuffix(string(doc), "}") + `,"components":{"schemas":{"Value":{"type":"number","format":"float","nullable":true}}}}`)
}

func TestReferencedScalarFieldDefaultsOptimizationAndLock(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, mode := range []string{"dedup", "merge", "combined"} {
			r := request(scalarDefaultOperations("1"), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = true
			r.Options.DeduplicateModels = mode != "merge"
			r.Options.MergeIdenticalModels = mode != "dedup"
			r.Options.HoistSharedShapes = mode == "combined"
			r.Options.LockHoistedNames = mode == "combined"
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			alias := "BetaResponseDto"
			if mode == "combined" {
				alias = "AlphaResponseDto"
			}
			if !strings.Contains(source(t, got, alias), "typealias "+alias) {
				t.Fatal("reuse lost", mode)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			r.Document = scalarDefaultOperations("1.000000001")
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("mapped Float identity", mode, err)
			}
			r.Document = scalarDefaultOperations("2")
			changed, err := core.GenerateWithDiagnostics(r)
			if mode == "combined" {
				if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
					t.Fatal("lock drift", err)
				}
			} else if err != nil || reflect.DeepEqual(got, changed) {
				t.Fatal("initializer identity", err)
			}
			r.PreviousNameLock = nil
			doc := string(scalarDefaultOperations("1"))
			at := strings.Index(doc, `"/b":`)
			r.Document = []byte(doc[:at] + strings.Replace(doc[at:], `"default":1`, `"default":2`, 1))
			different, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"AlphaResponseDto", "BetaResponseDto"} {
				if strings.Contains(source(t, different, name), "typealias "+name) {
					t.Fatal("different defaults collapsed", mode)
				}
			}
		}
	}
	// A required annotation supplies no runtime shape identity.
	r := request(schemas(`{"Value":{"type":"integer"},"Holder":{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Value","default":1}}}}`), "Dto")
	first, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Document = []byte(strings.ReplaceAll(string(r.Document), `"default":1`, `"default":2`))
	second, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("required initializer", err)
	}
}

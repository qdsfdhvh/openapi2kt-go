package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func falseNominalSignatureInput(count int, required bool) []byte {
	props := map[string]any{}
	names := []string{}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("p%03d", i)
		props[name] = false
		names = append(names, name)
	}
	model := map[string]any{"properties": props}
	if required {
		model["required"] = names
	}
	data, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": model}}})
	return data
}
func TestFalseNominalJVMSignatureLimits(t *testing.T) {
	packageName := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	for _, required := range []bool{false, true} {
		for _, tc := range []struct {
			style  core.ModelStyle
			count  int
			accept bool
		}{{core.DataClass, 188, true}, {core.DataClass, 189, false}, {core.PlainClass, 190, true}, {core.PlainClass, 191, false}} {
			for mask := 0; mask < 8; mask++ {
				original := falseNominalSignatureInput(tc.count, required)
				rq := request(original, "Dto")
				rq.Options.DeduplicateModels = mask&1 != 0
				rq.Options.HoistSharedShapes = mask&2 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				rq.Options.MergeIdenticalModels = mask&4 != 0
				rq.Options.PackageName = packageName
				rq.Options.ModelStyle = tc.style
				report, err := core.GenerateWithDiagnostics(rq)
				if !bytes.Equal(original, falseNominalSignatureInput(tc.count, required)) {
					t.Fatal("input mutated")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if !reflect.DeepEqual(report, again) || !reflect.DeepEqual(err, e) {
					t.Fatal("nondeterministic", err, e)
				}
				if tc.accept {
					if err != nil {
						t.Fatal(tc, required, err)
					}
					continue
				}
				if err == nil || len(report.Artifacts.Files) != 0 {
					t.Errorf("impossible actual false nominal signature emitted: %+v required=%v err=%v", tc, required, err)
					continue
				}
				ds, ok := err.(core.Diagnostics)
				if !ok || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Wide" {
					t.Fatal(err)
				}
			}

		}
	}
}

func TestFalseNominalGuardCountsActualNestedAndRootClassifiers(t *testing.T) {
	packageName := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	for _, named := range []bool{false, true} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			if named {
				props[fmt.Sprintf("p%03d", i)] = map[string]any{"$ref": "#/components/schemas/Blocked"}
			} else {
				props[fmt.Sprintf("p%03d", i)] = false
			}
		}
		schemas := map[string]any{"Wide/~": map[string]any{"properties": props}}
		if named {
			schemas["Blocked"] = false
		}
		data, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(data, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.PackageName = packageName
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(report.Artifacts.Files) != 0 {
				t.Fatal("root/nested false overflow admitted", named, style, err)
			}
			ds := err.(core.Diagnostics)
			if ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Wide~1~0" {
				t.Fatal(err)
			}
			rq.Options.PackageName = "org.example.short"
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("original240 slots reduced", named, style, err)
			}
		}
	}
}

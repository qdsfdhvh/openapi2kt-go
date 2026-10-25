package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func falseGenericSignatureInput(count int, required bool) []byte {
	props := map[string]any{}
	names := []string{}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("p%03d", i)
		props[name] = map[string]any{"type": "array", "items": false}
		names = append(names, name)
	}
	model := map[string]any{"properties": props}
	if required {
		model["required"] = names
	}
	data, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": model}}})
	return data
}
func TestActualFalseGenericSignatureLimits(t *testing.T) {
	pkg := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	for _, required := range []bool{false, true} {
		for _, tc := range []struct {
			style  core.ModelStyle
			count  int
			accept bool
		}{{core.PlainClass, 180, true}, {core.PlainClass, 181, false}, {core.DataClass, 179, true}, {core.DataClass, 180, false}} {
			for mask := 0; mask < 8; mask++ {
				original := falseGenericSignatureInput(tc.count, required)
				before := append([]byte(nil), original...)
				rq := request(original, "Dto")
				rq.Options.DeduplicateModels = mask&1 != 0
				rq.Options.HoistSharedShapes = mask&2 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				rq.Options.MergeIdenticalModels = mask&4 != 0
				rq.Options.PackageName = pkg
				rq.Options.ModelStyle = tc.style
				report, err := core.GenerateWithDiagnostics(rq)
				if !bytes.Equal(original, before) {
					t.Fatal("input mutated")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if !reflect.DeepEqual(report, again) || !reflect.DeepEqual(err, e) {
					t.Fatal("generic limit nondeterministic", err, e)
				}
				if tc.accept {
					if err != nil {
						t.Fatal(tc, required, err)
					}
					continue
				}
				if err == nil || len(report.Artifacts.Files) != 0 {
					t.Errorf("uncompilable knownfalse generic signature emitted %+v required=%v err=%v", tc, required, err)
					continue
				}
				d := err.(core.Diagnostics)[0]
				if d.Code != "E_MODEL_LIMIT" || d.Pointer == nil || *d.Pointer != "/components/schemas/Wide" {
					t.Fatal(err)
				}
			}

		}
	}
}

func TestFalseGenericRootAndOriginalPointers(t *testing.T) {
	pkg := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	for _, named := range []bool{false, true} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			var leaf any = false
			if named {
				leaf = map[string]any{"$ref": "#/components/schemas/Blocked"}
			}
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "array", "items": leaf}
		}
		schemas := map[string]any{"Wide/~": map[string]any{"properties": props}}
		if named {
			schemas["Blocked"] = false
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": schemas}})
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(input, "Dto")
			rq.Options.PackageName = pkg
			rq.Options.ModelStyle = style
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(report.Artifacts.Files) != 0 {
				t.Fatal("actual root/nested false generic overflow admitted", named, style, err)
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != "E_MODEL_LIMIT" || d.Pointer == nil || *d.Pointer != "/components/schemas/Wide~1~0" {
				t.Fatal("generic limit originalPointer lost", d)
			}
			rq.Options.PackageName = "org.example.short"
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("original240 slots reduced", named, style, e)
			}
		}
	}
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func explicitNullOperations(constant string) []byte {
	object := `{"properties":{"kotlinx":{"type":"string"},"value":{"type":"null","enum":[null]},"fixed":` + constant + `},"required":["value"]}`
	op := func(id, shape string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"nested":` + shape + `},"required":["nested"]}}}}}}}`
	}
	other := strings.Replace(object, constant, `{"type":"string","const":"other"}`, 1)
	return []byte(`{"openapi":"3.1.0","paths":{"/a":` + op("alpha", object) + `,"/b":` + op("beta", object) + `,"/c":` + op("gamma", other) + `}}`)
}

func TestExplicitNullLiteralsKeepEffectiveReuseAndOwnedLockIdentity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for _, qualified := range []bool{false, true} {
				r := request(explicitNullOperations(`{"type":"null","const":null}`), "Dto")
				r.Options.Validation = checked
				r.Options.ModelStyle = style
				r.Options.HoistSharedShapes = true
				r.Options.DeduplicateModels = true
				r.Options.MergeIdenticalModels = true
				r.Options.LockHoistedNames = true
				r.Options.QualifyHoistedNames = qualified
				got, e := core.GenerateWithDiagnostics(r)
				if e != nil || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
					t.Fatal(style, checked, qualified, e)
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				again, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Original null-owned lock replay", e)
				}
				inferred := r
				inferred.Document = []byte(strings.ReplaceAll(string(r.Document), `"type":"null",`, ""))
				implicit, e := core.GenerateWithDiagnostics(inferred)
				if e != nil || !reflect.DeepEqual(got, implicit) {
					t.Fatal("Inferred null changed actual owned reuse or lock identity", e)
				}
				// A null-only vocabulary is the same actual scalar domain/type as type:null.
				r.Document = []byte(strings.ReplaceAll(string(r.Document), `,"enum":[null]`, ""))
				control, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("Equivalent null carrier changed reuse/lock", e)
				}
				r.Document = explicitNullOperations(`{"type":"string","const":"changed"}`)
				changed, e := core.GenerateWithDiagnostics(r)
				if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
					t.Fatal("Owned const kind drift ignored", e)
				}
			}
		}
	}
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedBoundsOperations(minimum int) []byte {
	object := `{"type":"object","allOf":[{"$ref":"#/components/schemas/Lower"},{"type":"object","allOf":[{"properties":{"value":{"type":"string","maxLength":3}}}]}]}`
	if minimum == 2 {
		object = strings.ReplaceAll(object, `"maxLength":3`, `"maxLength":4`)
	}
	op := func(id, schema string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"nested":` + schema + `},"required":["nested"]}}}}}}}`
	}
	other := strings.ReplaceAll(object, `"maxLength":3`, `"maxLength":5`)
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Lower":{"properties":{"value":{"type":"string","minLength":2}},"required":["value"]}}},"paths":{"/a":` + op("alpha", object) + `,"/b":` + op("beta", object) + `,"/c":` + op("gamma", other) + `}}`)
}

func TestReferencedNestedAllOfBoundsKeepEffectiveReuseAndLockIdentity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			r := request(referencedBoundsOperations(1), "Dto")
			r.Options.Validation = true
			r.Options.ModelStyle = style
			r.Options.HoistSharedShapes = true
			r.Options.DeduplicateModels = true
			r.Options.MergeIdenticalModels = true
			r.Options.LockHoistedNames = true
			r.Options.QualifyHoistedNames = qualified
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
				t.Fatal(style, qualified, "Effective bounds collapsed", e)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Original lock replay", e)
			}
			r.Document = referencedBoundsOperations(2)
			changed, e := core.GenerateWithDiagnostics(r)
			if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || len(changed.Artifacts.Files) != 0 {
				t.Fatal("Real owned bounds drift ignored", e)
			}
		}
	}
}

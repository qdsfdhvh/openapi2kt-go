package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedCountOperations(maximum int) []byte {
	input := string(referencedBoundsOperations(1))
	input = strings.Replace(input, `"Lower":{"properties":`, `"Lower":{"minProperties":1,"properties":`, 1)
	input = strings.ReplaceAll(input, `"type":"object","allOf":[{"properties":`, `"type":"object","maxProperties":2,"allOf":[{"properties":`)
	if maximum == 3 {
		input = strings.ReplaceAll(input, `"maxProperties":2`, `"maxProperties":3`)
	}
	return []byte(input)
}

func TestReferencedNestedAllOfCountsKeepEffectiveReuseAndOwnedLock(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			r := request(referencedCountOperations(2), "Dto")
			r.Options.Validation = true
			r.Options.ModelStyle = style
			r.Options.HoistSharedShapes = true
			r.Options.DeduplicateModels = true
			r.Options.MergeIdenticalModels = true
			r.Options.LockHoistedNames = true
			r.Options.QualifyHoistedNames = qualified
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
				t.Fatal(style, qualified, e)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Original whole-object lock replay", e)
			}
			r.Document = referencedCountOperations(3)
			changed, e := core.GenerateWithDiagnostics(r)
			if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("Owned effective count drift ignored", e)
			}
		}
	}
}

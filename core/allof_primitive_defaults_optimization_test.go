package core_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfPrimitiveDefaultsKeepEffectiveReuseAndLockIdentity(t *testing.T) {
	input := string(referencedBoundsOperations(1))
	input = strings.ReplaceAll(input, `,"required":["value"]`, "")
	input = strings.ReplaceAll(input, `"minLength":2`, `"minLength":2,"default":"ab"`)
	input = strings.ReplaceAll(input, `"maxLength":3`, `"maxLength":3,"default":"abc"`)
	input = strings.ReplaceAll(input, `"maxLength":5`, `"maxLength":3,"default":"abd"`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			r := request([]byte(input), "Dto")
			r.Options.Validation = true
			r.Options.ModelStyle = style
			r.Options.HoistSharedShapes = true
			r.Options.DeduplicateModels = true
			r.Options.MergeIdenticalModels = true
			r.Options.LockHoistedNames = true
			r.Options.QualifyHoistedNames = qualified
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
				t.Fatal("Different winning defaults collapsed", style, qualified, err)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Default-bearing owned lock replay failed", err)
			}
			// Only the source component's own default changes. The operation
			// initializer still comes from the last contribution.
			r.Document = []byte(strings.ReplaceAll(input, `"default":"ab"`, `"default":"longer"`))
			changedSource, err := core.GenerateWithDiagnostics(r)
			if err != nil || !bytes.Equal(got.Artifacts.NameLock, changedSource.Artifacts.NameLock) || source(t, got, "AlphaResponseDto") != source(t, changedSource, "AlphaResponseDto") {
				t.Fatal("Discarded annotation entered effective owned identity", err)
			}
			r.Document = []byte(strings.ReplaceAll(input, `"default":"abc"`, `"default":"abe"`))
			failed, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || len(failed.Artifacts.Files) != 0 {
				t.Fatal("Changed winning owned initializer ignored", err)
			}
		}
	}
}

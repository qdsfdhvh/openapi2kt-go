package core_test

import (
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestHoistActualReservedRuntimeAndComponents(t *testing.T) {
	input, e := os.ReadFile("../tests/fixtures/http/hoist-reservations.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, moved := range []bool{false, true} {
			for _, qualified := range []bool{false, true} {
				req := request(input, "")
				req.Options.Validation = true
				req.Options.ModelStyle = style
				if moved {
					req.Options.RuntimePackage = "org.example.shared.runtime"
				}
				base, e := core.GenerateWithDiagnostics(req)
				if e != nil {
					t.Fatal(e)
				}
				req.Options.HoistSharedShapes = true
				req.Options.QualifyHoistedNames = qualified
				req.Options.DeduplicateModels = true
				r, e := core.GenerateWithDiagnostics(req)
				if e != nil || !reflect.DeepEqual(base.Warnings, r.Warnings) {
					t.Fatal("options/warnings", e)
				}
				if source(t, r, "Response") != source(t, base, "Response") || source(t, r, "Response2") != source(t, base, "Response2") {
					t.Fatal("component was renamed/absorbed")
				}
				for _, role := range []string{"Response", "Patch", "PatchSerializer", "LongAsStringSerializer"} {
					target := role
					if qualified {
						target = "Alpha" + role
					} else if role == "Response" {
						target += "3"
					} else if !moved {
						target += "2"
					}
					if !strings.Contains(source(t, r, "Beta"+role), "public typealias Beta"+role+" = org.example.models.models."+target) {
						t.Fatal(role, moved, qualified, target, source(t, r, "Beta"+role))
					}
				}
			}
		}
	}
}

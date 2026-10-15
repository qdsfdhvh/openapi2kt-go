package core_test

import (
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestHoistWholeEnvelopeOptions(t *testing.T) {
	input, e := os.ReadFile("../tests/fixtures/http/deduplicate-models.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualify := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = true
			req.Options.ModelStyle = style
			base, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			req.Options.HoistSharedShapes = true
			req.Options.QualifyHoistedNames = qualify
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) || !reflect.DeepEqual(r.Warnings, base.Warnings) || len(r.Artifacts.Files) != len(base.Artifacts.Files)+3 {
				t.Fatal("parity/group/determinism", e)
			}
			for _, role := range []string{"Response", "Input", "Mode"} {
				shared := role + "Dto"
				if qualify {
					shared = "Alpha" + shared
				}
				for _, owner := range []string{"Alpha", "Beta"} {
					text := source(t, r, owner+role+"Dto")
					if !strings.Contains(text, "public typealias "+owner+role+"Dto = org.example.models.models."+shared) {
						t.Fatal(text)
					}
				}
				found := false
				for _, f := range r.Artifacts.Files {
					if f.Path == "org/example/models/models/"+shared+".kt" {
						found = true
						if strings.Contains(f.Content, "public typealias") {
							t.Fatal("shared is alias", f.Content)
						}
					}
				}
				if !found {
					t.Fatal(shared)
				}
			}
			req.Options.DeduplicateModels = true
			combined, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, combined) {
				t.Fatal("pass order changed hoist/dedup", e)
			}
			req.Options.HoistSharedShapes = false
			req.Options.QualifyHoistedNames = false
			req.Options.DeduplicateModels = false
			off, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(base, off) {
				t.Fatal("off changed output", e)
			}
		}
	}
	req := request([]byte("{"), "")
	req.Options.QualifyHoistedNames = true
	r, e := core.GenerateWithDiagnostics(req)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_OPTIONS" || len(r.Artifacts.Files) != 0 {
		t.Fatal("dependency not before document load", e)
	}
}

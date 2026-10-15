package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestOperationProviderIncomingNestedOwnerPreserved(t *testing.T) {
	data, e := os.ReadFile("../tests/fixtures/http/operation-native-dependency.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, hoist := range []bool{false, true} {
			req := request(data, "Dto")
			req.Options.PackageName = "org.example.dependencies"
			req.Options.Validation = true
			req.Options.ModelStyle = style
			req.Options.DeduplicateModels = true
			req.Options.HoistSharedShapes = hoist
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(source(t, r, "BetaResponseDto"), "public typealias") {
				t.Fatal("incoming provider classifier erased")
			}
			if !strings.Contains(source(t, r, "CallerDto"), "org.example.dependencies.api.models.BetaResponseDto.Child") {
				t.Fatal("opaque provider rewritten")
			}
		}
	}
}

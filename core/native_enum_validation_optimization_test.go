package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumChecksParticipateInReuseAndBranchFieldIdentity(t *testing.T) {
	schemasJSON := `{"openapi":"3.1.0","components":{"schemas":{"Alpha":{"properties":{"code":{"type":"integer","enum":[1,2]}}},"Beta":{"properties":{"code":{"type":"integer","enum":[1,2]}}},"Gamma":{"properties":{"code":{"type":"integer","enum":[1,3]}}},"Choice":{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"code":{"type":"integer","enum":[1,2]}}},{"properties":{"kind":{"type":"string","const":"b"},"code":{"type":"integer","enum":[1,3]}}}]}}}}`
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		r := request([]byte(schemasJSON), "Dto")
		r.Options.Validation = true
		r.Options.ModelStyle = style
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
			t.Fatal("enum membership absent from merge identity")
		}
		choice := source(t, got, "ChoiceDto")
		for _, part := range []string{"value == 1 || value == 2", "value == 1 || value == 3"} {
			if !strings.Contains(choice, part) {
				t.Fatal("variant vocabulary dropped", part, choice)
			}
		}
		repeat, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, repeat) {
			t.Fatal(err)
		}
	}
}

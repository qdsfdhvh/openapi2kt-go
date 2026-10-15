package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestBorrowedCollectionEnumLeavesOriginalCanonicalBacking(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/borrowed-collection-enum-leaves.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle, rq.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []struct{ name, backing string }{{"RowsBridgeDto", "kotlin.collections.List<org.example.models.models.StateDto?>"}, {"MapBridgeDto", "kotlin.collections.Map<kotlin.String, org.example.models.models.OwnerDto.State?>"}, {"AgainBridgeDto", "kotlin.collections.List<org.example.models.models.StateDto?>"}} {
				if !strings.Contains(source(t, got, want.name), "public typealias "+want.name+" = "+want.backing+"\n") {
					t.Fatal(want, source(t, got, want.name))
				}
			}
			if strings.Contains(source(t, got, "DefaultApi"), "RowsBridgeDto.Inner") {
				t.Fatal("borrowed alias invented peer ownership")
			}
		}
	}
}

func borrowedEnumGraduation(t *testing.T, input []byte, canonical string) {
	t.Helper()
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("original ordinaryDTO borrowed collection positive", checked, err)
		}
		all := ""
		for _, file := range got.Artifacts.Files {
			all += file.Content
		}
		if !strings.Contains(all, "kotlin.collections.List<"+canonical+">") {
			t.Fatal("lost actual nominal leaf backing", canonical, all)
		}
	}
}

package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestComposedNestedModelActualOwners(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/composed-nested-owners.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, validation := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct {
			name, owner string
			nullable    bool
		}{{"ComposedNode", "ComposedDto.Node", false}, {"ChoiceNode", "ChoiceDto.First.Node", false}, {"MaybeComposedNode", "ComposedDto.Node", true}, {"NarrowChoiceNode", "ChoiceDto.First.Node", false}, {"ComposedRow", "ComposedDto.RowsInner", false}, {"ComposedValue", "ComposedDto.EntriesValue", false}, {"MixedNode", "ObjectMixedChoiceDto.Node", false}, {"AnyNode", "AnyChoiceDto.A.Node", false}, {"ChoiceBranch", "ChoiceDto.First", false}, {"RootRowNode", "RootRowsDtoInner.R.Node", false}, {"RootMapNode", "RootMapDtoValue.M.Node", false}} {
			want := "public typealias " + tc.name + "Dto = org.example.models.models." + tc.owner
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.name+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.name+"Dto"))
			}
		}
		if !strings.Contains(source(t, r, "ComposedDto"), "public val peer: org.example.models.models.ChoiceDto.First.Node? = null,") || !strings.Contains(source(t, r, "ChoiceDto"), "public val peer: org.example.models.models.ComposedDto.Node? = null,") {
			t.Fatal("mutual source-owned references")
		}
	}
}

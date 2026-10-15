package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestEnumNominalRetainedVocabularyOwners(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/enum-nominal-owners.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, checked := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = checked
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct {
			alias, owner string
			nullable     bool
		}{
			{"NestedState", "OwnerDto.State", true}, {"NestedStateFalse", "OwnerDto.State", false}, {"RowState", "OwnerDto.RowsInner", false}, {"MapState", "OwnerDto.EntriesValue", false}, {"VariantState", "ChoiceDto.A.State", false}, {"RootRowState", "RootRowsDtoInner", false}, {"NestedBridge", "OwnerDto.State", true}, {"NamedBridge", "NamedStateDto", false},
		} {
			want := "public typealias " + tc.alias + "Dto = org.example.models.models." + tc.owner
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.alias+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.alias+"Dto"))
			}
		}
	}
}

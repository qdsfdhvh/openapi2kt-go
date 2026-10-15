package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func hierarchyNullableAliasesInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/hierarchy-nullable-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHierarchyNullableAliasesCanonicalNearestIntent(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(hierarchyNullableAliasesInput(t), suffix)
			req.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(err)
			}
			for _, pair := range []struct {
				alias, target string
				nullable      bool
			}{{"ARoot", "Root", true}, {"BRoot", "Root", true}, {"CRoot", "Root", false}, {"DRoot", "Root", true}, {"AGroup", "Group", true}, {"AFirst", "First", true}, {"AReused", "ReusedRoot", true}, {"AReusedLeaf", "ReusedFirst", true}} {
				s := source(t, got, pair.alias+suffix)
				want := "public typealias " + pair.alias + suffix + " = org.example.models.models." + pair.target + suffix
				if pair.nullable {
					want += "?"
				}
				if !strings.Contains(s, want) {
					t.Fatal(want, s)
				}
			}
			holder := source(t, got, "AliasHolder"+suffix)
			for _, part := range []string{"public val inherited: org.example.models.models.Root" + suffix + "?", "public val narrowed: org.example.models.models.Root" + suffix + ",", "public val leaf: org.example.models.models.First" + suffix + "?", "kotlin.collections.List<org.example.models.models.Root" + suffix + "?>"} {
				if !strings.Contains(holder, part) {
					t.Fatal(part, holder)
				}
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
		}
	}
}

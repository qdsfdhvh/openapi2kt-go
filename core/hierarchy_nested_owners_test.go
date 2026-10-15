package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHierarchyNestedDTOActualCanonicalOwners(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/hierarchy-nested-owners.json")
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
		}{{"Node", "FirstDto.Node", false}, {"NodeChain", "FirstDto.Node", false}, {"NodeMaybe", "FirstDto.Node", true}, {"NodeFalse", "FirstDto.Node", false}, {"Row", "FirstDto.RowsInner", false}, {"Value", "FirstDto.EntriesValue", false}} {
			want := "public typealias " + tc.name + "Dto = org.example.models.models." + tc.owner
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.name+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.name+"Dto"))
			}
		}
		first := source(t, r, "FirstDto")
		second := source(t, r, "SecondDto")
		if !strings.Contains(first, "public data class Node(") || !strings.Contains(first, "public val peer: org.example.models.models.SecondDto.Node? = null,") || !strings.Contains(second, "public val peer: org.example.models.models.FirstDto.Node? = null,") {
			t.Fatal("actual nested mutual nominal identity", first, second)
		}
	}
}

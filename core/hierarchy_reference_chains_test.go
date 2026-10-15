package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalChainsKeepNamedHierarchyCatalogAndNullableIntent(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/hierarchy-reference-chains.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, validation := range []bool{false, true} {
		r := request(b, "Dto")
		r.Options.Validation = validation
		got, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct {
			name, physical string
			nullable       bool
		}{{"ParentBridge", "Root", true}, {"ParentBridgeChain", "Root", true}, {"ParentFalse", "Root", false}, {"ParentReNull", "Root", true}, {"LeafBridge", "First", true}, {"LeafFalse", "First", false}, {"GroupBridge", "Group", true}} {
			want := "public typealias " + tc.name + "Dto = org.example.models.models." + tc.physical + "Dto"
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, got, tc.name+"Dto"), want) {
				t.Fatal(tc)
			}
		}
		if !strings.Contains(source(t, got, "FirstDto"), "public val ancestor: org.example.models.models.RootDto? = null,") || !strings.Contains(source(t, got, "FirstDto"), "public val peer: org.example.models.models.FirstDto? = null,") {
			t.Fatal("nominal recursive chain identity")
		}
	}
}

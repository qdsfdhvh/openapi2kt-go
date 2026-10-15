package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestValueClassInternalChainsCanonicalIdentity(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/valueclass-reference-chains.json")
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
			alias, target string
			nullable      bool
		}{{"CountBridge", "Count", true}, {"CountBridgeFalse", "Count", false}, {"CountBridgeTrue", "Count", true}, {"NameBridge", "Name", true}} {
			want := "public typealias " + tc.alias + "Dto = org.example.models.models." + tc.target + "Dto"
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.alias+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.alias+"Dto"))
			}
		}
		if !strings.Contains(source(t, r, "CountDto"), "public value class CountDto") || !strings.Contains(source(t, r, "HolderDto"), "narrow: org.example.models.models.CountDto") {
			t.Fatal("canonical wrapper lost")
		}
	}
}

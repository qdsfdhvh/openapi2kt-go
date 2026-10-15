package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfNominalConstructorTerminals(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/allof-nominal-terminals.json")
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
		}{{"AllNested", "AllContainerDto.Node", false}, {"AllMaybe", "AllContainerDto.Node", true}, {"AllFalse", "AllContainerDto.Node", false}, {"AllRootBridge", "AllModelDto", false}, {"AllNestedBridge", "AllContainerDto.Node", true}, {"AllVariant", "AllChoiceDto.All", false}, {"IntrinsicAllNode", "IntrinsicAllContainerDto.Node", true}, {"IntrinsicAllFalse", "IntrinsicAllContainerDto.Node", false}, {"InheritedAllNode", "InheritedAllContainerDto.Node", true}, {"InheritedAllFalse", "InheritedAllContainerDto.Node", false}} {
			want := "public typealias " + tc.alias + "Dto = org.example.models.models." + tc.owner
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.alias+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.alias+"Dto"))
			}
		}
		if !strings.Contains(source(t, r, "AllModelDto"), "nested: org.example.models.models.AllContainerDto.Node? = null") || !strings.Contains(source(t, r, "AllContainerDto"), "peer: org.example.models.models.AllModelDto? = null") {
			t.Fatal("canonical allOf mutual recursion")
		}
	}
}

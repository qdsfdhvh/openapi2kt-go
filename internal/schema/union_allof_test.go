package schema

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestUnionAllOfBorrowedFieldIdentityAndDepth(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/union-allof.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	a := &analyzer{root: root, components: map[string]*component{}, options: Options{PackageName: "org.example.unionallof", Suffix: "Dto"}}
	if _, e := a.analyze(root); e != nil {
		t.Fatal(e)
	}
	s := get(get(root, "components"), "schemas")
	v := get(s, "Composed")
	view, e := a.objectCompositionUnion(v, "/components/schemas/Composed", 0)
	if e != nil || view == nil || len(view.Branches) != 2 {
		t.Fatal(view, e)
	}
	id := get(get(get(s, "Common"), "properties"), "id")
	for _, b := range view.Branches {
		f := shapeField(b.Shape, "id")
		if f.Schema != id || f.Pointer != "/components/schemas/Common/properties/id" || f.Depth != 3 {
			t.Fatal("source identity/depth", f)
		}
		if b.Schema != v || b.Pointer != "/components/schemas/Composed" {
			t.Fatal(b)
		}
	}
}

func TestUnionAllOfDetectorUsesBoundedOriginalGraph(t *testing.T) {
	a := jsonGraphAnalyzer(t, `"Root":{"allOf":[{"$ref":"#/components/schemas/Root"}]}`)
	s := get(get(a.root, "components"), "schemas")
	a.components = map[string]*component{"Root": {Schema: get(s, "Root"), Wire: "Root", Pointer: "/components/schemas/Root"}}
	if a.containsObjectCompositionUnion(get(s, "Root"), 0, map[*document.Value]bool{}) {
		t.Fatal("ordinary cyclic allOf changed route")
	}
	var stack []string
	for i := 0; i < 100; i++ {
		stack = append(stack, `{"type":"object"}`)
	}
	a = jsonGraphAnalyzer(t, `"Root":{"allOf":[`+strings.Join(stack, ",")+`]}`)
	if a.containsObjectCompositionUnion(get(get(get(a.root, "components"), "schemas"), "Root"), 0, map[*document.Value]bool{}) {
		t.Fatal("pure allOf changed route")
	}
}

package schema

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestHierarchyFieldsSourceIdentityAndCanonicalEdgeBodies(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/hierarchy-fields.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	a := &analyzer{root: root, components: map[string]*component{}, options: Options{PackageName: "org.example.hierarchyfields", Suffix: "Dto"}}
	declarations, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	schemas := get(get(root, "components"), "schemas")
	leaf := get(schemas, "MissingFirst")
	id := get(get(leaf, "properties"), "id")
	if a.components["MissingFirst"].Schema != leaf || get(get(leaf, "properties"), "kind") != nil {
		t.Fatal("generated mapping body mutated original schema")
	}
	for _, d := range declarations {
		if h, ok := d.(*Hierarchy); ok && h.ClassName == "DeepFirstDto" {
			if len(h.Leaf.Constants) != 2 || h.Leaf.Constants[0].WireName != "family" || h.Leaf.Constants[1].WireName != "stage" {
				t.Fatal(h.Leaf)
			}
		}
	}
	if get(get(leaf, "properties"), "id") != id {
		t.Fatal("source field replaced")
	}
	p := &hierarchyPlanner{a: a, graph: map[string]*hierarchyBranches{}, shapes: map[string]*shape{}, parents: map[string]string{}, heights: map[string]int{}}
	for _, key := range []string{"DeepRoot", "MissingRoot", "FixedRoot", "VaryRoot"} {
		if e := p.visit(key, 1, nil, a.components[key].Pointer); e != nil {
			t.Fatal(e)
		}
	}
	for _, tc := range []struct{ key, wire, value, pointer string }{
		{"MissingFirst", "kind", "first/a~b", "/components/schemas/MissingRoot/oneOf/0/$ref"},
		{"DeepFirst", "family", "group", "/components/schemas/DeepRoot/oneOf/0/$ref"},
		{"DeepSecond", "family", "group", "/components/schemas/DeepRoot/oneOf/0/$ref"},
		{"DeepFirst", "stage", "first", "/components/schemas/DeepGroup/oneOf/0/$ref"},
	} {
		edge, ok := p.mappedField(tc.key, tc.wire)
		if !ok || edge.Value != tc.value || edge.Pointer != tc.pointer {
			t.Fatal(tc, edge, ok)
		}
	}
	f := shapeField(p.shapes["MissingFirst"], "id")
	if f.Schema != id || f.Pointer != "/components/schemas/MissingFirst/properties/id" || f.Depth != 3 {
		t.Fatal(f)
	}
	if len(p.shapes["MissingFirst"].Fields) != 1 || len(get(leaf, "properties").Members()) != 1 {
		t.Fatal("synthetic source fields")
	}
}

func TestHierarchyFieldsBodyOrderAndNoEmptyObjectApproximation(t *testing.T) {
	a := jsonGraphAnalyzer(t, `"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"leaf":"Leaf"}}},"Leaf":{"type":"object"}`)
	a.options.PackageName = "org.example.hierarchyfields"
	a.components = map[string]*component{}
	if _, err := a.analyze(a.root); err != nil {
		t.Fatal(err)
	}
	for _, w := range a.warnings {
		if w.Code == "W_JSON_ELEMENT_APPROXIMATION" && w.Pointer != nil && strings.HasPrefix(*w.Pointer, "/components/schemas/Leaf") {
			t.Fatal("generated consuming leaf called raw", w)
		}
	}
}

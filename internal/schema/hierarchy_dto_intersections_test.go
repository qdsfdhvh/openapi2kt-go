package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestHierarchyNestedIntersectionOwnSourceAndParentTagIsolation(t *testing.T) {
	data, e := os.ReadFile("../../tests/fixtures/http/hierarchy-dto-intersections.json")
	if e != nil {
		t.Fatal(e)
	}
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.hierarchydto", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	target, steps, _, err := a.resolveIntersectionMember("#/components/schemas/Owner/properties/payload", "/use", 0)
	owner := a.components["Owner"]
	if err != nil || target.Catalog != nil || target.SourceCatalog != owner || !target.DirectLexical || target.ComposedLexical || steps != 1 || !a.hierarchySources[owner.Schema] || len(a.components) != len(catalog.Members()) {
		t.Fatal("namedleaf originallexical source lost", target, steps, err)
	}
	var leaf *Hierarchy
	for _, d := range ds {
		if h, ok := d.(*Hierarchy); ok && h.ClassName == owner.Name {
			leaf = h
		}
	}
	use := a.borrowedDTOLeaves[target.Schema]
	binding := a.deferredNominals[target.Schema]
	if leaf == nil || leaf.Leaf == nil || use == nil || use.Actual == nil || binding == nil || binding.Owner != owner || binding.Schema != target.Schema || use.Name != binding.Name || a.modelOrigins[use.Actual] != target.Schema || a.modelPointers[use.Actual] != target.Pointer || use.Actual == leaf.Leaf {
		t.Fatal("nestedtarget borrowed root/common or guessedsourceowner", leaf, use, binding)
	}
	for _, p := range use.Actual.Properties {
		if p.WireName == "kind" {
			t.Fatal("parenttag became nestedproperty")
		}
	}
	shape, err := a.objectShape(get(catalog, "Consumer"), "/components/schemas/Consumer", 0)
	if err != nil || shapeField(shape, "kind") != nil || shapeField(shape, "id").Schema != get(get(target.Schema, "properties"), "id") {
		t.Fatal("parentkind/source drift in intersection", err)
	}
	work, bytes := a.intersectionCaptures.Work, a.intersectionCaptures.Bytes
	originalBytes, _ := target.Schema.MarshalJSON()
	if bytes != len(originalBytes) || len(a.intersectionCaptures.Seen) != 1 {
		t.Fatal("leaf tag captured or target recharged", bytes)
	}
	context := a
	view := *target
	if _, _, _, err = context.internalDTOIntersection(&view, "/repeat", 0, 1, nil); err != nil || context.intersectionCaptures != a.intersectionCaptures || a.intersectionCaptures.Work != work || a.intersectionCaptures.Bytes != bytes {
		t.Fatal("namedleaf capture lost context sharing", err)
	}
	wrong := *target
	wrong.SourceCatalog = a.components["Root"]
	if _, _, _, err = a.internalDTOIntersection(&wrong, "/parent", 0, 1, nil); err == nil {
		t.Fatal("parentcatalog replaced leafsource")
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != owner.Name {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("consumer authorized missing realleaf", err)
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("hierarchy tag planning rewrote originalsource")
	}
}

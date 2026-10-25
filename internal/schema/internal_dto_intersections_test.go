package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestInternalIntersectionOriginalOwnerAndSharedCaptureBudget(t *testing.T) {
	data, e := os.ReadFile("../../tests/fixtures/http/internal-dto-intersections.json")
	if e != nil {
		t.Fatal(e)
	}
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.internaldto", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	owner := a.components["Owner"]
	node := get(get(owner.Schema, "properties"), "payload")
	target, steps, _, err := a.resolveIntersectionMember("#/components/schemas/Owner/properties/payload", "/use", 0)
	if err != nil || target.Catalog != nil || target.SourceCatalog != owner || target.Schema != node || target.Pointer != "/components/schemas/Owner/properties/payload" || steps != 1 || len(a.components) != len(catalog.Members()) {
		t.Fatal("internal intersection invented catalog/source or resetheight", target, steps, err)
	}
	shape, err := a.objectShape(get(catalog, "Consumer"), "/components/schemas/Consumer", 0)
	if err != nil || shapeField(shape, "id").Schema != get(get(node, "properties"), "id") || shapeField(shape, "id").Pointer != "/components/schemas/Owner/properties/payload/properties/id" {
		t.Fatal("borrowedfield lost originalnode/Pointer", err)
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	use := a.borrowedDTOLeaves[node]
	if use == nil || use.Actual == nil || a.modelOrigins[use.Actual] != node || a.modelPointers[use.Actual] != target.Pointer || use.Name != "org.example.internaldto.models.OwnerDto.Payload" {
		t.Fatal("borrowedcopy became canonicalowner", use)
	}
	work, bytes := a.intersectionCaptures.Work, a.intersectionCaptures.Bytes
	originalBytes, _ := node.MarshalJSON()
	if work <= 0 || bytes != len(originalBytes) || len(a.intersectionCaptures.Seen) != 1 {
		t.Fatal("capture duplicated/sharedoriginal root", work, bytes)
	}
	context := a
	if _, _, _, err = context.resolveIntersectionMember("#/components/schemas/Bridge", "/again", 0); err != nil || context.intersectionCaptures != a.intersectionCaptures || a.intersectionCaptures.Work != work || a.intersectionCaptures.Bytes != bytes {
		t.Fatal("copiedcontext reset/recharged source budget", err)
	}
	// A different overlapping descendant is charged independently, even though
	// its bytes/nodes were included in the parent's earlier capture.
	child, err := a.resolveSchema("#/components/schemas/Owner/properties/payload/properties/node", "/child")
	if err != nil {
		t.Fatal(err)
	}
	childBytes, _ := child.Schema.MarshalJSON()
	if _, _, _, err = a.internalDTOIntersection(child, "/child", 0, 1, nil); err != nil || a.intersectionCaptures.Work <= work || a.intersectionCaptures.Bytes != bytes+len(childBytes) {
		t.Fatal("overlapping source capture was free", err)
	}
	for _, which := range []string{"exact", "work", "bytes"} {
		b := &intersectionCaptureBudget{Seen: map[*document.Value]bool{}, Work: 16384 - work, Bytes: (8 << 20) - bytes}
		if which == "work" {
			b.Work++
		}
		if which == "bytes" {
			b.Bytes++
		}
		context := a
		context.intersectionCaptures = b
		view := *target
		_, _, _, err = context.internalDTOIntersection(&view, "/boundary", 0, 1, nil)
		if which == "exact" {
			if err != nil || b.Work != 16384 || b.Bytes != 8<<20 {
				t.Fatal("exact sharedbudget boundary lost", b, err)
			}
			if _, _, _, err = context.internalDTOIntersection(&view, "/repeated", 0, 1, nil); err != nil {
				t.Fatal("same target recharged", err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("sourcecapture cumulative cap reset", which, err)
		}
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != owner.Name {
			missing = append(missing, d)
		}
	}
	if err = a.validateInternalModels(missing); err == nil || err.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("borrowedconsumer authorized missingoriginalowner", err)
	}
	fake := *owner
	old := use.Owner
	use.Owner = &fake
	if err = a.validateInternalModels(ds); err == nil {
		t.Fatal("copiedcatalog authorized capture")
	}
	use.Owner = old
	clone, parseFailure := document.ParseObject(originalBytes, document.MaxDocumentBytes)
	if parseFailure != nil {
		t.Fatal(parseFailure)
	}
	view := *target
	view.Schema = clone
	if _, _, _, err = a.internalDTOIntersection(&view, "/copiednode", 0, 1, nil); err == nil {
		t.Fatal("copied schema authorized originalowner")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST changed")
	}
}

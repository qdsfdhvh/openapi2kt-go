package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestAllOfTargetIntersectionOriginalOwnerAndSharedCaptureBudget(t *testing.T) {
	data, e := os.ReadFile("../../tests/fixtures/http/allof-target-intersections.json")
	if e != nil {
		t.Fatal(e)
	}
	root, failure := document.Load(data, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	before, _ := root.MarshalJSON()
	catalog := get(get(root, "components"), "schemas")
	a := analyzer{root: root, options: Options{PackageName: "org.example.alloftarget", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if err != nil || shapeField(shape, "id").Schema != get(get(get(node, "allOf").Items()[0], "properties"), "id") || shapeField(shape, "id").Pointer != "/components/schemas/Owner/properties/payload/allOf/0/properties/id" {
		t.Fatal("borrowedfield lost originalnode/Pointer", err)
	}
	if err = a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	use := a.borrowedAllOfLeaves[node]
	if use == nil || use.Actual == nil || a.modelOrigins[use.Actual] != node || a.modelPointers[use.Actual] != target.Pointer || use.Name != "org.example.alloftarget.models.OwnerDto.Payload" {
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
	child, err := a.resolveSchema("#/components/schemas/Owner/properties/payload/allOf/0/properties/node", "/child")
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

func TestAllOfTargetIntersectionReferencedSourceAndCachedHeightIdentity(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Base":{"properties":{"description":{"type":"string"}},"required":["description"]},"BaseAlias":{"$ref":"#/components/schemas/Base"},"BaseAgain":{"$ref":"#/components/schemas/BaseAlias"},"Owner":{"properties":{"payload":{"allOf":[{"$ref":"#/components/schemas/BaseAgain"},{"properties":{"label":{"type":"string"}}}]}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/payload"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}}}`)
	root, load := document.Load(input, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.alloftarget", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	target, e := a.resolveSchema("#/components/schemas/Owner/properties/payload", "/probe")
	if e != nil {
		t.Fatal(e)
	}
	use := a.borrowedAllOfLeaves[target.Schema]
	if use == nil || use.Actual == nil || len(use.Refs) < 3 {
		t.Fatal("originalnamedreference graph not retained", use)
	}
	for _, which := range []string{"pointer", "catalog"} {
		view := *target
		if which == "pointer" {
			view.Pointer += "/different"
		} else {
			view.SourceCatalog = a.components["Base"]
		}
		if _, _, _, e := a.internalDTOIntersection(&view, "/foreign", 0, 1, nil); e == nil {
			t.Fatal("cachedcomposed source acceptedforeign", which)
		}
	}
	base := a.components["Base"]
	original := base.Schema
	bytes, _ := original.MarshalJSON()
	clone, parse := document.ParseObject(bytes, document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	base.Schema = clone
	if e := a.validateInternalModels(ds); e == nil {
		t.Fatal("samebytes substituted namedsource accepted")
	}
	base.Schema = original
	alias := a.components["BaseAlias"]
	height := alias.AliasHeight
	alias.AliasHeight++
	if e := a.validateInternalModels(ds); e == nil {
		t.Fatal("cachedshape aliasheight drift accepted")
	}
	alias.AliasHeight = height
	branches := get(target.Schema, "allOf")
	branch := branches.Items()[0]
	branchBytes, _ := branch.MarshalJSON()
	copied, parse := document.ParseObject(branchBytes, document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	branches.Items()[0] = copied
	if e := a.validateInternalModels(ds); e == nil {
		t.Fatal("samebytes substituted originalbranch accepted")
	}
	branches.Items()[0] = branch
	if e := a.validateInternalModels(ds); e != nil {
		t.Fatal(e)
	}
	cost := len(use.Refs)
	for _, node := range use.Nodes {
		cost += 1 + len(node.Members) + len(node.Items)
	}
	if a.intersectionCaptures.Work != cost || a.intersectionCaptures.Bytes != len(use.Bytes) {
		t.Fatal("namedsource graph did not charge exactoriginal work", a.intersectionCaptures, cost)
	}
	for _, spare := range []int{0, 1} {
		context := a
		context.intersectionCaptures = &intersectionCaptureBudget{Seen: map[*document.Value]bool{}, Work: 16384 - cost + spare, Bytes: (8 << 20) - len(use.Bytes)}
		view := *target
		_, _, _, e := context.internalDTOIntersection(&view, "/boundary", 0, 1, nil)
		if spare == 0 && (e != nil || context.intersectionCaptures.Work != 16384 || context.intersectionCaptures.Bytes != 8<<20) || spare == 1 && (e == nil || e.Code != "E_MODEL_LIMIT") {
			t.Fatal("namedsourcework boundary reset", spare, e)
		}
	}
}

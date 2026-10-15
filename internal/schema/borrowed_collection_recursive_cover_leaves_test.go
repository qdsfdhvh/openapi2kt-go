package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestBorrowedCollectionRecursiveCoverLeavesActualGraphAuthority(t *testing.T) {
	for _, tc := range []struct {
		body          string
		height, nodes int
	}{
		{`"Value":` + jsonGraphCover("Value", "Value"), 2, 1},
		{`"Value":` + jsonGraphCover("Other", "Other") + `,"Other":` + jsonGraphCover("Value", "Value"), 4, 2},
		{`"Value":` + jsonGraphCover("ToOther", "Other") + `,"ToOther":{"$ref":"#/components/schemas/Other"},"Other":` + jsonGraphCover("ToValue", "Value") + `,"ToValue":{"$ref":"#/components/schemas/Value"}`, 6, 4},
		{`"Value":` + jsonGraphCover("Other", "End") + `,"Other":` + jsonGraphCover("Value", "End") + `,"End":` + jsonGraphCover("End", "End"), 6, 3},
	} {
		root, err := document.ParseObject([]byte(`{"openapi":"3.1.0","components":{"schemas":{`+tc.body+`,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/Value"}}}},"RowsBridge":{"$ref":"#/components/schemas/Hops/properties/rows","nullable":false},"MapBridge":{"$ref":"#/components/schemas/Hops/properties/map"}}}}`), document.MaxDocumentBytes)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.coverchains", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(tc, err)
		}
		c := a.components["Value"]
		var original Declaration
		missing := []Declaration{}
		for _, d := range ds {
			if d.Name() == c.Name {
				original = d
			} else {
				missing = append(missing, d)
			}
		}
		origin := a.recursiveCoverOrigins[original]
		if origin.Height != tc.height || len(origin.Nodes) < tc.nodes || origin.Owner != c || origin.Schema != c.Schema || origin.Pointer != c.Pointer {
			t.Fatal("complete graph source lost", tc, origin)
		}
		use := a.borrowedRecursiveCoverLeaves[c]
		if use == nil || use.Origin.Height != tc.height || len(use.Origin.Nodes) < tc.nodes {
			t.Fatal("borrowed source lost complete SCC graph", tc, use)
		}
		savedHeight := a.recursiveJSONHeights[c.Schema]
		a.recursiveJSONHeights[c.Schema] = 1
		if err := a.validateBorrowedRecursiveCoverLeaves(ds); err == nil {
			t.Fatal("changed graph work cache still authorized")
		}
		a.recursiveJSONHeights[c.Schema] = savedHeight
		if err := a.validateBorrowedRecursiveCoverLeaves(missing); err == nil {
			t.Fatal("missing original cover authorized")
		}
		copied := &ValueAlias{ClassName: c.Name, Type: TypeRef{Kind: JSON}}
		if err := a.validateBorrowedRecursiveCoverLeaves(append(append([]Declaration{}, missing...), copied)); err == nil {
			t.Fatal("same-name alias replacement authorized")
		}
		if err := a.validateBorrowedRecursiveCoverLeaves(append(append([]Declaration{}, ds...), original)); err == nil {
			t.Fatal("duplicate original occurrence authorized")
		}
		a.recursiveCoverOrigins[copied] = origin
		if err := a.validateBorrowedRecursiveCoverLeaves(append(append([]Declaration{}, ds...), copied)); err == nil {
			t.Fatal("multiple original cover authorities")
		}
		delete(a.recursiveCoverOrigins, copied)
		alias := original.(*ValueAlias)
		alias.Type.Serializer = "synthetic.Override"
		if err := a.validateBorrowedRecursiveCoverLeaves(ds); err == nil {
			t.Fatal("changed actual cover codec authorized")
		}
		alias.Type.Serializer = ""
		// Replace the downstream graph owner with same bytes through a new source tree.
		savedRoot := a.root
		freshRoot, err := document.ParseObject(before, document.MaxDocumentBytes)
		if err != nil {
			t.Fatal(err)
		}
		a.root = freshRoot
		if err := a.validateBorrowedRecursiveCoverLeaves(ds); err == nil {
			t.Fatal("same-byte graph source replacements authorized")
		}
		a.root = savedRoot
		// Same-byte replacement of an original nested branch/ref node must not
		// inherit graph source authority merely because root bytes stay equal.
		branches := get(c.Schema, "oneOf").Items()
		savedBranch := branches[4]
		branchBytes, _ := savedBranch.MarshalJSON()
		copiedBranchRoot, parseErr := document.ParseObject(append(append([]byte(`{"value":`), branchBytes...), '}'), document.MaxDocumentBytes)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		branches[4] = get(copiedBranchRoot, "value")
		if err := a.validateBorrowedRecursiveCoverLeaves(ds); err == nil {
			t.Fatal("same-byte original branch/ref replacement authorized")
		}
		branches[4] = savedBranch
		// A changed terminal/descendant edge must not be hidden by old height caches.
		for node := range origin.Nodes {
			if !has(node, "oneOf") {
				continue
			}
			branches := get(node, "oneOf").Items()
			saved := branches[1]
			bad, err := document.ParseObject([]byte(`{"value":{"type":"string","minLength":1}}`), document.MaxDocumentBytes)
			if err != nil {
				t.Fatal(err)
			}
			branches[1] = get(bad, "value")
			if err := a.validateBorrowedRecursiveCoverLeaves(ds); err == nil {
				t.Fatal("changed descendant grammar hidden by cached height")
			}
			branches[1] = saved
		}
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("original graph mutated")
		}
	}
}

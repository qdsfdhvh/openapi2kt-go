package schema

import (
	"testing"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

func TestNullableInternalUnionMemberOriginalOwner(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		nullableInternalUnionMemberOriginalOwner(t, `{"properties":{"target":{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}}}`, "#/components/schemas/Owner/properties/target")
	})
	t.Run("composed", func(t *testing.T) {
		nullableInternalUnionMemberOriginalOwner(t, `{"allOf":[{"properties":{"target":{"nullable":true,"allOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}]}}}]}`, "#/components/schemas/Owner/allOf/0/properties/target")
	})
}

func nullableInternalUnionMemberOriginalOwner(t *testing.T, ownerSchema, reference string) {
	t.Helper()
	root := loadComponentDocument(t, `{"Owner":`+ownerSchema+`,"Alias":{"$ref":"`+reference+`"},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]}}`)
	a := objectDefaultTestAnalyzer(false)
	a.root, a.components = root, map[string]*component{}
	a.options.PackageName, a.options.Suffix = "org.example.members", "Dto"
	declarations, err := a.analyze(root)
	if err != nil {
		for _, use := range a.borrowedDTOLeaves {
			t.Log("source", use.Pointer, use.Source, "owner", use.Owner.Wire, "name", use.Name, "physical", a.physicalDTOName(use.Schema))
		}
		t.Fatal(err)
	}
	validate := func(ds []Declaration) *diagnostic.Diagnostic {
		if err := a.validateBorrowedDTOLeaves(ds); err != nil {
			return err
		}
		return a.validateBorrowedAllOfLeaves(ds)
	}
	if err := validate(declarations); err != nil {
		t.Fatal(err)
	}
	target, failure := a.lookupSchemaReference("#/components/schemas/Alias", "/late")
	originalTarget, originalFailure := a.lookupSchemaReference(reference, "/original")
	if failure != nil || originalFailure != nil || target.SourceCatalog != a.components["Owner"] || target.Schema != originalTarget.Schema || target.Pointer != originalTarget.Pointer || target.DirectLexical != originalTarget.DirectLexical || target.ComposedLexical != originalTarget.ComposedLexical {
		t.Fatal("alias lost original lexical source", target, failure)
	}
	var owner *Model
	var rest []Declaration
	for _, d := range declarations {
		if d.Name() == "OwnerDto" {
			owner = d.(*Model)
		} else {
			rest = append(rest, d)
		}
	}
	if owner == nil {
		t.Fatal("missing physical owner")
	}
	clone := *owner
	clone.Nested = []Declaration{&Model{ClassName: owner.Nested[0].Name(), Properties: owner.Nested[0].(*Model).Properties}}
	for _, replaced := range [][]Declaration{rest, append(append([]Declaration{}, rest...), &clone), append(append([]Declaration{}, declarations...), owner)} {
		if validate(replaced) == nil {
			t.Fatal("borrowed variant authorized missing, replaced or repeated original DTO")
		}
	}
	alias := a.components["Alias"]
	original := alias.Schema
	copied, e := document.Load([]byte(`{"$ref":"`+reference+`"}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	alias.Schema = copied
	if validate(declarations) == nil {
		t.Fatal("same-byte alias replacement authorized")
	}
	alias.Schema = original
	if err := validate(declarations); err != nil {
		t.Fatal(err)
	}
}

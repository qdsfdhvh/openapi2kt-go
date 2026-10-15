package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNativeEnumNullMemberInternalRootRetainsNullNodeAndCachedWork(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bridge":{"$ref":"#/components/schemas/Source/properties/code"},"Alias":{"$ref":"#/components/schemas/Bridge"},"Source":{"properties":{"code":{"type":["integer","null"],"enum":[null,1,2,3,4],"minimum":2}}}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.enums", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	origin := a.components["Bridge"].InternalModelTarget.BorrowedScalar
	if origin == nil || origin.Owner != a.components["Source"] || origin.Vocabulary != get(origin.Schema, "enum") || len(origin.Members) != 5 || origin.Checks.Minimum.Literal != "2" {
		t.Fatal(origin)
	}
	target, err := a.lookupSchemaReference("#/components/schemas/Alias", "/use")
	if err != nil || target.Steps != 3 {
		t.Fatal(target, err)
	}
	a.aliasPlanning = true
	no := false
	ty, err := a.referenceType(target, &no, 45, "/depth48")
	if err != nil || ty.Kind != Scalar || ty.Validation != origin.Checks || ty.SchemaSteps != 3 {
		t.Fatal(ty, err)
	}
	if _, err := a.referenceType(target, &no, 46, "/depth49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal(err)
	}
	a.aliasPlanning = false
	if err := a.validateBorrowedRootBindings(ds); err != nil {
		t.Fatal(err)
	}
	copied, e := document.Load([]byte(`{"value":null}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	old := origin.Vocabulary.Items()[0]
	origin.Vocabulary.Items()[0] = get(copied, "value")
	if err := a.validateBorrowedRootBindings(ds); err == nil {
		t.Fatal("same-byte replacement vocabulary authorized")
	}
	origin.Vocabulary.Items()[0] = old
	value := origin.Checks.Enum[0]
	origin.Checks.Enum[0] = "7"
	if err := a.validateBorrowedRootBindings(ds); err == nil {
		t.Fatal("mutated cached vocabulary authorized")
	}
	origin.Checks.Enum[0] = value
	if err := a.validateBorrowedRootBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(input) {
		t.Fatal("source rewritten")
	}
}

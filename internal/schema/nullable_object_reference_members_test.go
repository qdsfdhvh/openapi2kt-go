package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestNullableObjectReferenceMembersRequireActualSourceAndAliasOwners(t *testing.T) {
	input, err := os.ReadFile("../../tests/fixtures/http/nullable-object-reference-members.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(input, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.references", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	catalog := get(get(root, "components"), "schemas")
	original := get(catalog, "OriginalProduct")
	view, e := a.readUnion(get(catalog, "Product"), "/components/schemas/Product")
	if e != nil {
		t.Fatal(e)
	}
	if e = a.deriveObjectReferenceNullability(view, 1); e != nil || !view.Nullable || view.HasNull {
		t.Fatal("derived reference intent", view, e)
	}
	target, ok, e := a.nullableObjectReferenceMember(view.Branches[0], 1)
	if e != nil || !ok || target.Schema != original || target.Catalog != a.components["OriginalProduct"] || target.Pointer != "/components/schemas/OriginalProduct" {
		t.Fatal("reference source authority replaced", target, ok, e)
	}
	if len(a.components) != len(catalog.Members()) {
		t.Fatal("fake component introduced")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original nullable member nodes rewritten")
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d.Name() != "OriginalProductDto" {
			missing = append(missing, d)
		}
	}
	if e := a.validateBorrowedDTOLeaves(missing); e == nil || e.Code != "E_REF_UNSUPPORTED" {
		t.Fatal("borrowed variant authorized missing original source DTO", e)
	}
}

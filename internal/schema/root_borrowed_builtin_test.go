package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestRootBorrowedBuiltinSourceAuthorityAndRootDeclarationIdentity(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":{"type":["string","null"],"minLength":2},"hop":{"$ref":"#/components/schemas/Other/properties/leaf","nullable":true}}},"Other":{"properties":{"leaf":{"type":"integer","format":"int64","minimum":2}}},"ABridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"False":{"$ref":"#/components/schemas/ABridge","nullable":false},"Mixed":{"$ref":"#/components/schemas/Source/properties/hop","nullable":false},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/False"}},"required":["value"]}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowed", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog := get(get(root, "components"), "schemas")
	bridge, source := a.components["ABridge"], a.components["Source"]
	binding := bridge.InternalModelTarget
	origin := binding.BorrowedScalar
	if origin == nil || origin.Owner != source || origin.Schema != get(get(catalog, "Source"), "properties").Members()[0].Value || binding.Catalog != nil || binding.CanonicalBuiltin != nil || binding.ModelName != "" || binding.Deferred != nil || len(a.components) != len(catalog.Members()) {
		t.Fatal("fabricated canonical component/declaration or lost leaf", binding)
	}
	if origin.Pointer != "/components/schemas/Source/properties/leaf" || origin.Reference != "#/components/schemas/Source/properties/leaf" || origin.Type.Kind != Nullable || bare(origin.Type).Kind != Scalar || origin.Checks == nil || !a.emissionHints[origin.Checks] || a.validationSources[origin.Checks] != origin.Pointer {
		t.Fatal("borrowed checks/source provenance lost", origin)
	}
	mixed := a.components["Mixed"].InternalModelTarget.BorrowedScalar
	if mixed.Owner != a.components["Other"] || mixed.Pointer != "/components/schemas/Other/properties/leaf" || a.componentType(a.components["Mixed"]).Kind != Scalar {
		t.Fatal("entry owner replaced terminal owner", mixed)
	}
	cached, err := a.lookupSchemaReference("#/components/schemas/False", "/cache")
	if err != nil || cached.Catalog != a.components["False"] || cached.BorrowedScalar != origin || cached.Steps != 3 || cached.CanonicalBuiltin != nil {
		t.Fatal(cached, err)
	}
	a.aliasPlanning = true
	no := false
	ty, err := a.referenceType(cached, &no, 45, "/depth48")
	if err != nil || ty.Kind != Scalar || ty.SchemaSteps != 3 || ty.Validation != origin.Checks {
		t.Fatal(ty, err)
	}
	if _, err := a.referenceType(cached, &no, 46, "/depth49"); err == nil || err.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("borrowed false cache reset", err)
	}
	a.aliasPlanning = false
	// Historical borrowed uses remain per-use parsed objects, with same semantic
	// checks/hints; they do not gain a fictional canonical component check object.
	direct, err := a.resolveSchema(origin.Reference, "/direct")
	if err != nil || direct.Catalog != nil {
		t.Fatal(direct, err)
	}
	directType, err := a.referenceType(direct, nil, 0, "/direct")
	if err != nil || bare(directType).Validation == origin.Checks || !a.emissionHints[bare(directType).Validation] {
		t.Fatal("historical direct checks identity changed", directType, err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	missing := []Declaration{}
	var actual Declaration
	for _, d := range ds {
		if d.Name() == bridge.Name {
			actual = d
		} else {
			missing = append(missing, d)
		}
	}
	if err := a.validateBorrowedRootBindings(missing); err == nil {
		t.Fatal("missing actual root export authorized")
	}
	if err := a.validateBorrowedRootBindings(append(append([]Declaration{}, ds...), actual)); err == nil {
		t.Fatal("repeated original root declaration authorized")
	}
	copy := &ValueAlias{ClassName: bridge.Name, Type: a.componentType(bridge)}
	if err := a.validateBorrowedRootBindings(append(append([]Declaration{}, missing...), copy)); err == nil {
		t.Fatal("same-name root replacement authorized")
	}
	a.borrowedRoots[copy] = a.borrowedRoots[actual]
	if err := a.validateBorrowedRootBindings(append(append([]Declaration{}, ds...), copy)); err == nil {
		t.Fatal("multiple root declarations authorized")
	}
	delete(a.borrowedRoots, copy)
	originalOwnerSchema := source.Schema
	copyOwner, err := document.Load([]byte(`{"properties":{"leaf":{"type":["string","null"],"minLength":2},"hop":{"$ref":"#/components/schemas/Other/properties/leaf","nullable":true}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	source.Schema = copyOwner
	if err := a.validateBorrowedRootBindings(ds); err == nil {
		t.Fatal("same-byte replacement owner/leaf authorized")
	}
	source.Schema = originalOwnerSchema
	originalType := origin.Type
	bad := *origin.Type.Element
	copyChecks := *origin.Checks
	bad.Validation = &copyChecks
	origin.Type.Element = &bad
	if err := a.validateBorrowedRootBindings(ds); err == nil {
		t.Fatal("equivalent replacement parsed checks authorized")
	}
	origin.Type = originalType
	oldMin := *origin.Checks.MinLength
	*origin.Checks.MinLength = 3
	if err := a.validateBorrowedRootBindings(ds); err == nil {
		t.Fatal("mutated cached checks authorized")
	}
	*origin.Checks.MinLength = oldMin
	if err := a.validateBorrowedRootBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original borrowed source mutated")
	}
}

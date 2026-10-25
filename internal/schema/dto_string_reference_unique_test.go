package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestDTOStringReferenceOwnDefaultAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Value":{"type":"string","enum":["ready","done"]},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":"ready"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.refs", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal("analyze", e, *e.Pointer)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal("default plan", e, *e.Pointer)
	}
	src := a.dtoUniqueSources[a.components["Item"].Schema]
	if len(src.ReferenceFields) != 1 {
		t.Fatal("missing actual reference descriptor")
	}
	if e := a.validateDTOUniqueUses(&Plan{Models: ds}); e != nil {
		t.Fatal("barrier", e, *e.Pointer)
	}
}

func TestDTOStringReferenceInternalDefaultAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Value":{"$ref":"#/components/schemas/Hops/properties/leaf"},"Hops":{"properties":{"leaf":{"enum":["ready","done"]}}},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":"ready"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.refs", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal("analyze", e, *e.Pointer)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal("defaults", e, *e.Pointer)
	}
	src := a.dtoUniqueSources[a.components["Item"].Schema]
	if len(src.ReferenceFields) != 1 {
		t.Fatal("missing internal source descriptor")
	}
	if e := a.validateDTOUniqueUses(&Plan{Models: ds}); e != nil {
		t.Fatal("barrier", e, *e.Pointer)
	}
}

func TestDTOStringReferenceActualNestedEnumAuthority(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-string-reference-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.refs", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal(e)
	}
	plan := &Plan{Models: ds}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
	source := get(get(a.components["BorrowOwner"].Schema, "properties"), "mode/~")
	enum := a.dtoReferenceEnums[source]
	if enum == nil {
		t.Fatal("missing original nested enum")
	}
	reject := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign enum reference")
		}
	}
	oldName := enum.ClassName
	enum.ClassName = "Foreign"
	reject("changed physical classifier")
	enum.ClassName = oldName
	members := append([]EnumMember{}, enum.Members...)
	enum.Members[0].Wire = "foreign"
	reject("changed vocabulary wire")
	copy(enum.Members, members)
	enum.Members[0].Name = "FOREIGN"
	reject("changed member name")
	copy(enum.Members, members)
	origin := a.enumOrigins[enum]
	delete(a.enumOrigins, enum)
	reject("missing original enum origin")
	a.enumOrigins[enum] = origin
	alias := componentDeclaration(ds, "RefStateAliasDto").(*ValueAlias)
	oldType := alias.Type
	if alias.Type.Kind == Nullable {
		copy := *alias.Type.Element
		copy.Name = "Foreign"
		alias.Type.Element = &copy
	} else {
		alias.Type.Name = "Foreign"
	}
	reject("changed actual alias target type")
	alias.Type = oldType
	owner := componentDeclaration(ds, "BorrowOwnerDto").(*Model)
	nested := append([]Declaration{}, owner.Nested...)
	for i, d := range owner.Nested {
		if d == enum {
			copy := *enum
			a.enumOrigins[&copy] = origin
			owner.Nested[i] = &copy
			reject("samebyte copied nested enum")
			delete(a.enumOrigins, &copy)
			owner.Nested[i] = enum
		}
	}
	owner.Nested = append(owner.Nested, enum)
	reject("repeated actual enum")
	owner.Nested = nested
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored actual enum", e)
	}
}

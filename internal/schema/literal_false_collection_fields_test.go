package schema

import (
	"openapi-kt/internal/document"
	"slices"
	"testing"
)

func TestLiteralFalseCollectionActualSourceAndCodecAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"blocked":false,"rows":{"type":"array","maxItems":1,"items":{"additionalProperties":{"type":"array","nullable":true,"items":false}}}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.literalcollection", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var model *Model
	for m := range a.literalFalseFieldOrigins {
		if m.ClassName == "HolderDto" {
			model = m
		}
	}
	if model == nil {
		t.Fatal("missing actual owner")
	}
	origin := a.literalFalseFieldOrigins[model]
	index := -1
	for i, f := range origin.Fields {
		if f.Plan.Collection.Leaf != nil {
			index = i
		}
	}
	if index < 0 || len(origin.Fields) != 2 || len(model.Nested) != 1 || !origin.Helper.AbsentSerializer {
		t.Fatal("literal/collection fields lost shared actual helper")
	}
	snapshot := origin.Fields[index]
	p := snapshot.Property
	if p.OmitDefault || !p.DefaultNull || snapshot.Plan.Collection.Height != 3 {
		t.Fatal("container absence became strict false absence")
	}
	saved := p.OmitDefault
	p.OmitDefault = true
	if a.validateInternalModels(ds) == nil {
		t.Error("container silently acquired strict-absence policy")
	}
	p.OmitDefault = saved
	for _, mutate := range []func(*literalFalseCollectionSource){func(s *literalFalseCollectionSource) { s.Height++ }, func(s *literalFalseCollectionSource) { s.Pointer += "/fake" }, func(s *literalFalseCollectionSource) { s.Schema = s.Leaf }} {
		changed := origin
		changed.Fields = slices.Clone(origin.Fields)
		mutate(&changed.Fields[index].Plan.Collection)
		a.literalFalseFieldOrigins[model] = changed
		if a.validateInternalModels(ds) == nil {
			t.Error("changed original collection role authorized")
		}
		a.literalFalseFieldOrigins[model] = origin
	}
	savedType := p.Type
	p.Type = TypeRef{Kind: Nullable, Element: &TypeRef{Kind: JSON}}
	if a.validateInternalModels(ds) == nil {
		t.Error("raw substituted collection authorized")
	}
	p.Type = savedType
	savedElement := p.Type.Element
	p.Type.Element = &p.Type
	if a.validateInternalModels(ds) == nil {
		t.Error("cyclic collection type authorized")
	}
	p.Type.Element = savedElement
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	copiedRows := get(get(get(get(get(copied, "components"), "schemas"), "Holder"), "properties"), "rows")
	inner := get(get(snapshot.Plan.Field.Schema, "items"), "additionalProperties")
	for i, m := range inner.Members() {
		if m.Name == "items" {
			inner.Members()[i].Value = get(get(get(copiedRows, "items"), "additionalProperties"), "items")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-text original false leaf replaced")
			}
			inner.Members()[i] = m
		}
	}
	delete(a.literalFalseFieldOrigins, model)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing last collection source proof authorized")
	}
	a.literalFalseFieldOrigins[model] = origin
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}

package schema

import (
	"openapi-kt/internal/document"
	"slices"
	"testing"
)

func TestLiteralFalseCollectionDefaultCanonicalAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"rows":{"additionalProperties":{"additionalProperties":{"type":"array","nullable":true,"items":false}},"default":{"description":{"description":null}}}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.literaldefault", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var model *Model
	for m := range a.literalFalseFieldOrigins {
		model = m
	}
	if model == nil {
		t.Fatal("missing actual owner")
	}
	origin := a.literalFalseFieldOrigins[model]
	if len(origin.Fields) != 1 || !origin.Fields[0].Plan.Collection.OwnDefault {
		t.Fatal("missing actual field role")
	}
	changed := origin
	changed.Fields = slices.Clone(origin.Fields)
	changed.Fields[0].Plan.Encoded = "{}"
	a.literalFalseFieldOrigins[model] = changed
	if a.validateInternalModels(ds) == nil {
		t.Error("changed canonical literal proof authorized")
	}
	a.literalFalseFieldOrigins[model] = origin
	changed = origin
	changed.Fields = slices.Clone(origin.Fields)
	changed.Fields[0].Literal = nil
	a.literalFalseFieldOrigins[model] = changed
	if a.validateInternalModels(ds) == nil {
		t.Error("missing last ownliteral proof authorized")
	}
	a.literalFalseFieldOrigins[model] = origin
	p := origin.Fields[0].Property
	savedInitializer := p.Default
	p.Default = ""
	if a.validateInternalModels(ds) == nil {
		t.Error("removed actual initializer authorized")
	}
	p.Default = savedInitializer
	changed = origin
	changed.Fields = slices.Clone(origin.Fields)
	changed.Fields[0].Plan.Collection.OwnDefault = false
	a.literalFalseFieldOrigins[model] = changed
	if a.validateInternalModels(ds) == nil {
		t.Error("ownrole sourceflag removed authorized")
	}
	a.literalFalseFieldOrigins[model] = origin
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	copyField := get(get(get(get(get(copyRoot, "components"), "schemas"), "Holder"), "properties"), "rows")
	value := get(get(origin.Fields[0].Plan.Field.Schema, "default"), "description")
	for i, m := range value.Members() {
		if m.Name == "description" {
			value.Members()[i].Value = get(get(get(copyField, "default"), "description"), "description")
			if a.validateInternalModels(ds) == nil {
				t.Error("metadata-named data key hid original null replacement")
			}
			value.Members()[i] = m
		}
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}

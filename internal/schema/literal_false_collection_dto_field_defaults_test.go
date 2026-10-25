package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestLiteralFalseCollectionDTODeferredTargetAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Target":{"properties":{"rows":{"type":"array","items":false},"mapping":{"additionalProperties":{"type":"array","nullable":true,"items":false}},"reference":{"$ref":"#/components/schemas/Blocked"}},"default":{}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"mapping":{"description":null}}}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	defaults := []referencedFieldDefault{}
	a := analyzer{referencedDefaults: &defaults, root: root, options: Options{PackageName: "org.example.literalfalse", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	if err := a.validateFalseDTOFieldDefaults(ds); err != nil {
		t.Fatal(err)
	}
	var target *Model
	for m := range a.literalFalseFieldOrigins {
		if m.ClassName == "TargetDto" {
			target = m
		}
	}
	if target == nil {
		t.Fatal("missing actual target")
	}
	source := a.literalFalseFieldOrigins[target]
	saved := source.Helper.AbsentSerializer
	source.Helper.AbsentSerializer = !saved
	if a.validateFalseDTOFieldDefaults(ds) == nil {
		t.Error("deferred barrier authorized removed actual strict target codec")
	}
	source.Helper.AbsentSerializer = saved
	savedOmit := target.Properties[0].OmitDefault
	target.Properties[0].OmitDefault = !savedOmit
	if a.validateFalseDTOFieldDefaults(ds) == nil {
		t.Error("deferred barrier authorized target absence-policy change")
	}
	target.Properties[0].OmitDefault = savedOmit
	var fieldKey falseDTOFieldDefaultKey
	for key := range a.falseDTOFieldDefaults {
		fieldKey = key
	}
	fieldProof := a.falseDTOFieldDefaults[fieldKey]
	for _, literal := range []bool{false, true} {
		var saved falseCollectionFactoryOrigin
		if literal {
			saved = a.literalFalseObjectFactories[target]
			delete(a.literalFalseObjectFactories, target)
		} else {
			saved = a.falseObjectFactories[target]
			delete(a.falseObjectFactories, target)
		}
		if a.falseDTOFieldScope(fieldProof.Scope.Plan) != nil || a.validateFalseDTOFieldDefaults(ds) == nil {
			t.Error("one target getter role authorized another role's missing own factory", literal)
		}
		if literal {
			a.literalFalseObjectFactories[target] = saved
		} else {
			a.falseObjectFactories[target] = saved
		}
	}
	delete(a.falseDTOFieldDefaults, fieldKey)
	if a.validateFalseDTOFieldDefaults(ds) == nil {
		t.Error("missing last literal target consumer initializer proof authorized")
	}
	a.falseDTOFieldDefaults[fieldKey] = fieldProof
	delete(a.literalFalseFieldOrigins, target)
	if a.validateFalseDTOFieldDefaults(ds) == nil {
		t.Error("retained ref role bypassed missing actual literal getter proof")
	}
	a.literalFalseFieldOrigins[target] = source
	copyRoot, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	copiedDefault := get(get(get(get(get(get(copyRoot, "components"), "schemas"), "Consumer"), "properties"), "target"), "default")
	mapping := get(fieldProof.Literal.Value, "mapping")
	for i, member := range mapping.Members() {
		mapping.Members()[i].Value = get(get(copiedDefault, "mapping"), member.Name)
		if a.validateFalseDTOFieldDefaults(ds) == nil {
			t.Error("same-text consuming metadata-named literal data substituted")
		}
		mapping.Members()[i] = member
	}
	if err := a.validateFalseDTOFieldDefaults(ds); err != nil {
		t.Fatal(err)
	}
}

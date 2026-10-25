package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestFalseDTOIndependentDefaultsActualFactoryAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Blocked":false,"Target":{"properties":{"value":{"$ref":"#/components/schemas/Blocked","nullable":true}},"default":{"value":null}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}}}}`)
	root, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	defaults := []referencedFieldDefault{}
	a := analyzer{referencedDefaults: &defaults, root: root, options: Options{PackageName: "org.example.falsedtodefaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var consumer *Model
	for _, d := range ds {
		if value, ok := d.(*Model); ok && value.ClassName == "ConsumerDto" {
			consumer = value
		}
	}
	if consumer == nil || len(consumer.Properties) != 1 || consumer.Properties[0].Default == "" {
		t.Fatal("missing actual final field initializer")
	}
	var factoryModel *Model
	for _, d := range ds {
		if m, ok := d.(*Model); ok && m.ClassName == "TargetDto" {
			factoryModel = m
		}
	}
	if factoryModel == nil || factoryModel.ComponentDefault == "" {
		t.Fatal("missing independentlyvalidatedtargetfactory")
	}
	factory := factoryModel.ComponentDefault
	factoryModel.ComponentDefault = ""
	if a.validateInternalModels(ds) == nil {
		t.Error("missing actual target factory authorized fielddefault")
	}
	factoryModel.ComponentDefault = "{}"
	if a.validateInternalModels(ds) == nil {
		t.Error("changed actual target factory authorized fielddefault")
	}
	factoryModel.ComponentDefault = factory
	saved := consumer.Properties[0].Default
	consumer.Properties[0].Default = ""
	if a.validateInternalModels(ds) == nil {
		t.Error("removed final consuming initializer authorized")
	}
	consumer.Properties[0].Default = saved
	var key falseDTOFieldDefaultKey
	for candidate := range a.falseDTOFieldDefaults {
		key = candidate
	}
	proof := a.falseDTOFieldDefaults[key]
	delete(a.falseDTOFieldDefaults, key)
	if a.validateInternalModels(ds) == nil {
		t.Error("missing actual field default proof authorized")
	}
	a.falseDTOFieldDefaults[key] = proof
	savedType := consumer.Properties[0].Type
	consumer.Properties[0].Type = TypeRef{Kind: Nullable, Element: &TypeRef{Kind: JSON}}
	if a.validateInternalModels(ds) == nil {
		t.Error("raw replacement consumer codec authorized")
	}
	consumer.Properties[0].Type = savedType
	copied, e := document.ParseObject(input, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	schema := get(get(a.modelOrigins[consumer], "properties"), "target")
	copySchema := get(get(get(get(get(copied, "components"), "schemas"), "Consumer"), "properties"), "target")
	for i, m := range schema.Members() {
		if m.Name == "default" {
			schema.Members()[i].Value = get(copySchema, "default")
			if a.validateInternalModels(ds) == nil {
				t.Error("same-byte original consumer default replaced")
			}
			schema.Members()[i] = m
		}
	}
	without := []Declaration{}
	for _, d := range ds {
		if d != consumer {
			without = append(without, d)
		}
	}
	if a.validateInternalModels(without) == nil {
		t.Error("missing actual consumer authorized")
	}
	copiedModel := *consumer
	if a.validateInternalModels(append(without, &copiedModel)) == nil {
		t.Error("same-byte copied consumer authorized")
	}

	for _, mutate := range []func(){

		func() { consumer.Properties[0].DefaultNull = !consumer.Properties[0].DefaultNull },
		func() { consumer.ClassName += "Changed" },
		func() { proof.Scope.Target.Properties[0].Default = "null" },
	} {
		savedConsumer := *consumer
		savedProps := append([]Property(nil), consumer.Properties...)
		target := proof.Scope.Target
		savedTarget := *target
		savedTargetProps := append([]Property(nil), target.Properties...)
		mutate()
		if a.validateInternalModels(ds) == nil {
			t.Error("actual declaration mutation authorized")
		}
		*consumer = savedConsumer
		copy(consumer.Properties, savedProps)
		*target = savedTarget
		copy(target.Properties, savedTargetProps)
	}
	target := proof.Scope.Target
	withoutTarget := []Declaration{}
	for _, d := range ds {
		if d != target {
			withoutTarget = append(withoutTarget, d)
		}
	}
	if a.validateInternalModels(withoutTarget) == nil {
		t.Error("missing actual target authorized")
	}
	copiedTarget := *target
	if a.validateInternalModels(append(withoutTarget, &copiedTarget)) == nil {
		t.Error("copied target authorized")
	}

	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}

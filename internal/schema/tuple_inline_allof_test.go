package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleInlineAllOfSourceFactsAndRetainedConstructor(t *testing.T) {
	item := `{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"n":{"type":"integer","default":7}}}]}`
	root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[`+item+`]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[`+item+`],"default":[]}}}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.inlineallof", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	holder := componentModel(ds, "HolderDto")
	model := holder.Nested[0].(*Model)
	originalName := model.ClassName
	model.ClassName = "Other"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("renamed constructor retained old element authority")
	}
	model.ClassName = originalName
	copyModel := *model
	holder.Nested = []Declaration{&copyModel}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied constructor acquired original source authority")
	}
	holder.Nested = []Declaration{model, model}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("duplicate constructor retained unique authority")
	}
	holder.Nested = []Declaration{model}
	properties := model.Properties
	model.Properties = nil
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("empty constructor retained tuple authority")
	}
	model.Properties = properties
	outer := holder.Properties[0].Type.Element
	element := outer.Element
	copyType := *element
	outer.Element = &copyType
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied element address acquired retained tuple authority")
	}
	outer.Element = element
	facts := a.tupleObjects.Proof.Constructors
	a.tupleObjects.Proof.Constructors = nil
	if a.validateInternalModels(ds) == nil || a.validateTuplePlan(plan) == nil {
		t.Fatal("missing isolated constructor proof was accepted")
	}
	a.tupleObjects.Proof.Constructors = facts
	for source, fact := range facts {
		mutated := fact
		mutated.Height++
		facts[source] = mutated
		if a.validateInternalModels(ds) == nil {
			t.Fatal("changed cached original source height retained constructor proof")
		}
		facts[source] = fact
		break
	}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("inline allOf source analysis rewrote document")
	}
}

func TestTupleInlineAllOfDiscardedAnalysisIsIsolated(t *testing.T) {
	root := loadComponentDocument(t, `{"Choice":{"enum":["ready"]},"Base":{"properties":{"n":{"type":"integer","default":7}}},"Rows":{"type":"array","prefixItems":[{"allOf":[{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"ready"}}},{"$ref":"#/components/schemas/Base"}]}],"items":true}}`)
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.inlineallof", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 3 || len(pending) != 0 || len(a.modelOrigins) != 1 || len(a.components) != 3 {
		t.Fatal("discarded allOf source leaked registrations", len(ds), len(pending), len(a.modelOrigins), len(a.components))
	}
	for _, d := range ds {
		if strings.Contains(d.Name(), "TupleSource") {
			t.Fatal("isolated constructor escaped")
		}
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	base := a.components["Base"]
	basePointer := base.Pointer
	base.Pointer = "/components/schemas/Other"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed discarded shape reference catalog kept tuple permission")
	}
	base.Pointer = basePointer
	member := get(a.components["Rows"].Schema, "prefixItems").Items()[0]
	branch := get(member, "allOf").Items()[0]
	bytes, _ := branch.MarshalJSON()
	copyBranch, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	get(member, "allOf").Items()[0] = copyBranch
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte discarded allOf branch copy gained proof")
	}
}

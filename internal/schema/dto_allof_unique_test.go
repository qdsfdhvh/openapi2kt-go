package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func dtoAllOfUniqueAnalysis(t *testing.T, validate bool) (*analyzer, *Plan, *document.Value, *dtoUniqueSource) {
	t.Helper()
	root := loadComponentDocument(t, `{"Base":{"properties":{"id":{"type":"string"},"version":{"type":"integer","minimum":0,"default":0}},"required":["id"]},"Item":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"version":{"type":"integer","maximum":2,"default":1}}}]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`)
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.allofkeys", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal(e)
	}
	plan := &Plan{Models: ds}
	if validate {
		if e := a.validateDTOUniqueUses(plan); e != nil {
			t.Fatal(e)
		}
	}
	src := a.dtoUniqueSources[a.components["Item"].Schema]
	if src == nil || !src.Composed || len(src.Fields) != 2 || len(src.ShapeRefs) == 0 {
		t.Fatal("missing original allOf source")
	}
	return a, plan, root, src
}

func TestDTOAllOfUniqueActualOriginalConstructorAndShape(t *testing.T) {
	a, plan, root, src := dtoAllOfUniqueAnalysis(t, true)
	before, _ := root.MarshalJSON()
	reject := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign allOf constructor")
		}
	}
	for _, state := range src.ShapeRefs {
		owner := state.Owner
		old := owner.AliasHeight
		owner.AliasHeight++
		reject("changed original referenced source height")
		owner.AliasHeight = old
	}
	f := src.Fields[0]
	src.Fields[0].Pointer += "/foreign"
	reject("changed original winning field Pointer")
	src.Fields[0] = f
	required := src.Required["id"]
	src.Required["id"] = !required
	reject("changed original required field set")
	src.Required["id"] = required
	property := src.Actual.Properties[0]
	src.Actual.Properties[0].Name = "foreign"
	reject("changed actual getter API")
	src.Actual.Properties[0] = property
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored original shape", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST changed")
	}
}

func TestDTOAllOfUniqueFirstBarrierActualConstructor(t *testing.T) {
	a, plan, _, _ := dtoAllOfUniqueAnalysis(t, false)
	for i, d := range plan.Models {
		if d.Name() != "ItemDto" {
			continue
		}
		copy := *d.(*Model)
		plan.Models[i] = &copy
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal("same-byte allOf constructor replacement authorized before first barrier")
		}
		plan.Models[i] = d
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
}

func TestDTOAllOfUniqueOriginalAggregateCheckValues(t *testing.T) {
	a, plan, _, src := dtoAllOfUniqueAnalysis(t, true)
	for _, p := range src.Actual.Properties {
		if p.WireName != "version" {
			continue
		}
		if p.Validation == nil || p.Validation.Minimum == nil {
			t.Fatal("missing retained aggregate")
		}
		old := p.Validation.Minimum.Literal
		p.Validation.Minimum.Literal = "2"
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal("changed actual original aggregate check authorized")
		}
		p.Validation.Minimum.Literal = old
		return
	}
	t.Fatal("missing actual version getter")
}

func TestDTOAllOfUniqueFirstBarrierAggregateIdentityAndRestoration(t *testing.T) {
	a, plan, _, _ := dtoAllOfUniqueAnalysis(t, false)
	actual := componentDeclaration(plan.Models, "ItemDto").(*Model)
	for i := range actual.Properties {
		p := &actual.Properties[i]
		if p.WireName != "version" {
			continue
		}
		original := p.Validation
		copy := *original
		p.Validation = &copy
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal("same-byte aggregate check replacement authorized before first barrier")
		}
		p.Validation = original
		if e := a.validateDTOUniqueUses(plan); e != nil {
			t.Fatal("restored original aggregate", e)
		}
		return
	}
	t.Fatal("missing aggregate property")
}

func TestDTOAllOfUniqueOriginalActualInitializer(t *testing.T) {
	a, plan, _, src := dtoAllOfUniqueAnalysis(t, true)
	for i := range src.Actual.Properties {
		p := &src.Actual.Properties[i]
		if p.WireName != "version" {
			continue
		}
		old := p.Default
		p.Default = "0"
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal("changed actual aggregate constructor initializer authorized")
		}
		p.Default = old
		if e := a.validateDTOUniqueUses(plan); e != nil {
			t.Fatal("restored actual initializer", e)
		}
		return
	}
	t.Fatal("missing actual version getter")
}

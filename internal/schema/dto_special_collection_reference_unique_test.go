package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestDTOSpecialCollectionReferenceOriginalFormatAndPhysicalTypes(t *testing.T) {
	a, plan, root, src := dtoSpecialCollectionReferenceAnalysis(t, true)
	before, _ := root.MarshalJSON()
	reject := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign mapped collection getter")
		}
	}
	for _, name := range []string{"SpecialUuids", "SpecialInstants", "SpecialDays"} {
		component := a.components[name]
		edge := "items"
		if name == "SpecialInstants" {
			edge = "additionalProperties"
		}
		leaf := get(component.Schema, edge)
		for i, member := range leaf.Members() {
			if member.Name != "format" {
				continue
			}
			wrapper, e := document.ParseObject([]byte(`{"format":"uuid"}`), document.MaxDocumentBytes)
			if e != nil {
				t.Fatal(e)
			}
			if name == "SpecialUuids" {
				wrapper, e = document.ParseObject([]byte(`{"format":"date"}`), document.MaxDocumentBytes)
				if e != nil {
					t.Fatal(e)
				}
			}
			leaf.Members()[i].Value = get(wrapper, "format")
			reject("changed original format marker")
			leaf.Members()[i].Value = member.Value
			raw, _ := member.Value.MarshalJSON()
			wrapper, e = document.ParseObject(append(append([]byte(`{"format":`), raw...), '}'), document.MaxDocumentBytes)
			if e != nil {
				t.Fatal(e)
			}
			leaf.Members()[i].Value = get(wrapper, "format")
			reject("same-byte replaced format marker node")
			leaf.Members()[i].Value = member.Value
		}
		alias := componentDeclaration(plan.Models, component.Name).(*ValueAlias)
		origin, ok := a.namedCollectionOrigins[alias]
		if !ok {
			t.Fatal("missing physical mapped collection origin")
		}
		delete(a.namedCollectionOrigins, alias)
		reject("lost original collection origin")
		a.namedCollectionOrigins[alias] = origin
		for _, snapshot := range origin.Types {
			if snapshot.Value.Kind != Scalar {
				continue
			}
			old := *snapshot.Node
			snapshot.Node.Name = "kotlin.String"
			reject("mapped type replaced by text projection")
			*snapshot.Node = old
		}
	}
	for i, p := range src.Actual.Properties {
		if p.WireName != "specialInstant" {
			continue
		}
		src.Actual.Properties[i].Default = "kotlin.collections.emptyMap()"
		reject("replaced original mapped initializer")
		src.Actual.Properties[i] = p
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored mapped source rejected", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original source AST changed")
	}
}

func dtoSpecialCollectionReferenceAnalysis(t *testing.T, validate bool) (*analyzer, *Plan, *document.Value, *dtoUniqueSource) {
	t.Helper()
	data, err := os.ReadFile("../../tests/fixtures/http/dto-special-collection-reference-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.dtospecialcollectionreferenceunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || len(src.ReferenceFields) != 25 {
		t.Fatal("missing collection reference source")
	}
	return a, plan, root, src
}

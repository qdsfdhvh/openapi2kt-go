package schema

import (
	"openapi-kt/internal/document"
	"os"
	"testing"
)

func TestDTONativeCollectionReferenceOriginalVocabularyAndChecks(t *testing.T) {
	a, plan, root, src := dtoNativeCollectionReferenceAnalysis(t, true)
	before, _ := root.MarshalJSON()
	reject := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign native enum collection getter")
		}
	}
	for _, name := range []string{"NativeRefInts", "NativeRefLongs", "NativeRefFloats", "NativeRefDoubles", "NativeRefFlags"} {
		owner := a.components[name]
		edge := "items"
		if name == "NativeRefDoubles" || name == "NativeRefFlags" {
			edge = "additionalProperties"
		}
		leaf := get(owner.Schema, edge)
		vocabulary := get(leaf, "enum")
		if len(vocabulary.Items()) == 0 {
			t.Fatal("missing original enum vocabulary")
		}
		old := vocabulary.Items()[0]
		raw, _ := old.MarshalJSON()
		wrapper, e := document.ParseObject(append(append([]byte(`{"value":`), raw...), '}'), document.MaxDocumentBytes)
		if e != nil {
			t.Fatal(e)
		}
		vocabulary.Items()[0] = get(wrapper, "value")
		reject("same-byte replaced vocabulary value node")
		vocabulary.Items()[0] = old
		alias := componentDeclaration(plan.Models, owner.Name).(*ValueAlias)
		origin, ok := a.namedCollectionOrigins[alias]
		if !ok {
			t.Fatal("missing native collection alias origin")
		}
		found := false
		for _, snapshot := range origin.Types {
			if snapshot.Value.Validation == nil || len(snapshot.Value.Validation.Enum) == 0 {
				continue
			}
			found = true
			node := snapshot.Node
			checks := node.Validation
			literal := checks.Enum[0]
			checks.Enum[0] = "2"
			reject("changed actual enum membership checks")
			checks.Enum[0] = literal
			copy := *checks
			node.Validation = &copy
			reject("copied same-byte enum check object")
			node.Validation = checks
			node.Validation = nil
			reject("dropped retained enum membership")
			node.Validation = checks
		}
		if !found {
			t.Fatal("native collection source lost actual membership metadata", name)
		}
	}
	for i, p := range src.Actual.Properties {
		if p.WireName != "nativeRefFloats" {
			continue
		}
		src.Actual.Properties[i].Default = "kotlin.collections.emptyList()"
		reject("changed actual native initializer")
		src.Actual.Properties[i] = p
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored native source rejected", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST changed")
	}
}

func TestDTONativeCollectionReferenceFirstBarrierKeepsDeclarationIdentity(t *testing.T) {
	a, plan, _, _ := dtoNativeCollectionReferenceAnalysis(t, false)
	for i, d := range plan.Models {
		if d.Name() != "NativeRefFloatsDto" {
			continue
		}
		copy := *d.(*ValueAlias)
		plan.Models[i] = &copy
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal("same-byte native alias replacement before first barrier authorized")
		}
		plan.Models[i] = d
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
}

func dtoNativeCollectionReferenceAnalysis(t *testing.T, validate bool) (*analyzer, *Plan, *document.Value, *dtoUniqueSource) {
	t.Helper()
	data, err := os.ReadFile("../../tests/fixtures/http/dto-native-collection-reference-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.dtonativecollectionreferenceunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || len(src.ReferenceFields) != 33 {
		t.Fatal("missing collection reference source")
	}
	return a, plan, root, src
}

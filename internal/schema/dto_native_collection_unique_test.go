package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTONativeCollectionUniqueActualGetterTreeAndKeyWork(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-native-collection-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.dtonativecollectionunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	src := a.dtoUniqueSources[a.components["Item"].Schema]
	if src == nil || src.Actual == nil {
		t.Fatal("no actual getter source")
	}
	assertRejected := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign getter tree")
		}
	}
	var changeLeaf func(TypeRef) TypeRef
	changeLeaf = func(ty TypeRef) TypeRef {
		if ty.Element != nil {
			child := changeLeaf(*ty.Element)
			ty.Element = &child
		} else {
			ty.Name = "kotlin.Double"
		}
		return ty
	}
	for i, p := range src.Actual.Properties {

		if p.WireName == "nativeLongs" {
			var withoutChecks func(TypeRef) TypeRef
			withoutChecks = func(ty TypeRef) TypeRef {
				if ty.Element != nil {
					child := withoutChecks(*ty.Element)
					ty.Element = &child
				} else {
					ty.Validation = nil
				}
				return ty
			}
			src.Actual.Properties[i].Type = withoutChecks(p.Type)
			assertRejected("native enum getter checks erased")
			src.Actual.Properties[i] = p
		}

		if p.WireName == "nulls" || p.WireName == "enumNulls" {
			src.Actual.Properties[i].Type = changeLeaf(p.Type)
			assertRejected("changed actual JsonNull leaf")
			src.Actual.Properties[i] = p
			src.Actual.Properties[i].Default = "kotlin.collections.emptyList()"
			assertRejected("changed JsonNull constructor initializer")
			src.Actual.Properties[i] = p
		}
		if p.WireName == "lookup" {
			src.Actual.Properties[i].DefaultNull = !p.DefaultNull
			assertRejected("changed actual null initializer policy")
			src.Actual.Properties[i] = p
		}
		if p.WireName == "numbers" {
			src.Actual.Properties[i].Type = changeLeaf(p.Type)
			assertRejected("changed mapped leaf")
			src.Actual.Properties[i] = p
			src.Actual.Properties[i].Default = "kotlin.collections.listOf(9.0f)"
			assertRejected("changed actual constructor initializer")
			src.Actual.Properties[i] = p
			src.Actual.Properties[i].Type = TypeRef{Kind: JSON}
			assertRejected("raw projection")
			src.Actual.Properties[i] = p
		}
	}
	for i, f := range src.Fields {
		if f.Wire == "numbers" {
			if src.DefaultWork[i] != 3 {
				t.Fatal("initializer traversal work lost", src.DefaultWork[i])
			}
			src.DefaultWork[i] = 1
			assertRejected("reset initializer work")
			src.DefaultWork[i] = 3
			old := src.Defaults[i]
			wrong := `["wrong"]`
			src.Defaults[i] = &wrong
			assertRejected("wrong canonical constructor collection default")
			src.Defaults[i] = old
			value := get(f.Schema, "items")
			raw, _ := value.MarshalJSON()
			clone, parse := document.ParseObject(raw, document.MaxDocumentBytes)
			if parse != nil {
				t.Fatal(parse)
			}
			for j := range f.Schema.Members() {
				if f.Schema.Members()[j].Name == "items" {
					f.Schema.Members()[j].Value = clone
					assertRejected("samebytes item source replacement")
					f.Schema.Members()[j].Value = value
				}
			}
		}
	}
	parse := func(raw string) *document.Value {
		v, e := document.ParseObject([]byte(raw), document.MaxDocumentBytes)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	value := parse(`{"id":"a","requiredNull":null,"numbers":[0,1],"lookup":{"a":0,"b":1}}`)
	cost := 1 + len(src.Key.Fields) + 16 // List root+two leaves, Map root+two keys+two values.
	budget := &dtoUniqueDefaultKeys{Work: 16384 - cost, Seen: map[dtoUniqueDefaultCache]string{}, Sources: map[*DTOUniqueKey]*dtoUniqueSource{src.Key: src}}
	key, failure := budget.key(src.Key, value, "/value")
	if failure != nil || budget.Work != 16384 {
		t.Fatal("exact nested default-key work", budget.Work, cost, failure)
	}
	if again, failure := budget.key(src.Key, value, "/repeat"); failure != nil || again != key || budget.Work != 16384 {
		t.Fatal("identical literal was recharged", failure)
	}
	clone := parse(`{"id":"a","requiredNull":null,"numbers":[0,1],"lookup":{"a":0,"b":1}}`)
	if _, failure := budget.key(src.Key, clone, "/other"); failure == nil || failure.Code != "E_MODEL_LIMIT" {
		t.Fatal("distinct equalbytes literal escaped key work", failure)
	}
	omitted := parse(`{"id":"a","requiredNull":null,"lookup":{"b":1,"a":-0.0}}`)
	budget = &dtoUniqueDefaultKeys{Seen: map[dtoUniqueDefaultCache]string{}, Sources: map[*DTOUniqueKey]*dtoUniqueSource{src.Key: src}}
	omittedKey, failure := budget.key(src.Key, omitted, "/omitted")
	if failure != nil || omittedKey != key || budget.Work != cost {
		t.Fatal("omitted initializer/maporder/Float zero key or work mismatch", failure, budget.Work, cost)
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored original getter rejected", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
}

func TestDTONativeCollectionProbeKeepsMetadataPrivate(t *testing.T) {
	v, e := document.ParseObject([]byte(`{"type":"array","items":{"type":"integer","enum":[0,1],"minimum":0}}`), document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{options: Options{Validation: true}, validationSources: map[*PrimitiveValidation]string{}, emissionHints: EmissionHints{}}
	if ok, e := a.dtoUniqueContainerCandidate(v, "/source", 0, true); !ok || e != nil {
		t.Fatal("native getter source", e)
	}
	if len(a.validationSources) != 0 || len(a.emissionHints) != 0 || len(a.warnings) != 0 {
		t.Fatal("probe leaked runtime checker or warnings")
	}
}

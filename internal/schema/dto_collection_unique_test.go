package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTOCollectionUniqueActualGetterTreeAndKeyWork(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-collection-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.dtocollectionunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	cost := 1 + len(src.Key.Fields) + 6 // List root+two leaves, Map root+two keys+two values.
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

func TestDTOCollectionUniqueCanonicalEncodingSharedTrees(t *testing.T) {
	for _, wide := range []bool{false, true} {
		props := map[string]any{}
		if wide {
			for i := range 240 {
				props[fmt.Sprint("p", i)] = map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}
			}
		} else {
			props["values"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "default": []any{strings.Repeat("x", 32000)}}
		}
		holder := map[string]any{}
		for i := range 240 {
			holder[fmt.Sprint("v", i)] = map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"$ref": "#/components/schemas/Item"}}
		}
		data, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": props}, "Holder": map[string]any{"properties": holder}}}})
		if err != nil {
			t.Fatal(err)
		}
		root, load := document.Load(data, document.JSON, "")
		if load != nil {
			t.Fatal(load)
		}
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.dtocollectionbounds", Suffix: "Dto", Validation: true})
		if failure != nil {
			t.Fatal(wide, failure)
		}
		encoded, err := json.Marshal(plan.Models)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) > 1<<20 {
			t.Fatal("shared field/default trees expanded into canonical model metadata", wide, len(encoded))
		}
	}
}

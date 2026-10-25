package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTOConstUniqueActualBodyAndOriginalSource(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-const-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.dtoconstunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || src.Actual == nil || len(src.Actual.Constants) != 3 || len(src.Key.Fields) != len(src.Actual.Properties)+3 {
		t.Fatal("actual constant getters lost")
	}
	assertRejected := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized wrong body")
		}
	}
	constants := append([]Constant{}, src.Actual.Constants...)
	for _, mutation := range []string{"initializer", "type", "name", "wire"} {
		switch mutation {
		case "initializer":
			src.Actual.Constants[0].Initializer = `"foreign"`
		case "type":
			src.Actual.Constants[0].Type = TypeRef{Kind: Scalar, Name: "kotlin.Boolean"}
		case "name":
			src.Actual.Constants[0].Name = "foreign"
		case "wire":
			src.Actual.Constants[0].WireName = "foreign"
		}
		assertRejected(mutation)
		copy(src.Actual.Constants, constants)
	}
	src.Actual.Constants = constants[:2]
	assertRejected("missing body")
	src.Actual.Constants = append(append([]Constant{}, constants...), constants[0])
	assertRejected("repeated body")
	src.Actual.Constants = constants
	for i, f := range src.Fields {
		if !has(f.Schema, "const") {
			continue
		}
		defaultKey := src.Defaults[i]
		wrong := "wrong"
		src.Defaults[i] = &wrong
		assertRejected("wrong initializer key")
		src.Defaults[i] = defaultKey
		for _, member := range f.Schema.Members() {
			if member.Name == "const" {
				old := member.Value
				raw, _ := old.MarshalJSON()
				wrapper, parse := document.ParseObject(append(append([]byte(`{"v":`), raw...), '}'), document.MaxDocumentBytes)
				if parse != nil {
					t.Fatal(parse)
				}
				for j := range f.Schema.Members() {
					if f.Schema.Members()[j].Name == "const" {
						f.Schema.Members()[j].Value = get(wrapper, "v")
						assertRejected("samebytes source replacement")
						f.Schema.Members()[j].Value = old
					}
				}
				break
			}
		}
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored original rejected", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
}

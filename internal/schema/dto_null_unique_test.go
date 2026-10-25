package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTOJsonNullUniqueActualFieldsAndOriginalSource(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-null-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.dtonullunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || src.Actual == nil || len(src.Actual.Constants) != 5 {
		t.Fatal("actual null body/constructor identity lost")
	}
	assertRejected := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized raw/foreign null codec")
		}
	}
	for i, p := range src.Actual.Properties {
		if p.WireName != "requiredNull" && p.WireName != "optionalNull" {
			continue
		}
		src.Actual.Properties[i].Type = TypeRef{Kind: JSON}
		assertRejected("raw projection for typed null getter")
		src.Actual.Properties[i] = p
	}
	for i, c := range src.Actual.Constants {
		if c.Type.Name != "kotlinx.serialization.json.JsonNull" {
			continue
		}
		src.Actual.Constants[i].Type = TypeRef{Kind: Scalar, Name: "kotlinx.serialization.json.JsonElement"}
		assertRejected("foreign body codec")
		src.Actual.Constants[i] = c
		src.Actual.Constants[i].Initializer = "foreign"
		assertRejected("foreign null initializer")
		src.Actual.Constants[i] = c
	}
	for i, f := range src.Fields {
		if f.Wire == "fixedNull" {
			if has(f.Schema, "type") || src.Defaults[i] != nil {
				t.Fatal("null initializer/type was fabricated")
			}
			wrong := "null"
			src.Defaults[i] = &wrong
			assertRejected("non-null default key for null body")
			src.Defaults[i] = nil
		}
		if f.Wire == "nullEnum" {
			if has(f.Schema, "type") {
				t.Fatal("enum inference inserted type")
			}
			for _, origin := range a.enumOrigins {
				if origin.Schema == f.Schema {
					t.Fatal("null-only source acquired String enum declaration")
				}
			}
			values := get(f.Schema, "enum")
			old := values.Items()[0]
			wrapper, parse := document.ParseObject([]byte(`{"v":null}`), document.MaxDocumentBytes)
			if parse != nil {
				t.Fatal(parse)
			}
			values.Items()[0] = get(wrapper, "v")
			assertRejected("samebytes null vocabulary replacement")
			values.Items()[0] = old
		}
	}
	for _, use := range *a.dtoUniqueUses {
		if use.Source == src && !use.DefaultOnly {
			old := *use.Element
			*use.Element = TypeRef{Kind: JSON}
			assertRejected("raw collection element")
			*use.Element = old
			break
		}
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored typed source failed", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
}

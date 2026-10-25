package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTOEnumUniqueOriginalVocabularyAndActualDeclaration(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-enum-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.dtoenumunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || src.Actual == nil || len(src.EnumActual) != 1 {
		t.Fatal("actual enum declaration was not proven")
	}
	f := get(get(src.Schema, "properties"), "state")
	enum := src.EnumActual[f]
	if enum == nil {
		t.Fatal("original field has no actual enum")
	}
	assertRejected := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized a foreign enum codec")
		}
	}
	origin := a.enumOrigins[enum]
	changed := origin
	changed.Pointer += "/foreign"
	a.enumOrigins[enum] = changed
	assertRejected("foreign Pointer")
	a.enumOrigins[enum] = origin
	changed = origin
	changed.Schema = src.Schema
	a.enumOrigins[enum] = changed
	assertRejected("foreign original node")
	a.enumOrigins[enum] = origin
	oldName := enum.ClassName
	enum.ClassName = "Foreign"
	assertRejected("guessed classifier")
	enum.ClassName = oldName
	oldMembers := append([]EnumMember{}, enum.Members...)
	enum.Members[0].Wire = "foreign"
	assertRejected("changed emitted vocabulary")
	copy(enum.Members, oldMembers)
	enum.Members[0].Name = "Foreign"
	assertRejected("changed emitted member")
	copy(enum.Members, oldMembers)
	nested := append([]Declaration{}, src.Actual.Nested...)
	without := []Declaration{}
	for _, d := range nested {
		if d != enum {
			without = append(without, d)
		}
	}
	src.Actual.Nested = without
	assertRejected("missing actual enum")
	src.Actual.Nested = append(append([]Declaration{}, nested...), enum)
	assertRejected("repeated actual enum")
	copied := *enum
	a.enumOrigins[&copied] = origin
	src.Actual.Nested = append(append([]Declaration{}, without...), &copied)
	assertRejected("same members copied enum")
	delete(a.enumOrigins, &copied)
	src.Actual.Nested = nested
	vocab := get(f, "enum")
	old := vocab.Items()[0]
	raw, _ := old.MarshalJSON()
	clone, parse := document.ParseObject(append(append([]byte(`{"v":`), raw...), '}'), document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	vocab.Items()[0] = get(clone, "v")
	assertRejected("same bytes vocabulary element replacement")
	vocab.Items()[0] = old
	members := f.Members()
	for i, m := range members {
		if m.Name == "enum" {
			raw, _ := m.Value.MarshalJSON()
			clone, parse := document.ParseObject(append(append([]byte(`{"v":`), raw...), '}'), document.MaxDocumentBytes)
			if parse != nil {
				t.Fatal(parse)
			}
			members[i].Value = get(clone, "v")
			assertRejected("same bytes vocabulary array replacement")
			members[i].Value = vocab
			break
		}
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored original codec rejected", e)
	}

	work, bytes := a.dtoUniqueCaptures.Work, a.dtoUniqueCaptures.Bytes
	if _, e := a.dtoUniqueSource(src.Schema, src.Pointer, "/repeat"); e != nil || work != a.dtoUniqueCaptures.Work || bytes != a.dtoUniqueCaptures.Bytes {
		t.Fatal("enum source capture reset or recharged", e)
	}
	rootWork := 0
	for _, node := range src.Nodes {
		rootWork += 1 + len(node.Members) + len(node.Items)
	}
	for _, extra := range []int{0, 1} {
		context := a
		context.dtoUniqueSources = map[*document.Value]*dtoUniqueSource{}
		context.dtoUniqueCaptures = &dtoUniqueCaptureBudget{Work: 16384 - rootWork + extra, Bytes: (8 << 20) - len(src.Bytes)}
		_, e := context.dtoUniqueSource(src.Schema, src.Pointer, "/boundary")
		if extra == 0 && (e != nil || context.dtoUniqueCaptures.Work != 16384 || context.dtoUniqueCaptures.Bytes != 8<<20) || extra == 1 && (e == nil || e.Code != "E_MODEL_LIMIT") {
			t.Fatal("enum source work boundary changed", extra, e)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST mutated")
	}
}

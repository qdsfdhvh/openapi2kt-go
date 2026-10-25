package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestDTOUniqueOriginalConstructorCaptureAndDefaultIdentity(t *testing.T) {
	data, readErr := os.ReadFile("../../tests/fixtures/http/dto-unique.json")
	if readErr != nil {
		t.Fatal(readErr)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	before, _ := root.MarshalJSON()
	defaults := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.dtounique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	owner := a.components["Item"]
	src := a.dtoUniqueSources[owner.Schema]
	if src == nil || src.Actual == nil || a.modelOrigins[src.Actual] != src.Schema || a.modelPointers[src.Actual] != src.Pointer || src.Owner != owner || len(a.dtoUniqueSources) != 2 {
		t.Fatal("typedname replaced originalconstructor/source")
	}
	actual := src.Actual
	without := []Declaration{}
	for _, d := range ds {
		if d != actual {
			without = append(without, d)
		}
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, ds...), actual)} {
		if e := a.validateDTOUniqueUses(&Plan{Models: bad}); e == nil {
			t.Fatal("missing/repeatedrealconstructor authorized")
		}
	}
	copied := *actual
	a.modelOrigins[&copied] = src.Schema
	a.modelPointers[&copied] = src.Pointer
	if e := a.validateDTOUniqueUses(&Plan{Models: append(append([]Declaration{}, without...), &copied)}); e == nil {
		t.Fatal("samefields copiedconstructor replaced actualidentity")
	}
	delete(a.modelOrigins, &copied)
	delete(a.modelPointers, &copied)
	for i := range src.Key.Fields {
		if src.Fields[i].Wire == "version" {
			old := src.Defaults[i]
			wrong := "123"
			src.Defaults[i] = &wrong
			if e := a.validateDTOUniqueUses(plan); e == nil {
				t.Fatal("wrong constructor default key authorized")
			}
			src.Defaults[i] = old
		}
	}
	props := get(src.Schema, "properties")
	original := get(props, "id")
	raw, _ := original.MarshalJSON()
	clone, parse := document.ParseObject(raw, document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	for i := range props.Members() {
		if props.Members()[i].Name == "id" {
			props.Members()[i].Value = clone
			if e := a.validateDTOUniqueUses(plan); e == nil {
				t.Fatal("samebytes replaced fieldnode authorized")
			}
			props.Members()[i].Value = original
			break
		}
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
	work, bytes := a.dtoUniqueCaptures.Work, a.dtoUniqueCaptures.Bytes
	rootWork := 0
	for _, node := range src.Nodes {
		rootWork += 1 + len(node.Members) + len(node.Items)
	}
	rootBytes := len(src.Bytes)
	totalWork, totalBytes := 0, 0
	for _, source := range a.dtoUniqueSources {
		totalBytes += len(source.Bytes)
		for _, node := range source.Nodes {
			totalWork += 1 + len(node.Members) + len(node.Items)
		}
	}
	if totalWork != work || totalBytes != bytes {
		t.Fatal("distinct original sources not charged cumulatively", work, bytes, totalWork, totalBytes)
	}

	context := a
	if _, e := context.dtoUniqueSource(src.Schema, src.Pointer, "/repeat"); e != nil || context.dtoUniqueCaptures != a.dtoUniqueCaptures || a.dtoUniqueCaptures.Work != work || a.dtoUniqueCaptures.Bytes != bytes {
		t.Fatal("sharedsourcecache recharged/reset", e)
	}
	for _, overflow := range []string{"exact", "work", "bytes"} {
		context := a
		context.dtoUniqueSources = map[*document.Value]*dtoUniqueSource{}
		context.dtoUniqueCaptures = &dtoUniqueCaptureBudget{Work: 16384 - rootWork, Bytes: (8 << 20) - rootBytes}
		if overflow == "work" {
			context.dtoUniqueCaptures.Work++
		}
		if overflow == "bytes" {
			context.dtoUniqueCaptures.Bytes++
		}
		_, e := context.dtoUniqueSource(src.Schema, src.Pointer, "/boundary")
		if overflow == "exact" && (e != nil || context.dtoUniqueCaptures.Work != 16384 || context.dtoUniqueCaptures.Bytes != 8<<20) || overflow != "exact" && (e == nil || e.Code != "E_MODEL_LIMIT") {
			t.Fatal("unique source cap not exact", overflow, e)
		}
	}
	value, parse := document.ParseObject([]byte(`{"id":"a"}`), document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	budget := &dtoUniqueDefaultKeys{Work: 16384 - 1 - len(src.Key.Fields), Seen: map[dtoUniqueDefaultCache]string{}, Sources: map[*DTOUniqueKey]*dtoUniqueSource{src.Key: src}}
	if _, e := budget.key(src.Key, value, "/value"); e != nil || budget.Work != 16384 {
		t.Fatal("exactdefault key work", e)
	}
	if _, e := budget.key(src.Key, value, "/repeat"); e != nil || budget.Work != 16384 {
		t.Fatal("samevalue key was recharged", e)
	}
	cloneValue, parse := document.ParseObject([]byte(`{"id":"a"}`), document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	if _, e := budget.key(src.Key, cloneValue, "/different"); e == nil || e.Code != "E_MODEL_LIMIT" {
		t.Fatal("distinctsamebytes value escaped cumulative key work", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source/default AST mutated")
	}
}

func TestDTOUniqueCanonicalEncodingDoesNotExpandSharedDescriptor(t *testing.T) {
	for _, wide := range []bool{false, true} {
		item := map[string]any{}
		holder := map[string]any{}
		if wide {
			for i := range 240 {
				item[fmt.Sprint("p", i)] = map[string]any{"type": "string"}
			}
		} else {
			item["value"] = map[string]any{"type": "string", "default": strings.Repeat("x", 32000)}
		}
		for i := range 240 {
			holder[fmt.Sprint("v", i)] = map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}, "uniqueItems": true}
		}
		data, encode := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Item": map[string]any{"properties": item}, "Holder": map[string]any{"properties": holder}}}})
		if encode != nil {
			t.Fatal(encode)
		}
		root, load := document.Load(data, document.JSON, "")
		if load != nil {
			t.Fatal(load)
		}
		plan, _, failure := Analyze(root, Options{PackageName: "org.example.dtouniquebound", Suffix: "Dto", Validation: true})
		if failure != nil {
			t.Fatal(wide, failure)
		}
		encoded, encode := json.Marshal(plan.Models)
		if encode != nil {
			t.Fatal(encode)
		}
		if len(encoded) > 1<<20 {
			t.Fatal("shared DTO key/default metadata expanded into canonical model encoding", wide, len(encoded))
		}
	}
}

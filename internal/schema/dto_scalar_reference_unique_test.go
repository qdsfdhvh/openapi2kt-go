package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func dtoScalarReferenceAnalysis(t *testing.T) (*analyzer, *Plan, *document.Value, *dtoUniqueSource) {
	t.Helper()
	data, err := os.ReadFile("../../tests/fixtures/http/dto-scalar-reference-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, errLoad := document.Load(data, document.JSON, "")
	if errLoad != nil {
		t.Fatal(errLoad)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.dtoscalarreferenceunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || src.Actual == nil || len(src.ReferenceFields) != 11 {
		t.Fatal("missing original reference getter proof", src)
	}
	return a, plan, root, src
}

func TestDTOScalarReferenceActualSourceDeclarationAndInitializer(t *testing.T) {
	a, plan, root, src := dtoScalarReferenceAnalysis(t)
	before, _ := root.MarshalJSON()
	reject := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign scalar ref getter")
		}
	}
	for i, p := range src.Actual.Properties {
		if p.WireName != "refFloat" {
			continue
		}
		src.Actual.Properties[i].Default = "1.0f"
		reject("changed actual deferred initializer")
		src.Actual.Properties[i] = p
		src.Actual.Properties[i].DefaultNull = !p.DefaultNull
		reject("changed null initializer policy")
		src.Actual.Properties[i] = p
		src.Actual.Properties[i].Type = TypeRef{Kind: Scalar, Name: "kotlin.Double"}.Null()
		reject("changed actual mapped type")
		src.Actual.Properties[i] = p
	}
	value := a.components["RefFloat"]
	oldScalar := value.Scalar
	value.Scalar = "kotlin.Double"
	reject("mutated catalog type")
	value.Scalar = oldScalar
	alias := componentDeclaration(plan.Models, "RefFloatDto").(*ScalarAlias)
	alias.Scalar = "kotlin.Double"
	reject("changed exported scalar alias")
	alias.Scalar = oldScalar
	oldOrigin := a.builtinOrigins[alias]
	delete(a.builtinOrigins, alias)
	reject("lost actual builtin declaration authority")
	a.builtinOrigins[alias] = oldOrigin
	for i, d := range plan.Models {
		if d != alias {
			continue
		}
		clone := *alias
		plan.Models[i] = &clone
		a.builtinOrigins[&clone] = oldOrigin
		reject("samebytes copied retained declaration")
		delete(a.builtinOrigins, &clone)
		plan.Models[i] = alias
	}
	schema := value.Schema
	oldMembers := append([]document.Member{}, schema.Members()...)
	for i, m := range schema.Members() {
		if m.Name == "type" {
			v, e := document.ParseObject([]byte(`{"v":"number"}`), document.MaxDocumentBytes)
			if e != nil {
				t.Fatal(e)
			}
			schema.Members()[i].Value = get(v, "v")
			reject("samebytes terminal type node replacement")
			schema.Members()[i] = m
		}
	}
	if len(oldMembers) != len(schema.Members()) {
		t.Fatal("bad restore")
	}
	catalog := get(get(root, "components"), "schemas")
	for i, m := range catalog.Members() {
		if m.Name == "RefFloat" {
			raw, _ := schema.MarshalJSON()
			clone, e := document.ParseObject(raw, document.MaxDocumentBytes)
			if e != nil {
				t.Fatal(e)
			}
			catalog.Members()[i].Value = clone
			reject("cached target cannot authorize replaced document node")
			catalog.Members()[i] = m
		}
	}
	owner := a.components["BorrowOwner"]
	props := get(owner.Schema, "properties")
	raw, _ := props.MarshalJSON()
	clone, e := document.ParseObject(raw, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	for i, m := range owner.Schema.Members() {
		if m.Name == "properties" {
			owner.Schema.Members()[i].Value = clone
			reject("replaced original internal owner ancestor")
			owner.Schema.Members()[i] = m
		}
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored original source rejected", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("scalar ref proof rewrote source")
	}
}

func TestDTOScalarReferenceCaptureBudgetAndReuse(t *testing.T) {
	a, _, _, src := dtoScalarReferenceAnalysis(t)
	var f field
	var original Property
	for i, source := range src.Fields {
		if source.Wire == "refFloat" {
			f = source
			for _, p := range src.Actual.Properties {
				if p.WireName == f.Wire {
					original = p
				}
			}
			_ = i
		}
	}
	trial := *a
	trial.dtoUniqueCaptures = &dtoUniqueCaptureBudget{}
	proof, e := trial.dtoUniqueScalarReference(f, original.Type, original.Type, false, "/capture")
	if e != nil || proof == nil {
		t.Fatal(e)
	}
	work, bytes := trial.dtoUniqueCaptures.Work, trial.dtoUniqueCaptures.Bytes
	for _, over := range []bool{false, true} {
		context := *a
		context.dtoUniqueCaptures = &dtoUniqueCaptureBudget{Work: 16384 - work, Bytes: (8 << 20) - bytes}
		if over {
			context.dtoUniqueCaptures.Work++
		}
		_, e := context.dtoUniqueScalarReference(f, original.Type, original.Type, false, "/capture")
		if !over && (e != nil || context.dtoUniqueCaptures.Work != 16384 || context.dtoUniqueCaptures.Bytes != (8<<20)) || over && (e == nil || e.Code != "E_MODEL_LIMIT") {
			t.Fatal("reference capture exact budget", over, e, work, bytes)
		}
	}
	context := *a
	context.dtoUniqueCaptures = &dtoUniqueCaptureBudget{Bytes: (8 << 20) - bytes + 1}
	if _, e := context.dtoUniqueScalarReference(f, original.Type, original.Type, false, "/bytes"); e == nil || e.Code != "E_MODEL_LIMIT" {
		t.Fatal("reference bytes budget", e)
	}
	initial := a.dtoUniqueCaptures.Work
	initialBytes := a.dtoUniqueCaptures.Bytes
	if _, e := a.dtoUniqueSource(src.Schema, src.Pointer, "/again"); e != nil || a.dtoUniqueCaptures.Work != initial || a.dtoUniqueCaptures.Bytes != initialBytes {
		t.Fatal("identical DTO descriptor recharged", e)
	}
}

func TestDTOScalarReferenceRejectsAliasCopyBeforeFirstBarrier(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/http/dto-scalar-reference-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.refs", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal(e)
	}
	replaced := false
	for i, d := range ds {
		if d.Name() == "RefIntAliasDto" {
			original := d.(*ValueAlias)
			clone := *original
			ds[i] = &clone
			replaced = true
		}
	}
	if !replaced {
		t.Fatal("missing actual alias control")
	}
	if e := a.validateDTOUniqueUses(&Plan{Models: ds}); e == nil {
		t.Fatal("copied intermediate alias passed first actual-declaration barrier")
	}
}

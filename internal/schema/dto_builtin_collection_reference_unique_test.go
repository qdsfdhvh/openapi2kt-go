package schema

import (
	"os"
	"testing"

	"openapi-kt/internal/document"
)

func dtoBuiltinCollectionReferenceAnalysis(t *testing.T, validate bool) (*analyzer, *Plan, *document.Value, *dtoUniqueSource) {
	t.Helper()
	data, err := os.ReadFile("../../tests/fixtures/http/dto-builtin-collection-reference-unique.json")
	if err != nil {
		t.Fatal(err)
	}
	root, load := document.Load(data, document.JSON, "")
	if load != nil {
		t.Fatal(load)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.dtobuiltincollectionreferenceunique", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
	if src == nil || len(src.ReferenceFields) != 19 {
		t.Fatal("missing collection reference source")
	}
	return a, plan, root, src
}

func TestDTOBuiltinCollectionReferenceSourceAndPhysicalAuthority(t *testing.T) {
	a, plan, root, src := dtoBuiltinCollectionReferenceAnalysis(t, true)
	before, _ := root.MarshalJSON()
	reject := func(label string) {
		t.Helper()
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal(label, "authorized foreign collection reference getter")
		}
	}
	for i, p := range src.Actual.Properties {
		if p.WireName != "list" {
			continue
		}
		src.Actual.Properties[i].Default = "kotlin.collections.emptyList()"
		reject("changed original deferred initializer")
		src.Actual.Properties[i] = p
		src.Actual.Properties[i].DefaultNull = !p.DefaultNull
		reject("changed nullable initializer")
		src.Actual.Properties[i] = p
		q := p.Type
		if q.Kind == Nullable {
			child := *q.Element
			q.Element = &child
			child.Name += "Foreign"
		} else {
			q.Name += "Foreign"
		}
		src.Actual.Properties[i].Type = q
		reject("changed getter physical classifier")
		src.Actual.Properties[i] = p
		detached := p.Type
		if detached.Kind == Nullable {
			child := *detached.Element
			detached.Element = &child
		}
		aliasView := &detached
		if aliasView.Kind == Nullable {
			aliasView = aliasView.Element
		}
		if aliasView.Kind != Named || aliasView.Element == nil {
			t.Fatal("expected real named collection backing")
		}
		copyBacking := *aliasView.Element
		aliasView.Element = &copyBacking
		src.Actual.Properties[i].Type = detached
		reject("same-byte detached getter alias backing")
		src.Actual.Properties[i] = p
	}
	alias := componentDeclaration(plan.Models, "RefFloatsDto").(*ValueAlias)
	originalType := alias.Type
	alias.Type = TypeRef{Kind: List, Element: &TypeRef{Kind: Scalar, Name: "kotlin.Double"}}
	reject("changed actual alias backing")
	alias.Type = originalType
	for i, d := range plan.Models {
		if d != alias {
			continue
		}
		copy := *alias
		plan.Models[i] = &copy
		reject("copied same-byte alias object")
		plan.Models[i] = d
		plan.Models = append(plan.Models, d)
		reject("duplicate actual alias")
		plan.Models = plan.Models[:len(plan.Models)-1]
		break
	}
	for _, origin := range a.namedCollectionOrigins {
		if origin.Owner != a.components["RefFloats"] {
			continue
		}
		for _, snapshot := range origin.Types {
			if snapshot.Value.Kind != Scalar {
				continue
			}
			old := *snapshot.Node
			snapshot.Node.Name = "kotlin.Double"
			reject("mutated original shared leaf")
			*snapshot.Node = old
			if old.Validation != nil {
				v := *old.Validation
				snapshot.Node.Validation = &v
				reject("copied check object")
				*snapshot.Node = old
			}
		}
	}
	v := a.components["RefFloats"].Schema
	for i, m := range v.Members() {
		if m.Name != "items" {
			continue
		}
		raw, _ := m.Value.MarshalJSON()
		copy, e := document.ParseObject(raw, document.MaxDocumentBytes)
		if e != nil {
			t.Fatal(e)
		}
		v.Members()[i].Value = copy
		reject("same-byte original source replacement")
		v.Members()[i].Value = m.Value
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal("restored source rejected", e)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source AST changed")
	}
}

func TestDTOBuiltinCollectionReferenceBeforeFirstBarrier(t *testing.T) {
	a, plan, _, _ := dtoBuiltinCollectionReferenceAnalysis(t, false)
	for i, d := range plan.Models {
		if d.Name() != "RefFloatsDto" {
			continue
		}
		copy := *d.(*ValueAlias)
		plan.Models[i] = &copy
		if e := a.validateDTOUniqueUses(plan); e == nil {
			t.Fatal("copied alias before first barrier authorized")
		}
		plan.Models[i] = d
	}
	if e := a.validateDTOUniqueUses(plan); e != nil {
		t.Fatal(e)
	}
}

func TestDTOBuiltinCollectionReferenceKeyAndCaptureBudgets(t *testing.T) {
	root, e := document.Load([]byte(`{"openapi": "3.1.0", "components": {"schemas": {"Value": {"type": "object", "additionalProperties": {"type": "number", "format": "float"}}, "Item": {"properties": {"value": {"$ref": "#/components/schemas/Value", "default": {"a": 0, "b": 1}}}}, "Holder": {"properties": {"values": {"type": "array", "items": {"$ref": "#/components/schemas/Item"}, "uniqueItems": true}}}}}}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	defaults := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.budget", Suffix: "Dto", Validation: true}, components: map[string]*component{}, referencedDefaults: &defaults, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, e := a.analyze(root)
	if e != nil {
		t.Fatal(e)
	}
	if e := a.planReferencedFieldDefaults(); e != nil {
		t.Fatal(e)
	}
	if e := a.validateDTOUniqueUses(&Plan{Models: ds}); e != nil {
		t.Fatal(e)
	}
	src := a.dtoUniqueSources[a.components["Item"].Schema]
	if src == nil || src.DefaultWork[0] != 5 {
		t.Fatal("lost complete map initializer work")
	}
	parse := func(raw string) *document.Value {
		v, e := document.ParseObject([]byte(raw), document.MaxDocumentBytes)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	omitted := parse(`{}`)
	// One object + one field + map root/two keys/two values, with the field
	// already counting one root visit: six work units in either representation.
	budget := &dtoUniqueDefaultKeys{Work: 16384 - 6, Seen: map[dtoUniqueDefaultCache]string{}, Sources: map[*DTOUniqueKey]*dtoUniqueSource{src.Key: src}}
	key, e := budget.key(src.Key, omitted, "/omitted")
	if e != nil || budget.Work != 16384 {
		t.Fatal("exact referenced map key limit", budget.Work, e)
	}
	if again, e := budget.key(src.Key, omitted, "/repeat"); e != nil || again != key || budget.Work != 16384 {
		t.Fatal("same literal reset or recharged key budget")
	}
	if _, e := budget.key(src.Key, parse(`{}`), "/new"); e == nil || e.Code != "E_MODEL_LIMIT" {
		t.Fatal("distinct literal escaped cumulative work", e)
	}
	budget = &dtoUniqueDefaultKeys{Seen: map[dtoUniqueDefaultCache]string{}, Sources: map[*DTOUniqueKey]*dtoUniqueSource{src.Key: src}}
	explicit, e := budget.key(src.Key, parse(`{"value":{"b":1,"a":-0.0}}`), "/explicit")
	if e != nil || explicit != key || budget.Work != 6 {
		t.Fatal("referenced map order/Float zero/default key", budget.Work, e)
	}
	f, p := src.Fields[0], src.Actual.Properties[0]
	trial := *a
	trial.dtoUniqueCaptures = &dtoUniqueCaptureBudget{}
	if _, e := trial.dtoUniqueScalarReference(f, defaultBacking(p.Type), p.Type, false, "/capture"); e != nil {
		t.Fatal(e)
	}
	work, size := trial.dtoUniqueCaptures.Work, trial.dtoUniqueCaptures.Bytes
	for _, over := range []bool{false, true} {
		context := *a
		context.dtoUniqueCaptures = &dtoUniqueCaptureBudget{Work: 16384 - work, Bytes: (8 << 20) - size}
		if over {
			context.dtoUniqueCaptures.Work++
		}
		_, e := context.dtoUniqueScalarReference(f, defaultBacking(p.Type), p.Type, false, "/capture")
		if !over && (e != nil || context.dtoUniqueCaptures.Work != 16384 || context.dtoUniqueCaptures.Bytes != 8<<20) || over && (e == nil || e.Code != "E_MODEL_LIMIT") {
			t.Fatal("reference capture shared work/byte boundary", over, e)
		}
	}
	context := *a
	context.dtoUniqueCaptures = &dtoUniqueCaptureBudget{Bytes: (8 << 20) - size + 1}
	if _, e := context.dtoUniqueScalarReference(f, defaultBacking(p.Type), p.Type, false, "/bytes"); e == nil || e.Code != "E_MODEL_LIMIT" {
		t.Fatal("reference byte capture boundary", e)
	}
}

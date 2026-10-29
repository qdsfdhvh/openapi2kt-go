package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestDictionaryAllOfSourceAndDeclarationAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{
		"A":{"allOf":[{"$ref":"#/components/schemas/Middle","nullable":false}]},
		"Middle":{"$ref":"#/components/schemas/ZMap"},
		"Other":{"additionalProperties":{"type":"string","minLength":1}},
		"ZMap":{"nullable":true,"additionalProperties":{"type":"string","minLength":1}}
	}`)
	a := &analyzer{
		root: root, options: Options{PackageName: "org.example.dictionary", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	source := a.dictionaryAllOfSources[a.components["A"]]
	if source == nil || source.Canonical != a.components["ZMap"] || len(source.Bindings) != 3 {
		t.Fatal("original dictionary chain was not captured", source)
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if err := a.validateDictionaryAllOfSources(declarations); err == nil {
			t.Error("changed source retained authority:", name)
		}
		restore()
		if err := a.validateDictionaryAllOfSources(declarations); err != nil {
			t.Fatal("restored source rejected:", name, err)
		}
	}
	reject("intermediate target", func() func() {
		c := a.components["Middle"]
		old := c.BindingTarget
		c.BindingTarget = a.components["Other"]
		return func() { c.BindingTarget = old }
	})
	reject("cached source height", func() func() {
		c := a.components["A"]
		old := c.AliasHeight
		c.AliasHeight++
		return func() { c.AliasHeight = old }
	})
	reject("nullable intent", func() func() {
		c := a.components["A"]
		old := *c.BindingNullable
		*c.BindingNullable = !old
		return func() { *c.BindingNullable = old }
	})
	reject("same-byte member ref replacement", func() func() {
		copy, err := document.Load([]byte(`{"$ref":"#/components/schemas/Middle","nullable":false}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		fields := source.Member.Members()
		old := fields[0].Value
		fields[0].Value = get(copy, "$ref")
		return func() { fields[0].Value = old }
	})
	reject("same-byte wrapper member replacement", func() func() {
		copy, err := document.Load([]byte(`{"$ref":"#/components/schemas/Middle","nullable":false}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		items := source.Members.Items()
		old := items[0]
		items[0] = copy
		return func() { items[0] = old }
	})
	reject("same-byte nested target source replacement", func() func() {
		copy, err := document.Load([]byte(`{"minLength":1}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		fields := get(a.components["ZMap"].Schema, "additionalProperties").Members()
		old := fields[1].Value
		fields[1].Value = get(copy, "minLength")
		return func() { fields[1].Value = old }
	})
	reject("same-node nested member rename", func() func() {
		fields := get(a.components["ZMap"].Schema, "additionalProperties").Members()
		old := fields[1].Name
		fields[1].Name = "maxLength"
		return func() { fields[1].Name = old }
	})
	reject("copied intermediate declaration", func() func() {
		for i, declaration := range declarations {
			if declaration.Name() == "Middle" {
				old := declarations[i]
				copy := *old.(*ValueAlias)
				declarations[i] = &copy
				return func() { declarations[i] = old }
			}
		}
		t.Fatal("missing intermediate declaration")
		return nil
	})
	reject("duplicate wrapper declaration", func() func() {
		declarations = append(declarations, source.Bindings[0].Declaration)
		return func() { declarations = declarations[:len(declarations)-1] }
	})
	reject("canonical backing checks", func() func() {
		alias := source.Bindings[len(source.Bindings)-1].Declaration
		checks := alias.Type.WithoutRootNull().Element.Validation
		if checks == nil || checks.MinLength == nil {
			t.Fatal("missing canonical leaf checks")
		}
		old := *checks.MinLength
		*checks.MinLength++
		return func() { *checks.MinLength = old }
	})
}

func TestDictionaryAllOfCaptureBudget(t *testing.T) {
	budget := dictionaryAllOfBudget{Work: 16383, Bytes: (8 << 20) - 2}
	if err := budget.charge(1, 2, "/source"); err != nil || budget.Work != 16384 || budget.Bytes != 8<<20 {
		t.Fatal("exact budget rejected", budget, err)
	}
	if err := budget.charge(1, 0, "/source"); err == nil || err.Code != "E_MODEL_LIMIT" {
		t.Fatal("work overrun admitted", err)
	}
	budget = dictionaryAllOfBudget{Bytes: (8 << 20) - 1}
	if err := budget.charge(1, 2, "/source"); err == nil || err.Code != "E_MODEL_LIMIT" {
		t.Fatal("byte overrun admitted", err)
	}
}

func TestDictionaryAllOfNeutralMemberSourceAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{
		"A":{"allOf":[{"description":"note"},{"$ref":"#/components/schemas/ZMap"},{}]},
		"ZMap":{"additionalProperties":{"type":"string"}}
	}`)
	a := &analyzer{
		root: root, options: Options{PackageName: "org.example.dictionary", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	source := a.dictionaryAllOfSources[a.components["A"]]
	if source == nil || source.Pointer != "/components/schemas/A/allOf/1/$ref" {
		t.Fatal("the original middle reference was not captured", source)
	}
	items := source.Members.Items()
	items[0], items[2] = items[2], items[0]
	if err := a.validateDictionaryAllOfSources(declarations); err == nil {
		t.Error("reordered neutral sources retained authority")
	}
	items[0], items[2] = items[2], items[0]
	if err := a.validateDictionaryAllOfSources(declarations); err != nil {
		t.Fatal("restored member order rejected", err)
	}
	copy, err := document.Load([]byte(`{"description":"note"}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	old := items[0]
	items[0] = copy
	if err := a.validateDictionaryAllOfSources(declarations); err == nil {
		t.Error("equal replacement neutral source retained authority")
	}
	items[0] = old
	if err := a.validateDictionaryAllOfSources(declarations); err != nil {
		t.Fatal("restored neutral source rejected", err)
	}
}

func TestDictionaryAllOfIndependentFactoryAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{
		"A":{"allOf":[{},{"$ref":"#/components/schemas/M"}],"default":{"a":"one"}},
		"M":{"$ref":"#/components/schemas/Z","default":{"m":"two"}},
		"Z":{"additionalProperties":{"type":"string"},"default":{"z":"three"}}
	}`)
	a := &analyzer{
		root: root, options: Options{PackageName: "org.example.dictionary", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	source := a.dictionaryAllOfSources[a.components["A"]]
	if source == nil || len(source.Bindings) != 3 {
		t.Fatal("independent wrapper source chain not captured", source)
	}
	for index, want := range []string{`{"a":"one"}`, `{"m":"two"}`, `{"z":"three"}`} {
		if got := source.Bindings[index].Factory; got != want {
			t.Fatalf("factory %d: got %q want %q", index, got, want)
		}
	}
	alias := source.Bindings[0].Declaration
	old := alias.ComponentDefault
	alias.ComponentDefault = `{"a":"changed"}`
	if err := a.validateDictionaryAllOfSources(declarations); err == nil {
		t.Error("changed wrapper factory retained authority")
	}
	alias.ComponentDefault = old
	if err := a.validateDictionaryAllOfSources(declarations); err != nil {
		t.Fatal("restored wrapper factory rejected", err)
	}
}

func TestDictionaryAllOfSourceNodeCaptureBudget(t *testing.T) {
	root, err := document.Load([]byte(`{"metadata":[0,1,2]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	budget := dictionaryAllOfBudget{Work: 16384 - 5}
	nodes, failure := captureDictionaryAllOfNodes(root, &budget, "/source")
	if failure != nil || budget.Work != 16384 || len(nodes) != 5 || !dictionaryAllOfNodesMatch(root, nodes) {
		t.Fatal("exact node budget rejected", budget, failure)
	}
	budget = dictionaryAllOfBudget{Work: 16384 - 4}
	if _, failure := captureDictionaryAllOfNodes(root, &budget, "/source"); failure == nil || failure.Code != "E_MODEL_LIMIT" {
		t.Fatal("node work overrun admitted", failure)
	}
	wide, err := document.Load([]byte(`{"values":[`+strings.Repeat(`0,`, 16384)+`0]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	budget = dictionaryAllOfBudget{}
	if _, failure := captureDictionaryAllOfNodes(wide, &budget, "/source"); failure == nil || failure.Code != "E_MODEL_LIMIT" {
		t.Fatal("wide node capture escaped the work limit", failure)
	}
}

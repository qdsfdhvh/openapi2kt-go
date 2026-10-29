package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleUnionRetainedDeclarationAndSourceProof(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[{"oneOf":[{"type":"string"},{"type":"integer"}]}]}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.uniontuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	var union *PrimitiveUnion
	var rows *ValueAlias
	for _, d := range ds {
		if d, ok := d.(*PrimitiveUnion); ok {
			union = d
		}
		if d, ok := d.(*ValueAlias); ok && d.ClassName == "RowsDto" {
			rows = d
		}
	}
	if union == nil || rows == nil {
		t.Fatal("missing actual union or alias")
	}
	kinds := union.Kinds
	union.Kinds = []PrimitiveKind{PrimitiveBoolean}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("mutated JSON-kind selector kept tuple permission")
	}
	union.Kinds = kinds
	origin := a.unionOrigins[union]
	a.unionOrigins[union] = nominalOrigin{}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("union lost original nominal origin without losing tuple permission")
	}
	a.unionOrigins[union] = origin
	for i, d := range ds {
		if d == union {
			copy := *union
			ds[i] = &copy
			if a.validateTuplePlan(plan) == nil {
				t.Fatal("same-byte copied union acquired original authority")
			}
			ds[i] = union
		}
	}
	element := rows.Type.Element
	copyType := *element
	rows.Type.Element = &copyType
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied element address acquired tuple authority")
	}
	rows.Type.Element = element
	proof := a.tupleObjects.Proof.Unions
	for source, fact := range proof {
		changed := fact
		changed.Nullable = !fact.Nullable
		proof[source] = changed
		if a.validateInternalModels(ds) == nil || a.validateTuplePlan(plan) == nil {
			t.Fatal("changed source fact kept tuple authority")
		}
		proof[source] = fact
		break
	}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	member := get(a.components["Rows"].Schema, "prefixItems").Items()[0]
	branches := get(member, "oneOf")
	original := branches.Items()[0]
	bytes, _ := original.MarshalJSON()
	replacement, failure := document.Load(bytes, document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	branches.Items()[0] = replacement
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original branch replacement kept source proof")
	}
	branches.Items()[0] = original
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("union tuple analysis mutated original document")
	}
}

func TestTupleUnionDiscardedValidationIsolation(t *testing.T) {
	root := loadComponentDocument(t, `{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[{"oneOf":[{"properties":{"a":{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"ready"}}}}},{"properties":{"b":{"type":"integer"}}}]}],"items":true}}`)
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.uniontuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 || len(a.modelOrigins) != 0 || len(a.unionOrigins) != 0 || len(a.tupleObjects.UnionDeclarations) != 0 || len(pending) != 0 {
		t.Fatal("discarded union source leaked emission registrations")
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	plan, warnings, err := analyzeDocument(root, a.options, true, nil)
	if err != nil || plan != nil || warnings != nil {
		t.Fatal("source-only validation returned an emission candidate", err)
	}
}

func TestTupleUnionLiteralDataCapture(t *testing.T) {
	root := loadComponentDocument(t, `{"Choice":{"oneOf":[{"properties":{"kind":{"const":"text"},"name":{"type":"string"},"data":{"const":{"allOf":[{"$ref":"#/data-only"}],"description":"data"}}}},{"properties":{"kind":{"const":"count"},"n":{"type":"integer"}}}]},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Choice"}],"items":true}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.uniontuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	branch := get(a.components["Choice"].Schema, "oneOf").Items()[0]
	literal := get(get(get(branch, "properties"), "data"), "const")
	use := a.borrowedUnionLeaves[a.components["Choice"].Schema]
	if use == nil {
		t.Fatal("missing original borrowed union")
	}
	for _, node := range []*document.Value{literal, get(literal, "description"), get(get(literal, "allOf").Items()[0], "$ref")} {
		found := false
		for _, saved := range use.Nodes {
			if saved.Node == node {
				found = true
			}
		}
		if !found {
			t.Fatal("literal data node was dropped from source proof")
		}
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	old := literal.Members()[1].Value
	copyValue, failure := document.Load([]byte(`{"value":"data"}`), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	literal.Members()[1].Value = get(copyValue, "value")
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte metadata-named data replacement kept union authority")
	}
	literal.Members()[1].Value = old
}

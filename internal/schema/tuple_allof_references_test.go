package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleAllOfDiscardedPrefixOriginalShapeRefsAndActualModels(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Base":{"properties":{"description":{"type":"string"}},"required":["description"]},"BaseAlias":{"$ref":"#/components/schemas/Base"},"BaseAgain":{"$ref":"#/components/schemas/BaseAlias"},"Node":{"allOf":[{"$ref":"#/components/schemas/BaseAgain"},{"properties":{"label":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Node"},"Source":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"items":{"type":"integer"}}}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err.Code, err.Pointer)
	}
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.borrowedallof", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err.Code, *err.Pointer)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	var original *Model
	without := []Declaration{}
	for _, d := range ds {
		if d.Name() == "NodeDto" {
			original = d.(*Model)
		} else {
			without = append(without, d)
		}
	}
	if original == nil {
		t.Fatal("canonical allOf constructor absent")
	}
	for _, bad := range [][]Declaration{without, append(append([]Declaration{}, without...), &Model{ClassName: original.ClassName, Properties: original.Properties}), append(append([]Declaration{}, ds...), original)} {
		if a.validateInternalModels(bad) == nil {
			t.Fatal("missing/replaced/repeated original allOf constructor authorized")
		}
	}
	nullable := original.Nullable
	original.Nullable = !nullable
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed actual allOf nullable view authorized")
	}
	original.Nullable = nullable
	base := a.components["Base"]
	originalBase := base.Schema
	copied, loadErr := document.Load([]byte(`{"properties":{"description":{"type":"string"}},"required":["description"]}`), document.JSON, "")
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	base.Schema = copied
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original allOf shape-ref target replacement authorized")
	}
	base.Schema = originalBase
	shapeAlias := a.components["BaseAlias"]
	originalAliasSchema := shapeAlias.Schema
	copyShapeAlias, aliasErr := document.Load([]byte(`{"$ref":"#/components/schemas/Base"}`), document.JSON, "")
	if aliasErr != nil {
		t.Fatal(aliasErr)
	}
	shapeAlias.Schema = copyShapeAlias
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte intermediate allOf shape alias replacement authorized")
	}
	shapeAlias.Schema = originalAliasSchema
	branches := get(a.components["Node"].Schema, "allOf")
	originalBranch := branches.Items()[0]
	copyBranch, branchErr := document.Load([]byte(`{"$ref":"#/components/schemas/BaseAgain"}`), document.JSON, "")
	if branchErr != nil {
		t.Fatal(branchErr)
	}
	branches.Items()[0] = copyBranch
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte original allOf branch replacement authorized")
	}
	branches.Items()[0] = originalBranch
	alias := a.components["Alias"]
	originalAlias := alias.Schema
	aliasBytes, _ := originalAlias.MarshalJSON()
	aliasCopy, copyErr := document.Load(aliasBytes, document.JSON, "")
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	alias.Schema = aliasCopy
	if a.validateInternalModels(ds) == nil {
		t.Fatal("discarded tuple prefix accepted same-byte exported alias replacement")
	}
	alias.Schema = originalAlias
	height := alias.AliasHeight
	alias.AliasHeight++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed allOf leaf alias suffix cache authorized")
	}
	alias.AliasHeight = height
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	// A property named "description" is semantic schema, not annotation metadata.
	property := get(get(base.Schema, "properties"), "description")
	for _, use := range a.borrowedAllOfLeaves {
		found := false
		for _, node := range use.Nodes {
			found = found || node.Node == property
		}
		if !found {
			t.Fatal("allOf referenced property schema was omitted from source identity")
		}
	}
	after, _ := root.MarshalJSON()
	if string(after) != string(before) {
		t.Fatal("allOf borrowed binding mutated original input")
	}
}

func TestTupleAllOfSnapshotBudgetBoundaries(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Base":{"properties":{"value":{"type":"string"}}},"BaseAlias":{"$ref":"#/components/schemas/Base"},"Item":{"allOf":[{"$ref":"#/components/schemas/BaseAlias"},{"properties":{"label":{"type":"string"}}}]},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"items":{"type":"integer"}}}}}`)
	a, _ := referencedCollectionTestAnalyzer(t, input)
	terminal := a.components["Item"].Schema
	target, err := a.resolveSchema("#/components/schemas/Item", "/probe")
	if err != nil {
		t.Fatal(err)
	}
	use := a.borrowedAllOfLeaves[terminal]
	if use == nil {
		t.Fatal("missing source snapshot")
	}
	cost := len(use.Refs)
	for _, node := range use.Nodes {
		cost += 1 + len(node.Members) + len(node.Items)
	}
	for _, spare := range []int{cost, cost - 1} {
		budget := &tupleAllOfSourceBudget{Seen: map[*document.Value]bool{}, Work: 16384 - spare}
		ok, err := a.admitBorrowedAllOfModelBudget(target, "/probe", budget)
		if spare == cost {
			if !ok || err != nil || budget.Work != 16384 {
				t.Fatal("exact aggregate work boundary", ok, budget, err)
			}
			before := *budget
			if ok, err := a.admitBorrowedAllOfModelBudget(target, "/probe", budget); !ok || err != nil || budget.Work != before.Work || budget.Bytes != before.Bytes {
				t.Fatal("same target recharged", err)
			}
		} else if ok || err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("aggregate work overflow accepted", ok, err)
		}
	}
	for _, spare := range []int{len(use.Bytes), len(use.Bytes) - 1} {
		budget := &tupleAllOfSourceBudget{Seen: map[*document.Value]bool{}, Bytes: (8 << 20) - spare}
		ok, err := a.admitBorrowedAllOfModelBudget(target, "/probe", budget)
		if spare == len(use.Bytes) && (!ok || err != nil || budget.Bytes != 8<<20) || spare < len(use.Bytes) && (ok || err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("aggregate source byte boundary", ok, budget, err)
		}
	}
	// Fresh capture must stop before multiplying reference/edge snapshot slices,
	// even if the legacy per-target node count itself is still small.
	budget := &tupleAllOfSourceBudget{Seen: map[*document.Value]bool{}, Work: 16383}
	nodes, refs, err := a.borrowedAllOfSourceBudget(terminal, target.Pointer, "/probe", budget)
	if err == nil || err.Code != "E_MODEL_LIMIT" || nodes != nil || refs != nil {
		t.Fatal("fresh source capture ignored cumulative budget", err)
	}
}

func TestTupleAllOfOriginalAliasIncomingBudget(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"allOf":[{"properties":{"value":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Item","nullable":false},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"items":{"type":"integer"}}}}}`)
	a, _ := referencedCollectionTestAnalyzer(t, input)
	member := get(a.components["Probe"].Schema, "prefixItems").Items()[0]
	for _, spare := range []int{2, 1} {
		work := 16384 - spare
		signature, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work)
		if spare == 2 && (err != nil || signature.ModelSource != a.components["Item"].Schema || signature.Nullable || work != 16384) || spare == 1 && (err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("tuple allOf reset reference work at nominal barrier", signature, work, err)
		}
	}
	last := -1
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		work := 0
		_, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", depth, &work)
		if err == nil {
			last = depth
			continue
		}
		if last < 0 || depth <= last || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("cached allOf target lost incoming depth", depth, last, err)
		}
	}
	if last >= MaxSchemaDepth-1 {
		t.Fatal("allOf nominal barrier erased two original source hops", last)
	}
}

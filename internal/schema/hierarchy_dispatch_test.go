package schema

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func hierarchyDispatchAnalyzer(t *testing.T) (*analyzer, *document.Value) {
	t.Helper()
	b, err := os.ReadFile("../../tests/fixtures/http/hierarchy-dispatch.json")
	if err != nil {
		t.Fatal(err)
	}
	root, e := document.Load(b, document.JSON, "synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	a := &analyzer{root: root, components: map[string]*component{}, options: Options{PackageName: "org.example.hierarchydispatch", Suffix: "Dto"}}
	if _, e := a.analyze(root); e != nil {
		t.Fatal(e)
	}
	return a, get(get(root, "components"), "schemas")
}

func TestHierarchyCompoundOriginalPinsOrderAndBudget(t *testing.T) {
	a, schemas := hierarchyDispatchAnalyzer(t)
	before, err := schemas.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	p := &hierarchyPlanner{a: a, graph: map[string]*hierarchyBranches{}, shapes: map[string]*shape{}, parents: map[string]string{}, heights: map[string]int{}}
	if e := p.visit("Root", 1, nil, "/components/schemas/Root"); e != nil {
		t.Fatal(e)
	}
	infos, e := p.implicit("Group")
	if e != nil || len(infos) != 2 || infos[0].Wire != "tag" || infos[1].Wire != "mode" {
		t.Fatal(infos, e)
	}
	group := p.graph["Group"]
	result := &Hierarchy{}
	if e := p.compoundTags(group, infos, result); e != nil {
		t.Fatal(e)
	}
	if len(result.TagCombinations) != 3 || result.TagCombinations[0].Child != "FirstDto" || result.TagCombinations[1].Child != "SecondDto" || result.TagCombinations[2].Child != "RangeDto" {
		t.Fatal(result.TagCombinations)
	}
	rangeRule := result.TagCombinations[2]
	if !reflect.DeepEqual(rangeRule.Matches, []ObjectTagMatch{{Wire: "tag", Values: []string{"one", "three"}}, {Wire: "mode", Values: []string{"fast", "slow"}}}) {
		t.Fatal(rangeRule)
	}
	original := get(get(get(schemas, "Range"), "properties"), "mode")
	f := shapeField(p.shapes["Range"], "mode")
	if f.Schema != original || f.Pointer != "/components/schemas/Range/properties/mode" || f.Depth != 4 {
		t.Fatal(f)
	}
	after, err := schemas.MarshalJSON()
	if err != nil || string(before) != string(after) {
		t.Fatal("dispatch mutated original schema", err)
	}
	p.work = 16384
	result = &Hierarchy{}
	budgetError := p.compoundTags(group, infos, result)
	if budgetError == nil || budgetError.Code != "E_SCHEMA_LIMIT" || *budgetError.Pointer != "/components/schemas/Group/oneOf/0/$ref" || len(result.TagCombinations) != 0 {
		t.Fatal(budgetError, result)
	}
}

func TestHierarchyReusedTagHasOneCanonicalPropertyAndNearestMapping(t *testing.T) {
	a, _ := hierarchyDispatchAnalyzer(t)
	p := &hierarchyPlanner{a: a, graph: map[string]*hierarchyBranches{}, shapes: map[string]*shape{}, parents: map[string]string{}, heights: map[string]int{}}
	if e := p.visit("ReusedMissingRoot", 1, nil, "/components/schemas/ReusedMissingRoot"); e != nil {
		t.Fatal(e)
	}
	edge, ok := p.mappedField("ReusedMissingSecond", "tag")
	if !ok || edge.Value != "two" || !strings.HasPrefix(edge.Pointer, "/components/schemas/ReusedMissingGroup/oneOf/1/") {
		t.Fatal(edge, ok)
	}
	used := map[string]bool{}
	out := map[string]*Hierarchy{}
	if e := p.build("ReusedMissingRoot", map[string]hierarchyInfo{}, false, used, out); e != nil {
		t.Fatal(e)
	}
	group := out["ReusedMissingGroup"]
	if len(group.Enums) != 0 || len(group.Fields) != 1 || len(group.Inherited) != 1 || len(group.OwnFields) != 0 || group.Fields[0].Type.Name != "org.example.hierarchydispatch.models.ReusedMissingRootDtoTag" {
		t.Fatal(group)
	}
	leaf := out["ReusedMissingSecond"].Leaf
	if len(leaf.Constants) != 1 || leaf.Constants[0].Initializer != `org.example.hierarchydispatch.models.ReusedMissingRootDtoTag.Serializer.fromSerialName("two")` {
		t.Fatal(leaf)
	}
}

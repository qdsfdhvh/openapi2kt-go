package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestAllOfDefaultOverridesOriginalLiteralSourcesAndNoEmittedAggregation(t *testing.T) {
	root := loadComponentDocument(t, `{"Record":{"allOf":[{"properties":{"value":{"type":"integer","default":0}}},{"properties":{"value":{"type":"integer","minimum":1,"maximum":3,"default":2}}}],"default":{}}}`)
	v := get(get(get(root, "components"), "schemas"), "Record")
	pointer := "/components/schemas/Record"
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, objectDefaultSchemas: map[*document.Value]bool{v: true}}
	work := 0
	a.objectDefaultWork = &work
	shape, e := a.modelShape(v, pointer, 0, false)
	if e != nil {
		t.Fatal(e)
	}
	emitted, preserved, e := a.ordinaryPrimitiveAggregates(v, shape, true)
	if e != nil || len(emitted) != 0 || len(preserved) != 0 {
		t.Fatal("unbounded discarded fields introduced emitted aggregation", e)
	}
	node, e := a.ordinaryAllOfDefaultNode(v, pointer, 0)
	if e != nil {
		t.Fatal(e)
	}
	steps := work
	child := node.objectFields[0].node
	last := get(get(get(v, "allOf").Items()[1], "properties"), "value")
	if steps < 4 || child.schema != last || child.pointer != pointer+"/allOf/1/properties/value" || child.numericAggregate == nil || child.numericAggregate.endpoints[0].source != get(last, "minimum") || child.numericAggregate.endpoints[2].source != get(last, "maximum") {
		t.Fatal("lost original winning source/selected bounds", steps)
	}
	if len(a.emissionHints) != 0 {
		t.Fatal("literal-only override plan leaked emitted hints")
	}
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, e := a.ordinaryAllOfDefaultNode(v, pointer, 0)
		if spare == steps && (e != nil || work != maxNestedDefaultNodes) || spare < steps && (e == nil || e.Code != "E_MODEL_LIMIT") {
			t.Fatal("shared original default work reset", spare, work, e)
		}
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original AST changed")
	}
}

func TestAllOfDefaultOverridesRequireCompleteOriginalContributions(t *testing.T) {
	root := loadComponentDocument(t, `{"Record":{"allOf":[{"properties":{"value":{"type":"integer","default":0}}},{"properties":{"value":{"type":"integer","minimum":1,"default":2}}}]}}`)
	v := get(get(get(root, "components"), "schemas"), "Record")
	a := analyzer{root: root, options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}}
	shape, e := a.modelShape(v, "/components/schemas/Record", 0, false)
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"missing", "copied", "reordered"} {
		altered := *shape
		altered.Contributions = append([]field{}, shape.Contributions...)
		switch mode {
		case "missing":
			altered.Contributions = altered.Contributions[1:]
		case "copied":
			original := altered.Contributions[0]
			raw, _ := original.Schema.MarshalJSON()
			copy, e := document.ParseObject(raw, document.MaxDocumentBytes)
			if e != nil {
				t.Fatal(e)
			}
			altered.Contributions[0].Schema = copy
		case "reordered":
			altered.Contributions[0], altered.Contributions[1] = altered.Contributions[1], altered.Contributions[0]
		}
		if _, e := a.allOfDefaultPrimitiveAggregates(v, "/components/schemas/Record", &altered); e == nil {
			t.Fatal("incomplete original source group granted literal role", mode)
		}
	}
}

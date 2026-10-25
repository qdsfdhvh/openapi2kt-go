package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestAllOfUUIDDefaultOriginalSourcesAndSharedWork(t *testing.T) {
	root := loadComponentDocument(t, `{"Record":{"allOf":[{"properties":{"value":{"type":"string","format":"uuid","minLength":36,"pattern":"^123e4567-","default":"123e4567-e89b-12d3-a456-426614174001"}}},{"properties":{"value":{"type":"string","format":"uuid","maxLength":36,"default":"123e4567-e89b-12d3-a456-426614174000"}}}],"default":{"value":"123e4567-e89b-12d3-a456-426614174002"}}}`)
	v := get(get(get(root, "components"), "schemas"), "Record")
	pointer := "/components/schemas/Record"
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, objectDefaultSchemas: map[*document.Value]bool{v: true}}
	work := 0
	a.objectDefaultWork = &work
	node, err := a.ordinaryAllOfDefaultNode(v, pointer, 0)
	if err != nil {
		t.Fatal(err)
	}
	steps := work
	if steps < 4 || len(node.objectFields) != 1 || node.objectFields[0].node.schema != get(get(get(v, "allOf").Items()[1], "properties"), "value") || node.objectFields[0].node.pointer != pointer+"/allOf/1/properties/value" {
		t.Fatal("lost original winning source or discarded work", steps)
	}
	if len(a.emissionHints) != 0 {
		t.Fatal("literal-only plans leaked emission hints")
	}
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err := a.ordinaryAllOfDefaultNode(v, pointer, 0)
		if spare == steps && (err != nil || work != maxNestedDefaultNodes) || spare < steps && (err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("overlap work reset shared budget", work, err)
		}
	}
	work = 0
	s, err := a.modelShape(v, pointer, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	copyRoot, e := document.ParseObject([]byte(`{"field":{"type":"string","format":"uuid","minLength":36,"pattern":"^123e4567-","default":"123e4567-e89b-12d3-a456-426614174001"}}`), document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	s.Contributions[0].Schema = get(copyRoot, "field")
	if _, err := a.allOfDefaultPrimitiveAggregates(v, pointer, s); err == nil {
		t.Fatal("copied source gained discarded authority")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original AST mutated")
	}
}

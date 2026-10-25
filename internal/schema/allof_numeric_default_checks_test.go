package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestAllOfNumericDefaultOriginalSourcesAndSharedWork(t *testing.T) {
	root := loadComponentDocument(t, `{"Record":{"allOf":[{"properties":{"value":{"type":"integer","format":"int64","minimum":9007199254740993,"multipleOf":3}}},{"properties":{"value":{"type":"integer","format":"int64","maximum":9007199254741005,"default":9007199254740996}}}],"default":{"value":9007199254740999}}}`)
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
	child := node.objectFields[0].node
	first := get(get(get(v, "allOf").Items()[0], "properties"), "value")
	last := get(get(get(v, "allOf").Items()[1], "properties"), "value")
	p := child.numericAggregate
	if steps < 4 || child.schema != last || child.pointer != pointer+"/allOf/1/properties/value" || p == nil || p.endpoints[0].source != get(first, "minimum") || p.endpoints[0].pointer != pointer+"/allOf/0/properties/value/minimum" || p.factor.source != get(first, "multipleOf") || p.endpoints[2].source != get(last, "maximum") {
		t.Fatal("lost exact original winning/bound/factor source or shared work", steps, p)
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
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("original AST mutated")
	}
}

func TestAllOfNumericSelectedChecksMatchOriginalConjunction(t *testing.T) {
	for _, tc := range []struct{ version, kind, format, first, second string }{
		{"3.1.0", "integer", "int64", `"minimum":9007199254740993,"multipleOf":3`, `"maximum":9007199254741005,"exclusiveMinimum":9007199254740993`},
		{"3.0.3", "integer", "int32", `"minimum":-10,"exclusiveMinimum":true,"multipleOf":3`, `"maximum":10,"exclusiveMaximum":true`},
		{"3.1.0", "number", "float", `"minimum":16777217,"exclusiveMinimum":16777216`, `"maximum":16777222,"exclusiveMaximum":16777221`},
		{"3.1.0", "number", "double", `"minimum":9007199254740993,"exclusiveMinimum":9007199254740992`, `"maximum":9007199254741000,"exclusiveMaximum":9007199254740999`},
	} {
		t.Run(tc.version+tc.format, func(t *testing.T) {
			input := `{"openapi":"` + tc.version + `","allOf":[{"properties":{"value":{"type":"` + tc.kind + `","format":"` + tc.format + `",` + tc.first + `}}},{"properties":{"value":{"type":"` + tc.kind + `","format":"` + tc.format + `",` + tc.second + `}}}]}`
			root, e := document.Load([]byte(input), document.JSON, "")
			if e != nil {
				t.Fatal(e)
			}
			// The document's OpenAPI version belongs to the analyzer, while
			// modelShape consumes only the original schema object.
			version := root
			schema, e := document.Load([]byte(strings.Replace(input, `"openapi":"`+tc.version+`",`, "", 1)), document.JSON, "")
			if e != nil {
				t.Fatal(e)
			}
			a := analyzer{root: version, options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}}
			s, err := a.modelShape(schema, "", 0, false)
			if err != nil {
				t.Fatal(err)
			}
			plans := map[string]*numericDefaultAggregate{}
			if _, _, err := a.ordinaryPrimitiveAggregatesWithDefaults(schema, s, true, plans); err != nil {
				t.Fatal(err)
			}
			plan := plans["value"]
			if plan == nil {
				t.Fatal("missing numeric plan")
			}
			for _, number := range strings.Fields(`-11 -10 -9 -3 0 3 9 10 11 16777216 16777217 16777218 16777220 16777221 16777222 9007199254740992 9007199254740993 9007199254740994 9007199254740996 9007199254740999 9007199254741000 9007199254741005 9007199254741006`) {
				valueRoot, _ := document.ParseObject([]byte(`{"value":`+number+`}`), document.MaxDocumentBytes)
				value := get(valueRoot, "value")
				scalar, _ := scalarMapping(tc.kind, s.Fields[0].Schema, s.Fields[0].Pointer)
				if _, err := scalarLiteral(scalar, value, "/value"); err != nil {
					continue
				}
				accepted := true
				for _, f := range s.Contributions {
					_, check, err := a.boundedPrimitiveType(f.Schema, f.Pointer, f.Depth)
					if err != nil {
						t.Fatal(err)
					}
					accepted = accepted && validateDefaultChecksAt(value, f.Schema, check, scalar, f.Pointer, "/value") == nil
				}
				if got := plan.validate(value, scalar, "/value") == nil; got != accepted {
					t.Fatal("selected original checks differ from full conjunction", number, got, accepted)
				}
			}
		})
	}
}

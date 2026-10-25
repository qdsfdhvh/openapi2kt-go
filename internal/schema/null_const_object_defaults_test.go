package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNullConstObjectDefaultOriginalLiteralAndValueBudget(t *testing.T) {
	for _, raw := range []string{`{"const":null}`, `{"type":"null","const":null}`, `{"const":null,"nullable":true}`} {
		schema, parse := document.ParseObject([]byte(raw), document.MaxDocumentBytes)
		if parse != nil {
			t.Fatal(parse)
		}
		before, _ := schema.MarshalJSON()
		a := analyzer{}
		node, failure := a.objectDefaultConstNode(schema, "/source/fixed")
		if failure != nil {
			t.Fatal(failure)
		}
		literal := get(schema, "const")
		if node.schema != schema || node.constValue != literal || node.pointer != "/source/fixed" || node.constKind != "null" {
			t.Fatal("original null literal authority lost")
		}
		if text, _ := node.value(literal, "/default/fixed"); text != "null" {
			t.Fatal("null literal became a String token", text)
		}
		wrong, parse := document.ParseObject([]byte(`{"value":"null"}`), document.MaxDocumentBytes)
		if parse != nil {
			t.Fatal(parse)
		}
		if _, failure := node.value(get(wrong, "value"), "/default/fixed"); failure == nil || failure.Code != "E_SCHEMA_INVALID" || failure.Pointer == nil || *failure.Pointer != "/default/fixed" {
			t.Fatal("wrong mapped value lost Pointer", failure)
		}
		node.constValue = get(wrong, "value")
		if _, failure := node.value(literal, "/default/fixed"); failure == nil {
			t.Fatal("const kind hint authorized non-null source literal")
		}
		node.constValue = literal
		after, _ := schema.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("inference inserted or replaced source nodes")
		}
	}
	value, parse := document.ParseObject([]byte(`{"fixed":null}`), document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	for _, overflow := range []string{"exact", "work", "bytes"} {
		work, bytes := maxNestedDefaultNodes-2, maxNestedDefaultBytes-len(`{"fixed":null}`)
		if overflow == "work" {
			work++
		}
		if overflow == "bytes" {
			bytes++
		}
		failure := nestedDefaultBudget(value, "/value", 0, &work, &bytes)
		if overflow == "exact" && (failure != nil || work != maxNestedDefaultNodes || bytes != maxNestedDefaultBytes) || overflow != "exact" && (failure == nil || failure.Code != "E_MODEL_LIMIT") {
			t.Fatal("null member bypassed whole-value budget", overflow, work, bytes, failure)
		}
	}
}

package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestStructuredConstFieldRetainsOriginalLiteralNodeAndSchemaDepth(t *testing.T) {
	for _, input := range []string{`{"const":{"$ref":"data","type":"data","const":[null,true,1e2]}}`, `{"type":"array","const":[{"enum":[null]},"quoted"]}`} {
		root, err := document.Load([]byte(input), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		original := get(root, "const")
		value, failure := constantProfile("field", "field", root, "/original/~0~1source", 48, true)
		if failure != nil || !structuredConstant(*value) || get(root, "const") != original {
			t.Fatal(value, failure)
		}
		if _, failure := constantProfile("field", "field", root, "/original/~0~1source", 49, true); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("Structured body reset source depth", failure)
		}
		if _, failure := constantProfile("field", "field", root, "/original/~0~1source", 48, false); failure == nil {
			t.Fatal("Structured body permission leaked to a scalar/root planner")
		}
		encoded, marshalErr := root.MarshalJSON()
		if marshalErr != nil || string(encoded) != input {
			t.Fatal("Structured literal or schema rewritten", marshalErr)
		}
	}
}

package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestExplicitNullLiteralGrammarRetainsOriginalVocabularyAndTree(t *testing.T) {
	for _, input := range []string{`{"type":"null","enum":[null]}`, `{"type":"null","const":null}`} {
		root, e := document.Load([]byte(input), document.JSON, "")
		if e != nil {
			t.Fatal(e)
		}
		if has(root, "enum") {
			original := get(root, "enum").Items()[0]
			name, failure := nullOnlyEnumType(root, "/original")
			if failure != nil || name != "kotlinx.serialization.json.JsonNull" || get(root, "enum").Items()[0] != original {
				t.Fatal(name, failure)
			}
		} else {
			original := get(root, "const")
			value, failure := constant("value", "value", root, "/original", 48)
			if failure != nil || value.Type.Name != "kotlinx.serialization.json.JsonNull" || value.Initializer != value.Type.Name || get(root, "const") != original {
				t.Fatal(value, failure)
			}
			if _, failure := constant("value", "value", root, "/original", 49); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
				t.Fatal("Original const source budget", failure)
			}
		}
		encoded, marshalErr := root.MarshalJSON()
		if marshalErr != nil || string(encoded) != input {
			t.Fatal("Null source was rewritten", marshalErr)
		}
	}
	root, e := document.Load([]byte(`{"type":"null","enum":[null,false]}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, failure := nullOnlyEnumType(root, "/original/~0~1source"); failure == nil || failure.Code != "E_SCHEMA_INVALID" || *failure.Pointer != "/original/~0~1source/enum/1" {
		t.Fatal("Original vocabulary index/pointer lost", failure)
	}
}

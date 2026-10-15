package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestInferredNullLiteralsRetainOriginalNodesAndNullableIntent(t *testing.T) {
	for _, input := range []string{`{"const":null}`, `{"enum":[null]}`, `{"nullable":true,"enum":[null]}`, `{"nullable":false,"const":null}`} {
		root, err := document.Load([]byte(input), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		originalConst, originalEnum := get(root, "const"), get(root, "enum")
		kind, nullable, failure := schemaType(root, "/original/~0~1source")
		flag, _ := get(root, "nullable").Bool()
		if failure != nil || kind != "null" || nullable != flag || !inferredNullLiteral(root) || has(root, "type") {
			t.Fatal(kind, nullable, failure)
		}
		if originalEnum != nil {
			name, failure := nullOnlyEnumType(root, "/original/~0~1source")
			if failure != nil || name != "kotlinx.serialization.json.JsonNull" {
				t.Fatal(name, failure)
			}
		} else {
			value, failure := constant("value", "value", root, "/original/~0~1source", 48)
			if failure != nil || value.Type.Name != "kotlinx.serialization.json.JsonNull" {
				t.Fatal(value, failure)
			}
			if _, failure := constant("value", "value", root, "/original/~0~1source", 49); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
				t.Fatal("Original const source budget lost", failure)
			}
		}
		encoded, marshalErr := root.MarshalJSON()
		if marshalErr != nil || string(encoded) != input || get(root, "const") != originalConst || get(root, "enum") != originalEnum {
			t.Fatal("Null inference copied or rewrote original source", marshalErr)
		}
	}
}

func TestNullInferenceDoesNotReplaceExplicitTypesOrNativeProfiles(t *testing.T) {
	for _, input := range []string{`{"type":"string","enum":[null]}`, `{"type":"null","const":null}`, `{"enum":["ready",null],"nullable":true}`, `{"const":"null"}`, `{"enum":[]}`, `{"const":null,"format":""}`, `{"const":null,"x-native-type":"custom.Value"}`, `{"enum":[null],"x-kotlin-value-class":true}`} {
		root, err := document.Load([]byte(input), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		if inferredNullLiteral(root) {
			t.Fatal("Inferred null replaced an unrelated source profile", input)
		}
	}
}

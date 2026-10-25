package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestImplicitStringEnumNullUsesOriginalVocabularyIntent(t *testing.T) {
	for _, tc := range []struct {
		input    string
		kind     string
		nullable bool
	}{
		{`{"enum":[null,"ready","null"]}`, "", true},
		{`{"enum":["ready",null],"nullable":true}`, "", true},
		{`{"enum":["ready",null],"nullable":false}`, "", false},
		{`{"enum":["ready",null],"type":"string"}`, "string", false},
		{`{"enum":["ready",null],"type":["string","null"],"nullable":false}`, "string", true},
		{`{"enum":[null]}`, "null", false},
		{`{"enum":["ready"]}`, "", false},
		{`{"enum":["ready",null],"format":""}`, "", false},
		{`{"enum":["ready",null],"x-native-type":"custom.State"}`, "", false},
		{`{"enum":["ready",null],"x-kotlin-value-class":true}`, "", false},
		{`{"enum":["ready",null],"const":"ready"}`, "", false},
		{`{"enum":["ready",null],"oneOf":[]}`, "", false},
		{`{"enum":["ready",null],"items":{}}`, "", false},
		{`{"enum":["ready",null],"properties":{}}`, "object", false},
		{`{"enum":[null,"ready",1]}`, "", false},
	} {
		node, err := document.Load([]byte(tc.input), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		before, _ := node.MarshalJSON()
		vocabulary := get(node, "enum")
		member := vocabulary.Items()[0]
		kind, nullable, failure := schemaType(node, "/original")
		if failure != nil || kind != tc.kind || nullable != tc.nullable {
			t.Fatal(tc.input, kind, nullable, failure)
		}
		after, _ := node.MarshalJSON()
		if string(before) != string(after) || has(node, "nullable") != strings.Contains(tc.input, `"nullable"`) || vocabulary != get(node, "enum") || member != vocabulary.Items()[0] {
			t.Fatal("inference rewrote original source intent or vocabulary")
		}
	}
}

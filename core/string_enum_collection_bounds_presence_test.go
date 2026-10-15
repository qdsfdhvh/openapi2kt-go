package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumCollectionBoundsPatchAnnotationsAndEscapedSources(t *testing.T) {
	for _, field := range []string{
		`{"type":"array","nullable":true,"items":{"enum":["one","x"],"minLength":2},"default":["x"]}`,
		`{"type":"object","nullable":true,"additionalProperties":{"enum":["one","x"],"minLength":2},"default":{"a/~":"x"}}`,
	} {
		tail := "/default/0"
		if strings.Contains(field, "additionalProperties") {
			tail = "/default/a~1~0"
		}
		doc := []byte(`{"openapi":"3.1.0","paths":{"/a~b/c":{"patch":{"operationId":"update","requestBody":{"content":{"application/json":{"schema":{"properties":{"a/~":` + field + `}}}}},"responses":{"204":{}}}}}}`)
		enumBoundsDiagnostic(t, doc, true, "E_SCHEMA_INVALID", "/paths/~1a~0b~1c/patch/requestBody/content/application~1json/schema/properties/a~1~0"+tail)
		// Known schema keyword spellings are opaque successful dictionary data.
		if strings.Contains(field, "additionalProperties") {
			doc = []byte(strings.Replace(string(doc), `{"a/~":"x"}`, `{"$ref":"one","type":"one","a/~":"one"}`, 1))
			r := request(doc, "Dto")
			r.Options.Validation = true
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal(err)
			}
		}
	}
}

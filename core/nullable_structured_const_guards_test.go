package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableStructuredConstDoesNotHideOriginalGrammarBehindFalseViews(t *testing.T) {
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":["object","null"],"const":[]}`, "E_SCHEMA_INVALID", "/const"},
		{`{"const":{},"nullable":true,"default":null}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"const":[],"nullable":true,"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "/items"},
		{`{"const":{},"nullable":true,"format":"json"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"const":[],"nullable":true,"x-native-type":"custom.Value"}`, "E_SCHEMA_UNSUPPORTED", "/x-native-type"},
		{`{"const":{},"nullable":"bad"}`, "E_SCHEMA_INVALID", "/nullable"},
		{`{"const":{},"nullable":true,"title":false}`, "E_SCHEMA_INVALID", "/title"},
		{`{"const":{},"nullable":true,"minProperties":0}`, "E_SCHEMA_UNSUPPORTED", "/minProperties"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(schemas(`{"Value":`+tc.schema+`,"Alias":{"$ref":"#/components/schemas/Value","nullable":false}}`), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Narrowing erased source grammar", tc, validation, err)
			}
		}
	}
}

func TestNullableStructuredConstRetainsPinnedAndDiscardedSourceRejections(t *testing.T) {
	for _, input := range []string{
		`{"Holder":{"properties":{"fixed":{"type":"string"}},"additionalProperties":{"const":{},"nullable":true}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":{},"nullable":true}}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":{"type":"array","items":{"const":{},"nullable":true,"default":null}}}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if strings.Contains(input, `"Choice":`) {
			assertPinnedStructuredConstGraduation(t, got, err)
			continue
		}
		if strings.Contains(input, `"additionalProperties":{"const":`) {
			assertDiscardedStructuredConstGraduation(t, got, err)
			continue
		}
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Nullable value role leaked or discarded source stopped validating", input, err)
		}
	}
}

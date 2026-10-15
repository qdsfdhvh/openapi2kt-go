package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSpecialScalarGuardsAndOwnership(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"string","format":false}`, "E_SCHEMA_INVALID", ""},
		{`{"type":"string","format":"unsupported-format"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"type":"string","format":"date-time","pattern":"x"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"type":"string","format":"binary","maxLength":10}`, "E_SCHEMA_UNSUPPORTED", "/maxLength"},
		{`{"type":"string","x-native-type":false}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"type":"string","x-native-type":""}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"type":"string","x-native-type":"org.example.Bad<Type>"}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"type":"string","x-native-type":"org.example.Type; injected"}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"type":"string","x-native-type":"org.example.class"}`, "E_SCHEMA_INVALID", "/x-native-type"},
		{`{"type":"string","x-native-type":"bigint","pattern":"x"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"type":"null","format":"unknown"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"type":"null","minimum":0}`, "E_SCHEMA_UNSUPPORTED", "/minimum"},
		{`{"type":"string","format":"date","const":"2024-02-30"}`, "E_SCHEMA_INVALID", "/const"},
		{`{"type":"string","format":"date-time","const":"missing-zone"}`, "E_SCHEMA_INVALID", "/const"},
		{`{"type":"string","format":"date-time","const":123}`, "E_SCHEMA_INVALID", "/const"},
		{`{"x-native-type":"date","const":"2024-02-30T00:00:00Z"}`, "E_SCHEMA_INVALID", "/const"},
		{`{"x-native-type":"bigint","const":"9223372036854775808"}`, "E_SCHEMA_INVALID", "/const"},
		{`{"type":"string","format":"binary","const":"ambiguous-bytes"}`, "E_SCHEMA_UNSUPPORTED", "/const"},
		{`{"x-native-type":"org.example.scalarvalues.Money","const":"12.50"}`, "E_SCHEMA_UNSUPPORTED", "/const"},
		{`{"type":"string","format":"date","enum":["day"],"pattern":"x"}`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`{"type":"array","items":{"type":"string","format":"date-time","pattern":"x"}}`, "E_SCHEMA_UNSUPPORTED", "/items/pattern"},
	} {
		failure(t, fieldDoc(tc.field), tc.code, "/components/schemas/Record/properties/value"+tc.tail)
	}
	for _, scalar := range []string{`{"type":"string","x-native-type":"org.example.types.Money"}`, `{"type":"string","format":"date-time"}`, `{"type":"string","format":"binary"}`} {
		r := report(t, fieldDoc(scalar), "Dto")
		if len(r.Artifacts.Files) != 1 || strings.Contains(source(t, r, "RecordDto"), "LongAsStringSerializer") {
			t.Fatal("unneeded scalar helper", r)
		}
	}
	// A bigint alias is not an annotated field and does not own a helper.
	r := report(t, schemas(`{"Big":{"type":"string","x-native-type":"bigint"}}`), "")
	if len(r.Artifacts.Files) != 1 || !strings.Contains(source(t, r, "Big"), "= kotlin.Long") {
		t.Fatal(r)
	}
	failure(t, schemas(`{"LongAsStringSerializer":{"type":"object","properties":{"big":{"type":"string","x-native-type":"bigint"}}}}`), "E_NAME_COLLISION", "/components/schemas/LongAsStringSerializer")
	// Map/list bigint elements deliberately have no direct-field serializer.
	r = report(t, fieldDoc(`{"type":"object","additionalProperties":{"type":"string","x-native-type":"bigint"}}`), "")
	if len(r.Artifacts.Files) != 1 || strings.Contains(source(t, r, "Record"), "LongAsStringSerializer") {
		t.Fatal(r)
	}
	// Primitive unions retain their established kind carrier; special string
	// formats do not silently retarget the accepted String payload API.
	r = report(t, fieldDoc(`{"oneOf":[{"type":"string","format":"date"},{"type":"integer"}]}`), "")
	if !strings.Contains(source(t, r, "Record"), "public val value: kotlin.String") {
		t.Fatal(source(t, r, "Record"))
	}
}

func TestSpecialScalarSlots(t *testing.T) {
	for _, n := range []int{120, 121} {
		fields, required := []string{}, []string{}
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string","x-native-type":"bigint"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		r, err := core.Generate(request(schemas(`{"Wide":{"type":"object","properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), ""))
		if n == 120 {
			if err != nil || len(r.Files) != 2 {
				t.Fatal(n, r, err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Files) != 0 {
			t.Fatal(n, r, err)
		}
	}
}

package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestModelDefaultsRetainRequestPresenceAndOriginalValidation(t *testing.T) {
	input := httpDoc(`{"operationId":"defaults","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"required":{"type":["string","null"],"default":"ready"},"patch":{"type":["string","null"],"default":"ready"},"optional":{"type":"string","default":"ready"}},"required":["required"]}}}},"responses":{"204":{}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "DefaultsInputDto")
	for _, want := range []string{`val required: kotlin.String?,`, `Patch<kotlin.String> = org.example.models.models.Patch.Absent,`, `val optional: kotlin.String? = "ready",`} {
		if !strings.Contains(text, want) {
			t.Fatal("presence contract lost", want, text)
		}
	}
	requiredNull := request(schemas(`{"Record":{"properties":{"value":{"type":["string","null"],"default":"ready"}},"required":["value"]}}`), "Dto")
	accepted, e := core.GenerateWithDiagnostics(requiredNull)
	if e != nil || !strings.Contains(source(t, accepted, "RecordDto"), "val value: kotlin.String?,") {
		t.Fatal("required nullable default changed constructor presence", accepted, e)
	}
	for _, field := range []string{
		`{"type":"string","default":"x","minLength":2}`,
		`{"type":"string","default":"b","pattern":"^a$"}`,
		`{"type":"integer","default":3,"multipleOf":2}`,
		`{"type":"number","format":"float","default":0.300000004,"exclusiveMinimum":0.300000001}`,
		`{"type":"number","default":0.5,"multipleOf":0.25}`,
	} {
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		if e == nil || len(r.Artifacts.Files) != 0 || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default" {
			t.Fatal("default validation lost", field, r, e)
		}
	}
}

func TestModelDefaultsCommonFieldsOverridesAndOptimization(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(schemas(`{"A":{"properties":{"value":{"type":"string","default":"first"}}},"B":{"properties":{"value":{"type":"string","default":"second"}}},"C":{"properties":{"value":{"type":"string","default":"first"}}},"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"type":"string","default":"first"}}},{"properties":{"kind":{"const":"b"},"value":{"type":"string","default":"second"}}}]},"Override":{"allOf":[{"properties":{"value":{"type":"string","default":"first"}}},{"properties":{"value":{"type":"string","default":"second"}}}]}}`), "Dto")
		rq.Options.ModelStyle = style
		rq.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(source(t, got, "BDto"), "typealias") || !strings.Contains(source(t, got, "CDto"), "typealias CDto = org.example.models.models.ADto") {
			t.Fatal("different default merged or identical rejected", got)
		}
		for _, value := range []string{` = "first",`, ` = "second",`} {
			if !strings.Contains(source(t, got, "ChoiceDto"), value) {
				t.Fatal("variant default lost", value, got)
			}
		}
		if !strings.Contains(source(t, got, "OverrideDto"), ` = "second",`) && !strings.Contains(source(t, got, "OverrideDto"), "= org.example.models.models.BDto") {
			t.Fatal("later default override lost", got)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("default output is nondeterministic", err)
		}
	}
	bad := schemas(`{"Record":{"allOf":[{"properties":{"value":{"type":"integer","default":"invalid"}}},{"properties":{"value":{"type":"string","default":"ready"}}}]}}`)
	failure(t, bad, "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/0/properties/value/default")
}

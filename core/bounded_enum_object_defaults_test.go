package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBoundedEnumObjectDefaults(t *testing.T) {
	for _, metadata := range []string{"", `,"format":"date"`, `,"x-native-type":"org.example.UnusedProvider"`} {
		field := `{"type":"string","enum":["one","x","toolong"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"` + metadata + `}`
		shape := `{"properties":{"state":` + field + `,"words":{"type":"array","items":` + field + `},"labels":{"additionalProperties":` + field + `}},"required":["state"]}`
		value := `{"state":"one","words":["one"],"labels":{"a/~":"one"}}`
		for _, body := range []string{
			`{"Value":` + strings.TrimSuffix(shape, "}") + `,"default":` + value + `}}`,
			`{"Value":` + shape + `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":` + value + `}}}}`,
			`{"Holder":{"properties":{"value":` + strings.TrimSuffix(shape, "}") + `,"default":` + value + `}}}}`,
			`{"Value":` + shape + `,"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"default":[` + value + `]}}}}`,
			`{"Holder":{"properties":{"rows":{"type":"array","items":` + shape + `,"default":[` + value + `]}}}}`,
			`{"Value":{"allOf":[{"properties":{"state":` + field + `}},{"properties":{"other":{"type":"string"}}}],"default":{"state":"one","other":"given"}}}`,
		} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("bounded lexical enum object annotation", metadata, body, err)
			}
			if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("bounded enum object default nondeterminism", err)
			}
		}
	}
}

func TestBoundedEnumObjectDefaultsStrictValues(t *testing.T) {
	leaf := `{"type":"string","format":"date","enum":["one","two","x","UPPER","toolong"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}`
	for _, tc := range []struct{ value, tail string }{
		{`{"state":"x"}`, "/state"},
		{`{"state":"UPPER"}`, "/state"},
		{`{"state":"toolong"}`, "/state"},
		{`{"state":"future"}`, "/state"},
		{`{"state":null}`, "/state"},
		{`{"state":"one","words":["x"]}`, "/words/0"},
		{`{"state":"one","labels":{"z/~":"UPPER","a":"one"}}`, "/labels/z~1~0"},
	} {
		body := `{"Value":{"properties":{"state":` + leaf + `,"words":{"type":"array","items":` + leaf + `},"labels":{"additionalProperties":` + leaf + `}},"required":["state"],"default":` + tc.value + `}}`
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Value/default"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("bounded object annotation lost strict vocabulary/check/Pointer", tc, err)
		}
	}
}

func TestBoundedEnumObjectDefaultsIndependentOwners(t *testing.T) {
	for _, outer := range []string{`{"state":"two"}`, `{"state":"one"}`} {
		body := `{"Value":{"properties":{"state":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"default":"one"}},"required":["state"],"default":` + outer + `},"Alias":{"$ref":"#/components/schemas/Value","default":{"state":"two"}},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}}}`
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("independent bounded field/root/alias own annotations", err)
		}
		if !strings.Contains(source(t, got, "AliasDto"), "public fun `AliasDto default`()") || !strings.Contains(source(t, got, "MaybeDto"), "public fun `MaybeDto default`()") || !strings.Contains(source(t, got, "HolderDto"), "ValueDto? = null") {
			t.Fatal("bounded object factory ownership or no-inheritance changed")
		}
	}
}

func TestObjectDefaultAnnotatedTargetIndependence(t *testing.T) {
	for _, field := range []string{`{"type":"string","default":"inner"}`, `{"type":"string","format":"date","enum":["one","two"],"minLength":2,"default":"one"}`} {
		root, outer := `"root"`, `"outer"`
		if strings.Contains(field, `"enum"`) {
			root, outer = `"one"`, `"two"`
		}
		body := `{"Value":{"properties":{"state":` + field + `},"required":["state"],"default":{"state":` + root + `}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":{"state":` + outer + `}},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"},"default":[{"state":` + outer + `}]},"unannotated":{"$ref":"#/components/schemas/Value"}}}}`
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("independent target and consuming object annotations", field, err)
		}
	}
}

func TestAnnotatedObjectTargetFormerGuards(t *testing.T) {
	// The original accepted target grammar now retains each independent default.
	// Keep former refusal inputs verbatim rather than silently removing coverage.
	for _, body := range []string{
		`{"Inner":{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":"t"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"a":"x"}}}}}`,
		`{"Item":{"type":"object","properties":{"name":{"type":"string"}},"default":{"name":"t"}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"name":"x"}]}}}}`,
		`{"Item":{"type":"object","properties":{"name":{"type":"string"}},"default":{"name":"t"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[{"name":"x"}]}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("independent original target/consumer annotations did not migrate", checked, body, err)
			}
		}
	}
}

func TestAnnotatedObjectAliasConsumerDefaults(t *testing.T) {
	rq := request(schemas(`{"Value":{"properties":{"state":{"type":"string","format":"date","enum":["one","two"],"minLength":2}},"required":["state"],"default":{"state":"one"}},"Alias":{"$ref":"#/components/schemas/Value","default":{"state":"two"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":{"state":"one"}},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":[{"state":"one"}]}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("independent alias factory and consuming object annotations", err)
	}
}

func TestBoundedEnumObjectDefaultsSourceBeforeEmptyOrNull(t *testing.T) {
	for _, outer := range []string{`{}`, `null`} {
		for _, tc := range []struct{ field, code, tail string }{
			{`{"enum":["one"],"minLength":4,"maxLength":2}`, "E_SCHEMA_INVALID", "/maxLength"},
			{`{"enum":["one","one"],"minLength":1}`, "E_SCHEMA_INVALID", "/enum/1"},
			{`{"enum":["one","x"],"minLength":2,"default":"x"}`, "E_SCHEMA_INVALID", "/default"},
		} {
			rq := request(schemas(`{"Value":{"type":"object","nullable":true,"properties":{"state":`+tc.field+`},"default":`+outer+`}}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Value/properties/state"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("empty/null factory hid original bounded enum source", outer, tc, err)
			}
		}
	}
	for _, field := range []string{
		`{"type":"array","items":{"enum":["one"],"minLength":2,"default":"one"}}`,
	} {
		rq := request(schemas(`{"Value":{"properties":{"state":{"enum":["one"],"minLength":2},"bad":`+field+`},"default":{"state":"one"}}}`), "Dto")
		rq.Options.Validation = true
		inlineEnumChildGraduation(t, rq, `,"default":"one"`, `,"default":"other"`, "/components/schemas/Value/properties/bad/items/default")
	}
}

func TestBoundedEnumObjectDefaultsPatchAnnotation(t *testing.T) {
	for _, required := range []bool{false, true} {
		for _, value := range []string{"one", "x"} {
			body := `{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":true,"default":{"state":"` + value + `"}}}`
			if required {
				body += `,"required":["value"]`
			}
			body += `}`
			doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Target":{"properties":{"state":{"type":"string","format":"date","enum":["one","x"],"minLength":2}},"required":["state"],"default":{"state":"one"}},"Alias":{"$ref":"#/components/schemas/Target","default":{"state":"one"}}}},"paths":{"/update":{"patch":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + body + `}}},"responses":{"204":{}}}}}}`)
			rq := request(doc, "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if value == "one" {
				if err != nil {
					t.Fatal("valid bounded object request annotation", required, err)
				}
				continue
			}
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/paths/~1update/patch/requestBody/content/application~1json/schema/properties/value/default/state" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("required/Patch suppression hid bounded object annotation", required, err)
			}
		}
	}
}

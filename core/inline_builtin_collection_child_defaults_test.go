package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Preserve each former refusal as an unchanged positive input while ensuring
// that an invalid annotation still fails at the original child Pointer.
func inlineBuiltinChildGraduation(t *testing.T, rq core.GenerationRequest, annotation, invalid, pointer string) {
	t.Helper()
	document := string(rq.Document)
	if !strings.Contains(document, annotation) {
		t.Fatal("missing original child annotation", annotation)
	}
	rq.Document = []byte(strings.Replace(document, annotation, "", 1))
	control, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("unannotated collection rejected", err)
	}
	rq.Document = []byte(document)
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(got, control) {
		t.Fatal("independent child annotation changed output", err)
	}
	rq.Document = []byte(strings.Replace(document, annotation, invalid, 1))
	report, err := core.GenerateWithDiagnostics(rq)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
		*ds[0].Pointer != pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatal("invalid child must fail at its original default", err)
	}
}

func TestInlineBuiltinCollectionChildDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, field, valid, invalid, pointer string
		checked                              bool
	}{
		{"String list", `{"type":"array","items":{"type":"string"@},"default":[]}`, `"ready"`, `true`, "/items/default", false},
		{"Int list", `{"type":"array","items":{"type":"integer"@}}`, `1`, `2147483648`, "/items/default", false},
		{"Long map", `{"additionalProperties":{"type":"integer","format":"int64"@},"default":{}}`, `9223372036854775807`, `1.0`, "/additionalProperties/default", false},
		{"Float list", `{"type":"array","items":{"type":"number","format":"float"@}}`, `0.5`, `1e300`, "/items/default", false},
		{"Double map", `{"additionalProperties":{"type":"number"@}}`, `1.25`, `"1.25"`, "/additionalProperties/default", false},
		{"Boolean list", `{"type":"array","items":{"type":"boolean"@}}`, `true`, `"true"`, "/items/default", false},
		{"nullable item", `{"type":"array","items":{"type":["integer","null"]@}}`, `null`, `false`, "/items/default", false},
		{"checked string", `{"type":"array","items":{"type":"string","minLength":2@},"default":[]}`, `"ok"`, `"x"`, "/items/default", true},
		{"checked integer", `{"additionalProperties":{"type":"integer","minimum":2@},"default":{}}`, `2`, `1`, "/additionalProperties/default", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Holder":{"properties":{"value":` + tc.field + `}}}`
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					if tc.checked && !checked {
						continue
					}
					rq := request(schemas(strings.Replace(body, "@", "", 1)), "Dto")
					rq.Options.ModelStyle = style
					rq.Options.Validation = checked
					control, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("unannotated collection rejected", err)
					}
					rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.valid, 1))
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, control) {
						t.Fatal("independent child annotation changed output", style, checked, err)
					}
					rq.Document = schemas(strings.Replace(body, "@", `,"default":`+tc.invalid, 1))
					report, err := core.GenerateWithDiagnostics(rq)
					ds, ok := err.(core.Diagnostics)
					want := "/components/schemas/Holder/properties/value" + tc.pointer
					if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil ||
						*ds[0].Pointer != want || !reflect.DeepEqual(report, core.GenerationReport{}) {
						t.Fatal("invalid child must fail at its original default", style, checked, err)
					}
				}
			}
		})
	}
}

func TestInlineBuiltinCollectionChildDefaultBoundaries(t *testing.T) {
	for _, body := range []string{
		`{"Rows":{"type":"array","items":{"type":"string","default":"x"}}}`,
		`{"Rows":{"additionalProperties":{"type":"integer","default":1}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"type":"string","default":"x"}}}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","prefixItems":[{"type":"string","default":"x"}],"items":false}}}}`,
		`{"Holder":{"properties":{"value":{"type":"object","properties":{"fixed":{"type":"string"}},"additionalProperties":{"type":"string","default":"x"}}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("unadmitted role gained child permission", checked, body, err)
			}
		}
	}
	body := `{"Holder":{"properties":{"a/~":{"type":"array","items":{"type":"integer","default":"bad"},"default":[]}}}}`
	carrierDefaultFailure(t, schemas(body), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/a~1~0/items/default")
	formerRefusal := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","default":"x"}}},"default":{}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(formerRefusal), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":"x"`, `,"default":true`,
			"/components/schemas/Holder/properties/value/items/default")
	}
	dateRefusal := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","format":"date","default":"2024-02-29"}}}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(dateRefusal), "Dto")
		rq.Options.Validation = checked
		inlineBuiltinChildGraduation(t, rq, `,"default":"2024-02-29"`, `,"default":"2023-02-29"`,
			"/components/schemas/Holder/properties/value/items/default")
	}
}

func TestInlineBuiltinCollectionChildDefaultHierarchyBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, field, annotation, suffix string
	}{
		{"List", `{"type":"array","items":{"type":"string"@}}`, `,"default":"x"`, "/items/default"},
		{"Map", `{"type":"object","additionalProperties":{"type":"integer"@}}`, `,"default":1`, "/additionalProperties/default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"leaf":"Leaf"}}},"Leaf":{"type":"object","properties":{"kind":{"type":"string","const":"leaf"},"rows":` + tc.field + `}}}`
			for _, checked := range []bool{false, true} {
				rq := request(schemas(strings.Replace(body, "@", "", 1)), "Dto")
				rq.Options.Validation = checked
				if _, err := core.GenerateWithDiagnostics(rq); err != nil {
					t.Fatal("unannotated hierarchy rejected", checked, err)
				}
				carrierDefaultFailure(t, schemas(strings.Replace(body, "@", tc.annotation, 1)), checked,
					"E_SCHEMA_UNSUPPORTED", "/components/schemas/Leaf/properties/rows"+tc.suffix)
			}
		})
	}
}

func TestInlineBuiltinCollectionChildDefaultNullOuterStillValidates(t *testing.T) {
	for _, tc := range []struct {
		field, suffix string
	}{
		{`{"type":["array","null"],"items":{"type":"integer","default":"bad"},"default":null}`, "/items/default"},
		{`{"type":["object","null"],"additionalProperties":{"type":"integer","default":"bad"},"default":null}`, "/additionalProperties/default"},
	} {
		body := `{"Holder":{"properties":{"value":` + tc.field + `},"required":["value"]}}`
		for _, checked := range []bool{false, true} {
			carrierDefaultFailure(t, schemas(body), checked, "E_SCHEMA_INVALID",
				"/components/schemas/Holder/properties/value"+tc.suffix)
		}
	}
}

func TestInlineBuiltinCollectionChildDefaultBudget(t *testing.T) {
	for _, size := range []int{32750, 32768} {
		word := strings.Repeat("a", size)
		body := `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","default":"` + word + `"}}}}}`
		report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
		if size == 32750 && err != nil {
			t.Fatal("default below the byte limit rejected", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
			t.Fatal("default escaped the byte limit", err)
		}
	}
}

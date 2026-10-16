package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const composedUnionParts = `"Choice":{"oneOf":[
 {"type":"object","required":["kind","body"],"properties":{"kind":{"const":"note"},"body":{"type":"string"}}},
 {"type":"object","required":["kind","count"],"properties":{"kind":{"const":"count"},"count":{"type":"integer"}}}]},
 "Mode":{"anyOf":[
 {"type":"object","required":["mode","fast"],"properties":{"mode":{"const":"fast"},"fast":{"type":"boolean"}}},
 {"type":"object","required":["mode","slow"],"properties":{"mode":{"const":"slow"},"slow":{"type":"boolean"}}}]},
 "Product":{"allOf":[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Mode"}]}`

func composedHolderDocument(field string) []byte {
	return schemas(`{` + composedUnionParts + `,"Holder":{"properties":{"v":` + field + `}}}`)
}

func TestComposedUnionFieldDefaults(t *testing.T) {
	inline := `{"allOf":[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Mode"}],"default":{"kind":"note","mode":"fast","fast":true,"body":"x"}}`
	got, err := core.GenerateWithDiagnostics(request(composedHolderDocument(inline), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "HolderDto")
	if !strings.Contains(text, "public val v: V? = `openapi-kt model default org example models models HolderDto V`(\"{\\\"body\\\":\\\"x\\\",\\\"fast\\\":true,\\\"kind\\\":\\\"note\\\",\\\"mode\\\":\\\"fast\\\"}\")") {
		t.Fatal("inline composed default helper", text)
	}
	referenced := `{"$ref":"#/components/schemas/Product","default":{"kind":"count","mode":"slow","slow":false,"count":7}}`
	got, err = core.GenerateWithDiagnostics(request(composedHolderDocument(referenced), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text = source(t, got, "HolderDto")
	if !strings.Contains(text, "org.example.models.models.ProductDto? = `openapi-kt model default org example models models ProductDto`(") {
		t.Fatal("referenced composed default helper", text)
	}
	// A required field validates the annotation and keeps its constructor
	// argument; an invalid annotation still fails generation.
	requiredDoc := strings.Replace(string(composedHolderDocument(inline)), `{"properties"`, `{"required":["v"],"properties"`, 1)
	got, err = core.GenerateWithDiagnostics(request([]byte(requiredDoc), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(source(t, got, "HolderDto"), " = `openapi-kt model default") {
		t.Fatal("required composed default emitted an initializer", source(t, got, "HolderDto"))
	}
	badRequired := strings.Replace(requiredDoc, `"mode":"fast"`, `"mode":"bogus"`, 1)
	carrierDefaultFailure(t, []byte(badRequired), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
}

func TestComposedUnionDefaultSelectionMirrorsDispatch(t *testing.T) {
	inline := `{"allOf":[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Mode"}],"default":`
	for name, tc := range map[string]struct {
		value string
		code  string
		tail  string
	}{
		"unknown combination": {`{"kind":"note","mode":"bogus","fast":true,"body":"x"}`, "E_SCHEMA_INVALID", ""},
		"missing pin":         {`{"kind":"note","body":"x","fast":true}`, "E_SCHEMA_INVALID", ""},
		"unknown key":         {`{"kind":"note","mode":"fast","fast":true,"body":"x","extra":1}`, "E_SCHEMA_INVALID", "/extra"},
		"missing required":    {`{"kind":"note","mode":"fast","body":"x"}`, "E_SCHEMA_INVALID", ""},
		"wrong leaf kind":     {`{"kind":"note","mode":"fast","fast":"yes","body":"x"}`, "E_SCHEMA_INVALID", "/fast"},
		"non object":          {`["note"]`, "E_SCHEMA_INVALID", ""},
	} {
		t.Run(name, func(t *testing.T) {
			if !json.Valid([]byte(tc.value)) {
				t.Fatal("invalid test value")
			}
			carrierDefaultFailure(t, composedHolderDocument(inline+tc.value+`}`), true, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
			carrierDefaultFailure(t, composedHolderDocument(inline+tc.value+`}`), false, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
		})
	}
	// Null defaults follow the existing object-union role: inline field unions
	// have no null variant, a nullable root component does.
	carrierDefaultFailure(t, composedHolderDocument(inline+`null}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	nullableComponent := schemas(`{` + composedUnionParts + `,"Nullable":{"allOf":[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Mode"}],"nullable":true},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Nullable","default":null}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(nullableComponent, "Dto")); err != nil {
		t.Fatal("nullable composed component null default", err)
	}
	// Canonical byte budget applies to composed defaults as well.
	oversize := `"` + strings.Repeat("a", 32767) + `"`
	carrierDefaultFailure(t, composedHolderDocument(inline+`{"kind":"note","mode":"fast","fast":true,"body":`+oversize+`}}`), true, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/body")
}

func TestComposedUnionDefaultExclusions(t *testing.T) {
	// An ordinary allOf merge keeps its historical default rejection.
	carrierDefaultFailure(t, schemas(`{`+composedUnionParts+`,"Plain":{"allOf":[{"type":"object","properties":{"a":{"type":"string"}}}]},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Plain"}],"default":{"a":"x"}}}}}`), true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/properties/v/default")
	// Reference chains to a composed component stay excluded.
	input := schemas(`{` + composedUnionParts + `,"Chain":{"$ref":"#/components/schemas/Product"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Chain","default":{"kind":"note","mode":"fast","fast":true,"body":"x"}}}}}`)
	r := request(input, "Dto")
	report, err := core.GenerateWithDiagnostics(r)
	if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatalf("composed reference chain default leaked: %v", err)
	}
}

func TestComposedUnionDiscardedContributionDefaults(t *testing.T) {
	// A later override discards the composed contribution; its annotation still
	// validates and its scratch plan must not leak declarations or warnings.
	valid := schemas(`{` + composedUnionParts + `,"Holder":{"allOf":[{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Mode"}],"default":{"kind":"note","mode":"fast","fast":true,"body":"x"}}}},{"properties":{"v":{"type":"object","properties":{"note":{"type":"string"}}}}}]}}`)
	for _, checked := range []bool{false, true} {
		r := request(valid, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("valid discarded composed default rejected", err)
		}
		text := source(t, got, "HolderDto")
		if strings.Contains(text, "openapi-kt model default") || strings.Contains(text, "Validation") {
			t.Fatal("scratch composed declarations leaked", text)
		}
		// The component unions in the document warn on their own; the discarded
		// contribution must not add warnings beyond that baseline.
		baseline := []byte(strings.Replace(string(valid), `,"default":{"kind":"note","mode":"fast","fast":true,"body":"x"}`, ``, 1))
		quiet, err := core.GenerateWithDiagnostics(request(baseline, "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Warnings, quiet.Warnings) {
			t.Fatal("discarded composed default changed warnings", got.Warnings, quiet.Warnings)
		}
		invalid := []byte(strings.Replace(string(valid), `"mode":"fast","fast":true`, `"mode":"bogus","fast":true`, 1))
		carrierDefaultFailure(t, invalid, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default")
	}
}

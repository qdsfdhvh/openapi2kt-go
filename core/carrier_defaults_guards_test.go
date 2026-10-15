package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func carrierDefaultDocument(field string, required bool) []byte {
	requirement := ""
	if required {
		requirement = `,"required":["v"]`
	}
	return schemas(`{"Holder":{"properties":{"v":` + field + `}` + requirement + `}}`)
}

func carrierDefaultFailure(t *testing.T, input []byte, checked bool, code, pointer string) {
	t.Helper()
	if !json.Valid(input) {
		t.Fatal("invalid regression document")
	}
	r := request(input, "Dto")
	r.Options.Validation = checked
	report, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != code || ds[0].Pointer == nil || *ds[0].Pointer != pointer || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatalf("want %s at %s (validation=%v), got %v", code, pointer, checked, err)
	}
}

func TestRawJSONDefaultNullability(t *testing.T) {
	for name, shape := range map[string]string{
		"free":  ``,
		"empty": `"type":"object",`,
		"open":  `"type":"object","additionalProperties":true,`,
	} {
		for _, required := range []bool{false, true} {
			for _, checked := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/required=%v/checked=%v", name, required, checked), func(t *testing.T) {
					for _, intent := range []string{"", `"nullable":false,`} {
						carrierDefaultFailure(t, carrierDefaultDocument(`{`+shape+intent+`"default":null}`, required), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
					}
					r := request(carrierDefaultDocument(`{`+shape+`"nullable":true,"default":null}`, required), "Dto")
					r.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal("explicit nullable raw default", err)
					}
					text := source(t, got, "HolderDto")
					if strings.Contains(text, " = null") == required {
						t.Fatal("required/default presence changed", text)
					}
				})
			}
		}
	}
	// The 3.1 type spelling independently grants nullability.
	if _, err := core.GenerateWithDiagnostics(request(carrierDefaultDocument(`{"type":["object","null"],"additionalProperties":true,"default":null}`, false), "Dto")); err != nil {
		t.Fatal(err)
	}
}

func TestCarrierDefaultByteBudgets(t *testing.T) {
	oversize := `"` + strings.Repeat("a", 32767) + `"`
	objectUnion := `"oneOf":[{"type":"object","required":["kind","s"],"properties":{"kind":{"enum":["a"]},"s":{"type":"string"}}},{"type":"object","required":["kind","n"],"properties":{"kind":{"enum":["b"]},"n":{"type":"integer"}}}]`
	for name, tc := range map[string]struct{ field, tail string }{
		"raw scalar":   {`{"default":` + oversize + `}`, ""},
		"raw member":   {`{"additionalProperties":true,"default":{"z/~":` + oversize + `,"a":0}}`, "/z~1~0"},
		"raw escaping": {`{"default":"` + strings.Repeat("<", 5462) + `"}`, ""},
		"primitive":    {`{"oneOf":[{"type":"string"},{"type":"integer"}],"default":` + oversize + `}`, ""},
		"object union": {`{` + objectUnion + `,"default":{"kind":"a","s":` + oversize + `}}`, "/s"},
	} {
		t.Run(name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				carrierDefaultFailure(t, carrierDefaultDocument(tc.field, false), checked, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default"+tc.tail)
			}
		})
	}
	for _, checked := range []bool{false, true} {
		// Referenced root unions must use the same value preflight.
		input := schemas(`{"V":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/V","default":` + oversize + `}}}}`)
		carrierDefaultFailure(t, input, checked, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default")
	}
	// Exact boundary: compact JSON includes both string quotes.
	for _, prefix := range []string{`{`, `{"oneOf":[{"type":"string"},{"type":"integer"}],`} {
		input := carrierDefaultDocument(prefix+`"default":"`+strings.Repeat("a", 32766)+`"}`, false)
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
			t.Fatal("32KiB boundary rejected", err)
		}
	}
}

func TestRawJSONDefaultNodeBudget(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Array root + 16,383 one-byte values is 16,384 nodes / 32,767 bytes.
		members := strings.Repeat("0,", 16382) + "0"
		r := request(carrierDefaultDocument(`{"default":[`+members+`]}`, false), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("16,384 node boundary rejected", err)
		}
		carrierDefaultFailure(t, carrierDefaultDocument(`{"default":[`+members+`,0]}`, false), checked, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/16383")
	}
}

func TestDiscardedUnionDefaultsUseScratchDeclarations(t *testing.T) {
	for _, checked := range []bool{false, true} {
		input := schemas(`{"Holder":{"allOf":[{"properties":{"v":{"oneOf":[{"type":"string","enum":["x"]},{"type":"integer"}],"default":"x"}}},{"properties":{"v":{"type":"string"}}}]}}`)
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("valid discarded union default rejected", err)
		}
		text := source(t, got, "HolderDto")
		if strings.Contains(text, "Validation") || strings.Contains(text, "sealed class") || strings.Contains(text, "openapi-kt model default") {
			t.Fatal("scratch declarations leaked", text)
		}
		for _, warning := range got.Warnings {
			if strings.HasPrefix(warning.Code, "W_PRIMITIVE_UNION") {
				t.Fatal("scratch warning leaked", warning)
			}
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"default":"x"`, `"default":"other"`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default")
	}
}

func carrierRequestDocument(field string, required bool) []byte {
	requirement := ""
	if required {
		requirement = `,"required":["v"]`
	}
	return []byte(`{"openapi":"3.1.0","paths":{"/p":{"post":{"operationId":"p","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"v":` + field + `}` + requirement + `}}}},"responses":{"204":{"description":"ok"}}}}}}`)
}

func TestCarrierDefaultsValidateBeforeSuppression(t *testing.T) {
	value := `"` + strings.Repeat("a", 32767) + `"`
	for _, checked := range []bool{false, true} {
		for _, field := range []string{`{"nullable":true,"default":` + value + `}`, `{"nullable":true,"oneOf":[{"type":"string"},{"type":"integer"}],"default":` + value + `}`} {
			for _, required := range []bool{false, true} {
				carrierDefaultFailure(t, carrierRequestDocument(field, required), checked, "E_MODEL_LIMIT", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default")
				carrierDefaultFailure(t, carrierDefaultDocument(field, required), checked, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default")
			}
			input := schemas(`{"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, input, checked, "E_MODEL_LIMIT", "/components/schemas/Holder/allOf/0/properties/v/default")
		}
		// Request projection nullability comes from the union, including null branches.
		carrierDefaultFailure(t, carrierRequestDocument(`{"oneOf":[{"type":"string"},{"type":"integer"}],"default":null}`, true), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default")
		r := request(carrierRequestDocument(`{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"null"}],"default":null}`, true), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("request projection null branch rejected", err)
		}
	}
}

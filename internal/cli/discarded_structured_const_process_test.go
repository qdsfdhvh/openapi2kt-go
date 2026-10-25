package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscardedStructuredConstNativeOriginalSourcesAndFailurePreservation(t *testing.T) {
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			for _, body := range []string{
				`{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{}}}}`,
				`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"}},"additionalProperties":{"const":{}}},{"properties":{"tag":{"const":"two"}}}]}}`,
				`{"Choice":{"oneOf":[{"allOf":[{"properties":{"payload":{"const":{"source":true}}}},{"properties":{"payload":{"type":"string"},"tag":{"const":"one"}}}]},{"properties":{"tag":{"const":"two"}}}]}}`,
			} {
				p := newProject(t)
				put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":`+body+`}}`))
				flags := []string{"--model-style", style}
				if checked {
					flags = append(flags, "--validation")
				}
				good := result(t, p.generate(t, flags...), 0)
				for _, w := range good.Warnings {
					if w.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
						t.Fatal("Native discarded literal leaked typed kind/body warning", good)
					}
				}
				for _, f := range good.Files {
					if filepath.Ext(f) == ".kt" {
						code := read(t, filepath.Join(p.Output, f))
						if bytes.Contains(code, []byte("openapi-kt object const")) || bytes.Contains(code, []byte("AdditionalValue")) {
							t.Fatal("Native scratch declaration escaped", f)
						}
					}
				}
				before := snapshot(t, p.Output)
				result(t, p.generate(t, append(flags, "--check")...), 0)
				result(t, p.generate(t, flags...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Discarded literal no-op/check changed owned bytes or attrs")
				}
				put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{},"default":{}}}}}}`))
				bad := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
				if bad.Diagnostics[0].Pointer == nil || *bad.Diagnostics[0].Pointer != "/components/schemas/Holder/additionalProperties/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("Discarded-source failure changed prior output", bad)
				}
			}
		}
	}
}

func TestDiscardedStructuredConstNativeValidateAndWarningPolicy(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{"$ref":"literal data"}}}}}}`))
	good := result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	denied := warningDenied(t, p.generate(t, "--deny-warnings"))
	if len(good.Warnings) != 1 || len(denied.Diagnostics) != 1 || denied.Diagnostics[0].Pointer == nil || *denied.Diagnostics[0].Pointer != "/components/schemas/Holder/additionalProperties" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Projection policy or old output changed", good, denied)
	}
	result(t, p.generate(t, "--deny-warnings", "--allow-warning=W_ADDITIONAL_PROPERTIES_APPROXIMATION", "--check"), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format=json"), 0)
	warningDenied(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format=json", "--deny-warnings"))
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Validate/policy touched owned output")
	}
}

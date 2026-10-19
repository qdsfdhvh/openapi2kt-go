package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

// A dedicated request model is the only place a read-only field can be projected:
// the server owns its value, so a request omits it unless the caller overrides.
func TestReadOnlyFieldProjectsToPatchAbsentInRequestModels(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/r":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["id","name"],"properties":{"id":{"type":"string","readOnly":true},"name":{"type":"string"},"note":{"type":"string","readOnly":true}}}}}},"responses":{"204":{}}}}}}`)
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(input, "Dto")
			r.Options.Validation, r.Options.ModelStyle = validation, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(validation, style, err)
			}
			text := source(t, got, "SubmitInputDto")
			for _, want := range []string{
				// Required and read-only: still absent by default, still settable.
				"public val id: org.example.models.models.Patch<kotlin.String> = org.example.models.models.Patch.Absent,",
				// Required and writable: unchanged required constructor argument.
				"public val name: kotlin.String,",
				// Optional and read-only: absent by default.
				"public val note: org.example.models.models.Patch<kotlin.String> = org.example.models.models.Patch.Absent,",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("request model lost %q:\n%s", want, text)
				}
			}
		}
	}
}

// A component shared with responses has no request projection, so readOnly keeps
// the declared requiredness there. The annotation must not change emitted bytes.
func TestReadOnlyFieldIsInertOnSharedComponentModels(t *testing.T) {
	plain := schemas(`{"Record":{"type":"object","required":["id"],"properties":{"id":{"type":"string"},"note":{"type":"string"}}}}`)
	annotated := schemas(`{"Record":{"type":"object","required":["id"],"properties":{"id":{"type":"string","readOnly":true},"note":{"type":"string","readOnly":true}}}}`)
	for _, validation := range []bool{false, true} {
		a, b := request(plain, "Dto"), request(annotated, "Dto")
		a.Options.Validation, b.Options.Validation = validation, validation
		plainReport, plainErr := core.GenerateWithDiagnostics(a)
		annotatedReport, annotatedErr := core.GenerateWithDiagnostics(b)
		if plainErr != nil || annotatedErr != nil {
			t.Fatalf("validation=%v: %v / %v", validation, plainErr, annotatedErr)
		}
		if plainReport.Artifacts.Files[0].Content != annotatedReport.Artifacts.Files[0].Content {
			t.Fatalf("readOnly changed a shared component model:\n%s", annotatedReport.Artifacts.Files[0].Content)
		}
	}
}

// Only the literal true selects presence; false and a non-boolean value are
// handled as ordinary annotation validation.
func TestReadOnlyFieldRequiresBooleanAndFalseIsANoOp(t *testing.T) {
	falseDoc := []byte(`{"openapi":"3.1.0","paths":{"/r":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["id"],"properties":{"id":{"type":"string","readOnly":false}}}}}},"responses":{"204":{}}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(falseDoc, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if text := source(t, got, "SubmitInputDto"); !strings.Contains(text, "public val id: kotlin.String,") {
		t.Fatalf("readOnly false still projected the field:\n%s", text)
	}
	badDoc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"type":"object","properties":{"id":{"type":"string","readOnly":"yes"}}}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(badDoc, "Dto")); err == nil {
		t.Fatal("A non-boolean readOnly was accepted")
	}
}

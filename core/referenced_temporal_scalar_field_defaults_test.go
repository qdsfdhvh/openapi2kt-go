package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedTemporalInput(t *testing.T, own bool) []byte {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/referenced-temporal-scalar-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	if own {
		return input
	}
	var doc any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	var strip func(any)
	strip = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			delete(v, "default")
			for _, child := range v {
				strip(child)
			}
		case []any:
			for _, child := range v {
				strip(child)
			}
		}
	}
	strip(doc)
	input, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestReferencedTemporalScalarFieldDefaultsBaseline(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedTemporalInput(t, false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("actual no-default reference control", checked, style, err)
			}
			for _, want := range []string{"val required: kotlinx.datetime.LocalDate,", "val stamp: kotlin.time.Instant? = null", "val internal: kotlin.time.Instant? = null", "val bridge: kotlin.time.Instant? = null"} {
				if !strings.Contains(source(t, got, "HolderDto"), want) {
					t.Fatal(want, got)
				}
			}
			if !strings.Contains(source(t, got, "BridgeDto"), "typealias BridgeDto = kotlin.time.Instant") || !strings.Contains(source(t, got, "StampDto"), "typealias StampDto = kotlin.time.Instant?") {
				t.Fatal("actual public alias backing")
			}
			t.Logf("accepted no-default checked=%v style=%s warnings=%d", checked, style, len(got.Warnings))
		}
	}
}

func TestReferencedTemporalScalarFieldDefaultsOwnAnnotation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedTemporalInput(t, true), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("field-own temporal default", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{"val required: kotlinx.datetime.LocalDate,", "val day: kotlinx.datetime.LocalDate? = `openapi-kt default date`(\"2024-02-29\")", "val stamp: kotlin.time.Instant? = `openapi-kt default instant`(\"2024-02-29T12:34:56.123456789+02:30\")", "val nil: kotlin.time.Instant? = null"} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			if strings.Contains(text, "default tree") || strings.Contains(text, "init {") {
				t.Fatal("scalar default introduced barrier", text)
			}
			required := source(t, got, "RequiredNullableDto")
			if !strings.Contains(required, "val value: kotlin.time.Instant?,") || strings.Contains(required, "default instant") {
				t.Fatal(required)
			}
			patch := source(t, got, "PatchInputDto")
			if !strings.Contains(patch, "Patch.Absent") || !strings.Contains(patch, "val ordinary: kotlin.time.Instant? = `openapi-kt default instant`") {
				t.Fatal(patch)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
		}
	}
}

package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDiscardedStructuredConstOriginalExtraAndPinnedContributions(t *testing.T) {
	for _, tc := range []struct{ name, input, owner string }{
		{"ordinary-extra", `{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{}}}}`, "HolderDto"},
		{"pinned-extra", `{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"}},"additionalProperties":{"const":{}}},{"properties":{"tag":{"const":"two"}}}]}}`, "ChoiceDto"},
		{"pinned-overridden", `{"Choice":{"oneOf":[{"allOf":[{"properties":{"payload":{"const":{"source":true}}}},{"properties":{"payload":{"type":"string"},"tag":{"const":"one"}}}]},{"properties":{"tag":{"const":"two"}}}]}}`, "ChoiceDto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(schemas(tc.input), "Dto")
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal("Valid original discarded structured literal should validate without acquiring a body/value role", e)
			}
			code := source(t, got, tc.owner)
			if strings.Contains(code, "openapi-kt object const") || strings.Contains(code, "openapi-kt array const") || strings.Contains(code, "AdditionalValue") {
				t.Fatal("Discarded source leaked declaration/initializer/helper", code)
			}
			for _, w := range got.Warnings {
				if w.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
					t.Fatal("Discarded const leaked value/body warning", w)
				}
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Discarded source report drift", e)
			}
		})
	}
}

// Preserve the original extra-key negative inputs, now validated-only sources.
func assertDiscardedStructuredConstGraduation(t *testing.T, got core.GenerationReport, err error) {
	t.Helper()
	if err != nil {
		t.Fatal("Original valid extra-key literal did not graduate", err)
	}
	code := source(t, got, "HolderDto")
	if strings.Contains(code, "openapi-kt object const") || strings.Contains(code, "JsonObject") || strings.Contains(code, "AdditionalValue") {
		t.Fatal("Ignored literal acquired body/value state", code)
	}
	found := false
	for _, w := range got.Warnings {
		if w.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
			t.Fatal("Ignored kind is not retained as a typed value", w)
		}
		found = found || w.Code == "W_ADDITIONAL_PROPERTIES_APPROXIMATION" && w.Pointer != nil && *w.Pointer == "/components/schemas/Holder/additionalProperties"
	}
	if !found {
		t.Fatal("Existing extra-key projection warning lost", got.Warnings)
	}
}

func TestDiscardedStructuredConstMalformedContributionsKeepOriginalPointers(t *testing.T) {
	for _, tc := range []struct{ value, code, tail string }{
		{`{"const":{},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"const":{},"type":"array"}`, "E_SCHEMA_INVALID", "const"},
		{`{"const":[],"type":"object"}`, "E_SCHEMA_INVALID", "const"},
		{`{"const":{},"format":"json"}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"const":[],"x-native-type":"example.Value"}`, "E_SCHEMA_UNSUPPORTED", "x-native-type"},
		{`{"const":{},"minProperties":0}`, "E_SCHEMA_UNSUPPORTED", "minProperties"},
		{`{"const":[],"items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED", "items"},
		{`{"const":{},"nullable":"bad"}`, "E_SCHEMA_INVALID", "nullable"},
		{`{"const":{},"title":false}`, "E_SCHEMA_INVALID", "title"},
	} {
		for _, extra := range []bool{false, true} {
			for _, checked := range []bool{false, true} {
				body := `{"Choice":{"oneOf":[{"allOf":[{"properties":{"payload":` + tc.value + `}},{"properties":{"payload":{"type":"string"},"tag":{"const":"one"}}}]},{"properties":{"tag":{"const":"two"}}}]}}`
				pointer := "/components/schemas/Choice/oneOf/0/allOf/0/properties/payload/" + tc.tail
				if extra {
					body = `{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":` + tc.value + `}}`
					pointer = "/components/schemas/Holder/additionalProperties/" + tc.tail
				}
				r := request(schemas(body), "Dto")
				r.Options.Validation = checked
				got, e := core.GenerateWithDiagnostics(r)
				if e == nil {
					t.Fatal(tc, extra, checked)
				}
				d := e.(core.Diagnostics)[0]
				if d.Code != tc.code || d.Pointer == nil || *d.Pointer != pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal(tc, extra, checked, e)
				}
			}
		}
	}
}

func TestDiscardedStructuredConstDoesNotChangeGeneratedKotlinAgainstProjectionControls(t *testing.T) {
	for _, tc := range []struct{ input, control string }{
		{`{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{"$ref":"literal data","type":"data","enum":[null]}}}}`, `{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":true}}`},
		{`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":{"type":"string"}},"additionalProperties":{"const":[null]}},{"properties":{"tag":{"const":"two"}}}]}}`, `{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":{"type":"string"}},"additionalProperties":true},{"properties":{"tag":{"const":"two"}}}]}}`},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				makeRequest := func(body string) core.GenerationRequest {
					r := request(schemas(body), "Dto")
					r.Options.ModelStyle = style
					r.Options.Validation = checked
					return r
				}
				got, e := core.GenerateWithDiagnostics(makeRequest(tc.input))
				if e != nil {
					t.Fatal(e)
				}
				control, e := core.GenerateWithDiagnostics(makeRequest(tc.control))
				if e != nil {
					t.Fatal(e)
				}
				if !reflect.DeepEqual(got.Artifacts, control.Artifacts) {
					t.Fatal("Discarded JSON literal changed generated files or public naming", style, checked)
				}
			}
		}
	}
}

func TestDiscardedStructuredConstPreservesSourceAndMemberBudgets(t *testing.T) {
	for _, depth := range []int{47, 48} {
		value := `{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{"type":"literal data"}}}`
		for i := range depth {
			value = fmt.Sprintf(`{"properties":{"n%d":%s}}`, i, value)
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{"Holder":`+value+`}`), "Dto"))
		if (e == nil) != (depth == 47) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Discarded literal reset source depth", depth, e)
		}
	}
	for _, width := range []int{240, 241} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`},"additionalProperties":{"const":[null]}}}`), "Dto"))
		if (e == nil) != (width == 240) || e != nil && (e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Ignored literal changed real member budget", width, e)
		}
	}
	// A discarded annotation is not a Kotlin initializer or a default factory.
	large := strings.Repeat("a", 65536)
	got, e := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"value":{"type":"string"}},"additionalProperties":{"const":{"literal":"`+large+`"}}}}`), "Dto"))
	if e != nil || strings.Contains(source(t, got, "HolderDto"), large) || len(source(t, got, "HolderDto")) > 4096 {
		t.Fatal("Discarded data was emitted as a JVM constant/default", e)
	}
}

func TestDiscardedStructuredConstFixtureProfilesKeepActualRoleWarnings(t *testing.T) {
	input, e := os.ReadFile("../tests/fixtures/http/discarded-structured-const.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for _, optimized := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.ModelStyle = style
				r.Options.Validation = checked
				r.Options.HoistSharedShapes = optimized
				r.Options.DeduplicateModels = optimized
				r.Options.MergeIdenticalModels = optimized
				r.Options.LockHoistedNames = optimized
				got, e := core.GenerateWithDiagnostics(r)
				if e != nil {
					t.Fatal(style, checked, optimized, e)
				}
				locations := map[string]bool{}
				for _, w := range got.Warnings {
					if w.Code == "W_CONST_STRUCTURED_APPROXIMATION" {
						if w.Pointer == nil {
							t.Fatal(w)
						}
						if *w.Pointer != "/components/schemas/Body/properties/payload/const" && *w.Pointer != "/components/schemas/Dictionary/additionalProperties/const" {
							t.Fatal("Discarded source acquired a typed kind/body warning", w)
						}
						locations[*w.Pointer] = true
					}
				}
				if len(locations) != 2 {
					t.Fatal("Actual body/dictionary value role lost", locations)
				}
				for _, name := range []string{"ExtraDto", "ExtraArrayDto", "ChoiceDto", "AlphaDto", "BetaDto"} {
					code := source(t, got, name)
					if strings.Contains(code, "openapi-kt object const") || strings.Contains(code, "openapi-kt array const") || strings.Contains(code, "AdditionalValue") {
						t.Fatal("Discarded source leaked Kotlin state", name, code)
					}
				}
				if !strings.Contains(source(t, got, "BodyDto"), "openapi-kt object const") || !strings.Contains(source(t, got, "DictionaryDto"), "JsonObject") {
					t.Fatal("Actual retained value/body projection lost")
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				again, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Profile/name-lock replay changed", e)
				}
			}
		}
	}
}

func TestDiscardedStructuredConstKeepsRequiredWinnerConstructor(t *testing.T) {
	for _, literal := range []string{`{}`, `[null]`} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(schemas(`{"Choice":{"oneOf":[{"allOf":[{"properties":{"payload":{"const":`+literal+`,"nullable":true}},"required":["payload"]},{"properties":{"payload":{"type":"string"},"tag":{"const":"one"}}}]},{"properties":{"tag":{"const":"two"}}}]}}`), "Dto")
			r.Options.ModelStyle = style
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal(e)
			}
			code := source(t, got, "ChoiceDto")
			if !strings.Contains(code, "public val payload: kotlin.String,") || strings.Contains(code, "public val payload: kotlin.String =") || strings.Contains(code, "openapi-kt object const") || strings.Contains(code, "openapi-kt array const") {
				t.Fatal("Required final winner inherited discarded body's default or nullable", literal, style, code)
			}
		}
	}
}

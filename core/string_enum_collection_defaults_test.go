package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumCollectionDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/string-enum-collection-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{"data_class", "plain_class"} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, marker := range []string{
				"public val required: kotlin.collections.List<RequiredInner>,",
				"public val codes: kotlin.collections.List<CodesInner?>? =",
				"public val labels: kotlin.collections.Map<kotlin.String, LabelsValue>? =",
				"public sealed class CodesInner", "public sealed class LabelsValue", "class Unrecognized",
				"kotlinx.serialization.serializer<kotlin.collections.List<org.example.models.models.HolderDto.CodesInner?>>()",
				"kotlinx.serialization.serializer<kotlin.collections.Map<kotlin.String, org.example.models.models.HolderDto.LabelsValue>>()",
			} {
				if !strings.Contains(text, marker) {
					t.Fatal("Lost lexical enum API/default", marker, text)
				}
			}
			maps := source(t, got, "RequiredMapsDto")
			if !strings.Contains(maps, "public val labels: kotlin.collections.Map<kotlin.String, LabelsValue>,") || !strings.Contains(maps, "public val nullableLabels: kotlin.collections.Map<kotlin.String, NullableLabelsValue>?,") || strings.Contains(maps, "`openapi-kt default enum") {
				t.Fatal("Required Map acquired an initializer", maps)
			}
			patch := source(t, got, "PatchInputDto")
			if !strings.Contains(patch, "Patch<kotlin.collections.List<CodesInner>>") || !strings.Contains(patch, "Patch.Absent") || !strings.Contains(patch, "public val required: kotlin.collections.List<RequiredInner>?,") {
				t.Fatal(patch)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Nondeterministic enum collection", err)
			}
		}
	}
}

func TestStringEnumCollectionDefaultsOriginalDiagnostics(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"type":"string","enum":["one"]},"default":["future"]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"additionalProperties":{"enum":["one"]},"default":{"a/~":"future"}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{`{"additionalProperties":{"enum":["one"]},"default":{"z":"future","a":false}}`, "E_SCHEMA_INVALID", "/default/z"},
		{`{"type":"array","items":{"enum":["one"]},"default":[null]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"enum":["one"]},"default":[1]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"enum":[]},"default":[]}`, "E_SCHEMA_INVALID", "/items/enum"},
		{`{"additionalProperties":{"enum":["one",false]},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/enum/1"},
		{`{"type":"array","nullable":true,"items":{"enum":["one","one"]},"default":null}`, "E_SCHEMA_INVALID", "/items/enum/1"},
		{`{"type":"array","items":{"enum":["one",null]},"default":[]}`, "E_SCHEMA_INVALID", "/items/enum/1"},
		{`{"additionalProperties":{"enum":["one",null,null],"nullable":true},"default":{}}`, "E_SCHEMA_INVALID", "/additionalProperties/enum/2"},
		{`{"type":"array","items":{"enum":["one"],"default":"one"},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/default"},
		{`{"type":"array","items":{"type":"string","format":"uuid","enum":["one"]},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/enum"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(tc.field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, checked, err)
			}
		}
	}
}

func TestStringEnumCollectionDefaultsChecksAndExcludedRoles(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","items":{"enum":["one"]},"minItems":1,"default":[]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"additionalProperties":{"enum":["one"]},"maxProperties":0,"default":{"a":"one"}}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"enum":["one"]},"uniqueItems":true,"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/uniqueItems"},
		{`{"additionalProperties":{"enum":["one"],"maximum":1},"default":{}}`, "E_SCHEMA_UNSUPPORTED", "/additionalProperties/maximum"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value"+tc.tail {
			t.Fatal(tc, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("Checks silently dropped", tc)
		}
	}
	field := `{"type":"array","items":{"type":"string","enum":["one","two"]},"default":["one"]}`
	for _, input := range []string{
		`{"Codes":` + field + `}`,
		`{"Code":{"type":"string","enum":["one"]},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"default":["one"]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":` + field + `}}}}`,
		`{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
		`{"Holder":{"allOf":[{"properties":{"value":` + strings.Replace(field, `"default":["one"]`, `"default":["bad"]`, 1) + `}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Expanded default role or lost original", input, checked, err)
			}
		}
	}
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			values := make([]string, count)
			for i := range values {
				values[i] = fmt.Sprintf(`"v%d"`, i)
			}
			r := request(fieldDoc(`{"type":"array","items":{"enum":[`+strings.Join(values, ",")+`]},"default":[]}`), "Dto")
			r.Options.Validation = checked
			_, err := core.GenerateWithDiagnostics(r)
			if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT") {
				t.Fatal(count, err)
			}
		}
	}
}

func TestStringEnumCollectionDefaultsFrozenIdentity(t *testing.T) {
	field := `{"additionalProperties":{"enum":["one","two"]},"default":{"b":"two","a":"one"}}`
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	for _, checked := range []bool{false, true} {
		r := request([]byte(`{"openapi":"3.1.0","paths":{"/a":`+op+`,"/b":`+op+`}}`), "Dto")
		r.Options.Validation = checked
		r.Options.HoistSharedShapes = true
		r.Options.LockHoistedNames = true
		r.Options.DeduplicateModels = true
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || len(got.Artifacts.NameLock) == 0 {
			t.Fatal(err)
		}
		r.PreviousNameLock = got.Artifacts.NameLock
		r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"b":"two","a":"one"}`, `{"a":"one","b":"two"}`))
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Key order drifted", err)
		}
		r.Document = []byte(strings.ReplaceAll(string(r.Document), `{"a":"one","b":"two"}`, `{"a":"two","b":"two"}`))
		changed, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
			t.Fatal("Changed default reused frozen identity", err)
		}
	}
}

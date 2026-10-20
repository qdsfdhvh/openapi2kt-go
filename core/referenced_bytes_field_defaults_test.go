package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func fileContent(t *testing.T, artifacts core.GenerationArtifacts, path string) string {
	t.Helper()
	for _, file := range artifacts.Files {
		if strings.HasSuffix(file.Path, path) {
			return file.Content
		}
	}
	t.Fatal("missing generated file", path)
	return ""
}

func TestReferencedBytesFieldDefaults(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/referenced-bytes-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			files, err := core.Generate(r)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			holder := fileContent(t, files, "models/HolderDto.kt")
			for _, want := range []string{"val requiredBlob: kotlin.ByteArray,", "val blob: kotlin.ByteArray? = `openapi-kt default tree Scalar ByteArray`(\"[-128,0,127]\")", "val nil: kotlin.ByteArray? = null", "private fun `openapi-kt default tree Scalar ByteArray`", "kotlinx.serialization.serializer<kotlin.ByteArray>()"} {
				if !strings.Contains(holder, want) {
					t.Fatal(want, holder)
				}
			}
			if strings.Count(holder, "private fun `openapi-kt default tree Scalar ByteArray`") != 1 {
				t.Fatal("duplicate helper", holder)
			}
			required := fileContent(t, files, "models/RequiredNullableDto.kt")
			if !strings.Contains(required, "val blob: kotlin.ByteArray?,") || strings.Contains(required, "private fun `openapi-kt default tree") {
				t.Fatal(required)
			}
			if strings.Contains(fileContent(t, files, "models/DiscardedDto.kt"), "default tree") {
				t.Fatal("discarded initializer survived")
			}
			again, err := core.Generate(r)
			if err != nil || !reflect.DeepEqual(files, again) {
				t.Fatal("nondeterministic", err)
			}
			// The unchanged reference graph must also work without annotations.
			var document map[string]any
			if err := json.Unmarshal(input, &document); err != nil {
				t.Fatal(err)
			}
			// Marshal both controls with the same property ordering.
			ordered, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			r.Document = ordered
			owned, err := core.Generate(r)
			if err != nil {
				t.Fatal(err)
			}
			var remove func(any)
			remove = func(v any) {
				switch v := v.(type) {
				case map[string]any:
					delete(v, "default")
					for _, child := range v {
						remove(child)
					}
				case []any:
					for _, child := range v {
						remove(child)
					}
				}
			}
			remove(document)
			plain, _ := json.Marshal(document)
			r.Document = plain
			control, err := core.Generate(r)
			if err != nil {
				t.Fatal("reference admission changed", err)
			}
			for _, name := range []string{"BytesDto", "BinaryDto", "AliasDto", "BridgeDto", "RowsDto", "RowsAliasDto", "BagsDto", "NestedDto", "CappedDto", "HopsDto", "DiscardedDto"} {
				path := "models/" + name + ".kt"
				if fileContent(t, control, path) != fileContent(t, owned, path) {
					t.Fatal("referenced target changed output", style, checked, path)
				}
			}
			patch := fileContent(t, owned, "api/models/PatchInputDto.kt")
			if !strings.Contains(patch, "Patch.Absent") || !strings.Contains(patch, "val requiredBlob: kotlin.ByteArray?,") || strings.Contains(patch, "default tree") {
				t.Fatal("annotation changed Patch presence", patch)
			}
		}
	}
}

func TestReferencedBytesFieldDefaultOriginalValueDiagnostics(t *testing.T) {
	for _, tc := range []struct{ terminal, value, tail, override string }{
		{`{"type":"string","format":"byte"}`, `"AQI="`, "", ""},
		{`{"type":"string","format":"binary"}`, `[1.0]`, "/0", ""},
		{`{"type":"string","format":"byte"}`, `[-0]`, "/0", ""},
		{`{"type":"string","format":"byte"}`, `[128]`, "/0", ""},
		{`{"type":"string","format":"binary"}`, `[-129]`, "/0", ""},
		{`{"type":"string","format":"byte"}`, `[null]`, "/0", ""},
		{`{"type":"string","format":"byte"}`, `[true]`, "/0", ""},
		{`{"type":"string","format":"byte","nullable":true}`, `null`, "", ""},
		{`{"type":"array","items":{"type":"string","format":"byte"}}`, `["AQI="]`, "/0", ""},
		{`{"type":"array","items":{"type":"string","format":"byte","nullable":true}}`, `[[127,null]]`, "/0/1", ""},
		{`{"additionalProperties":{"type":"string","format":"binary"}}`, `{"a/~":[-128,128]}`, "/a~1~0/1", ""},
		{`{"type":"array","items":{"additionalProperties":{"type":"string","format":"binary"}}}`, `[{"$ref":[false]}]`, "/0/$ref/0", ""},
		{`{"type":"array","items":{"type":"string","format":"byte"}}`, `[null]`, "/0", ""},
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(tc.terminal, tc.value, tc.override), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			want := "/components/schemas/AHolder/properties/value/default" + tc.tail
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal(tc, checked, err)
			}
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != want {
				t.Fatal(tc, checked, want, err)
			}
		}
	}
}

func TestReferencedBytesFieldDefaultNullabilityAndCounts(t *testing.T) {
	for _, terminal := range []string{
		`{"type":"string","format":"binary"}`,
		`{"type":"string","format":"byte","nullable":true}`,
		`{"type":"array","items":{"type":"string","format":"byte"}}`,
		`{"additionalProperties":{"type":"string","format":"binary"}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(terminal, "null", `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			if _, err := core.Generate(r); err != nil {
				t.Fatal("nearest true did not grant annotation null", terminal, checked, err)
			}
		}
	}
	for _, tc := range []struct{ terminal, value string }{
		{`{"type":"array","minItems":2,"items":{"type":"string","format":"byte"}}`, `[[1]]`},
		{`{"minProperties":2,"additionalProperties":{"type":"string","format":"binary"}}`, `{"a":[1]}`},
	} {
		r := request(referencedFieldDoc(tc.terminal, tc.value, ""), "Dto")
		r.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/AHolder/properties/value/default" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("retained container count lost", tc, err)
		}
	}
}

func TestReferencedBytesFieldDefaultRolesStayBounded(t *testing.T) {
	for _, terminal := range []string{
		`{"type":"string","format":"byte","default":[1]}`,
		`{"type":"string","format":"binary","minLength":1}`,
		`{"type":"string","format":"byte","enum":["AQI="]}`,
		`{"type":"string","format":"byte","x-native-type":"kotlin.ByteArray"}`,
		`{"type":"array","items":{"type":"string","format":"byte","default":[1]}}`,
		`{"type":"array","uniqueItems":true,"items":{"type":"string","format":"byte"}}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(terminal, `[1]`, ""), "Dto")
			r.Options.Validation = checked
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal(terminal, checked, err)
			}
		}
	}
	for _, body := range []string{
		`{"Bytes":{"type":"string","format":"byte"},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Bytes"}],"default":[1]}}}}`,
		`{"Bytes":{"type":"string","format":"byte"},"Alias":{"$ref":"#/components/schemas/Bytes","default":[1]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":[2]}}}}`,
		`{"Bytes":{"type":"string","format":"byte"},"Holder":{"properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Bytes","default":[1]}}}}}`,
	} {
		if report, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal(body, err)
		}
	}
	// Original discarded and required annotations still fail before suppression.
	for _, holder := range []string{
		`{"properties":{"v":{"$ref":"#/components/schemas/Bytes","default":[128]}},"required":["v"]}`,
		`{"allOf":[{"properties":{"v":{"$ref":"#/components/schemas/Bytes","default":[128]}}},{"properties":{"v":{"type":"string"}}}]}`,
	} {
		if report, err := core.GenerateWithDiagnostics(request(schemas(`{"Bytes":{"type":"string","format":"byte"},"Holder":`+holder+`}`), "Dto")); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal(holder, err)
		}
	}
}

func TestReferencedBytesFieldDefaultBudgets(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{`[` + strings.Repeat("0,", 16382) + `0]`, true},
		{`[` + strings.Repeat("0,", 16383) + `0]`, false},
		{`[` + strings.Repeat("-128,", 6553) + `0]`, true},
		{`[` + strings.Repeat("-128,", 6553) + `10]`, false},
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedFieldDoc(`{"type":"string","format":"byte"}`, tc.value, ""), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			if tc.valid {
				if err != nil {
					t.Fatal(len(tc.value), checked, err)
				}
			} else if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal(len(tc.value), checked, err)
			} else if ds, ok := err.(core.Diagnostics); !ok || len(ds) != 1 || ds[0].Code != "E_MODEL_LIMIT" {
				t.Fatal("wrong budget diagnostic", len(tc.value), checked, err)
			}
		}
	}
	// The collection root and bytes leaf share the same whole-annotation budget.
	for _, n := range []int{16382, 16383} {
		annotation := `[[` + strings.Repeat("0,", n-1) + `0]]`
		r := request(referencedFieldDoc(`{"type":"array","items":{"type":"string","format":"binary"}}`, annotation, ""), "Dto")
		_, err := core.Generate(r)
		if (err == nil) != (n == 16382) {
			t.Fatal(n, err)
		}
	}
}

func TestReferencedBytesFieldDefaultEffectiveIdentity(t *testing.T) {
	for _, flag := range []string{"dedup", "merge", "hoist"} {
		model := func(value string) string {
			return `{"properties":{"blob":{"$ref":"#/components/schemas/Bytes","default":` + value + `}}}`
		}
		input := []byte(fmt.Sprintf(`{"openapi":"3.1.0","components":{"schemas":{"Bytes":{"type":"string","format":"byte"},"Alpha":%s,"Beta":%s,"Different":%s}},"paths":{"/a":{"get":{"responses":{"200":{"content":{"application/json":{"schema":%s}}}}}},"/b":{"get":{"responses":{"200":{"content":{"application/json":{"schema":%s}}}}}}}}`, model(`[1]`), model(`[1]`), model(`[2]`), model(`[1]`), model(`[1]`)))
		r := request(input, "Dto")
		r.Options.DeduplicateModels = flag == "dedup"
		r.Options.MergeIdenticalModels = flag == "merge"
		r.Options.HoistSharedShapes = flag == "hoist"
		files, err := core.Generate(r)
		if err != nil {
			t.Fatal(flag, err)
		}
		different := fileContent(t, files, "models/DifferentDto.kt")
		if !strings.Contains(different, "`openapi-kt default tree Scalar ByteArray`(\"[2]\")") {
			t.Fatal(flag, different)
		}
		if flag == "merge" {
			if !strings.Contains(fileContent(t, files, "models/BetaDto.kt"), "typealias BetaDto = org.example.models.models.AlphaDto") {
				t.Fatal("same effective initializer not merged", fileContent(t, files, "models/BetaDto.kt"))
			}
		}
	}
}

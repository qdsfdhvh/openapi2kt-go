package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func request(input []byte, suffix string) core.GenerationRequest {
	return core.GenerationRequest{Document: input, Format: core.JSON, SourceName: "input.json", Options: core.GenerationOptions{PackageName: "org.example.models", ModelNameSuffix: suffix}}
}
func schemas(text string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":` + text + `}}`)
}
func fieldDoc(field string) []byte {
	return schemas(`{"Record":{"type":"object","properties":{"value":` + field + `},"required":["value"]}}`)
}
func report(t testing.TB, input []byte, suffix string) core.GenerationReport {
	t.Helper()
	r, err := core.GenerateWithDiagnostics(request(input, suffix))
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func source(t testing.TB, r core.GenerationReport, name string) string {
	t.Helper()
	for _, f := range r.Artifacts.Files {
		if strings.HasSuffix(f.Path, "/"+name+".kt") {
			return f.Content
		}
	}
	t.Fatal("missing source", name)
	return ""
}
func failure(t testing.TB, input []byte, code, pointer string) {
	t.Helper()
	r, err := core.GenerateWithDiagnostics(request(input, ""))
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != code || ds[0].Pointer == nil || *ds[0].Pointer != pointer {
		t.Fatalf("want %s at %s, got %v", code, pointer, err)
	}
	if len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
		t.Fatal("failed generation returned partial artifacts/warnings")
	}
}

func TestGenerationCorpus(t *testing.T) {
	for _, c := range testutil.Corpus(t) {
		t.Run(c.ID, func(t *testing.T) {
			name := c.Input
			if c.SourceName != nil {
				name = *c.SourceName
			}
			rq := core.GenerationRequest{Document: testutil.Read(t, c.Input), Format: core.InputFormat(c.InputFormat), SourceName: name,
				Options: core.GenerationOptions{PackageName: c.PackageName, ModelNameSuffix: c.ModelNameSuffix}}
			r, err := core.GenerateWithDiagnostics(rq)
			if testutil.DefaultGraduation(c) {
				if err != nil {
					t.Fatal("original default fixture failed", err)
				}
				files := map[string]string{}
				for _, file := range r.Artifacts.Files {
					files[file.Path] = file.Content
				}
				testutil.CheckDefaultGraduation(t, c, files)
				return
			}
			if testutil.BooleanValueGraduation(c) {
				if err != nil {
					t.Fatal("graduated boolean value failed", err)
				}
				files := map[string]string{}
				for _, file := range r.Artifacts.Files {
					files[file.Path] = file.Content
				}
				testutil.CheckBooleanValueGraduation(t, c, files, r.Warnings)
				return
			}
			if testutil.NullableRawGraduation(c) {
				if err != nil {
					t.Fatal("graduated nullable raw failed", err)
				}
				files := map[string]string{}
				for _, file := range r.Artifacts.Files {
					files[file.Path] = file.Content
				}
				testutil.CheckNullableRawGraduation(t, c, files)
				if len(r.Warnings) != 1 || r.Warnings[0].Code != "W_JSON_ELEMENT_APPROXIMATION" || *r.Warnings[0].Pointer != "/components/schemas/Record" {
					t.Fatal(r.Warnings)
				}
				return
			}
			if testutil.ArrayGraduation(c) {
				if err != nil {
					t.Fatal("graduated array failed", err)
				}
				files := map[string]string{}
				for _, file := range r.Artifacts.Files {
					files[file.Path] = file.Content
				}
				testutil.CheckArrayGraduation(t, c, files)
				return
			}
			if testutil.ScalarGraduation(c) {
				if err != nil {
					t.Fatal("graduated scalar failed", err)
				}
				files := map[string]string{}
				for _, file := range r.Artifacts.Files {
					files[file.Path] = file.Content
				}
				testutil.CheckScalarGraduation(t, c, files)
				return
			}
			if c.Expect.Kind == "error" {
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) != 1 || ds[0].Code != c.Expect.Code || !reflect.DeepEqual(ds[0].Pointer, c.Expect.Pointer) {
					t.Fatalf("want %s %v, got %#v", c.Expect.Code, c.Expect.Pointer, err)
				}
				if len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
					t.Fatal("partial result on error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			actual, expected := map[string]string{}, map[string]string{}
			for _, f := range r.Artifacts.Files {
				if _, duplicate := actual[f.Path]; duplicate {
					t.Fatal("duplicate generated file")
				}
				actual[f.Path] = f.Content
			}
			for path, golden := range c.Expect.Files {
				expected[path] = string(testutil.CurrentUUIDSnapshot(t, c, path, testutil.Read(t, golden)))
			}
			if !reflect.DeepEqual(actual, expected) {
				for path, text := range expected {
					if actual[path] != text {
						t.Errorf("snapshot mismatch %s\nactual:\n%s\nexpected:\n%s", path, actual[path], text)
					}
				}
				t.Fatalf("file set mismatch: actual=%d expected=%d", len(actual), len(expected))
			}
			filesOnly, err := core.Generate(rq)
			if err != nil || !reflect.DeepEqual(filesOnly, r.Artifacts) {
				t.Fatal("files-only/report APIs differ", err)
			}
		})
	}
}

func TestWarningsAreOrderedSourceLocatedAndNotPayloadScans(t *testing.T) {
	r := report(t, schemas(`{"Record":{"type":"object","example":{"oneOf":[{"type":"null"},{"type":"null"}]},"properties":{"n":{"oneOf":[{"type":"string"},{"type":"null"},{"type":"null"}]},"any":{},"obj":{"type":"object"},"native":{"type":"integer","enum":[1,2]},"tag":{"type":"string","const":"private-marker"}}},"Empty":{"type":"object"}}`), "")
	var codes, pointers []string
	for _, w := range r.Warnings {
		codes = append(codes, w.Code)
		pointers = append(pointers, *w.Pointer)
		if strings.Contains(w.Message, "private-marker") {
			t.Fatal("warning leaks payload")
		}
	}
	if !reflect.DeepEqual(codes, []string{"W_JSON_ELEMENT_APPROXIMATION", "W_ONEOF_NULL_OVERLAP", "W_JSON_ELEMENT_APPROXIMATION", "W_ENUM_NATIVE_APPROXIMATION"}) || !reflect.DeepEqual(pointers, []string{"/components/schemas/Empty", "/components/schemas/Record/properties/n/oneOf", "/components/schemas/Record/properties/obj", "/components/schemas/Record/properties/native/enum"}) {
		t.Fatal(codes, pointers)
	}
	defaults := report(t, schemas(`{"A":{"type":"object"},"Z":{"type":"object","properties":{"v":{"type":"string","default":"private-marker"}}}}`), "")
	if len(defaults.Warnings) != 1 || defaults.Warnings[0].Code != "W_JSON_ELEMENT_APPROXIMATION" || strings.Contains(defaults.Warnings[0].Message, "private-marker") || !strings.Contains(source(t, defaults, "Z"), ` = "private-marker",`) {
		t.Fatal("default payload scanned or lost", defaults)
	}
	failure(t, schemas(`{"A":{"type":"object"},"Z":{"type":"object","properties":{"v":{"type":"string","default":true}}}}`), "E_SCHEMA_INVALID", "/components/schemas/Z/properties/v/default")
}

func TestSchemaTraversalAndSerializationSlotBoundaries(t *testing.T) {
	for _, depth := range []int{47, 48} {
		for _, composition := range []bool{false, true} {
			field := `{"type":"string"}`
			for range depth {
				if composition {
					field = `{"anyOf":[` + field + `,{"type":"null"}]}`
				} else {
					field = `{"type":"array","items":` + field + `}`
				}
			}
			_, err := core.Generate(request(fieldDoc(field), ""))
			if (err == nil) != (depth == 47) {
				t.Fatalf("depth=%d composition=%t: %v", depth, composition, err)
			}
			if err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct {
		count                             int
		required, nullable, constant, ref bool
		ok                                bool
	}{
		{120, true, false, false, false, true}, {121, true, false, false, false, false},
		{240, true, true, false, false, true}, {241, true, true, false, false, false},
		{240, false, false, false, false, true}, {241, false, false, false, false, false},
		{120, false, false, true, false, true}, {121, false, false, true, false, false},
		{120, true, false, false, true, true}, {121, true, false, false, true, false},
		{240, true, true, false, true, true}, {241, true, true, false, true, false},
	} {
		properties := map[string]any{}
		var required []string
		for i := range tc.count {
			name := fmt.Sprintf("field%d", i)
			field := map[string]any{"type": "integer", "format": "int64", "nullable": tc.nullable}
			if tc.constant {
				field["const"] = i
			}
			if tc.ref {
				field = map[string]any{"$ref": "#/components/schemas/Scalar", "nullable": tc.nullable}
			}
			properties[name] = field
			if tc.required {
				required = append(required, name)
			}
		}
		if required == nil {
			required = []string{}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Record": map[string]any{"type": "object", "properties": properties, "required": required}, "Scalar": map[string]any{"type": "integer", "format": "int64"}}}})
		_, err := core.Generate(request(input, ""))
		if (err == nil) != tc.ok || err != nil && err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}

func TestEnumsConstNativeAndAnnotationValidation(t *testing.T) {
	values := `{"Status":{"enum":["Unrecognized","Serializer","Companion","in-progress","in progress","in_progress","雪","🚀","", " "]}}`
	s := source(t, report(t, schemas(values), "Dto"), "StatusDto")
	for _, name := range []string{"UNRECOGNIZED_VALUE", "SERIALIZER_VALUE", "COMPANION_VALUE", "IN_PROGRESS", "IN_PROGRESS_VALUE", "IN_PROGRESS_VALUE2", "U96EA", "UD83D_UDE80", "VALUE", "VALUE_VALUE"} {
		if !strings.Contains(s, "public data object "+name+" :") {
			t.Fatal("missing enum member", name)
		}
	}
	for _, count := range []int{240, 241} {
		values := make([]string, count)
		for i := range count {
			values[i] = fmt.Sprintf("v%d", i)
		}
		encoded, _ := json.Marshal(values)
		_, err := core.Generate(request(schemas(`{"Status":{"type":"string","enum":`+string(encoded)+`}}`), ""))
		if (err == nil) != (count == 240) || err != nil && err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" {
			t.Fatal(count, err)
		}
	}
	for _, tc := range []struct{ field, code, tail string }{
		{`{"const":"1"}`, "E_SCHEMA_UNSUPPORTED", "const"}, {`{"const":"-0X1p1F"}`, "E_SCHEMA_UNSUPPORTED", "const"},
		{`{"const":1,"format":"int64"}`, "E_SCHEMA_UNSUPPORTED", "format"},
		{`{"type":"integer","const":2147483648}`, "E_SCHEMA_INVALID", "const"},
		{`{"type":"number","format":"float","const":1e100}`, "E_SCHEMA_INVALID", "const"},
		{`{"type":"boolean","const":1}`, "E_SCHEMA_INVALID", "const"},
		{`{"type":"number","enum":[1e0,1.0]}`, "E_SCHEMA_INVALID", "enum/1"},
		{`{"type":"integer","enum":[1.0]}`, "E_SCHEMA_INVALID", "enum/0"},
		{`{"type":"string","title":false}`, "E_SCHEMA_INVALID", "title"},
		{`{"oneOf":[{"type":"string"},{"type":"null","const":null}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/1/const"},
		{`{"anyOf":[true,{"type":"null"}]}`, "E_SCHEMA_INVALID", "anyOf/0"},
		{`{"anyOf":null}`, "E_SCHEMA_INVALID", "anyOf"},
	} {
		failure(t, fieldDoc(tc.field), tc.code, "/components/schemas/Record/properties/value/"+tc.tail)
	}
	input := schemas(`{"Bool":{"type":"boolean","enum":[true,false]},"Pinned":{"type":"integer","format":"int64","const":-9223372036854775808},"Record":{"type":"object","properties":{"v":{"$ref":"#/components/schemas/Pinned","nullable":true},"b":{"$ref":"#/components/schemas/Bool"},"items":{"type":"array","items":{"type":"number","const":-0.0}}}}}`)
	r := report(t, input, "Dto")
	s = source(t, r, "RecordDto")
	if !strings.Contains(s, "public val v: kotlin.Long = kotlin.Long.MIN_VALUE") || !strings.Contains(s, "public val b: kotlin.Boolean? = null") || !strings.Contains(s, "kotlin.collections.List<kotlin.Double>") {
		t.Fatal(s)
	}
	if len(r.Warnings) != 3 {
		t.Fatal(r.Warnings)
	}
}

func TestCompositionSourcePointersOverridesCyclesWorkAndDepth(t *testing.T) {
	r := report(t, schemas(`{"Base":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"required":["a"]},"Record":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"b":{"type":"integer","format":"int64"},"c":{"type":"boolean"}},"required":["c"]}],"properties":{"b":{"type":"string","const":"final"},"d":{"type":"string"}}}}`), "Dto")
	s := source(t, r, "RecordDto")
	if !strings.Contains(s, "val a: kotlin.String,") || !strings.Contains(s, "val c: kotlin.Boolean,") || !strings.Contains(s, `val b: kotlin.String = "final"`) {
		t.Fatal(s)
	}
	defaultOverride := report(t, schemas(`{"Record":{"allOf":[{"properties":{"a":{"type":"string","default":"bad"}}},{"properties":{"a":{"type":"string"}}}]}}`), "")
	if !strings.Contains(source(t, defaultOverride, "Record"), "val a: kotlin.String? = null,") || strings.Contains(source(t, defaultOverride, "Record"), `"bad"`) {
		t.Fatal("discarded original default was retained instead of winning field", defaultOverride)
	}
	failure(t, schemas(`{"Record":{"allOf":[{"properties":{"a":{"type":"string","default":true}}},{"properties":{"a":{"type":"string"}}}]}}`), "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/0/properties/a/default")
	failure(t, schemas(`{"Record":{"allOf":[{"$ref":"#/components/schemas/Record"},{"properties":{"id":{"type":"string"}}}]}}`), "E_REF_CYCLE", "/components/schemas/Record/allOf/0/$ref")
	failure(t, schemas(`{"Base":{"type":"object"},"Root":{"allOf":[{"$ref":"#/components/schemas/Base"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/allOf")
	failure(t, schemas(`{"Unused":{"allOf":[{"$ref":"#/components/schemas/Missing"},{"type":"object"}]}}`), "E_REF_MISSING", "/components/schemas/Unused/allOf/0")
	for _, tc := range []struct {
		count  int
		repeat bool
		code   string
	}{{47, false, ""}, {48, false, "E_SCHEMA_DEPTH"}, {14, true, "E_SCHEMA_LIMIT"}} {
		components := map[string]any{"Model0": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}}
		for i := 1; i <= tc.count; i++ {
			ref := map[string]any{"$ref": fmt.Sprintf("#/components/schemas/Model%d", i-1)}
			second := any(map[string]any{"properties": map[string]any{fmt.Sprintf("field%d", i): map[string]any{"type": "string"}}})
			if tc.repeat {
				second = ref
			}
			components[fmt.Sprintf("Model%d", i)] = map[string]any{"allOf": []any{ref, second}}
		}
		encoded, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": components}})
		_, err := core.Generate(request(encoded, ""))
		if (err == nil) != (tc.code == "") || err != nil && err.(core.Diagnostics)[0].Code != tc.code {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}

// The exporter only writes into an explicitly supplied verification directory.
// It calls the real pure Go core; generated Kotlin is never hand-authored.
func TestExportModelSmoke(t *testing.T) {
	root := os.Getenv("OPENAPI_KT_TEST_EXPORT")
	if root == "" {
		t.Skip("opt-in Kotlin verification exporter")
	}
	exportModelSmoke(t, root)
}

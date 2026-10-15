package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestHTTPRequestKindProjection(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "json-kind-request-unions.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	if len(r.Artifacts.Files) != 10 {
		t.Fatal(r)
	}
	raw := source(t, r, "RawChoiceInputDto")
	for _, want := range []string{"sealed class RawChoiceInputDto", "data object Variant1", "data object Variant5", "return org.example.models.api.models.UnknownRawChoiceInputDto.serializer()"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q: %s", want, raw)
		}
	}
	if strings.Contains(raw, "public val value:") || strings.Contains(raw, "element.jsonObject") || strings.Contains(raw, "NullRawChoiceInputDto") {
		t.Fatal("root request variants must not become native payload/null wrappers", raw)
	}
	tagged := source(t, r, "TaggedChoiceInputDto")
	for _, want := range []string{"data object Variant1", "data class Known", "val obj = element.jsonObject", `tag == "known"`, "memo: kotlin.String? = null"} {
		if !strings.Contains(tagged, want) {
			t.Errorf("missing %q: %s", want, tagged)
		}
	}
	presence := source(t, r, "PresenceChoiceInputDto")
	if !strings.Contains(presence, `"label" in obj`) || !strings.Contains(presence, "data object Variant1") {
		t.Fatal(presence)
	}
	api := source(t, r, "DefaultApi")
	for _, want := range []string{"input: org.example.models.api.models.RawChoiceInputDto,", "input: org.example.models.api.models.OptionalRawInputDto? = null,"} {
		if !strings.Contains(api, want) {
			t.Error(api)
		}
	}
	for _, file := range r.Artifacts.Files {
		if strings.Contains(file.Content, "Patch<") || strings.HasSuffix(file.Path, "/Patch.kt") {
			t.Fatal("root request kind context must stay Shared", file)
		}
	}
	unreachable, projected := 0, 0
	for _, warning := range r.Warnings {
		if warning.Code == "W_UNION_UNREACHABLE_VARIANT" {
			unreachable++
		}
		if warning.Code == "W_HTTP_REQUEST_UNION_PROJECTION" {
			projected++
		}
	}
	if unreachable != 14 || projected != 15 {
		t.Fatal(r.Warnings)
	}
	if again := report(t, input, "Dto"); !reflect.DeepEqual(r, again) {
		t.Fatal("request-kind generation is not deterministic")
	}
	rq := request(input, "Dto")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	excluded, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(excluded.Artifacts.Files) != 3 {
		t.Fatal(excluded, err)
	}
}

func TestHTTPRequestKindFallbackOwnership(t *testing.T) {
	paths := map[string]any{}
	for _, param := range []string{"a", "b"} {
		paths["/same/{"+param+"}"] = map[string]any{"post": map[string]any{
			"parameters":  []any{map[string]any{"name": param, "in": "path", "required": true, "schema": map[string]any{"type": "string"}}},
			"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "boolean"}}}}}},
			"responses":   map[string]any{"204": map[string]any{}},
		}}
	}
	input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for _, name := range []string{"PostSameInputDto", "PostSame2InputDto"} {
		if !strings.Contains(source(t, r, name), "data object Variant2") || !strings.Contains(source(t, r, "DefaultApi"), ".api.models."+name+",") {
			t.Fatal(r)
		}
	}
	projections := 0
	for _, w := range r.Warnings {
		if w.Code == "W_HTTP_REQUEST_UNION_PROJECTION" {
			projections++
		}
	}
	if projections != 4 {
		t.Fatal("fallback replanning duplicated/lost warnings", r.Warnings)
	}
}

func TestHTTPRequestKindGuards(t *testing.T) {
	base := "/paths/~1apply/post/requestBody/content/application~1json/schema"
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"anyOf":[{"type":"string"},{"type":"array"}]}`, "E_SCHEMA_INVALID", "/anyOf/1"},
		{`{"anyOf":[{"type":"boolean"},{"type":"array","items":false}]}`, "E_SCHEMA_INVALID", "/anyOf/1/items"},
		{`{"anyOf":[{"type":"string","pattern":"x"},{"type":"boolean"}]}`, "E_SCHEMA_UNSUPPORTED", "/anyOf/0/pattern"},
		{`{"anyOf":[{"type":"string","enum":[false]},{"type":"boolean"}]}`, "E_SCHEMA_INVALID", "/anyOf/0/enum/0"},
		{`{"anyOf":[{"$ref":"#/components/schemas/Missing"},{"type":"boolean"}]}`, "E_REF_MISSING", "/anyOf/0"},
		{`{"anyOf":[{"type":"string"},false]}`, "E_SCHEMA_INVALID", "/anyOf/1"},
	} {
		input := presenceDocument(t, `{"content":{"application/json":{"schema":`+tc.schema+`}}}`, nil)
		failure(t, input, tc.code, base+tc.tail)
		// Exclusions validate all schemas before discarding owned files.
		rq := request(input, "")
		rq.Options.ExcludeAPIs = []string{"DefaultApi"}
		r, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
			t.Fatal(r, err)
		}
	}
	for _, count := range []int{240, 241} {
		branches := make([]string, count)
		for i := range branches {
			branches[i] = `{"type":"string"}`
		}
		branches[0] = `{"type":"boolean"}`
		input := presenceDocument(t, `{"required":true,"content":{"application/json":{"schema":{"oneOf":[`+strings.Join(branches, ",")+`]}}}}`, nil)
		if count == 240 {
			r := report(t, input, "")
			if !strings.Contains(source(t, r, "PostApplyInput"), "data object Variant240") {
				t.Fatal(r)
			}
		} else {
			failure(t, input, "E_UNION_LIMIT", base+"/oneOf")
		}
	}
	// Projected array items still observe the existing depth cap.
	item := map[string]any{"type": "string"}
	for i := 0; i < 49; i++ {
		item = map[string]any{"type": "array", "items": item}
	}
	schema, err := json.Marshal(map[string]any{"anyOf": []any{map[string]any{"type": "boolean"}, item}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = core.Generate(request(presenceDocument(t, `{"content":{"application/json":{"schema":`+string(schema)+`}}}`, nil), ""))
	if err == nil || !strings.Contains(err.Error(), "E_SCHEMA_DEPTH") {
		t.Fatal(err)
	}
	// The projection does not bypass constructor limits of actual object branches.
	for _, count := range []int{120, 121} {
		var fields, required []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"integer","format":"int64"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		input := presenceDocument(t, `{"content":{"application/json":{"schema":{"anyOf":[{"type":"string"},{"type":"object","properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(required, ",")+`]}]}}}}`, nil)
		if count == 120 {
			report(t, input, "")
		} else {
			failure(t, input, "E_MODEL_LIMIT", base+"/anyOf/1")
		}
	}
}

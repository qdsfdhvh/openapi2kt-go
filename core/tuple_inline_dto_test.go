package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleInlineDTOSelectedAndDiscarded(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, want string }{
		{`[{"properties":{"name":{"type":"string"}},"required":["name"]}]`, "", "ProbeDtoInner"},
		{`[{"properties":{"name":{"type":"string"}}},{"properties":{"name":{"type":"string"}}}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[{"properties":{"name":{"type":"string"}}}]`, `,"items":true`, "kotlinx.serialization.json.JsonElement"},
		{`[{"properties":{"name":{"type":"string"}}}]`, `,"items":false`, "ProbeDtoInner"},
		{`[{"properties":{"name":{"type":"string"}}},{"type":"integer"}]`, `,"items":{"properties":{"name":{"type":"string"}}}`, "ProbeDtoInner"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				rq := request(schemas(`{"Probe":{"type":"array","prefixItems":`+tc.prefix+tc.tail+`}}`), "Dto")
				rq.Options.Validation = checked
				rq.Options.ModelStyle = style
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(tc, checked, err)
				}
				if !strings.Contains(source(t, got, "ProbeDto"), "List<"+tc.want+">") {
					t.Fatal(tc, source(t, got, "ProbeDto"))
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("inline DTO tuple nondeterminism", e)
				}
			}
		}
	}
}

func TestTupleInlineDTODeferredAnnotationAndCountGuards(t *testing.T) {
	for _, tc := range []struct{ field, tail, pointer, code string }{
		{`{"$ref":"#/components/schemas/Choice","default":"future"}`, `,"items":true`, "/properties/value/default", "E_SCHEMA_INVALID"},
		{`{"$ref":"#/components/schemas/Choice","default":"future"}`, `,"items":{"type":"string"}`, "/properties/value/default", "E_SCHEMA_INVALID"},
		{`{"$ref":"#/components/schemas/Item","default":{}}`, `,"items":false`, "/properties/value/default", "E_SCHEMA_INVALID"},
		{`{"type":"integer","minimum":1,"default":0}`, `,"items":true`, "/properties/value/default", "E_SCHEMA_INVALID"},
	} {
		input := schemas(`{"Choice":{"enum":["ready"]},"Item":{"properties":{"name":{"type":"string"}},"required":["name"]},"Rows":{"type":"array","prefixItems":[{"properties":{"value":` + tc.field + `},"required":["value"]}]` + tc.tail + `}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/prefixItems/0"+tc.pointer {
			t.Fatal(tc, err)
		}
	}
	for _, value := range []string{`[]`, `null`} {
		input := schemas(`{"Rows":{"type":"array","nullable":true,"prefixItems":[{"properties":{"n":{"type":"integer","default":"bad"}}}],"items":true,"default":` + value + `}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Rows/prefixItems/0/properties/n/default")
	}
	for _, value := range []string{`[{"name":"outer"}]`, `[{"name":"outer","n":1}]`} {
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[{"properties":{"name":{"type":"string"},"n":{"type":"integer"}},"required":["name"],"minProperties":2}],"default":`+value+`}}`), "Dto")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if strings.Contains(value, `"n"`) && err != nil || !strings.Contains(value, `"n"`) && (err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/default/0") {
			t.Fatal("inline constructor count default", value, err)
		}
	}
}

func TestTupleInlineDTOSourceDepthAndCumulativeLimits(t *testing.T) {
	for _, depth := range []int{47, 48} {
		schema := `{"type":"array","prefixItems":[{"properties":{"name":{"type":"string"}}}]}`
		for i := 1; i < depth; i++ {
			schema = `{"type":"array","items":` + schema + `}`
		}
		_, err := core.Generate(request(schemas(`{"Rows":`+schema+`}`), "Dto"))
		if depth == 47 && err != nil || depth == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("inline DTO source depth", depth, err)
		}
	}
	fields := make([]string, 240)
	for i := range fields {
		fields[i] = fmt.Sprintf(`"v%d":{"type":"integer"}`, i)
	}
	entries := make([]string, 32)
	for i := range entries {
		entries[i] = fmt.Sprintf(`"Rows%d":{"type":"array","prefixItems":[{"properties":{%s}}],"items":true}`, i, strings.Join(fields, ","))
	}
	if _, err := core.Generate(request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")); err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
		t.Fatal("independent discarded DTOs reset source capture", err)
	}
}

func TestTupleInlineDTOFullSourceDefaultsAndReferences(t *testing.T) {
	for _, tail := range []string{"", `,"items":true`, `,"items":false`, `,"items":{"type":"string"}`} {
		input := schemas(`{"Choice":{"enum":["ready","paused"]},"Item":{"properties":{"label":{"type":"string"}},"required":["label"]},"Rows":{"type":"array","prefixItems":[{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"ready"},"child":{"$ref":"#/components/schemas/Item","default":{"label":"inner"}},"list":{"type":"array","items":{"type":"integer"},"default":[1]}}}]` + tail + `}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("ordinary DTO source graph rejected", tail, err)
		}
	}
	for _, member := range []struct{ schema, pointer, code string }{
		{`{"properties":{"value":{"type":"integer","default":"bad"}}}`, "/properties/value/default", "E_SCHEMA_INVALID"},
		{`{"properties":{"value":{"$ref":"#/components/schemas/Choice","default":"future"}}}`, "/properties/value/default", "E_SCHEMA_INVALID"},
		{`{"properties":{"value":{"type":"integer","unknownTuple":true}}}`, "/properties/value/unknownTuple", "E_SCHEMA_UNSUPPORTED"},
	} {
		input := schemas(`{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[` + member.schema + `],"items":true}}`)
		failure(t, input, member.code, "/components/schemas/Rows/prefixItems/0"+member.pointer)
	}
}

func TestTupleInlineDTOOwnDefaults(t *testing.T) {
	tuple := `{"type":"array","prefixItems":[{"properties":{"name":{"type":"string"},"n":{"type":"integer","default":7}},"required":["name"]}],"default":[{"name":"outer"}]}`
	input := schemas(`{"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":` + tuple + `}}}`)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("inline DTO tuple own defaults", err)
	}
}

func TestTupleInlineDTODirectionalAndHTTPOnly(t *testing.T) {
	item := `{"properties":{"name":{"type":"string"},"secret":{"type":"string","writeOnly":true},"id":{"type":"integer","readOnly":true}},"required":["name","secret","id"]}`
	for _, shared := range []bool{false, true} {
		root := `{"properties":{"rows":{"type":"array","prefixItems":[` + item + `]}},"required":["rows"]}`
		components := ""
		if shared {
			components = `,"components":{"schemas":{"Holder":` + root + `}}`
			root = `{"$ref":"#/components/schemas/Holder"}`
		}
		input := `{"openapi":"3.1.0"` + components + `,"paths":{"/echo":{"post":{"operationId":"echo","requestBody":{"required":true,"content":{"application/json":{"schema":` + root + `}}},"responses":{"200":{"description":"synthetic","content":{"application/json":{"schema":` + root + `}}}}}}}}`
		rq := request([]byte(input), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("directional or HTTP-only source registry", shared, err)
		}
		again, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("directional source nondeterminism", e)
		}
		var requestSource, responseSource string
		for _, file := range got.Artifacts.Files {
			if strings.Contains(file.Path, "InputDto.kt") || strings.Contains(file.Path, "EchoRequest") {
				requestSource += file.Content
			}
			if strings.Contains(file.Path, "EchoPublicResponse") {
				responseSource += file.Content
			}
		}
		if !strings.Contains(requestSource, "val secret:") || !strings.Contains(requestSource, "Patch<kotlin.Int>") || !strings.Contains(responseSource, "val id:") || strings.Contains(responseSource, "val secret:") {
			t.Fatal("tuple selected DTO lost directional views", shared, requestSource, responseSource)
		}
	}
}

func TestHTTPOnlyOrdinaryDirectionalSourceAuthority(t *testing.T) {
	// A non-tuple control protects the common initialization fix: source origin
	// must exist even when there is no components section at all.
	item := `{"properties":{"name":{"type":"string"},"secret":{"type":"string","writeOnly":true},"id":{"type":"integer","readOnly":true}},"required":["name","secret","id"]}`
	input := `{"openapi":"3.1.0","paths":{"/echo":{"post":{"operationId":"echo","requestBody":{"required":true,"content":{"application/json":{"schema":` + item + `}}},"responses":{"200":{"description":"synthetic","content":{"application/json":{"schema":` + item + `}}}}}}}}`
	got, err := core.GenerateWithDiagnostics(request([]byte(input), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	api := source(t, got, "DefaultApi")
	if !strings.Contains(api, "EchoPublicResponseDto") {
		t.Fatal("HTTP-only response bypassed directional source planning", api)
	}
	response := source(t, got, "EchoPublicResponseDto")
	if strings.Contains(response, "val secret:") || !strings.Contains(response, "val id:") {
		t.Fatal("HTTP-only scalar control lost response projection", response)
	}
	inputSource := source(t, got, "EchoInputDto")
	if !strings.Contains(inputSource, "val secret:") || !strings.Contains(inputSource, "Patch<kotlin.Int>") {
		t.Fatal("HTTP-only scalar control lost request presence", inputSource)
	}
}

package core_test

import (
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func directionalDoc(properties string) []byte {
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Account":{"type":"object","required":["id","password","name"],"properties":{` + properties + `}}}},"paths":{"/account":{"post":{"operationId":"saveAccount","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Account"}}}},"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Account"}}}}}}}}}`)
}

func TestDirectionalSharedDTOs(t *testing.T) {
	input := directionalDoc(`"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true},"name":{"type":"string"}`)
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(input, "Dto")
			r.Options.Validation = validation
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			shared := source(t, got, "AccountDto")
			if !strings.Contains(shared, "public val password: kotlin.String") || !strings.Contains(shared, "public val id: kotlin.String") {
				t.Fatal(shared)
			}
			req := source(t, got, "SaveAccountRequestDto")
			if !strings.Contains(req, "id: org.example.models.models.Patch<kotlin.String>") || !strings.Contains(req, "password: kotlin.String") {
				t.Fatal(req)
			}
			res := source(t, got, "SaveAccountPublicResponseDto")
			if strings.Contains(res, "password") || !strings.Contains(res, "public val id: kotlin.String") {
				t.Fatal(res)
			}
			api := source(t, got, "DefaultApi")
			if !strings.Contains(api, "SaveAccountRequestDto") || !strings.Contains(api, "SaveAccountPublicResponseDto") {
				t.Fatal(api)
			}
		}
	}
}

func TestDirectionalRecursiveFixtureAndOptions(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/directional-models.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, optimized := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.DeduplicateModels = optimized
		r.Options.HoistSharedShapes = optimized
		r.Options.MergeIdenticalModels = optimized
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("directional output drift", err)
		}
		for _, artifact := range got.Artifacts.Files {
			if strings.Contains(artifact.Content, "??") {
				t.Fatal("nested nullable alias wrapper", artifact.Path)
			}
		}
		if !optimized {
			api := source(t, got, "DefaultApi")
			if !strings.Contains(api, "List<org.example.models.api.models.GetAliasesPublicResponseDto>?") {
				t.Fatal("nullable alias response lost its typed view", api)
			}
			response := source(t, got, "SaveAccountPublicResponseDto")
			if strings.Contains(response, "password") || !strings.Contains(response, "List<org.example.models.api.models.SaveAccountPublicResponseDto>") {
				t.Fatal(response)
			}
			empty := source(t, got, "GetSecretPublicResponseDto")
			if strings.Contains(empty, "JsonElement") || !strings.Contains(empty, "public class GetSecretPublicResponseDto") {
				t.Fatal(empty)
			}
		}
	}
}

func TestDirectionalFalseAnnotationsAndOpaqueExamples(t *testing.T) {
	plain := directionalDoc(`"id":{"type":"string"},"password":{"type":"string"},"name":{"type":"string"}`)
	annotated := []byte(strings.ReplaceAll(string(plain), `"type":"string"`, `"type":"string","writeOnly":false`))
	a, b := report(t, plain, "Dto"), report(t, annotated, "Dto")
	if !reflect.DeepEqual(a, b) {
		t.Fatal("writeOnly:false changed existing output")
	}
	example := []byte(strings.Replace(string(plain), `"type":"string"`, `"type":"string","example":{"writeOnly":true}`, 1))
	if got := report(t, example, "Dto"); !reflect.DeepEqual(a, got) {
		t.Fatal("example was interpreted as schema")
	}
}

func TestDirectionalInlineRequestKeepsSinglePatch(t *testing.T) {
	input := directionalDoc(`"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true},"name":{"type":"string"}`)
	inline := `{"type":"object","properties":{"id":{"type":"string","readOnly":true},"note":{"type":"string","nullable":true},"account":{"$ref":"#/components/schemas/Account"}}}`
	input = []byte(strings.Replace(string(input), `"schema":{"$ref":"#/components/schemas/Account"}`, `"schema":`+inline, 1))
	got := report(t, input, "Dto")
	text := source(t, got, "SaveAccountRequestDto")
	if strings.Contains(text, "Patch<org.example.models.models.Patch") || !strings.Contains(text, "note: org.example.models.models.Patch<kotlin.String>") || !strings.Contains(text, "id: org.example.models.models.Patch<kotlin.String>") {
		t.Fatal(text)
	}
}

func TestDirectionalDroppedFieldsStillValidate(t *testing.T) {
	input := directionalDoc(`"id":{"type":"string"},"password":{"type":"string","format":"unsupported","writeOnly":true},"name":{"type":"string"}`)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Account/properties/password/format")
}

func TestDirectionalCompositionRejectsRatherThanLeaks(t *testing.T) {
	input := directionalDoc(`"id":{"type":"string"},"password":{"type":"string","writeOnly":true},"name":{"type":"string"}`)
	input = []byte(strings.Replace(string(input), `"Account":{"type":"object",`, `"Account":{"allOf":[{"properties":{"token":{"type":"string","writeOnly":true}}}],"type":"object",`, 1))
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Account/properties/password/writeOnly")
}

func TestDirectionalObjectCountsRemainExplicitlyUnsupported(t *testing.T) {
	input := directionalDoc(`"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true},"name":{"type":"string"}`)
	input = []byte(strings.Replace(string(input), `"Account":{"type":"object",`, `"Account":{"minProperties":1,"type":"object",`, 1))
	r := request(input, "Dto")
	r.Options.Validation = true
	_, err := core.GenerateWithDiagnostics(r)
	diagnostics, ok := err.(core.Diagnostics)
	if !ok || len(diagnostics) != 1 || diagnostics[0].Code != "E_SCHEMA_UNSUPPORTED" || diagnostics[0].Pointer == nil || *diagnostics[0].Pointer != "/components/schemas/Account/minProperties" {
		t.Fatal("directional counts lost their source diagnostic", err)
	}
}

func TestDirectionalAnnotationScanPreservesReferenceDiagnostics(t *testing.T) {
	input := httpDoc(`{"responses":{"200":{"content":{"application/json":{"schema":{"anyOf":[{},{"type":"array","items":{"$ref":12}}]}}}}}}`)
	failure(t, input, "E_SCHEMA_INVALID", "/paths/~1ping/get/responses/200/content/application~1json/schema/anyOf/1/items")
}

func TestWriteOnlyAnnotationGuards(t *testing.T) {
	for _, annotation := range []string{`"writeOnly":"yes"`, `"writeOnly":true,"readOnly":true`} {
		failure(t, schemas(`{"Record":{"properties":{"secret":{"type":"string",`+annotation+`}}}}`), "E_SCHEMA_INVALID", "/components/schemas/Record/properties/secret/writeOnly")
	}
}

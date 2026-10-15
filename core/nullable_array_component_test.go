package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableArraySimpleBothFlags(t *testing.T) {
	for _, root := range []string{`"type":"array","nullable":true`, `"type":["array","null"]`} {
		for _, flag := range []bool{false, true} {
			req := request(schemas(`{"Root":{`+root+`,"items":{"type":["string","null"]}}}`), "")
			req.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(req)
			if err != nil || len(r.Artifacts.Files) != 1 || len(r.Warnings) != 0 {
				t.Fatal(flag, err, r)
			}
			if !strings.Contains(source(t, r, "Root"), "public typealias Root = kotlin.collections.List<kotlin.String?>?") {
				t.Fatal(r)
			}
		}
	}
}

func TestNullableArrayEnvelopeDeterminism(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/nullable-array-components.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "Dto"} {
		req := request(input, suffix)
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err != nil {
			t.Fatal(err)
		}
		again, err := core.GenerateWithDiagnostics(req)
		if err != nil || !reflect.DeepEqual(r, again) {
			t.Fatal("nullable alias determinism", err)
		}
		for _, file := range r.Artifacts.Files {
			if strings.Contains(file.Path, "NonNull") {
				t.Fatal("no extra exported alias", file.Path)
			}
		}
		api := source(t, r, "DefaultApi")
		for _, name := range []string{"Rows", "States"} {
			if !strings.Contains(api, "kotlin.collections.List<org.example.models.models."+name+suffix+"Inner>") {
				t.Fatal("API peer qualification", api)
			}
		}
	}
}

func TestNullableArrayAliasForwardViews(t *testing.T) {
	for _, root := range []string{`"type":"array","nullable":true`, `"type":["array","null"]`} {
		doc := schemas(`{"ACarrier":{"properties":{"inherit":{"$ref":"#/components/schemas/ZRows"},"narrow":{"$ref":"#/components/schemas/ZRows","nullable":false},"wrapped":{"allOf":[{"$ref":"#/components/schemas/ZRows"}]},"wrappedNarrow":{"allOf":[{"$ref":"#/components/schemas/ZRows"}],"nullable":false}},"required":["inherit","narrow","wrapped","wrappedNarrow"]},"ZRows":{` + root + `,"minItems":1,"items":{"properties":{"id":{"type":"string","minLength":1}},"required":["id"]}}}`)
		req := request(doc, "Dto")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err != nil {
			t.Fatal(err)
		}
		alias := source(t, r, "ZRowsDto")
		if !strings.Contains(alias, "public typealias ZRowsDto = kotlin.collections.List<ZRowsDtoInner>?") {
			t.Fatal(alias)
		}
		carrier := source(t, r, "ACarrierDto")
		for _, want := range []string{
			"inherit: org.example.models.models.ZRowsDto? = null",
			"wrapped: org.example.models.models.ZRowsDto? = null",
			"narrow: kotlin.collections.List<org.example.models.models.ZRowsDtoInner>,",
			"wrappedNarrow: kotlin.collections.List<org.example.models.models.ZRowsDtoInner>,",
			"has fewer entries than minItems",
		} {
			if !strings.Contains(carrier, want) {
				t.Errorf("missing %q: %s", want, carrier)
			}
		}
		for _, f := range r.Artifacts.Files {
			if strings.Contains(f.Path, "NonNull") {
				t.Fatal("view must not introduce a public alias", f.Path)
			}
		}
	}
}

func TestNullableArrayHistoricalDictionaryControl(t *testing.T) {
	doc := schemas(`{"Dict":{"nullable":true,"additionalProperties":{"type":"string"}},"Consumer":{"properties":{"direct":{"$ref":"#/components/schemas/Dict","nullable":false},"wrapped":{"allOf":[{"$ref":"#/components/schemas/Dict"}],"nullable":false}},"required":["direct","wrapped"]}}`)
	r := report(t, doc, "")
	text := source(t, r, "Consumer")
	for _, want := range []string{"direct: org.example.models.models.Dict? = null", "wrapped: org.example.models.models.Dict? = null"} {
		if !strings.Contains(text, want) {
			t.Fatal("nullable array views must not migrate dictionaries", text)
		}
	}
}

func TestNullableArrayBudgetsAndRejections(t *testing.T) {
	for _, n := range []int{240, 241} {
		var properties []string
		for i := range n {
			properties = append(properties, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/ZValues","nullable":false}`, i))
		}
		doc := schemas(`{"AConsumer":{"properties":{` + strings.Join(properties, ",") + `}},"ZValues":{"type":"array","nullable":true,"minItems":1,"items":{"type":["string","null"],"minLength":1}}}`)
		if n == 240 {
			r := aggregationReport(t, doc)
			if strings.Count(source(t, r, "AConsumer"), "private fun openapiKtValidatePrimitive") != 240 {
				t.Fatal("nullable views lost width helpers")
			}
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/AConsumer")
		}
	}
	for _, n := range []int{16, 17} {
		field := `{"$ref":"#/components/schemas/ZValues","nullable":false}`
		for range n - 1 {
			field = `{"type":"array","items":` + field + `}`
		}
		doc := schemas(`{"AConsumer":{"properties":{"v":` + field + `}},"ZValues":{"type":"array","nullable":true,"minItems":1,"items":{"type":"string","minLength":1}}}`)
		if n == 16 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/ZValues")
		}
	}
	for _, n := range []int{46, 47} {
		root := `{"type":"string"}`
		for range n - 1 {
			root = `{"type":"array","items":` + root + `}`
		}
		root = `{"type":"array","nullable":true,"items":` + root + `}`
		doc := schemas(`{"AConsumer":{"properties":{"v":{"$ref":"#/components/schemas/ZValues","nullable":false}}},"ZValues":` + root + `}`)
		if n == 46 {
			report(t, doc, "")
		} else {
			failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/AConsumer/properties/v")
		}
	}
	aliasFailure(t, schemas(`{"Root":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Root","nullable":false}}}`), "E_REF_CYCLE", "/components/schemas/Root")
	report(t, schemas(`{"Root":{"type":"object","nullable":true,"properties":{"id":{"type":"string"}}}}`), "") // Intrinsic direct ordinary DTO profile graduated independently.
	failure(t, schemas(`{"AArray":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/ZDict"}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/AArray","nullable":false}}},"ZDict":{"additionalProperties":{"type":"string","minLength":1}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/ZDict/additionalProperties/minLength")
}

func TestNullableArrayRequestPresenceAndAPIPeers(t *testing.T) {
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Rows":{"type":"array","nullable":true,"items":{"type":"string","enum":["open","closed"]}}}},"paths":{"/patch":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"optional":{"$ref":"#/components/schemas/Rows"},"wrapped":{"allOf":[{"$ref":"#/components/schemas/Rows"}]},"requiredNull":{"$ref":"#/components/schemas/Rows"},"requiredNarrow":{"$ref":"#/components/schemas/Rows","nullable":false}},"required":["requiredNull","requiredNarrow"]}}}},"responses":{"204":{}}}},"/rows":{"post":{"operationId":"echo","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Rows","nullable":false}}}},"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Rows","nullable":false}}}}}}}}}`)
	r := report(t, doc, "Dto")
	model := source(t, r, "SubmitInputDto")
	for _, want := range []string{
		"optional: org.example.models.models.Patch<kotlin.collections.List<org.example.models.models.RowsDtoInner>>",
		"wrapped: org.example.models.models.Patch<kotlin.collections.List<org.example.models.models.RowsDtoInner>>",
		"requiredNull: org.example.models.models.RowsDto?,",
		"requiredNarrow: kotlin.collections.List<org.example.models.models.RowsDtoInner>,",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("missing %q: %s", want, model)
		}
	}
	api := source(t, r, "DefaultApi")
	if strings.Count(api, "kotlin.collections.List<org.example.models.models.RowsDtoInner>") < 2 {
		t.Fatal("narrow body/response must qualify root peer", api)
	}
}

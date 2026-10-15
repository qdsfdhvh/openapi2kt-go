package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalScalarChainGuards(t *testing.T) {
	use := pointerRef("Source/properties/hop")
	for _, tc := range []struct{ hop, leaf, code, at string }{
		{pointerRef("Source/properties/hop"), `{"type":"string"}`, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v"},
		{pointerRef("Source/properties/leaf"), pointerRef("Source/properties/hop"), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v"},
		{pointerRef("Source"), `{"type":"string"}`, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v"},
		{pointerRef("Source/properties/missing"), `{"type":"string"}`, "E_REF_MISSING", "/components/schemas/Source/properties/hop/$ref"},
		{`{"$ref":true}`, `{"type":"string"}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/hop/$ref"},
		{`{"$ref":"#/components/schemas/Source/properties/leaf","nullable":0}`, `{"type":"string"}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/hop/nullable"},
		{`{"$ref":"#/components/schemas/Source/properties/leaf","minLength":1}`, `{"type":"string"}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/hop/minLength"},
		{pointerRef("Source/properties/leaf"), `{"type":"string","enum":["ok"]}`, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v"},
		{pointerRef("Source/properties/leaf/example"), `{"type":"string","example":{"type":"string"}}`, "E_REF_UNSUPPORTED", "/components/schemas/Source/properties/hop/$ref"},
	} {
		doc := schemas(`{"Consumer":{"properties":{"v":` + use + `}},"Source":{"properties":{"hop":` + tc.hop + `,"leaf":` + tc.leaf + `}}}`)
		if tc.hop == pointerRef("Source") {
			// Explicit graduation: the unchanged named ordinary self-owner is
			// a nominal recursion barrier, not a transparent scalar-ref cycle.
			for _, validation := range []bool{false, true} {
				rq := request(doc, "Dto")
				rq.Options.Validation = validation
				r, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.SourceDto? = null") {
					t.Fatal("named DTO terminal graduation", err)
				}
			}
			continue
		}
		if tc.leaf == `{"type":"string","enum":["ok"]}` {
			// The original chain now binds the retained nested vocabulary.
			for _, validation := range []bool{false, true} {
				rq := request(doc, "Dto")
				rq.Options.Validation = validation
				r, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.SourceDto.Leaf? = null") {
					t.Fatal("nested enum chain graduation", err)
				}
			}
			continue
		}
		aliasFailure(t, doc, tc.code, tc.at)
	}
	// Disabled validation cannot silently discard the terminal constraint.
	doc := schemas(`{"Consumer":{"properties":{"v":` + use + `}},"Source":{"properties":{"hop":` + pointerRef("Source/properties/leaf") + `,"leaf":{"type":"string","minLength":1}}}}`)
	failure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minLength")
}

func TestSchemaInternalScalarChainNoDropAndCheckedDepth(t *testing.T) {
	prefix := `{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"hop":` + pointerRef("Source/properties/leaf") + `,"leaf":{"type":"string","minLength":1}}}}},"paths":`
	for _, paths := range []string{
		`{ "/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"field":` + pointerRef("Source/properties/hop") + `}}}],"responses":{"204":{}}}}}`,
		`{ "/v":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"array","items":{"properties":{"field":` + pointerRef("Source/properties/hop") + `}}}}}},"responses":{"204":{}}}}}`,
	} {
		aliasFailure(t, []byte(prefix+paths+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minLength")
	}
	for _, n := range []int{16, 17} {
		ty := pointerRef("Source/properties/hop")
		for range n {
			ty = `{"type":"array","items":` + ty + `}`
		}
		doc := schemas(`{"Consumer":{"properties":{"v":` + ty + `}},"Source":{"properties":{"hop":` + pointerRef("Source/properties/leaf") + `,"leaf":{"type":"string","minLength":1}}}}`)
		if n == 16 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/leaf")
		}
	}
	// Scalar-checked cycles are excluded without unbounded traversal or recursion.
	doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Source/properties/hop") + `}},"Source":{"properties":{"hop":` + pointerRef("Source/properties/hop") + `}}}`)
	aliasFailure(t, doc, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
	if strings.Contains(source(t, aggregationReport(t, scalarChainDocument(1, 0)), "Consumer"), "typealias") {
		t.Fatal("chain introduced nominal declaration")
	}
}

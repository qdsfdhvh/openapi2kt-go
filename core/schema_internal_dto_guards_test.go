package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalDTOExcludedOwners(t *testing.T) {
	use := "/components/schemas/Consumer/properties/v"
	for _, tc := range []struct{ owner, path string }{
		{`{"properties":{"node":{}}}`, "Owner/properties/node"},
		{`{"properties":{"node":{"properties":{"id":{"type":"string","const":"fixed"}}}}}`, "Owner/properties/node"},
		{`{"properties":{"node":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}}`, "Owner/properties/node"},
		{`{"properties":{"nodes":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}}}`, "Owner/properties/nodes/items"},
		{`{"properties":{"nodes":{"additionalProperties":{"properties":{"id":{"type":"string"}}}}}}`, "Owner/properties/nodes/additionalProperties"},
		{`{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}]}`, "Owner/allOf/0/properties/node"},
		{`{"properties":{"hop":{"$ref":"#/components/schemas/Owner/properties/node"},"node":{"properties":{"id":{"type":"string"}}}}}`, "Owner/properties/hop"},
	} {
		doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef(tc.path) + `}},"Owner":` + tc.owner + `}`)
		canonical := map[string]string{
			"Owner/properties/hop":                        "OwnerDto.Node",
			"Owner/allOf/0/properties/node":               "OwnerDto.Node",
			"Owner/properties/nodes/items":                "OwnerDto.NodesInner",
			"Owner/properties/nodes/additionalProperties": "OwnerDto.NodesValue",
		}[tc.path]
		if strings.Contains(tc.owner, `"node":{"allOf"`) {
			canonical = "OwnerDto.Node"
		}
		if canonical != "" {
			// Explicit graduation of these unchanged, individually contracted inputs.
			for _, flag := range []bool{false, true} {
				rq := request(doc, "Dto")
				rq.Options.Validation = flag
				r, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models."+canonical+"? = null") {
					t.Fatal("canonical nominal chain graduation", flag, err)
				}
			}
			continue
		}
		aliasFailure(t, doc, "E_REF_UNSUPPORTED", use)
	}
	owner := `"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}`
	// Explicit graduation of the unchanged direct root/internal DTO input.
	for _, flag := range []bool{false, true} {
		rq := request(schemas(`{"Alias":`+pointerRef("Owner/properties/node")+`,`+owner+`}`), "Dto")
		rq.Options.Validation = flag
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, r, "AliasDto"), "public typealias AliasDto = org.example.models.models.OwnerDto.Node") {
			t.Fatal("root/internal DTO graduation", flag, err)
		}
	}
	aliasFailure(t, schemas(`{"Consumer":{"allOf":[`+pointerRef("Owner/properties/node")+`,{"properties":{"local":{"type":"string"}}}]},`+owner+`}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/allOf/0")
}

func TestSchemaInternalDTOConsumerDepthAndNoDrop(t *testing.T) {
	for _, outer := range []int{46, 47} {
		field := pointerRef("Owner/properties/node")
		for range outer {
			field = `{"type":"array","items":` + field + `}`
		}
		doc := schemas(`{"Consumer":{"properties":{"v":` + field + `}},"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}}`)
		if outer == 46 {
			report(t, doc, "")
		} else {
			failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v"+strings.Repeat("/items", outer))
		}
	}
	// Raw HTTP object/parameter projections must inspect canonical source checks.
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"node":{"properties":{"id":{"type":"string","minLength":2}}}}}}},"paths":{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"node":` + pointerRef("Owner/properties/node") + `}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Owner/properties/node/properties/id/minLength")
	// A nominal-looking source with only const-ref fields has no ctor properties.
	doc = schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Owner/properties/node") + `}},"Literal":{"type":"string","const":"fixed"},"Owner":{"properties":{"node":{"properties":{"id":{"$ref":"#/components/schemas/Literal"}}}}}}`)
	aliasFailure(t, doc, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
}

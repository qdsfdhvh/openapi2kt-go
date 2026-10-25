package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestInternalRootBridgeToNamedDTOIntersection(t *testing.T) {
	input := schemas(`{"Value":{"properties":{"id":{"type":"string"}},"required":["id"]},"Hops":{"properties":{"first":{"$ref":"#/components/schemas/Value"},"second":{"$ref":"#/components/schemas/Hops/properties/first"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/second"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"boolean"}}}]}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		report, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal("named-terminal internal bridge intersection rejected", e)
		}
		if !strings.Contains(source(t, report, "ConsumerDto"), "val id: kotlin.String,") {
			t.Fatal("actual original fields not retained")
		}
	}
}

func TestNamedDTOBridgeIntersectionSourceRolesAndNullableBoundary(t *testing.T) {
	for _, terminal := range []string{
		`{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}`,
		`{"allOf":[{"properties":{"id":{"type":"string"}}}]}`,
		`{"oneOf":[{"properties":{"id":{"type":"string"}}},{"properties":{"ok":{"type":"boolean"}}}]}`,
		`{"type":"object"}`,
		`{"properties":{"pin":{"const":"fixed"}}}`,
		`{"additionalProperties":{"type":"string"}}`,
	} {
		ref := "Node"
		if strings.Contains(terminal, `"node"`) {
			ref = "Node/properties/node"
		}
		input := schemas(`{"Node":` + terminal + `,"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/` + ref + `"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"boolean"}}}]}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			if terminal == `{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}` {
				r, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") || !strings.Contains(source(t, r, "BridgeDto"), "NodeDto.Node") {
					t.Fatal("original nested-source bridge lost originalfields/owner", e)
				}
				continue
			}
			if report, e := core.GenerateWithDiagnostics(rq); e == nil || len(report.Artifacts.Files) != 0 {
				t.Fatal("bridge granted unrelated shape role", terminal, e)
			}
		}
	}
	for _, count := range []int{44, 45, 46} {
		fields := []string{`"h0":{"$ref":"#/components/schemas/Value"}`}
		for i := 1; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/Hops/properties/h%d"}`, i, i-1))
		}
		input := schemas(`{"Value":{"properties":{"id":{"type":"string"}}},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(count-1) + `"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"boolean"}}}]}}`)
		_, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count <= 45 && e != nil {
			t.Fatal("accepted cachedsource bridge height rejected", count, e)
		}
		if count == 46 && (e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("bridge source cap lost", count, e)
		}
	}
}

func TestNamedDTOBridgeIntersectionConstructorWidth(t *testing.T) {
	for _, width := range []int{239, 240} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		input := schemas(`{"Value":{"properties":{` + strings.Join(fields, ",") + `}},"Hops":{"properties":{"v":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/v"},"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"extra":{"type":"string"}}}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if width == 239 && e != nil {
			t.Fatal("240slot aggregate rejected", e)
		}
		if width == 240 && (e == nil || len(r.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("combinedbridge width cap lost", e)
		}
	}
}

func TestNamedDTOBridgeIntersectionMemberNullViewAndOverride(t *testing.T) {
	base := `"Value":{"properties":{"id":{"type":"string"}}},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"}`
	for _, member := range []string{``, `,"nullable":false`, `,"nullable":true`} {
		input := schemas(`{` + base + `,"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"` + member + `},{"nullable":true,"properties":{"extra":{"type":"boolean"}}}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Consumer"}},"required":["value"]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))

		if e != nil {
			t.Fatal(member, e)
		}
		text := source(t, r, "HolderDto")
		nullable := strings.Contains(text, "ConsumerDto? = null")
		if nullable != (member == `,"nullable":true`) {
			t.Fatal("member null intent lost", member, text)
		}
	}
}

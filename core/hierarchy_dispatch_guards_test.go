package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func compoundGuardInput(fields int, vocabulary int) []byte {
	nodes := []string{`"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"family","mapping":{"group":"Group","other":"Other"}}}`, `"Group":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}]}`, `"Other":{"properties":{"family":{"type":"string","const":"other"}}}`}
	for i, name := range []string{"First", "Second"} {
		tag := fmt.Sprintf(`{"type":"string","const":"tag%d"}`, i)
		if vocabulary > 0 {
			start, end := 0, 120
			if i == 1 {
				start, end = 120, vocabulary
			}
			values := []string{}
			for n := start; n < end; n++ {
				values = append(values, fmt.Sprint("v", n))
			}
			encoded, _ := json.Marshal(values)
			tag = `{"enum":` + string(encoded) + `}`
		}
		props := []string{`"family":{"type":"string","const":"group"}`, `"tag":` + tag, fmt.Sprintf(`"mode":{"type":"string","const":"mode%d"}`, i)}
		required := []string{}
		for n := range fields {
			wire := fmt.Sprint("p", n)
			props = append(props, fmt.Sprintf(`%q:{"type":"boolean"}`, wire))
			required = append(required, fmt.Sprintf("%q", wire))
		}
		nodes = append(nodes, fmt.Sprintf(`%q:{"properties":{%s},"required":[%s]}`, name, strings.Join(props, ","), strings.Join(required, ",")))
	}
	return schemas("{" + strings.Join(nodes, ",") + "}")
}

func TestHierarchyCompoundSerializedSlotsAndPeerVocabulary(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{237, 238} {
			req := request(compoundGuardInput(count, 0), "Dto")
			req.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(req)
			if count == 237 {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/First" {
				t.Fatal(count, checked, got, err)
			}
		}
		for _, count := range []int{240, 241} {
			req := request(compoundGuardInput(1, count), "Dto")
			req.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(req)
			if count == 240 {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" {
				t.Fatal(count, checked, got, err)
			}
		}
	}
}

func TestHierarchyReusedOriginalDepthAndUnchangedOwnershipErrors(t *testing.T) {
	for _, count := range []int{48, 49} {
		nodes := []string{}
		for i := range count - 2 {
			nodes = append(nodes, fmt.Sprintf(`"D%d":{"oneOf":[{"$ref":"#/components/schemas/D%d"}],"discriminator":{"propertyName":"tag","mapping":{"one":"D%d"}}}`, i, i+1, i+1))
		}
		nodes = append(nodes, fmt.Sprintf(`"D%d":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"tag","mapping":{"one":"First","two":"Second"}}}`, count-2), `"First":{"properties":{"id":{"type":"string"}}}`, `"Second":{"properties":{"id":{"type":"string"}}}`)
		got, err := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(nodes, ",")+"}"), "Dto"))
		if count == 48 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(count, got, err)
		}
	}
	// Shared ownership is still a separate representation contract.
	failure(t, schemas(`{"A":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},"B":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},"Leaf":{"properties":{"id":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/B/oneOf/0/$ref")
	failure(t, schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"}],"discriminator":{"propertyName":"family"}},"Group":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},"Leaf":{"properties":{"family":{"type":"string","const":"one"},"tag":{"type":"integer"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Leaf/properties/tag")
}

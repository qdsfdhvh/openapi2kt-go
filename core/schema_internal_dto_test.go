package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalDTOEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-dtos.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			rq := request(input, suffix)
			rq.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatalf("internal DTOs flag=%t suffix=%q: %v", flag, suffix, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("nondeterministic", err)
			}
			owner := "org.example.models.models.TreeKey" + suffix
			for _, want := range []string{"node: " + owner + ".Node,", "narrowed: " + owner + ".Maybe,", "inherited: " + owner + ".Maybe? = null", "List<" + owner + ".Node>", "Map<kotlin.String, " + owner + ".Node>"} {
				if got := source(t, r, "Consumer"+suffix); !strings.Contains(got, want) {
					t.Errorf("missing %q: %s", want, got)
				}
			}
			if got := source(t, r, "DefaultApi"); !strings.Contains(got, "input: "+owner+".Node") {
				t.Fatal("direct HTTP root lost canonical DTO", got)
			}
			got := source(t, r, "TreeKey"+suffix)
			if strings.Count(got, "public data class Node(") != 1 || !strings.Contains(got, "peer: "+owner+".Maybe? = null") || !strings.Contains(got, "back: "+owner+".Node? = null") {
				t.Fatal("canonical recursive owners", got)
			}
			for _, want := range []string{"Patch<" + owner + ".Maybe>", "requiredNull: " + owner + ".Maybe?,", "requiredFalse: " + owner + ".Maybe,"} {
				if got := source(t, r, "SubmitInput"+suffix); !strings.Contains(got, want) {
					t.Errorf("missing presence %q: %s", want, got)
				}
			}
		}
	}
}

func TestSchemaInternalDTOSelfNestedAndChecks(t *testing.T) {
	doc := schemas(`{"Consumer":{"required":["v"],"properties":{"v":` + pointerRef("Owner/properties/node/properties/child") + `}},"Owner":{"properties":{"node":{"properties":{"child":{"required":["id"],"properties":{"id":{"type":"string","minLength":2},"self":` + pointerRef("Owner/properties/node/properties/child") + `}}}}}}}`)
	r := aggregationReport(t, doc)
	if got := source(t, r, "Consumer"); !strings.Contains(got, "v: org.example.models.models.Owner.Node.Child,") {
		t.Fatal(got)
	}
	if got := source(t, r, "Owner"); strings.Count(got, "public data class Child(") != 1 || !strings.Contains(got, "length >= 2") {
		t.Fatal(got)
	}
	failure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Owner/properties/node/properties/child/properties/id/minLength")
}

func TestSchemaInternalDTOLimits(t *testing.T) {
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":%s`, i, pointerRef("Owner/properties/node")))
		}
		doc := schemas(`{"Consumer":{"properties":{` + strings.Join(props, ",") + `}},"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}}`)
		if n == 240 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
}

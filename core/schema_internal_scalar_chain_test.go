package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalScalarChainEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-scalar-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.Validation = true
	a, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatalf("desired chain success: %v", err)
	}
	b, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("chain rerun differs", err)
	}
	record := source(t, a, "RecordDto")
	for _, want := range []string{"value: kotlin.String,", "nullable: kotlin.String?", "kotlin.collections.List<kotlin.String?>", "kotlin.collections.Map<kotlin.String, kotlin.String?>", "length >= 2", "duplicate entries for uniqueItems"} {
		if !strings.Contains(record, want) {
			t.Errorf("missing %q: %s", want, record)
		}
	}
	inputText := source(t, a, "SubmitInputDto")
	for _, want := range []string{"requiredNull: kotlin.String?,", "requiredFalse: kotlin.String,", "inherited: org.example.models.models.Patch<kotlin.String>", "wrapped: org.example.models.models.Patch<kotlin.String>", "memberFalse: kotlin.String? = null"} {
		if !strings.Contains(inputText, want) {
			t.Errorf("presence missing %q: %s", want, inputText)
		}
	}
}

// Equivalent terminal references must already generate before admitting chains.
// Keep the complete HTTP/SSE envelope and collapse only internal scalar hops.
func TestSchemaInternalScalarChainStructuralControl(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-scalar-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(input, &root); err != nil {
		t.Fatal(err)
	}
	props := root["components"].(map[string]any)["schemas"].(map[string]any)["Source"].(map[string]any)["properties"].(map[string]any)
	var expand func(any) any
	expand = func(node any) any {
		switch v := node.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/schemas/Source/properties/") {
				copy := map[string]any{"$ref": ref}
				if flag, ok := v["nullable"]; ok {
					copy["nullable"] = flag
				}
				for {
					key := strings.TrimPrefix(ref, "#/components/schemas/Source/properties/")
					key = strings.ReplaceAll(strings.ReplaceAll(key, "~1", "/"), "~0", "~")
					target := props[key].(map[string]any)
					next, ok := target["$ref"].(string)
					if !ok {
						break
					}
					if _, set := copy["nullable"]; !set {
						if flag, ok := target["nullable"]; ok {
							copy["nullable"] = flag
						}
					}
					ref = next
					copy["$ref"] = ref
				}
				return copy
			}
			out := map[string]any{}
			for k, child := range v {
				out[k] = expand(child)
			}
			return out
		case []any:
			out := make([]any, len(v))
			for i, child := range v {
				out[i] = expand(child)
			}
			return out
		}
		return node
	}
	control, err := json.Marshal(expand(root))
	if err != nil {
		t.Fatal(err)
	}
	aggregationReport(t, control)
}

func scalarChainDocument(edges, containers int) []byte {
	props := []string{`"terminal":{"type":"string"}`}
	next := "terminal"
	for i := edges - 1; i >= 0; i-- {
		key := fmt.Sprintf("hop%d", i)
		props = append(props, fmt.Sprintf(`%q:%s`, key, pointerRef("Source/properties/"+next)))
		next = key
	}
	use := pointerRef("Source/properties/" + next)
	for range containers {
		use = `{"type":"array","items":` + use + `}`
	}
	return schemas(`{"Consumer":{"required":["v"],"properties":{"v":` + use + `}},"Source":{"properties":{` + strings.Join(props, ",") + `}}}`)
}

func TestSchemaInternalScalarChainDepthAndSource(t *testing.T) {
	for _, flag := range []bool{false, true} {
		rq := request(scalarChainDocument(46, 0), "")
		rq.Options.Validation = flag
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("48 boundary", flag, err)
		}
	}
	aliasFailure(t, scalarChainDocument(47, 0), "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v")
	// Resolving the same target earlier must not reset a later nested use's budget.
	aliasFailure(t, scalarChainDocument(46, 1), "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v/items")
	ref := pointerRef("Source/properties/first")
	for _, tc := range []struct{ leaf, code, at string }{
		{`{"type":"string","minLength":-1}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/leaf/minLength"},
		{`{"type":"string","pattern":"(?=a)"}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/pattern"},
	} {
		doc := schemas(`{"Consumer":{"properties":{"v":` + ref + `}},"Source":{"properties":{"first":` + pointerRef("Source/properties/leaf") + `,"leaf":` + tc.leaf + `}}}`)
		aliasFailure(t, doc, tc.code, tc.at)
	}
}

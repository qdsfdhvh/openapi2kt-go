package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestNamedCollectionReferenceChainsFirstHTTPUnionAndSlots(t *testing.T) {
	for _, terminal := range []string{`{"type":"array","nullable":true,"items":{"type":"string"}}`, `{"additionalProperties":{"type":"string"},"nullable":true}`, `{"additionalProperties":true,"nullable":true}`, `{"type":"array","items":{"properties":{"id":{"type":"string"}},"required":["id"]}}`} {
		base := `"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":true}}}`
		for _, paths := range []string{
			`{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}}}}}}}`,
			`{"/write":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/value"}}}},"responses":{"204":{}}}}}`,
			`{"/query":{"get":{"parameters":[{"name":"value","in":"query","schema":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}}],"responses":{"204":{}}}}}`,
			`{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"oneOf":[{"properties":{"event":{"const":"value"},"data":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["data"]}]}}}}}}}}`,
		} {
			input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": json.RawMessage(`{` + base + `}`)}, "paths": json.RawMessage(paths)})
			if err != nil {
				t.Fatal(err)
			}
			for _, checked := range []bool{false, true} {
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || strings.Contains(source(t, got, "DefaultApi"), "models.ValueDto>") {
					t.Fatal("latecollectionlostrealbacking", checked, terminal, err)
				}
			}
		}
		input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"const":"a"},"value":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}},"required":["value"]},{"properties":{"kind":{"const":"b"},"id":{"type":"string"}},"required":["id"]}]}}`)
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
			t.Fatal("firstunionfieldmetadata lost namedcollection", terminal, err)
		}
	}
	for _, nullable := range []bool{false, true} {
		for _, n := range []int{240, 241} {
			props, req, params := []string{}, []string{}, []string{}
			for i := 0; i < n; i++ {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, nullable))
				req = append(req, fmt.Sprintf(`"p%d"`, i))
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":%t}}`, i, nullable))
			}
			base := `"Value":{"type":"array","nullable":true,"items":{"type":"string"}},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
			for _, tc := range []struct {
				input []byte
				code  string
			}{{schemas(`{` + base + `,"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(req, ",") + `]}}`), "E_MODEL_LIMIT"}, {[]byte(`{"openapi":"3.1.0","components":{"schemas":{` + base + `}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":{"204":{}}}}}}`), "E_OPERATION_LIMIT"}} {
				got, err := core.GenerateWithDiagnostics(request(tc.input, "Dto"))
				if n == 240 {
					if err != nil {
						t.Fatal(nullable, n, tc.code, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != tc.code || len(got.Artifacts.Files) != 0 {
					t.Fatal("collectionsingleslotchanged", nullable, n, tc.code, err)
				}
			}
		}
	}
}

// The actual checked Kotlin240 Map-parameter compile exposed a JVM method
// bytecode failure. Preserve that input and require checks in non-inline helpers
// before request callbacks, including cold SSE collection.
func TestNamedCollectionReferenceChainsWideCheckedOperationBytecode(t *testing.T) {
	var params []string
	for i := 0; i < 240; i++ {
		params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":false}}`, i))
	}
	base := `"Value":{"additionalProperties":{"type":"string","minLength":2},"minProperties":1},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
	for _, stream := range []bool{false, true} {
		count := 240
		response := `{"204":{}}`
		if stream {
			count = 239
			response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
		}
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + base + `}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params[:count], ",") + `],"responses":` + response + `}}}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		api := source(t, got, "DefaultApi")
		if strings.Count(api, "val length =") != 1 || !strings.Contains(api, "private fun `openapi-kt validate input") {
			t.Fatalf("wide checked operation still repeats inline leaf checks (stream=%v, length blocks=%d)", stream, strings.Count(api, "val length ="))
		}
		if strings.Count(api, "`openapi-kt validate input 1 1`(") != count+1 {
			t.Fatal("missing unchanged original per-input checks")
		}
		first := strings.Index(api, "`openapi-kt validate input 1 1`(p0)")
		request := strings.Index(api, "client.request(")
		if stream {
			request = strings.Index(api, "client.sse(")
		}
		if first < 0 || request <= first || !strings.Contains(api, "require(value.size >= 1)") || !strings.Contains(api, "length >= 2") {
			t.Fatal("request preflight order/checks lost")
		}
	}
}

func TestNamedCollectionReferenceChainsWideCheckedModelBytecode(t *testing.T) {
	var fields, names []string
	for i := 0; i < 240; i++ {
		fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":false}`, i))
		names = append(names, fmt.Sprintf(`"p%d"`, i))
	}
	input := schemas(`{"Value":{"additionalProperties":{"type":"string","minLength":2},"minProperties":1},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(names, ",") + `]}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	model := source(t, got, "HolderDto")
	init := strings.SplitN(strings.SplitN(model, "init {", 2)[1], "private fun", 2)[0]
	if strings.Contains(init, "val length =") || !strings.Contains(model, "private fun openapiKtValidatePrimitive") {
		t.Fatal("wide checked named Map constructor still inlines leaf loops")
	}
	if !strings.Contains(init, "this.openapiKtValidatePrimitive240(this.p239)") || !strings.Contains(model, "length >= 2") || !strings.Contains(model, "require(value.size >= 1)") {
		t.Fatal("original map field checks dropped")
	}
}

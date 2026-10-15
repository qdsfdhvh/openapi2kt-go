package core_test

import (
	"fmt"
	"strings"
	"testing"
)

// A named use adds one ref hop to its real schema nesting. Nullable wrappers
// add no structural edge; aliases are transparent, nominal DTOs are barriers.
func aliasDepthTree(n int, mixed bool) string {
	v := `{"type":["string","null"]}`
	for i := range n {
		if mixed && i%2 == 0 {
			v = `{"type":"object","nullable":true,"additionalProperties":` + v + `}`
		} else {
			v = `{"type":"array","nullable":true,"items":` + v + `}`
		}
	}
	// Direct array roots are not yet nullable; inner nullability adds no depth.
	return strings.Replace(v, `"nullable":true,`, "", 1)
}

func TestCollectionAliasConsumerCachedMixedAndBarriers(t *testing.T) {
	for _, n := range []int{44, 45} {
		z := aliasDepthTree(n, true)
		doc := schemas(`{"A":{"type":"array","items":{"$ref":"#/components/schemas/Z"}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/A"}}},"Z":` + z + `}`)
		if n == 44 {
			report(t, doc, "")
		} else {
			failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v")
		}
	}
	report(t, schemas(`{"Root":`+aliasDepthTree(48, false)+`}`), "")
	report(t, schemas(`{"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}},"Node":{"properties":{"next":{"$ref":"#/components/schemas/Rows"}}}}`), "")
}

func TestCollectionAliasConsumerHTTPAndSSEDepth(t *testing.T) {
	for _, sse := range []bool{false, true} {
		limit := 47 // HTTP body schema starts at zero; one named ref hop.
		if sse {
			limit = 46
		} // Event data is one schema field below the root.
		for _, n := range []int{limit, limit + 1} {
			pointer := "/paths/~1submit/post/requestBody/content/application~1json/schema"
			paths := `{"/submit":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Z"}}}},"responses":{"204":{}}}}}`
			if sse {
				pointer = "/paths/~1watch/get/responses/200/content/text~1event-stream/schema/properties/data"
				paths = `{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"type":"object","required":["data"],"properties":{"event":{"type":"string","const":"value"},"data":{"$ref":"#/components/schemas/Z"}}}}}}}}}}`
			}
			doc := []byte(`{"openapi":"3.0.3","components":{"schemas":{"Z":` + aliasDepthTree(n, false) + `}},"paths":` + paths + `}`)
			if n == limit {
				report(t, doc, "")
			} else {
				failure(t, doc, "E_SCHEMA_DEPTH", pointer)
			}
		}
	}
}

func TestCollectionAliasConsumerSchemaDepth(t *testing.T) {
	for _, tc := range []struct{ alias, wrappers int }{{46, 0}, {47, 0}, {45, 1}, {46, 1}, {42, 4}, {43, 4}} {
		t.Run(fmt.Sprintf("%d+%d", tc.alias, tc.wrappers), func(t *testing.T) {
			alias := `{"type":"string"}`
			for range tc.alias {
				alias = `{"type":"array","items":` + alias + `}`
			}
			field := `{"$ref":"#/components/schemas/ZAlias"}`
			for range tc.wrappers {
				field = `{"type":"array","items":` + field + `}`
			}
			doc := schemas(`{"Consumer":{"properties":{"v":` + field + `}},"ZAlias":` + alias + `}`)
			if 1+tc.wrappers+1+tc.alias <= 48 {
				report(t, doc, "")
			} else {
				failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v"+strings.Repeat("/items", tc.wrappers))
			}
		})
	}
}

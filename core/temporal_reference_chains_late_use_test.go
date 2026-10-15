package core_test

import (
	"strings"
	"testing"
)

func TestTemporalReferenceChainsFirstHTTPUse(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Time":{"type":"string","format":"date-time","nullable":true},"Hops":{"properties":{"time":{"$ref":"#/components/schemas/Time"}}}}},"paths":{"/time":{"get":{"operationId":"readTime","responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Hops/properties/time","nullable":false}}}}}}}}}`)
	r := report(t, input, "Dto")
	api := source(t, r, "DefaultApi")
	if !strings.Contains(api, "public suspend fun readTime(") || !strings.Contains(api, "): kotlin.time.Instant {") {
		t.Fatal("first late temporal use did not bind canonical non-null native scalar", api)
	}
}

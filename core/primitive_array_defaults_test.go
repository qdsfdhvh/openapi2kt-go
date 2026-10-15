package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveArrayDefaultsUseMappedFloatingIdentity(t *testing.T) {
	generate := func(value string) core.GenerationReport {
		t.Helper()
		input := schemas(`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"number","format":"float"},"default":` + value + `}}}}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		return report
	}
	first := generate(`[1.25,1]`)
	if !reflect.DeepEqual(first, generate(`[1.250,1.000000001]`)) {
		t.Fatal("Equivalent mapped Float defaults changed emitted identity")
	}
	if reflect.DeepEqual(first, generate(`[1,1.25]`)) {
		t.Fatal("Default array ordering disappeared")
	}
}

func TestPrimitiveArrayFieldDefaultsRetainTypedRequiredNullableAndOriginalMembers(t *testing.T) {
	input := schemas(`{"Holder":{"properties":{"kotlin":{"type":"string"},"required":{"type":"array","items":{"type":"string"},"default":["ready"]},"optional":{"type":"array","items":{"type":"integer","format":"int64"},"default":[-9223372036854775808,2]},"nullable":{"type":"array","nullable":true,"items":{"type":"boolean"},"default":null},"nullableItems":{"type":"array","items":{"type":"number","format":"float","nullable":true},"default":[1.25,null]},"empty":{"type":"array","items":{"type":"string"},"default":[]}},"required":["required"]}}`)
	for _, validation := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{"public val required: kotlin.collections.List<kotlin.String>,", "public val optional: kotlin.collections.List<kotlin.Long>? =", "public val nullable: kotlin.collections.List<kotlin.Boolean>? = null,", "kotlin.collections.List<kotlin.Float?>? =", "public val empty: kotlin.collections.List<kotlin.String>? ="} {
			if !strings.Contains(text, marker) {
				t.Fatal("Array annotation changed typed required/nullable/default policy", marker, text)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Array default source generation drifted", err)
		}
	}
}

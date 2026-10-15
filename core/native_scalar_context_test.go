package core_test

import (
	"strings"
	"testing"
)

func TestUntypedNativeScalarContexts(t *testing.T) {
	for _, tc := range []struct{ native, kotlin string }{
		{"date", "kotlin.time.Instant"},
		{"bigint", "kotlin.Long"},
		{"org.example.scalarvalues.Money", "org.example.scalarvalues.Money"},
	} {
		t.Run(tc.native, func(t *testing.T) {
			r := report(t, schemas(`{"Native":{"x-native-type":"`+tc.native+`"},"Record":{"type":"object","properties":{"value":{"$ref":"#/components/schemas/Native"},"direct":{"x-native-type":"`+tc.native+`"},"list":{"type":"array","items":{"x-native-type":"`+tc.native+`"}}},"required":["direct","list"]}}`), "Dto")
			if !strings.Contains(source(t, r, "NativeDto"), "public typealias NativeDto = "+tc.kotlin) {
				t.Fatal(source(t, r, "NativeDto"))
			}
			record := source(t, r, "RecordDto")
			if !strings.Contains(record, "kotlin.collections.List<"+tc.kotlin+">") {
				t.Fatal(record)
			}
			if tc.native == "bigint" {
				if strings.Count(record, "LongAsStringSerializer::class") != 2 {
					t.Fatal("direct/ref fields need annotation, list item must not", record)
				}
			} else if len(r.Artifacts.Files) != 2 {
				t.Fatal("caller owned type must not generate stand-in or helper", r)
			}
		})
	}
}

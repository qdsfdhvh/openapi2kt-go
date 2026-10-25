package schema

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestDiscardedConstValidationMatchesConstructedDiagnosticsAndOriginalNodes(t *testing.T) {
	for _, input := range []string{
		`{"const":{"$ref":"data","type":"data","enum":[null]}}`,
		`{"const":[null,{"value":1e2}],"nullable":true}`,
		`{"const":{},"default":{},"nullable":"bad"}`,
		`{"const":[],"type":"object","format":"json"}`,
		`{"const":{},"type":"array","x-native-type":"custom.Value"}`,
		`{"const":{},"type":false,"unknown":true}`,
		`{"const":{},"title":false,"nullable":"bad"}`,
		`{"const":[],"nullable":"bad","items":{"type":"string"}}`,
		`{"const":{},"format":false,"x-native-type":false}`,
		`{"const":"one"}`, `{"const":true}`, `{"const":null,"type":"null"}`,
	} {
		for _, depth := range []int{0, 48, 49} {
			node, e := document.Load([]byte(input), document.JSON, "")
			if e != nil {
				t.Fatal(e)
			}
			original := get(node, "const")
			before, _ := node.MarshalJSON()
			_, bodyError := constantProfile("validation", "field", node, "/actual/~0~1source", depth, true)
			if got := validateDiscardedConstant(node, "/actual/~0~1source", depth); !reflect.DeepEqual(got, bodyError) {
				t.Fatal("Discarded body changed original diagnostic ordering", input, depth, got, bodyError)
			}
			if structuredLiteral(node) {
				_, valueError := valueConstant("validation", "field", node, "/actual/~0~1source", depth)
				if got := validateDiscardedStructuredValue(node, "/actual/~0~1source", depth); !reflect.DeepEqual(got, valueError) {
					t.Fatal("Discarded extra changed value diagnostic ordering", input, depth, got, valueError)
				}
			}
			after, _ := node.MarshalJSON()
			if get(node, "const") != original || string(before) != string(after) {
				t.Fatal("Validation rewrote original schema/literal")
			}
		}
	}
}

func BenchmarkDiscardedStructuredConstValidation(b *testing.B) {
	node, e := document.Load([]byte(`{"const":{"payload":"`+strings.Repeat("a", 2*1024*1024)+`"}}`), document.JSON, "")
	if e != nil {
		b.Fatal(e)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if e := validateDiscardedStructuredValue(node, "/original/additionalProperties", 1); e != nil {
			b.Fatal(e)
		}
	}
}

func BenchmarkStructuredConstTemporaryInitializer(b *testing.B) {
	node, e := document.Load([]byte(`{"const":{"payload":"`+strings.Repeat("a", 2*1024*1024)+`"}}`), document.JSON, "")
	if e != nil {
		b.Fatal(e)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, e := valueConstant("validation", "extra", node, "/original/additionalProperties", 1); e != nil {
			b.Fatal(e)
		}
	}
}

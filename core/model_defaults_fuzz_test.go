package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzModelDefaults(f *testing.F) {
	f.Add(fieldDoc(`{"type":"string","default":"ready"}`))
	f.Add(fieldDoc(`{"type":"integer","default":2,"multipleOf":2}`))
	f.Add(fieldDoc(`{"type":"number","format":"float","default":0.300000004,"exclusiveMinimum":0.300000001}`))
	f.Add(schemas(`{"T":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"M":{"$ref":"#/components/schemas/T"},"N":{"$ref":"#/components/schemas/M","nullable":true,"default":null}}`))
	f.Add(schemas(`{"T":{"type":"array","items":{"type":"string"}},"N":{"$ref":"#/components/schemas/T","nullable":true,"default":["a"]}}`))
	f.Add(schemas(`{"T":{"type":["array","null"],"items":{"type":["string","null"]}},"M":{"$ref":"#/components/schemas/T","nullable":true},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":[null,"a"]}}`))
	f.Add(schemas(`{"T":{"type":"array","items":{"type":"string"},"default":["target"]},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":["outer"]}}`))
	f.Add(schemas(`{"T":{"type":"integer","minimum":0,"maximum":10,"multipleOf":2,"default":4},"M":{"$ref":"#/components/schemas/T","default":6}}`))
	f.Add(schemas(`{"T":{"type":"number","format":"float","default":0.300000004},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":0.5}}`))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 16384 {
			return
		}
		original := bytes.Clone(input)
		rq := core.GenerationRequest{Document: input, Options: core.GenerationOptions{PackageName: "org.example.fuzz", ModelNameSuffix: "Dto", Validation: true}}
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (ea == nil) != (eb == nil) || !reflect.DeepEqual(a, b) {
			t.Fatal("input mutation or nondeterministic default generation")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed generation returned artifacts")
		}
	})
}

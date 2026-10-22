package core_test

import (
	"bytes"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzJSONUnique(f *testing.F) {
	f.Add(fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"type":"string","format":"date-time","nullable":true}},"default":[{"a":"2026-01-02T03:04:05Z","b":null},{"b":null,"a":"2026-01-02T05:04:05+02:00"}]}`))
	f.Add(fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"type":"string","format":"uuid"}},"default":[{"a":"550e8400-e29b-41d4-a716-446655440000"},{"a":"550E8400E29B41D4A716446655440000"}]}`))
	f.Add(fieldDoc(`{"type":"array","items":true,"uniqueItems":true}`))
	f.Add(fieldDoc(`{"type":"array","items":{"additionalProperties":true},"uniqueItems":true}`))
	f.Add(fieldDoc(`{"type":"array","items":{"type":"string","format":"date-time"},"uniqueItems":true,"default":["2026-01-02T03:04:05Z","2026-01-02T05:04:05+02:00"]}`))
	f.Add(fieldDoc(`{"type":"array","items":{"type":"array","items":{"type":"string","format":"date","nullable":true},"uniqueItems":true},"default":[["2024-02-29",null]]}`))
	f.Add(fieldDoc(`{"type":"array","items":{"enum":["ready",null],"nullable":true},"uniqueItems":true,"default":["ready",null]}`))
	f.Add(fieldDoc(`{"type":"array","items":{"type":"array","items":{"enum":["ready"]},"uniqueItems":true},"default":[["ready","\u0072eady"]]}`))
	f.Add(schemas(`{"Code":{"enum":["ready",null],"nullable":true},"Alias":{"$ref":"#/components/schemas/Code"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"uniqueItems":true},"Record":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":["ready",null]}}}}`))
	f.Add(schemas(`{"Code":{"type":"integer","format":"int64","enum":[0,1]},"Alias":{"$ref":"#/components/schemas/Code"},"Rows":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Alias"}},"Record":{"properties":{"rows":{"$ref":"#/components/schemas/Rows","default":[0,-0]}}}}`))
	f.Add(fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"number","format":"float"},"default":[1,1.000000001]}`))
	f.Add(fieldDoc(`{"type":"array","items":{"type":"array","uniqueItems":true,"items":{"type":"number","enum":[0,-0.0,1,2]}},"default":[[0,-0.0]]}`))
	f.Add(schemas(`{"Raw":true,"Alias":{"$ref":"#/components/schemas/Raw","nullable":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"uniqueItems":true},"Record":{"properties":{"values":{"$ref":"#/components/schemas/Rows"}}}}`))
	f.Add(schemas(`{"Value":` + coverChainRoot("Other", "Other") + `,"Other":` + coverChainRoot("Value", "Value") + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Record":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"uniqueItems":true}}}}`))
	f.Add(schemas(`{"Value":{"allOf":[{"type":"object"},{"properties":{}}]},"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Record":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"uniqueItems":true}}}}`))
	f.Add(fieldDoc(`{"type":"array","items":{"type":"string","format":"uuid","nullable":true},"uniqueItems":true,"default":["550e8400-e29b-41d4-a716-446655440000","550E8400E29B41D4A716446655440000"]}`))
	f.Add(fieldDoc(`{"type":"array","uniqueItems":true,"items":{"type":"object","additionalProperties":{"type":"string","nullable":true}},"default":[{"a":"x","b":null},{"b":null,"a":"x"}]}`))
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

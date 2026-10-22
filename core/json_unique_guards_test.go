package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestJSONUniqueSourceEligibilityAndLimits(t *testing.T) {
	for _, leaf := range []string{`false`, `{"x-native-type":"kotlinx.serialization.json.JsonElement"}`, `{"type":"object","properties":{"v":{"type":"string"}}}`} {
		rq := request(schemas(`{"Raw":true,"Record":{"properties":{"value":{"type":"array","items":`+leaf+`,"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("type spelling granted JSON equality authority", leaf, got, err)
		}
	}
	mapUniqueGraduation(t, schemas(`{"Raw":true,"Record":{"properties":{"value":{"type":"array","items":{"type":"object","additionalProperties":{"type":"string"}},"uniqueItems":true}}}}`))
	for _, edges := range []int{16, 17} {
		field := `{"type":"array","items":true,"uniqueItems":true}`
		for i := 1; i < edges; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (edges == 16) {
			t.Fatal(edges, err)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal(got, err)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"type":"array","items":true,"uniqueItems":true}`, i)
		}
		rq := request(schemas(`{"Record":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (count == 240) {
			t.Fatal(count, err)
		}
		if err == nil && strings.Count(source(t, got, "RecordDto"), "private fun `openapi-kt unique JSON items`") != 1 {
			t.Fatal("wide helper duplicated", got)
		}
		if err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal(got, err)
		}
	}
	if _, err := core.Generate(request(fieldDoc(`{"type":"array","items":true,"uniqueItems":true}`), "Dto")); err == nil {
		t.Fatal("missing opt-in accepted")
	}
}

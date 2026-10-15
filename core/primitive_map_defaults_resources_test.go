package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveMapDefaultsSourceAndConstructorBudgets(t *testing.T) {
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":{"type":"object","additionalProperties":{"type":"integer"},"default":{"n":%d}}`, i, i)
		}
		report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if count == 240 && err != nil {
			t.Fatal(err)
		}
		if count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(report.Artifacts.Files) != 0) {
			t.Fatal("map bypassed constructor budget", err)
		}
	}
	for _, length := range []int{47, 48} {
		field := `{"type":"object","additionalProperties":{"type":"string"},"default":{"$ref":"#/data"}}`
		for i := 0; i < length; i++ {
			field = fmt.Sprintf(`{"properties":{"n%d":%s}}`, i, field)
		}
		report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":`+field+`}`), "Dto"))
		if length == 47 && err != nil {
			t.Fatal(err)
		}
		if length == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(report.Artifacts.Files) != 0) {
			t.Fatal("map source depth bypass", err)
		}
	}
	for _, literal := range []string{strings.Repeat("a", 70000), strings.Repeat("🙂", 18000)} {
		encoded, _ := json.Marshal(map[string]string{literal: literal})
		input := schemas(`{"Record":{"properties":{"value":{"type":"object","additionalProperties":{"type":"string"},"default":` + string(encoded) + `}}}}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil || !strings.Contains(source(t, report, "RecordDto"), literal) {
			t.Fatal("large key/value lost", err)
		}
	}
}

func FuzzPrimitiveMapDefaults(f *testing.F) {
	for _, s := range []string{`{}`, `{"a":1}`, `{"a/~":null}`, `{"$ref":"#/data"}`, `{"a":1.000000001}`, `{"a/~":"2024-02-29","nil":null}`, `{"stamp":"2024-02-29T12:34:56.123456789+02:30"}`, `{"bad":"2023-02-29"}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 || !json.Valid([]byte(value)) {
			return
		}
		for _, kind := range []string{`"integer"`, `"string"`, `"number","format":"float"`, `"string","format":"date"`, `"string","format":"date-time"`} {
			r := request(fieldDoc(`{"type":"object","additionalProperties":{"type":`+kind+`,"nullable":true},"default":`+value+`}`), "Dto")
			first, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				if len(first.Artifacts.Files) != 0 {
					t.Fatal("partial output")
				}
				continue
			}
			second, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatal("nondeterministic map", err)
			}
		}
	})
}

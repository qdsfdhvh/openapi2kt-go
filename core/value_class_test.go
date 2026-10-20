package core_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestValueClassGuardsAndSlots(t *testing.T) {
	for _, tc := range []struct{ schema, code, pointer string }{
		{`{"type":"string","x-kotlin-value-class":"yes"}`, "E_SCHEMA_INVALID", "/x-kotlin-value-class"},
		{`{"type":"string","x-kotlin-value-class":null}`, "E_SCHEMA_INVALID", "/x-kotlin-value-class"},
		{`{"type":"string","enum":["x"],"x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/enum"},
		{`{"type":"integer","const":1,"x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/const"},
		{`{"type":"string","format":"date","x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"type":"string","format":"binary","x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"type":"string","x-native-type":"bigint","x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/x-native-type"},
		{`{"type":"integer","default":1,"x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"type":"object","x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/x-kotlin-value-class"},
		{`{"type":"array","items":{"type":"string"},"x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/x-kotlin-value-class"},
	} {
		failure(t, schemas(`{"Id":`+tc.schema+`}`), tc.code, "/components/schemas/Id"+tc.pointer)
	}
	failure(t, fieldDoc(`{"type":"string","x-kotlin-value-class":true}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/x-kotlin-value-class")
	for _, format := range []string{"int64", "double"} {
		kind := "integer"
		if format == "double" {
			kind = "number"
		}
		for _, n := range []int{120, 121} {
			fields, required := make([]string, n), make([]string, n)
			for i := range n {
				fields[i] = fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Wide"}`, i)
				required[i] = fmt.Sprintf(`"p%d"`, i)
			}
			input := schemas(`{"Wide":{"type":"` + kind + `","format":"` + format + `","x-kotlin-value-class":true},"Record":{"type":"object","properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
			if n == 120 {
				report(t, input, "")
			} else {
				failure(t, input, "E_MODEL_LIMIT", "/components/schemas/Record")
			}
		}
	}
	composed := schemas(`{"Id":{"type":"string","x-kotlin-value-class":true},"Record":{"type":"object","properties":{"nullable":{"anyOf":[{"$ref":"#/components/schemas/Id"},{"type":"null"}]},"wrapped":{"allOf":[{"$ref":"#/components/schemas/Id"}],"nullable":true}}}}`)
	if strings.Count(source(t, report(t, composed, "Dto"), "RecordDto"), "org.example.models.models.IdDto? = null") != 2 {
		t.Fatal("nullable/ref-wrapper lost nominal identity")
	}
	failure(t, []byte(`{"openapi":"3.1.0",
 "components":{"schemas":{"Id":{"type":"string","x-kotlin-value-class":true}}},
 "paths":{"/watch":{"get":{
  "parameters":[{"name":"Last-Event-ID","in":"header","schema":{"$ref":"#/components/schemas/Id"}}],
  "responses":{"200":{"content":{"text/event-stream":{"schema":{"type":"object","properties":{}}}}}}
 }}}
}`), "E_OPERATIONS_UNSUPPORTED", "/paths/~1watch/get/parameters")
	input := schemas(`{"Id":{"type":"string","x-kotlin-value-class":false}}`)
	expected := schemas(`{"Id":{"type":"string"}}`)
	if !reflect.DeepEqual(report(t, input, "Dto"), report(t, expected, "Dto")) {
		t.Fatal("false changed aliases")
	}
}

func TestValueClassMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		good, bad := 120, 121
		if stream {
			good, bad = 119, 120
		}
		for _, n := range []int{good, bad} {
			params := make([]string, n)
			for i := range n {
				params[i] = fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Wide"}}`, i)
			}
			response := `{"204":{}}`
			if stream {
				response = `{"200":{"content":{"text/event-stream":{"schema":{"type":"object","properties":{}}}}}}`
			}
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Wide":{"type":"integer","format":"int64","x-kotlin-value-class":true}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
			if n == good {
				report(t, input, "")
			} else {
				failure(t, input, "E_OPERATION_LIMIT", "/paths/~1wide/get/parameters")
			}
		}
	}
}

func TestPrimitiveValueClasses(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "value-classes.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	for _, tc := range []struct{ name, scalar string }{{"UserId", "String"}, {"OtherId", "uuid.Uuid"}, {"Counter", "Long"}, {"Index", "Int"}, {"Enabled", "Boolean"}, {"Ratio", "Double"}, {"Weight", "Float"}} {
		text := source(t, r, tc.name+"Dto")
		for _, token := range []string{"@kotlinx.serialization.Serializable", "@kotlin.jvm.JvmInline", "public value class " + tc.name + "Dto(", "public val rawValue: kotlin." + tc.scalar} {
			if !strings.Contains(text, token) {
				t.Fatal(token, text)
			}
		}
	}
	if !strings.Contains(source(t, r, "PlainDto"), "public typealias PlainDto = kotlin.String") {
		t.Fatal(r)
	}
	record := source(t, r, "RecordDto")
	for _, token := range []string{"id: org.example.models.models.UserIdDto", "counter: org.example.models.models.CounterDto", "org.example.models.models.UserIdDto?>", "org.example.models.models.OtherIdDto? = null"} {
		if !strings.Contains(record, token) {
			t.Fatal(token, record)
		}
	}
	api := source(t, r, "TypesApi")
	for _, token := range []string{"id.rawValue.toString().encodeURLPathPart()", "it?.rawValue", "value?.rawValue", "it.rawValue", "deepValue.active.rawValue", "Flow<org.example.models.api.models.WatchEventDto>", "decodeFromString<org.example.models.models.CounterDto>"} {
		if !strings.Contains(api, token) {
			t.Fatal(token, api)
		}
	}
	for _, w := range r.Warnings {
		if w.Code == "W_HTTP_PARAMETER_PROJECTION" {
			t.Fatal("simple wrapper treated as structured", w)
		}
	}
	if !reflect.DeepEqual(r, report(t, input, "Dto")) {
		t.Fatal("not deterministic")
	}
}

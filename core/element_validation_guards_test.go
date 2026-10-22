package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDirectElementValidationGuards(t *testing.T) {
	for _, tc := range []struct{ leaf, code, suffix string }{
		{`{"type":"string","minLength":-1}`, "E_SCHEMA_INVALID", "minLength"},
		{`{"type":"string","maxLength":true}`, "E_SCHEMA_INVALID", "maxLength"},
		{`{"type":"string","minLength":3,"maxLength":2}`, "E_SCHEMA_INVALID", "maxLength"},
		{`{"type":"boolean","minimum":0}`, "E_SCHEMA_UNSUPPORTED", "minimum"},
		{`{"type":"integer","format":"int64","minimum":1.5}`, "E_SCHEMA_INVALID", "minimum"},
		{`{"type":"number","minimum":1e-999999}`, "E_SCHEMA_INVALID", "minimum"},
		{`{"type":"string","minimum":0}`, "E_SCHEMA_UNSUPPORTED", "minimum"},
		{`{"type":"number","pattern":"a"}`, "E_SCHEMA_UNSUPPORTED", "pattern"},
		{`{"type":"string","format":"date","minLength":1}`, "E_SCHEMA_UNSUPPORTED", "minLength"},
		{`{"type":"integer","default":0}`, "E_SCHEMA_UNSUPPORTED", "default"},
		{`{"type":"string","pattern":"(?<=a)b"}`, "E_SCHEMA_UNSUPPORTED", "pattern"},
	} {
		for _, container := range []struct{ schema, pointer string }{
			{`{"type":"array","items":` + tc.leaf + `}`, "/items/"},
			{`{"type":"object","additionalProperties":` + tc.leaf + `}`, "/additionalProperties/"},
		} {
			aliasFailure(t, fieldDoc(container.schema), tc.code, "/components/schemas/Record/properties/value"+container.pointer+tc.suffix)
		}
	}
	floatingUniqueGraduation(t, fieldDoc(`{"type":"array","items":{"type":"number","minimum":1},"uniqueItems":true}`))

	counted := fieldDoc(`{"type":"object","properties":{"v":{"type":"string"}},"minProperties":1}`)
	countedRequest := request(counted, "Dto")
	countedRequest.Options.Validation = true
	countedReport, err := core.GenerateWithDiagnostics(countedRequest)
	if err != nil {
		t.Fatal("ordinary nested DTO must retain JSON counts", err)
	}
	for _, token := range []string{"KeepGeneratedSerializer", "generatedSerializer()", "element.size < 1"} {
		if !strings.Contains(source(t, countedReport, "RecordDto"), token) {
			t.Fatal("nested count propagation lost", token)
		}
	}
	failure(t, counted, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/minProperties")
	// Overridden fields and ignored fixed-object extras have no checking slot.
	aliasFailure(t, schemas(`{"Record":{"allOf":[{"type":"object","properties":{"v":{"type":"array","items":{"type":"string","minLength":1}}}}],"type":"object","properties":{"v":{"type":"array","items":{"type":"string"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/v/items/minLength")
	aliasFailure(t, schemas(`{"Record":{"type":"object","properties":{"v":{"type":"string"}},"additionalProperties":{"type":"array","items":{"type":"string","minLength":1}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/additionalProperties/items/minLength")
	// Non-object root request union variants become unchecked empty singletons.
	aliasFailure(t, httpDoc(`{"requestBody":{"content":{"application/json":{"schema":{"anyOf":[{"type":"array","items":{"type":"string","minLength":1}},{"type":"boolean"}]}}}},"responses":{"204":{}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1ping/get/requestBody/content/application~1json/schema/anyOf/0/items/minLength")
}

func TestDirectElementValidationResourcesAndDefaults(t *testing.T) {
	input := fieldDoc(`{"type":"array","items":{"type":"string","minLength":1}}`)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/items/minLength")
	plain := fieldDoc(`{"type":"array","items":{"type":["integer","null"]}}`)
	req := request(plain, "")
	before := report(t, plain, "")
	req.Options.Validation = true
	after, err := core.GenerateWithDiagnostics(req)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("unbounded output changed", err)
	}
	for _, depth := range []int{16, 17, 48} {
		leaf := `{"type":"string","minLength":1}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		req := request(fieldDoc(leaf), "")
		req.Options.Validation = true
		if depth == 16 {
			if _, err := core.Generate(req); err != nil {
				t.Fatal("supported depth", err)
			}
		} else {
			aliasFailure(t, req.Document, "E_SCHEMA_DEPTH", "/components/schemas/Record/properties/value"+strings.Repeat("/items", depth))
		}
	}
	// Transparent component aliases do not reset the checked collection depth.
	leaf := `{"type":"string","minLength":1}`
	for i := 0; i < 16; i++ {
		leaf = `{"type":"object","additionalProperties":` + leaf + `}`
	}
	aliasFailure(t, schemas(`{"Bag":`+leaf+`,"Record":{"type":"object","properties":{"v":{"type":"array","items":{"$ref":"#/components/schemas/Bag"}}}}}`), "E_SCHEMA_DEPTH", "/components/schemas/Bag"+strings.Repeat("/additionalProperties", 16))
	for _, count := range []int{240, 241} {
		var fields []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"array","items":{"type":"integer","minimum":0}}`, i))
		}
		input := schemas(`{"Record":{"type":"object","properties":{` + strings.Join(fields, ",") + `}}}`)
		req := request(input, "")
		req.Options.Validation = true
		if count == 240 {
			if _, err := core.Generate(req); err != nil {
				t.Fatal("checks added JVM slots", err)
			}
		} else {
			aliasFailure(t, input, "E_MODEL_LIMIT", "/components/schemas/Record")
		}
	}
}

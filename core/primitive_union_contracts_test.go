package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func semanticFixture(t testing.TB, name string) []byte {
	t.Helper()
	input, err := os.ReadFile(filepath.Join(testutil.Root(), "..", "model-semantics", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestPrimitiveUnionNativePayloadScopeAndPinWarnings(t *testing.T) {
	for _, suffix := range []string{"", "Dto"} {
		input := semanticFixture(t, "primitive-unions")
		r := report(t, input, suffix)
		s := source(t, r, "Record"+suffix)
		for _, part := range []string{"public sealed class Value", "public data class IntValue(", "public val value: kotlin.Long", "public val value: kotlin.Double", "public data class BooleanValue(", "public data class StringValue(", "public val onlyString: kotlin.String,", "public val items: kotlin.collections.List<ItemsInner?>,", "public val nullable: Nullable? = null", "public data class ObjectMixed(", "public class ObjectConstantObject", `public val tag: kotlin.String = "object"`} {
			if !strings.Contains(s, part) {
				t.Fatal("missing", part)
			}
		}
		if strings.Index(s, "element.booleanOrNull") > strings.Index(s, "element.longOrNull") || strings.Index(s, "element.longOrNull") > strings.Index(s, "element.doubleOrNull") {
			t.Fatal("JSON kind dispatch precedence")
		}
		if !reflect.DeepEqual(r, report(t, input, suffix)) {
			t.Fatal("non-deterministic primitive report")
		}
	}
	for _, pin := range []string{`"const":"tag"`, `"enum":["tag"]`} {
		r := report(t, fieldDoc(`{"oneOf":[{"type":"string",`+pin+`},{"type":"integer"}]}`), "")
		s := source(t, r, "Record")
		if !strings.Contains(s, "public data class StringValue(") || strings.Contains(s, `public val value: kotlin.String =`) {
			t.Fatal("pin became body default")
		}
		if len(r.Warnings) != 2 || r.Warnings[0].Code != "W_PRIMITIVE_UNION_PIN_APPROXIMATION" {
			t.Fatal(r.Warnings)
		}
	}
}

func TestPrimitiveUnionOriginalErrorsAndOwnedPeers(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"oneOf":[{"type":"string"},{"type":"number","minimum":0}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/1/minimum"},
		{`{"oneOf":[{"type":"string","format":"unsupported-format"},{"type":"integer"}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/0/format"},
		{`{"oneOf":[{"type":"string"},false]}`, "E_SCHEMA_INVALID", "oneOf/1"},
		{`{"oneOf":[{"type":"string","nullable":true},{"type":"integer"}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf/0/nullable"},
		{`{"oneOf":[{"type":"integer","format":"int32"},{"type":"integer","format":"int64"}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf"},
		{`{"oneOf":[{"type":"string","enum":["a"]},{"type":"string","enum":["b"]}]}`, "E_SCHEMA_UNSUPPORTED", "oneOf"},
		{`{"oneOf":[{"type":"string","const":"a","deprecated":1},{"type":"integer"}]}`, "E_SCHEMA_INVALID", "oneOf/0/deprecated"},
		{`{"oneOf":[{"type":"string"},{"type":"integer","const":2147483648}]}`, "E_SCHEMA_INVALID", "oneOf/1/const"},
	} {
		failure(t, fieldDoc(tc.field), tc.code, "/components/schemas/Record/properties/value/"+tc.tail)
	}
	failure(t, schemas(`{"Record":{"type":"object","properties":{"value":{"oneOf":[{"type":"string"},{"type":"integer"}]},"stringValue":{"type":"object","properties":{"id":{"type":"string"}}}}}}`), "E_NAME_COLLISION", "/components/schemas/Record/properties/stringValue")
	failure(t, schemas(`{"Root":{"oneOf":[{"type":"string"},{"type":"integer"}]},"StringRoot":{"type":"object"}}`), "E_NAME_COLLISION", "/components/schemas/Root/oneOf")
}

func TestPrimitiveUnionBranchDepthAndNamedSlots(t *testing.T) {
	for _, count := range []int{240, 241} {
		branches := make([]string, count)
		for i := range count {
			branches[i] = `{"type":"string"}`
		}
		_, err := core.Generate(request(fieldDoc(`{"oneOf":[`+strings.Join(branches, ",")+`]}`), ""))
		if (err == nil) != (count == 240) || err != nil && err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" {
			t.Fatal(count, err)
		}
	}
	for _, count := range []int{46, 47} {
		field := `{"oneOf":[{"type":"string"},{"type":"boolean"}]}`
		for i := range count {
			field = `{"type":"object","properties":{"level` + fmt.Sprint(i) + `":` + field + `}}`
		}
		_, err := core.Generate(request(fieldDoc(field), ""))
		if (err == nil) != (count == 46) || err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(count, err)
		}
	}
	for _, last := range []string{`{"type":"boolean"}`, `{"type":"integer","format":"int64"}`} {
		var props, required []string
		for i := range 119 {
			name := fmt.Sprintf("field%d", i)
			props = append(props, fmt.Sprintf(`%q:{"type":"integer","format":"int64"}`, name))
			required = append(required, fmt.Sprintf("%q", name))
		}
		props = append(props, `"union":{"oneOf":[{"type":"string"},{"type":"integer"}]}`, `"last":`+last)
		required = append(required, `"union"`, `"last"`)
		input := schemas(`{"Record":{"type":"object","properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		_, err := core.Generate(request(input, ""))
		ok := strings.Contains(last, "boolean")
		if (err == nil) != ok || err != nil && err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
			t.Fatal(last, err)
		}
	}
}

func TestNestedNormalizationOpenVocabWidthAndSourceConstraints(t *testing.T) {
	r := report(t, fieldDoc(`{"type":"array","items":{"oneOf":[{"enum":["queued","ready"]},{"type":"string","enum":["ready","complete"]},{"type":"string","pattern":"^[a-z]+$"},{"type":"null"}]}}`), "Dto")
	s := source(t, r, "RecordDto")
	if !strings.Contains(s, "kotlin.collections.List<ValueInner?>") || strings.Count(s, "data object READY") != 1 || strings.Index(s, "data object QUEUED") > strings.Index(s, "data object READY") || strings.Index(s, "data object READY") > strings.Index(s, "data object COMPLETE") {
		t.Fatal(s)
	}
	if r.Warnings[0].Code != "W_UNION_CONSTRAINT_APPROXIMATION" || *r.Warnings[0].Pointer != "/components/schemas/Record/properties/value/items/oneOf/2/pattern" {
		t.Fatal(r.Warnings)
	}
	wide := `{"anyOf":[{"type":"integer","minimum":-2147483649},{"type":"integer","maximum":2147483648}]}`
	if !strings.Contains(source(t, report(t, fieldDoc(wide), ""), "Record"), "public val value: kotlin.Long,") {
		t.Fatal("lost range width")
	}
	for _, count := range []int{120, 121} {
		var props, required []string
		for i := range count {
			name := fmt.Sprintf("field%d", i)
			props = append(props, fmt.Sprintf("%q:%s", name, wide))
			required = append(required, fmt.Sprintf("%q", name))
		}
		_, err := core.Generate(request(schemas(`{"Record":{"type":"object","properties":{`+strings.Join(props, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), ""))
		if (err == nil) != (count == 120) || err != nil && err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
			t.Fatal(count, err)
		}
	}
	for _, tc := range []struct{ field, code, tail string }{
		{`{"anyOf":[{"type":"string","minLength":"bad"},{"type":"string"}]}`, "E_SCHEMA_INVALID", "anyOf/0/minLength"},
		{`{"anyOf":[{"type":"string","minLength":1.0},{"type":"string"}]}`, "E_SCHEMA_INVALID", "anyOf/0/minLength"},
		{`{"anyOf":[{"type":"string","minLength":9,"maxLength":1},{"type":"string"}]}`, "E_SCHEMA_INVALID", "anyOf/0/maxLength"},
		{`{"anyOf":[{"type":"number","minimum":false},{"type":"number"}]}`, "E_SCHEMA_INVALID", "anyOf/0/minimum"},
		{`{"anyOf":[{"type":"number","minimum":9,"maximum":1},{"type":"number"}]}`, "E_SCHEMA_INVALID", "anyOf/0/maximum"},
		{`{"anyOf":[{"type":"string","format":false},{"type":"string"}]}`, "E_SCHEMA_INVALID", "anyOf/0/format"},
		{`{"anyOf":[{"type":"string","default":"bad"},{"type":"string"}]}`, "E_SCHEMA_UNSUPPORTED", "anyOf/0/default"},
		{`{"anyOf":[{"enum":["a","a"]},{"type":"string"}]}`, "E_SCHEMA_INVALID", "anyOf/0/enum/1"},
	} {
		failure(t, fieldDoc(tc.field), tc.code, "/components/schemas/Record/properties/value/"+tc.tail)
	}
	vocabulary := make([]string, 240)
	for i := range vocabulary {
		vocabulary[i] = fmt.Sprintf("v%d", i)
	}
	encoded, _ := json.Marshal(vocabulary)
	good := `{"anyOf":[{"enum":` + string(encoded) + `},{"type":"string"}]}`
	if !strings.Contains(source(t, report(t, fieldDoc(good), ""), "Record"), "data object V239") {
		t.Fatal("lost merged enum boundary")
	}
	bad := `{"anyOf":[{"enum":` + string(encoded) + `},{"enum":["extra"]},{"type":"string"}]}`
	failure(t, fieldDoc(bad), "E_ENUM_LIMIT", "/components/schemas/Record/properties/value/anyOf/1/enum")
	_ = report(t, fieldDoc(`{"anyOf":[{"enum":["a"],"examples":[{"anyOf":false,"type":false}]},{"type":"string"}]}`), "")
}

func TestPrimitiveRootNamedNullAndOriginalPinSources(t *testing.T) {
	r := report(t, schemas(`{"NativeFlag":{"type":"boolean","const":false},"Choice":{"oneOf":[{"type":"string","enum":["a"]},{"type":"integer","format":"int32","const":12},{"type":"number","format":"float","enum":[1.5]},{"$ref":"#/components/schemas/NativeFlag"},{"type":"null"}]},"Holder":{"type":"object","required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Choice"},"items":{"type":"array","items":{"$ref":"#/components/schemas/Choice"}}}}}`), "Dto")
	s := source(t, r, "ChoiceDto")
	for _, part := range []string{"public data class IntChoiceDto(", "public val value: kotlin.Long", "public val value: kotlin.Double", "public data object NullChoiceDto", "Expected JSON null for a root null variant."} {
		if !strings.Contains(s, part) {
			t.Fatal("missing", part)
		}
	}
	if !strings.Contains(source(t, r, "HolderDto"), "public val v: org.example.models.models.ChoiceDto,") {
		t.Fatal("root ref incorrectly nullable")
	}
	seen := false
	for _, w := range r.Warnings {
		if w.Code == "W_PRIMITIVE_UNION_PIN_APPROXIMATION" && *w.Pointer == "/components/schemas/NativeFlag/const" {
			seen = true
		}
	}
	if !seen {
		t.Fatal("lost original ref pin source")
	}
	for _, kind := range []string{"string", "integer", "boolean"} {
		r := report(t, schemas(`{"Root":{"anyOf":[{"type":"`+kind+`"},{"type":"null"}]}}`), "")
		if !strings.Contains(source(t, r, "Root"), "public data object NullRoot") || strings.Contains(source(t, r, "Root"), "typealias") {
			t.Fatal("root was nested-folded")
		}
	}
	failure(t, schemas(`{"Root":{"anyOf":[{"type":"string"},{"type":"string"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/anyOf")
}

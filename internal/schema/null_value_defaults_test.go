package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestJsonNullDefaultLeavesKeepOriginalSourcePermission(t *testing.T) {
	for _, raw := range []string{`{"type":"null"}`, `{"enum":[null]}`, `{"type":"null","enum":[null],"nullable":true}`} {
		v, parse := document.ParseObject([]byte(raw), document.MaxDocumentBytes)
		if parse != nil {
			t.Fatal(parse)
		}
		before, _ := v.MarshalJSON()
		a := analyzer{options: Options{Validation: true}, validationSources: map[*PrimitiveValidation]string{}, emissionHints: EmissionHints{}}
		declarations := []Declaration{}
		node, failure := a.nestedDefaultSchema(v, "Unemitted", "/source/leaf", &declarations, map[string]bool{}, 0, false)
		if failure != nil || node == nil || !node.nullOnly || node.schema != v || node.pointer != "/source/leaf" || node.ty.WithoutRootNull().Name != "kotlinx.serialization.json.JsonNull" || len(declarations) != 0 || len(a.enumOrigins) != 0 {
			t.Fatal("null grammar invented type/declaration/source", failure)
		}
		wrapper, parse := document.ParseObject([]byte(`{"value":null,"wrong":"null"}`), document.MaxDocumentBytes)
		if parse != nil {
			t.Fatal(parse)
		}
		literal := get(wrapper, "value")
		if text, failure := node.value(literal, "/value"); failure != nil || text != "null" {
			t.Fatal("JsonNull domain required Kotlin nullable intent", failure, text)
		}
		if _, failure := node.value(get(wrapper, "wrong"), "/wrong"); failure == nil || failure.Pointer == nil || *failure.Pointer != "/wrong" {
			t.Fatal("wrong null token was coerced", failure)
		}
		fake := nestedDefaultNode{ty: TypeRef{Kind: Scalar, Name: "kotlinx.serialization.json.JsonNull"}}
		if _, failure := fake.value(literal, "/fake"); failure == nil {
			t.Fatal("type spelling alone granted null-value role")
		}
		key := nestedDefaultTypeKey(node.ty)
		if strings.Contains(key, ".") || !strings.Contains(key, "Scalar JsonNull") {
			t.Fatal("invalid or ambiguous JVM helper key", key)
		}
		after, _ := v.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("null inference mutated source")
		}
	}
	value, parse := document.ParseObject([]byte(`{"value":[null]}`), document.MaxDocumentBytes)
	if parse != nil {
		t.Fatal(parse)
	}
	for _, extra := range []int{0, 1} {
		work, bytes := maxNestedDefaultNodes-3+extra, maxNestedDefaultBytes-len(`{"value":[null]}`)
		failure := nestedDefaultBudget(value, "/value", 0, &work, &bytes)
		if extra == 0 && (failure != nil || work != maxNestedDefaultNodes || bytes != maxNestedDefaultBytes) || extra == 1 && (failure == nil || failure.Code != "E_MODEL_LIMIT") {
			t.Fatal("null leaf escaped whole-tree work", extra, work, bytes, failure)
		}
	}
}

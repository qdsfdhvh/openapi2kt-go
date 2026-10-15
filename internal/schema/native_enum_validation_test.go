package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNativeEnumFieldChecksRetainSourceOutsideSemanticIdentity(t *testing.T) {
	root, err := document.Load([]byte(`{"type":"number","format":"float","enum":[1,1.0,2.5]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := &analyzer{options: Options{Validation: true}, validationSources: map[*PrimitiveValidation]string{}, emissionHints: EmissionHints{}}
	ty, checks, failure := a.nativeEnumFieldType(field{Schema: root, Pointer: "/original/field", Depth: 1}, false)
	if failure != nil || ty.Kind != Scalar || ty.Name != "kotlin.Float" || checks == nil || len(checks.Enum) != 2 || checks.Enum[0] != "1.0f" || a.validationSources[checks] != "/original/field" || !a.emissionHints[checks] || len(a.warnings) != 0 {
		t.Fatal(ty, checks, failure)
	}
	data, e := root.MarshalJSON()
	if e != nil || string(data) != `{"type":"number","format":"float","enum":[1,1.0,2.5]}` {
		t.Fatal("original enum node changed", e)
	}
}

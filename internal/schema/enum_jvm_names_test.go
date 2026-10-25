package schema

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func TestEnumJVMNameProofUsesRetainedDeclarationsAndOriginalIndices(t *testing.T) {
	wire := strings.Repeat("/", 10923)
	jsonWire, _ := json.Marshal(wire)
	source, err := document.Load([]byte(`{"enum":[null,`+string(jsonWire)+`]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	enum := &StringEnum{ClassName: "State", Members: []EnumMember{{Name: naming.EnumMemberName(wire), Wire: wire}}}
	origin := nominalOrigin{Schema: source, Pointer: "/components/schemas/S~1~0", Nullable: true}
	plan := &Plan{enumDiagnosticOrigins: map[*StringEnum]nominalOrigin{enum: origin}}
	before, _ := json.Marshal(enum)
	if err := CheckRetainedEnumJVMNameLimits(plan); err != nil {
		t.Fatal("unretained validation-only enum caused an error", err)
	}
	plan.Models = []Declaration{&Model{ClassName: "Holder", Nested: []Declaration{enum, enum}}}
	failure := CheckRetainedEnumJVMNameLimits(plan)
	if failure == nil || failure.Pointer == nil || *failure.Pointer != "/components/schemas/S~1~0/enum/1" {
		t.Fatal("source member Pointer changed", failure)
	}
	after, _ := json.Marshal(enum)
	if string(before) != string(after) || !reflect.DeepEqual(plan.enumDiagnosticOrigins[enum], origin) || get(source, "enum").Items()[1].Kind() != document.String {
		t.Fatal("diagnostic walk changed declaration or source identity")
	}
	// A copied/unretained declaration has no recorded original source authority.
	copy := *enum
	plan.Models = []Declaration{&copy}
	if err := CheckRetainedEnumJVMNameLimits(plan); err != nil {
		t.Fatal("unrecorded copy acquired another enum's source Pointer", err)
	}
	// This check proves only the member-name lower bound. Package, enclosing,
	// filesystem and other classfile constraints remain independent.
	plan.Models = []Declaration{enum}
	enum.Members[0].Name = strings.Repeat("A", maxJVMUTF8ConstantBytes)
	if err := CheckRetainedEnumJVMNameLimits(plan); err != nil {
		t.Fatal("lower-bound proof overclaimed equality at the cap", err)
	}
	enum.Members[0].Name += "A"
	if err := CheckRetainedEnumJVMNameLimits(plan); err == nil {
		t.Fatal("strict overflow was not detected")
	}
}

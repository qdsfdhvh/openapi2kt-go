package schema

import (
	"reflect"
	"strings"
	"testing"
)

func lockPairPlan() *Plan {
	a := &Model{ClassName: "AlphaResponse", Properties: []Property{{Name: "id", WireName: "id", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}}}}
	b := &Model{ClassName: "BetaResponse", Properties: []Property{{Name: "id", WireName: "id", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}}}}
	return &Plan{Client: &Client{Models: []Declaration{a, b}}, EmissionHints: EmissionHints{}, operationRoles: map[string]string{a.Name(): "Response", b.Name(): "Response"}, operationOrigins: map[string][]string{a.Name(): {"/a"}, b.Name(): {"/b"}}}
}
func TestLockedHoistRejectsBeforeDeclarationMutation(t *testing.T) {
	for _, kind := range []string{"ambiguous", "component", "helper", "changed", "removed"} {
		p := lockPairPlan()
		state, _ := ParseNameLock(nil, "org.example", "", false)
		if e := HoistOperationModelsLocked(p, "org.example", false, state); e != nil {
			t.Fatal(e)
		}
		current := lockPairPlan()
		wantCode := "E_NAME_LOCK_CONFLICT"
		switch kind {
		case "ambiguous":
			for _, name := range []string{"GammaResponse", "DeltaResponse"} {
				current.Client.Models = append(current.Client.Models, &Model{ClassName: name, Properties: []Property{{Name: "other", WireName: "other", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}}}})
				current.operationRoles[name] = "Response"
				current.operationOrigins[name] = []string{"/a", "/d"}
			}
		case "component":
			current.Models = []Declaration{&StringEnum{ClassName: "response", Members: []EnumMember{{Name: "YES", Wire: "yes"}}}}
		case "helper":
			state.Entries[0].Name = "Patch"
			current.PatchPackage = "org.example.models"
		case "changed":
			current.Client.Models[1].(*Model).Properties[0].Name = "different"
			wantCode = "E_NAME_LOCK_CHANGED"
		case "removed":
			current.operationOrigins["BetaResponse"] = nil
			wantCode = "E_NAME_LOCK_REMOVED"
		}
		beforeNames := []string{current.Client.Models[0].Name(), current.Client.Models[1].Name()}
		beforeModels := append([]Declaration(nil), current.Models...)
		e := HoistOperationModelsLocked(current, "org.example", false, state)
		if e == nil || e.Code != wantCode {
			t.Fatal(kind, e)
		}
		if !reflect.DeepEqual(beforeModels, current.Models) || current.Client.Models[0].Name() != beforeNames[0] || current.Client.Models[1].Name() != beforeNames[1] {
			t.Fatal("partial relocation", kind)
		}
	}
}
func TestLockedHoistNoUnknownProvenanceAndOpaqueIdentity(t *testing.T) {
	p := lockPairPlan()
	p.operationOrigins = nil
	state, _ := ParseNameLock(nil, "org.example", "", false)
	if e := HoistOperationModelsLocked(p, "org.example", false, state); e != nil || len(state.Entries) != 0 || len(p.Models) != 0 {
		t.Fatal("invented provenance", e)
	}
	p = lockPairPlan()
	a := p.Client.Models[0].(*Model)
	a.Constants = []Constant{{Name: "marker", WireName: "marker", Type: TypeRef{Kind: Scalar, Name: "org.example.Provider"}, Initializer: "org.example.Provider.DEFAULT"}}
	first := nameLockShapeV1(a, "org.example")
	a.Constants[0].Initializer = "org.example.Provider.OTHER"
	if nameLockShapeV1(a, "org.example") == first {
		t.Fatal("provider constants erased")
	}
	if strings.Contains(first, "Provider") {
		t.Fatal("fingerprint leaked payload")
	}
}

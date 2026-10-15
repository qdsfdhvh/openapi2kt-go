package schema

import (
	"reflect"
	"testing"
)

func hoistModel(name string) *Model {
	return &Model{ClassName: name, Properties: []Property{{Name: "child", WireName: "child", Type: TypeRef{Kind: Named, Name: name + ".Child", SchemaSteps: 3}, Validation: &PrimitiveValidation{}}}, Nested: []Declaration{&Model{ClassName: "Child", Properties: []Property{{Name: "id", WireName: "id", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}}}}}}
}
func TestHoistOwnedScopeRolesHintsReservations(t *testing.T) {
	a, b := hoistModel("AlphaResponse"), hoistModel("BetaResponse")
	borrowed := &TypeRef{Kind: Named, Name: "AlphaResponse.Child", SchemaSteps: 7}
	a.Properties = append(a.Properties, Property{Name: "list", WireName: "list", Type: TypeRef{Kind: List, Element: borrowed}})
	b.Properties = append(b.Properties, Property{Name: "list", WireName: "list", Type: TypeRef{Kind: List, Element: &TypeRef{Kind: Named, Name: "BetaResponse.Child", SchemaSteps: 7}}})
	component := &StringEnum{ClassName: "response", Members: []EnumMember{{Name: "OK", Wire: "ok"}}}
	hints := EmissionHints{b.Properties[0].Validation: true}
	p := &Plan{Models: []Declaration{component}, Client: &Client{Models: []Declaration{b, a}}, EmissionHints: hints, operationRoles: map[string]string{"AlphaResponse": "Response", "BetaResponse": "Response"}}
	HoistOperationModels(p, "org.example.hoist", false)
	if len(p.Models) != 2 || p.Models[0] != component || p.Models[1] != a || a.Name() != "Response2" {
		t.Fatal("component overwritten/canonical wrong", p.Models)
	}
	if !hints[a.Properties[0].Validation] || a.Properties[0].Type.Name != "org.example.hoist.models.Response2.Child" || a.Properties[0].Type.SchemaSteps != 3 || a.Properties[1].Type.Element.SchemaSteps != 7 || borrowed.Name != "AlphaResponse.Child" {
		t.Fatal("owned reference/work/hints/borrowed view loss", a)
	}
	for _, d := range p.Client.Models {
		alias, ok := d.(*ScalarAlias)
		if !ok || alias.Scalar != "org.example.hoist.models.Response2" {
			t.Fatal(d)
		}
	}
}
func TestHoistRoleAndOpaqueExpressionsNeverGuessed(t *testing.T) {
	for _, change := range []string{"role", "unknownOrigin", "serializer", "initializer", "foreignNominal", "unsupportedNested"} {
		t.Run(change, func(t *testing.T) {
			a, b := hoistModel("AlphaResponse"), hoistModel("BetaResponse")
			roles := map[string]string{a.Name(): "Response", b.Name(): "Response"}
			switch change {
			case "role":
				roles[b.Name()] = "Input"
			case "unknownOrigin":
				delete(roles, b.Name())
			case "serializer":
				a.Properties[0].Type.Serializer = "AlphaResponse.GeneratedCodec"
				b.Properties[0].Type.Serializer = "BetaResponse.GeneratedCodec"
			case "initializer":
				a.Constants = []Constant{{Name: "tag", WireName: "tag", Initializer: "AlphaResponse.Child.VALUE"}}
				b.Constants = []Constant{{Name: "tag", WireName: "tag", Initializer: "BetaResponse.Child.VALUE"}}
			case "foreignNominal":
				a.Properties[0].Type.Name = "AlphaResponse.NotDeclared"
				b.Properties[0].Type.Name = "BetaResponse.NotDeclared"
			case "unsupportedNested":
				a.Nested = append(a.Nested, &ScalarAlias{ClassName: "Opaque", Scalar: "kotlin.String"})
				b.Nested = append(b.Nested, &ScalarAlias{ClassName: "Opaque", Scalar: "kotlin.String"})
			}
			p := &Plan{Client: &Client{Models: []Declaration{a, b}}, EmissionHints: EmissionHints{}, operationRoles: roles}
			HoistOperationModels(p, "org.example.hoist", false)
			if len(p.Models) != 0 || !reflect.DeepEqual(p.Client.Models, []Declaration{a, b}) || a.Name() != "AlphaResponse" || b.Name() != "BetaResponse" {
				t.Fatal("ineligible tree changed")
			}
		})
	}
}
func TestHoistRuntimeReservationsAndQualification(t *testing.T) {
	for _, shared := range []bool{false, true} {
		for _, qualify := range []bool{false, true} {
			a, b := hoistModel("AlphaPatch"), hoistModel("BetaPatch")
			pkg := "org.example.other.runtime"
			if shared {
				pkg = "org.example.hoist.models"
			}
			p := &Plan{PatchPackage: pkg, Client: &Client{Models: []Declaration{b, a}}, EmissionHints: EmissionHints{}, operationRoles: map[string]string{a.Name(): "Patch", b.Name(): "Patch"}}
			HoistOperationModels(p, "org.example.hoist", qualify)
			want := "Patch"
			if shared {
				want = "Patch2"
			}
			if qualify {
				want = "AlphaPatch"
			}
			if a.Name() != want {
				t.Fatal(want, a.Name())
			}
		}
	}
	a := &analyzer{options: Options{Suffix: "Dto"}}
	name, e := a.operationModelName("alpha", "Response", "/a")
	if e != nil || a.operationRoles[name] != "ResponseDto" {
		t.Fatal("missing original provenance", e)
	}
	// Distinct owner/role partitions can spell the same class; do not choose one.
	_, _ = a.operationModelName("alphaResponse", "Input", "/b")
	_, _ = a.operationModelName("alpha", "ResponseInput", "/c")
	if a.operationRoles["AlphaResponseInputDto"] != "" {
		t.Fatal("ambiguous allocation provenance guessed")
	}
}

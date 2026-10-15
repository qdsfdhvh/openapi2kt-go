package schema

import "testing"

func TestOperationOpaqueIncomingClassifierDependency(t *testing.T) {
	for _, hoist := range []bool{false, true} {
		a, b := hoistModel("AlphaResponse"), hoistModel("BetaResponse")
		external := &ScalarAlias{ClassName: "Caller", Scalar: "org.example.hoist.api.models.BetaResponse.Child"}
		p := &Plan{Models: []Declaration{external}, Client: &Client{Models: []Declaration{a, b}}, EmissionHints: EmissionHints{}, operationRoles: map[string]string{a.Name(): "Response", b.Name(): "Response"}}
		if hoist {
			HoistOperationModels(p, "org.example.hoist", false)
		} else {
			DeduplicateOperationModels(p, "org.example.hoist")
		}
		if p.Client.Models[1] != b || b.Name() != "BetaResponse" {
			t.Fatal("provider's previously valid nested classifier erased", hoist, p.Client.Models)
		}
	}
}

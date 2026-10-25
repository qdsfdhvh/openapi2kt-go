package schema

import (
	"reflect"
	"testing"
)

func TestLockedProjectionRetainsSingletonAndStrictReplay(t *testing.T) {
	original := lockPairPlan()
	state, _ := ParseNameLock(nil, "org.example", "", false)
	if err := HoistOperationModelsLocked(original, "org.example", false, state); err != nil {
		t.Fatal(err)
	}
	before := state.Entries[0]
	encoded, _ := state.Encode()
	singleton := func() *Plan {
		p := lockPairPlan()
		p.Client.Models = p.Client.Models[:1]
		delete(p.operationOrigins, "BetaResponse")
		delete(p.operationRoles, "BetaResponse")
		return p
	}
	strict, _ := ParseNameLock(encoded, "org.example", "", false)
	if err := HoistOperationModelsLocked(singleton(), "org.example", false, strict); err == nil || err.Code != "E_NAME_LOCK_REMOVED" {
		t.Fatal("ordinary generation silently projected", err)
	}
	projected, _ := ParseNameLock(encoded, "org.example", "", false)
	p := singleton()
	if err := HoistOperationModelsLockedProjection(p, "org.example", false, projected); err != nil {
		t.Fatal(err)
	}
	if len(projected.Entries) != 1 || projected.Entries[0].Name != before.Name || !reflect.DeepEqual(projected.Entries[0].Occurrences, []string{"/a"}) || len(p.Models) != 1 || p.Models[0].Name() != before.Name {
		t.Fatal("lost original singleton classifier", projected, p.Models)
	}
	next, _ := projected.Encode()
	replay, _ := ParseNameLock(next, "org.example", "", false)
	if err := HoistOperationModelsLocked(singleton(), "org.example", false, replay); err != nil {
		t.Fatal("projected persisted singleton cannot regenerate", err)
	}
}

func TestLockedProjectionRejectsChangedAndNewAllocationsBeforeMutation(t *testing.T) {
	for _, kind := range []string{"changed", "new", "origin", "conflict"} {
		initial := lockPairPlan()
		state, _ := ParseNameLock(nil, "org.example", "", false)
		if err := HoistOperationModelsLocked(initial, "org.example", false, state); err != nil {
			t.Fatal(err)
		}
		p := lockPairPlan()
		switch kind {
		case "changed":
			p.Client.Models[0].(*Model).Properties[0].Type.Name = "kotlin.Int"
		case "new":
			for _, d := range p.Client.Models {
				d.(*Model).Properties[0].Type.Name = "kotlin.Int"
			}
		case "origin":
			p.operationOrigins["AlphaResponse"] = []string{"/unrelated"}
		case "conflict":
			p.Models = []Declaration{&Model{ClassName: state.Entries[0].Name}}
		}
		oldNames := []string{p.Client.Models[0].Name(), p.Client.Models[1].Name()}
		if err := HoistOperationModelsLockedProjection(p, "org.example", false, state); err == nil {
			t.Fatal("invalid projection accepted", kind)
		}
		if p.Client.Models[0].Name() != oldNames[0] || p.Client.Models[1].Name() != oldNames[1] {
			t.Fatal("partial projection mutated declarations", kind)
		}
	}
}

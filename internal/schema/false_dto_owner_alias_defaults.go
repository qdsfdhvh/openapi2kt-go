package schema

import "openapi-kt/internal/document"

type falseDTOOwnerAliasState struct {
	Component, Target              *component
	Schema                         *document.Value
	Wire, Name, Pointer, Reference string
	Height                         int
	HasNullable, Nullable          bool
}

type falseDTOOwnerAliasStep struct {
	State falseDTOOwnerAliasState
	Nodes []borrowedAllOfNode
}

// Pure original component aliases borrow the canonical DTO and retain their
// own source/cache identity. Spelling and speculative backing types establish
// neither a target nor a nullable override.
func (a *analyzer) falseDTOOwnerAliasTarget(declared *component, pointer string) (*component, []falseDTOOwnerAliasStep, bool) {
	current := declared
	seen := map[*component]bool{}
	path := []falseDTOOwnerAliasStep{}
	for current != nil && has(current.Schema, "$ref") {
		if seen[current] || len(path) >= MaxSchemaDepth || a.components[current.Wire] != current || !a.componentRoot(current.Schema, current.Pointer) || current.InternalModelTarget != nil || has(current.Schema, "default") || checkKeys(current.Schema, []string{"$ref", "nullable"}, current.Pointer) != nil || current.Reference != text(get(current.Schema, "$ref")) {
			return nil, nil, false
		}
		seen[current] = true
		path = append(path, falseDTOOwnerAliasStep{State: falseDTOOwnerAliasState{Component: current, Target: current.BindingTarget, Schema: current.Schema, Wire: current.Wire, Name: current.Name, Pointer: current.Pointer, Reference: current.Reference, Height: current.AliasHeight}})
		next, err := a.resolveDeclared(current.Reference, pointer)
		if err != nil {
			return nil, nil, false
		}
		current = next
	}
	if current == nil || current.BindingTarget != nil || current.InternalModelTarget != nil {
		return nil, nil, false
	}
	var nullable *bool
	if current.NullableDTO {
		flag := true
		nullable = &flag
	}
	for i := len(path) - 1; i >= 0; i-- {
		state := &path[i].State
		if own := nullableOverride(state.Schema); own != nil {
			nullable = own
		}
		c := state.Component
		if c.BindingTarget != current || c.AliasHeight != len(path)-i || (c.BindingNullable != nil) != (nullable != nil) || nullable != nil && *c.BindingNullable != *nullable {
			return nil, nil, false
		}
		state.HasNullable = nullable != nil
		if nullable != nil {
			state.Nullable = *nullable
		}
	}
	return current, path, true
}

func sameFalseDTOOwnerAliasPath(saved, current []falseDTOOwnerAliasStep) bool {
	if len(saved) != len(current) {
		return false
	}
	for i, step := range saved {
		if step.State != current[i].State || len(step.Nodes) == 0 || step.Nodes[0].Node != step.State.Schema || !sameBorrowedHierarchyNodes(step.Nodes, step.Nodes) {
			return false
		}
	}
	return true
}

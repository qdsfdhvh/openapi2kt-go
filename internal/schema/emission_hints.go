package schema

// EmissionHints is a plan-only sidecar, never part of semantic declarations or
// their equality. Hint-bearing check sets use bounded non-inline private calls
// to keep JVM serialization constructors within the existing slot budget.
type EmissionHints map[*PrimitiveValidation]bool

// transferEquivalent ORs hints into the declaration retained by existing exact
// semantic equality. Sources/occurrence order must not determine helper safety.
// Call only after exact or owned-name-normalized semantic equality; declaration
// topology and enforcing checks must match at every nested position.
func (h EmissionHints) transferEquivalent(retained, candidate Declaration) {
	switch r := retained.(type) {
	case *Model:
		c := candidate.(*Model)
		for i, p := range r.Properties {
			if h[c.Properties[i].Validation] {
				h[p.Validation] = true
			}
			h.transferEquivalentType(p.Type, c.Properties[i].Type)
		}
		for i, n := range r.Nested {
			h.transferEquivalent(n, c.Nested[i])
		}
	case *PrimitiveUnion:
		c := candidate.(*PrimitiveUnion)
		if r.Object != nil {
			h.transferEquivalent(r.Object, c.Object)
		}
	case *ObjectUnion:
		c := candidate.(*ObjectUnion)
		for i, m := range r.Variants {
			h.transferEquivalent(m, c.Variants[i])
		}
	case *Hierarchy:
		c := candidate.(*Hierarchy)
		if r.Leaf != nil {
			h.transferEquivalent(r.Leaf, c.Leaf)
		}
	}
}

// Matching transparent trees are finite value plans, not nominal schema graphs.
// Exact semantic equality is checked by the caller before either transfer.
func (h EmissionHints) transferEquivalentType(retained, candidate TypeRef) {
	if h[candidate.Validation] {
		h[retained.Validation] = true
	}
	if retained.Element != nil {
		h.transferEquivalentType(*retained.Element, *candidate.Element)
	}
}

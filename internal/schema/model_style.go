package schema

// UsePlainClasses is a post-analysis presentation pass over owned declarations,
// never over referenced schema graphs. Resolved shapes and all budgets remain
// the analyzer's decisions. Shared peers are visited once.
func UsePlainClasses(plan *Plan) {
	seen := map[Declaration]bool{}
	var visit func(Declaration)
	visit = func(d Declaration) {
		if d == nil || seen[d] {
			return
		}
		seen[d] = true
		switch d := d.(type) {
		case *Model:
			d.Plain = true
			for _, nested := range d.Nested {
				visit(nested)
			}
		case *StringEnum:
			d.Plain = true
		case *PrimitiveUnion:
			d.Plain = true
			if d.Object != nil {
				visit(d.Object)
			}
		case *ObjectUnion:
			d.Plain = true
			for _, v := range d.Variants {
				visit(v)
			}
		case *Hierarchy:
			d.Plain = true
			if d.Leaf != nil {
				visit(d.Leaf)
			}
			for _, e := range d.Enums {
				visit(e)
			}
		case *SSEEvents:
			d.Plain = true
		case *ScalarAlias, *ValueAlias, *PrimitiveValueClass:
		default:
			panic("unhandled model style declaration")
		}
	}
	for _, d := range plan.Models {
		visit(d)
	}
	if plan.Client != nil {
		for _, d := range plan.Client.Models {
			visit(d)
		}
	}
}

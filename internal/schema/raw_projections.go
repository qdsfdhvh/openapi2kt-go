package schema

// Empty ordinary models emit JsonElement aliases. Discover that role only after
// original declarations have passed schema validation; hierarchy/union classes
// must never acquire a raw backing merely because their own fields are empty.
// Linking original TypeRefs also covers forward users analyzed before targets.
func (a *analyzer) linkRawProjections(declarations []Declaration) {
	raw := map[string]*TypeRef{}
	prefix := a.options.PackageName + ".models."
	byName := map[string]Declaration{}
	for _, d := range declarations {
		byName[d.Name()] = d
	}
	for _, c := range a.components {
		if c.BindingTarget != nil {
			continue
		}
		if model, ok := byName[c.Name].(*Model); ok && len(model.Properties) == 0 && len(model.Constants) == 0 {
			c.AliasType = TypeRef{Kind: JSON}
			raw[prefix+c.Name] = &c.AliasType
		}
	}
	if len(raw) == 0 {
		return
	}
	seen := map[*TypeRef]bool{}
	var link func(*TypeRef, bool)
	link = func(t *TypeRef, planning bool) {
		if seen[t] {
			return
		}
		seen[t] = true
		if t.Kind == Named && raw[t.Name] != nil {
			t.Element = raw[t.Name]
			// A raw reference is transparent source work, not a nominal barrier.
			// Existing lowered chain work already includes this hop.
			if planning && t.SchemaSteps == 0 {
				t.SchemaSteps = 1
			}
			return
		}
		if t.Element != nil {
			// Nullable adds no source hop. Its erased-work annotation already
			// includes the terminal raw reference; do not count that again on
			// the wrapped Named node. Collection edges keep their own work.
			link(t.Element, planning && !(t.Kind == Nullable && t.SchemaSteps != 0))
		}
	}
	var visit func(Declaration)
	visit = func(d Declaration) {
		switch d := d.(type) {
		case *Model:
			for i := range d.Properties {
				link(&d.Properties[i].Type, false)
			}
			for _, child := range d.Nested {
				visit(child)
			}
		case *ValueAlias:
			link(&d.Type, true)
		case *ObjectUnion:
			for i := range d.Common {
				link(&d.Common[i].Type, false)
			}
			for _, v := range d.Variants {
				visit(v)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		case *Hierarchy:
			for i := range d.Fields {
				link(&d.Fields[i].Type, false)
			}
			for i := range d.Inherited {
				link(&d.Inherited[i].Type, false)
			}
			if d.Leaf != nil {
				visit(d.Leaf)
			}
		}
	}
	// Planning work must be attached before shared collection nodes are visited
	// through ordinary model users. All pointers are cycle-safe even before the
	// transparent graph validator runs.
	for _, d := range declarations {
		if _, ok := d.(*ValueAlias); ok {
			visit(d)
		}
	}
	for _, c := range a.components {
		if collectionAliasSchema(c.Schema) {
			link(&c.AliasType, true)
		}
	}
	for _, d := range declarations {
		visit(d)
	}
}

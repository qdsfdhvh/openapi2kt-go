package schema

import "openapi-kt/internal/diagnostic"

// A common dictionary exposes the same public backing as each real branch.
// Branch planning retains checks and codecs; this public view owns neither.
func (a *analyzer) commonDictionaryType(f *field) (TypeRef, bool, *diagnostic.Diagnostic) {
	v := f.Schema
	var target *resolvedSchema
	if has(v, "$ref") {
		var err *diagnostic.Diagnostic
		target, err = a.resolveSchema(text(get(v, "$ref")), f.Pointer)
		if err != nil {
			return TypeRef{}, false, err
		}
		v = target.Schema
	}
	if !dictionarySchema(v) {
		return TypeRef{}, false, nil
	}
	context := *a
	context.warnings = nil // Probe warnings are emitted by the actual branches.
	var nested []Declaration
	ty, err := context.propertyType(f.Schema, "CommonDictionary", f.Pointer, &nested, map[string]bool{}, f.Depth)
	if err != nil {
		return TypeRef{}, true, err
	}
	// Inline nominal value declarations cannot acquire an abstract owner from
	// scratch planning. Named retained value types need no new declarations.
	if len(nested) != 0 {
		return TypeRef{}, true, unsupported(f.Pointer)
	}
	if target != nil {
		// Forward catalog backing is linked but may not be filled yet. Inspect
		// the original dictionary source through normal typed planning; the
		// original field use above retains its complete incoming alias budget.
		view, e := context.propertyType(target.Schema, "CommonDictionary", target.Pointer, &nested, map[string]bool{}, f.Depth+max(1, target.Steps))
		if e != nil {
			return TypeRef{}, true, e
		}
		if len(nested) != 0 {
			return TypeRef{}, true, unsupported(f.Pointer)
		}
		view = view.WithoutRootNull()
		if ty.Kind == Nullable {
			view = view.Null()
		}
		ty = view
	}
	view, err := context.commonDictionaryPublicView(ty, f.Depth, f.Pointer, map[*TypeRef]bool{})
	if err != nil {
		return TypeRef{}, true, err
	}
	if bare(view).Kind != Map && bare(view).Kind != JSONObject {
		return TypeRef{}, true, unsupported(f.Pointer)
	}
	return view, true, nil
}

func (a *analyzer) commonDictionaryPublicView(ty TypeRef, depth int, source string, active map[*TypeRef]bool) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, source); err != nil {
		return TypeRef{}, err
	}
	if ty.Kind == Named && ty.Element != nil {
		if active[ty.Element] {
			return TypeRef{}, diagnostic.At("E_REF_CYCLE", "Dictionary aliases cannot form a transparent recursive type; introduce a named object model.", source)
		}
		active[ty.Element] = true
		defer delete(active, ty.Element)
		target := *ty.Element
		for _, c := range a.components {
			if ty.Element != &c.AliasType || !collectionAliasSchema(c.Schema) {
				continue
			}
			// Only an actual canonical backing address authorizes forward
			// inspection. Matching Kotlin names never creates source identity.
			var nested []Declaration
			var err *diagnostic.Diagnostic
			target, err = a.propertyType(c.Schema, "CommonDictionary", c.Pointer, &nested, map[string]bool{}, depth+1)
			if err != nil {
				return TypeRef{}, err
			}
			if len(nested) != 0 {
				return TypeRef{}, unsupported(source)
			}
			break
		}
		if ty.ArrayAliasView == NonNullArrayAlias {
			target = target.WithoutRootNull()
		}
		return a.commonDictionaryPublicView(target, depth+1, source, active)
	}
	ty.Validation, ty.Collection, ty.SchemaSteps = nil, nil, 0
	if ty.Element != nil {
		childDepth := depth
		if ty.Kind == Map || ty.Kind == List {
			childDepth++
		}
		item, err := a.commonDictionaryPublicView(*ty.Element, childDepth, source, active)
		if err != nil {
			return TypeRef{}, err
		}
		ty.Element = &item
	}
	return ty, nil
}

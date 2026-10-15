package schema

import (
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Builtin bindings retain the original validated source, not scalar spelling.
// Legacy ScalarTarget aliases keep their historical lookup schema; only the new
// chain carrier flattens them, preserving the declared catalog and source work.
type builtinOrigin struct {
	Owner           *component
	Schema          *document.Value
	Pointer, Scalar string
	Validation      *PrimitiveValidation
}

func (a *analyzer) recordBuiltinOrigin(d Declaration, c *component) {
	if a.builtinOrigins != nil && builtinPrimitiveBinding(c) {
		a.builtinOrigins[d] = builtinOrigin{c, c.Schema, c.Pointer, c.Scalar, c.Validation}
	}
}

func (a *analyzer) bindNamedChainBuiltin(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	canonical := target.CanonicalBuiltin
	if canonical == nil {
		canonical = target.Catalog
		if canonical.BindingTarget != nil {
			canonical = canonical.BindingTarget
		} else if canonical.ScalarTarget != nil {
			canonical = canonical.ScalarTarget
		}
	}
	if !builtinPrimitiveBinding(canonical) {
		return false
	}
	original, pointer := canonical.Schema, canonical.Pointer
	if target.CanonicalBuiltin == nil && target.Catalog.ScalarTarget != nil {
		original, pointer = target.Catalog.Schema, target.Catalog.Pointer
	}
	if target.Schema != original || target.Pointer != pointer {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.builtinOrigins {
			if origin.Owner == canonical && origin.Schema == canonical.Schema && origin.Pointer == canonical.Pointer && origin.Scalar == canonical.Scalar && origin.Validation == canonical.Validation && builtinDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if target.Nullable == nil {
		target.Nullable = nullableOverride(target.Catalog.Schema)
	}
	target.Steps = max(target.Steps, 1+target.Catalog.AliasHeight)
	target.Schema, target.Pointer = canonical.Schema, canonical.Pointer
	target.CanonicalBuiltin = canonical
	if a.builtinUses == nil {
		a.builtinUses = map[*component]string{}
	}
	if previous, ok := a.builtinUses[canonical]; !ok || source < previous {
		a.builtinUses[canonical] = source
	}
	return true
}

func builtinDeclaration(d Declaration, origin builtinOrigin) bool {
	if d.Name() != origin.Owner.Name {
		return false
	}
	switch d := d.(type) {
	case *ScalarAlias:
		return !origin.Owner.NullablePrimitive && d.Scalar == origin.Scalar
	case *ValueAlias:
		return origin.Owner.NullablePrimitive && d.Type.Kind == Nullable && bare(d.Type).Kind == Scalar && bare(d.Type).Name == origin.Scalar && bare(d.Type).Validation == origin.Validation
	}
	return false
}

func (a *analyzer) validateBuiltinBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]bool{}
	for _, d := range declarations {
		actual[d] = true
	}
	used := []*component{}
	for c := range a.builtinUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.builtinUses[used[i]] < a.builtinUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.builtinOrigins {
			if actual[d] && origin.Owner == c && origin.Schema == c.Schema && origin.Pointer == c.Pointer && origin.Scalar == c.Scalar && origin.Validation == c.Validation && builtinPrimitiveBinding(c) && builtinDeclaration(d, origin) {
				matches++
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.builtinUses[c])
		}
	}
	return nil
}

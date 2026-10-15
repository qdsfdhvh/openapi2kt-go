package schema

import "openapi-kt/internal/diagnostic"

// Hierarchy edges select physical non-null children; member intent belongs to
// the parent typed view. General shape planners keep resolveNominal's separate
// nullable-borrowing guard. Actual ownership is still validated by planHierarchy.
func (a *analyzer) resolveHierarchyBranch(reference, pointer string, depth int) (*component, int, bool, *diagnostic.Diagnostic) {
	c, err := a.resolveDeclared(reference, pointer)
	if err != nil {
		return nil, 0, false, err
	}
	if c.InternalModelTarget != nil {
		return nil, 0, false, schemaReferenceUnsupported(pointer)
	}
	if c.NullableDTO || c.NullableEnum || c.NullableValueClass || c.NullableTemporal || c.NullableBigint || c.NullableBytes || c.NullableCallerNative || c.NullableRaw || c.NullablePrimitive || c.NullableNativeEnum {
		return nil, 0, false, unsupported(pointer)
	}
	target, steps, nullable := c, 1, false
	if c.BindingTarget != nil {
		target, steps = c.BindingTarget, 1+c.AliasHeight
		if target.NullableDTO || target.NullableEnum || target.NullableValueClass || target.NullableTemporal || target.NullableBigint || target.NullableBytes || target.NullableCallerNative || target.NullableRaw || target.NullablePrimitive || target.NullableNativeEnum {
			return nil, 0, false, unsupported(pointer)
		}
		if c.BindingNullable != nil && *c.BindingNullable && (!a.hierarchySources[target.Schema] || target.Scalar != "" || target.ValueClass || has(target.Schema, "enum")) {
			return nil, 0, false, unsupported(pointer)
		}
		// An alias carries explicit/inherited member intent. A direct child's
		// own intrinsic nullability alone retains the established edge policy.
		nullable = c.BindingNullable != nil && *c.BindingNullable
	}
	if err := depthCheck(depth+steps, pointer); err != nil {
		return nil, 0, false, err
	}
	return target, steps, nullable, nil
}

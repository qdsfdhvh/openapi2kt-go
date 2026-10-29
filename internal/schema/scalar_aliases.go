package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"slices"
)

// Named root bindings are a two-pass catalog operation: builtin scalar aliases
// keep their opt-in contract; ordinary DTO aliases retain canonical nominal
// identity. Forward references/cached heights never depend on declaration order.
// No raw nodes are copied or recursively expanded.
func (a *analyzer) resolveScalarAliases(schemas *document.Value) *diagnostic.Diagnostic {
	state := map[string]uint8{}
	var visit func(*component, int) *diagnostic.Diagnostic
	visit = func(c *component, depth int) *diagnostic.Diagnostic {
		pointer := c.Pointer + "/$ref"
		if depth+c.AliasHeight > MaxSchemaDepth {
			return diagnostic.At("E_SCHEMA_DEPTH", "Scalar alias depth exceeds the supported limit.", pointer)
		}
		if c.Reference == "" || state[c.Wire] == 2 {
			return nil
		}
		if state[c.Wire] == 1 {
			if !a.options.Validation {
				return unsupported(c.Pointer)
			}
			return diagnostic.At("E_REF_CYCLE", "Scalar reference aliases must not be cyclic.", pointer)
		}
		state[c.Wire] = 1
		if handled, err := a.resolveInternalRootAlias(c, depth, visit); handled {
			if err == nil {
				state[c.Wire] = 2
			}
			return err
		}
		target, err := a.resolveDeclared(c.Reference, pointer)
		if err != nil {
			return err
		}
		if err := visit(target, depth+1); err != nil {
			return err
		}
		c.AliasHeight = target.AliasHeight + 1
		if target.InternalModelTarget != nil {
			if err := a.bindInternalRootAlias(c, target.InternalModelTarget, depth, c.AliasHeight, target.BindingNullable); err != nil {
				return err
			}
			state[c.Wire] = 2
			return nil
		}
		if c.AliasHeight > MaxSchemaDepth {
			return diagnostic.At("E_SCHEMA_DEPTH", "Scalar alias depth exceeds the supported limit.", pointer)
		}
		canonical := target
		if target.BindingTarget != nil {
			canonical = target.BindingTarget
		} else if target.ScalarTarget != nil {
			canonical = target.ScalarTarget
		}
		jsonRoot := canonical.BooleanRoot || canonical.RecursiveJSON
		emptyRoot := canonical.FalseRoot
		hierarchy := a.hierarchySources[canonical.Schema]
		kind, nullable, e := schemaType(canonical.Schema, canonical.Pointer)
		if e != nil {
			return e
		}
		ordinary := (kind == "object" || has(canonical.Schema, "allOf")) && (!nullable || canonical.NullableDTO || canonical.NullableRaw) && !dictionarySchema(canonical.Schema) && !has(canonical.Schema, "enum") && !has(canonical.Schema, "oneOf") && !has(canonical.Schema, "anyOf") && !has(canonical.Schema, "discriminator") && canonical.Scalar == "" && !canonical.ValueClass
		collection := collectionAliasSchema(canonical.Schema)
		lexicalEnum := lexicalStringEnumBinding(canonical)
		temporal := temporalBinding(canonical)
		bigint := bigintBinding(canonical)
		bytes := bytesBinding(canonical)
		callerNative := callerNativeBinding(canonical)
		_, ownNullable, _ := schemaType(c.Schema, c.Pointer)
		union := unionNominalCandidate(canonical.Schema) || has(canonical.Schema, "allOf") && a.containsObjectCompositionUnion(canonical.Schema, 0, map[*document.Value]bool{})
		nativeEnum := a.options.Validation && nativeEnumBinding(canonical)
		primitive := canonical.NullablePrimitive || builtinPrimitiveBinding(canonical) && (ownNullable || target.BindingNullable != nil)
		structured := structuredConstBinding(canonical)
		if ownNullable && !collection && !ordinary && !lexicalEnum && !canonical.ValueClass && !temporal && !bigint && !bytes && !callerNative && !primitive && !nativeEnum && !structured && !jsonRoot && !emptyRoot && !hierarchy && !union {
			return unsupported(c.Pointer)
		}
		special := canonical.ValueClass || has(canonical.Schema, "enum") || has(canonical.Schema, "const") || has(canonical.Schema, "x-native-type") || canonical.Scalar != "" && !slices.Contains([]string{"kotlin.String", "kotlin.uuid.Uuid", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double"}, canonical.Scalar)
		if ordinary || collection || special || union || primitive || jsonRoot || emptyRoot || hierarchy {
			c.BindingTarget = canonical
			if collection || ordinary || lexicalEnum || canonical.ValueClass || temporal || bigint || bytes || callerNative || primitive || nativeEnum || structured || jsonRoot || emptyRoot || hierarchy || union {
				c.BindingNullable = nullableOverride(c.Schema)
				if c.BindingNullable == nil {
					c.BindingNullable = target.BindingNullable
					if c.BindingNullable == nil && (canonical.NullableDTO || canonical.NullableEnum || canonical.NullableValueClass || canonical.NullableTemporal || canonical.NullableBigint || canonical.NullableBytes || canonical.NullableCallerNative || canonical.NullableRaw || canonical.NullablePrimitive || canonical.NullableNativeEnum || canonical.NullableHierarchy || canonical.NullableStructuredConst) {
						inherited := true
						c.BindingNullable = &inherited
					}
				}
			}
		} else {
			if !a.options.Validation {
				return unsupported(c.Pointer)
			}
			if target.Scalar == "" || target.ValueClass || has(target.Schema, "const") || has(target.Schema, "enum") || has(target.Schema, "x-native-type") {
				return unsupported(pointer)
			}
			c.Scalar, c.Validation = target.Scalar, target.Validation
			c.ScalarTarget = canonical
		}
		state[c.Wire] = 2
		return nil
	}
	for _, entry := range schemas.Members() {
		if err := visit(a.components[entry.Name], 0); err != nil {
			return err
		}
	}
	return nil
}

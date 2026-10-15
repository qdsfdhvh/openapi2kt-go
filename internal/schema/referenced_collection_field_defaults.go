package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Admission and public spelling belong to the existing reference planner. Only
// a direct ordinary field's own annotation receives this additional role.
// Forward backing is read after the final source/declaration authority checks.
type referencedFieldDefault struct {
	Field field
	Type  TypeRef // Before optional presence adds incidental nullability.
	Model *Model  // nil for a discarded annotation that must still be validated.
	Index int
}

func (a *analyzer) referencedDefaultType(f field, ordinary bool) (TypeRef, *diagnostic.Diagnostic) {
	if !ordinary {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	if err := depthCheck(f.Depth, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	if err := checkKeys(f.Schema, []string{"$ref", "nullable", "default"}, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	if _, _, err := schemaType(f.Schema, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	ref, ok := get(f.Schema, "$ref").Text()
	if !ok {
		return TypeRef{}, invalid("$ref must be a string.", f.Pointer)
	}
	target, err := a.resolveSchema(ref, f.Pointer)
	if err != nil {
		return TypeRef{}, err
	}
	return a.referenceType(target, nullableOverride(f.Schema), f.Depth, f.Pointer)
}

// Resolve original source edges for default validation only. This does not
// authorize a typed reference: referencedDefaultType already did so. In
// particular, keep every named alias source rather than flattening lookup's
// canonical binding, so annotations on an intermediate cannot disappear.
func (a *analyzer) defaultSource(reference, source string) (*document.Value, string, *diagnostic.Diagnostic) {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(reference, prefix) && !strings.Contains(reference[len(prefix):], "/") {
		c, err := a.resolveDeclared(reference, source)
		if err != nil {
			return nil, "", err
		}
		return c.Schema, c.Pointer, nil
	}
	target, err := a.lookupSchemaReference(reference, source)
	if err != nil {
		return nil, "", err
	}
	return target.Schema, target.Pointer, nil
}

// Follow the actual linked alias backing, preserving the existing explicit
// false view and historical direct dictionary nullability behavior.
func defaultBacking(ty TypeRef) TypeRef {
	switch ty.Kind {
	case Nullable:
		return defaultBacking(*ty.Element).Null()
	case Named:
		if ty.Element != nil {
			target := *ty.Element
			if ty.ArrayAliasView == NonNullArrayAlias {
				target = target.WithoutRootNull()
			}
			return defaultBacking(target)
		}
	}
	return ty
}

// The own-default role originates at an admitted direct reference field. It is
// independent of scalar spelling: source grammar and actual linked backing must
// still agree after the reference planner's final authority checks.
type referencedDefaultSourceRole uint8

const (
	noReferencedDefaultRole referencedDefaultSourceRole = iota
	referencedScalarOwnDefault
	referencedCollectionOwnDefault
	referencedNamedOwnDefault
)

func (a *analyzer) referencedDefaultSchema(v *document.Value, pointer string, ty TypeRef, depth int, root bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	role := noReferencedDefaultRole
	if root && has(v, "$ref") && has(v, "default") {
		switch defaultBacking(ty).WithoutRootNull().Kind {
		case Scalar:
			role = referencedScalarOwnDefault
		case List, Map:
			role = referencedCollectionOwnDefault
		case Named:
			// Source grammar below must confirm a plain String enum or ordinary
			// DTO; a Named spelling alone never grants default admission.
			role = referencedNamedOwnDefault
		}
	}
	return a.referencedDefaultSchemaRole(v, pointer, ty, depth, root, role)
}

func (a *analyzer) referencedDefaultSchemaRole(v *document.Value, pointer string, ty TypeRef, depth int, root bool, role referencedDefaultSourceRole) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	if has(v, "$ref") {
		keys := []string{"$ref", "nullable"}
		if root {
			keys = append(keys, "default")
		}
		if err := checkKeys(v, keys, pointer); err != nil {
			return nil, err
		}
		source, p, err := a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, err
		}
		return a.referencedDefaultSchemaRole(source, p, ty, depth+1, false, role)
	}
	// This source grammar is deliberately narrower than the type planner. No
	// target defaults, nominal wrappers or additional nullable folds are granted.
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	actual := defaultBacking(ty)
	bare := actual.WithoutRootNull()
	node := &nestedDefaultNode{schema: v, pointer: pointer, ty: actual}
	keys := []string{"type", "nullable", "format"}
	edge := ""
	switch {
	case kind == "array":
		keys = a.collectionKeys([]string{"type", "nullable", "items"})
		edge = "items"
	case dictionarySchema(v):
		keys = a.collectionKeys([]string{"type", "nullable", "properties", "required", "additionalProperties"})
		edge = "additionalProperties"
	}
	if edge == "" && has(v, "enum") {
		if kind == "" || kind == "string" {
			if role != referencedCollectionOwnDefault && role != referencedNamedOwnDefault {
				return nil, unsupported(naming.Pointer(pointer, "enum"))
			}
			// Plain String enum terminals join this profile. Default members are
			// strict against the original source vocabulary while runtime stays
			// open; format and String bounds/pattern remain separate roles.
			if has(v, "format") || hasPrimitiveBounds(v) {
				return nil, unsupported(naming.Pointer(pointer, "enum"))
			}
			if err := checkKeys(v, []string{"type", "nullable", "enum"}, pointer); err != nil {
				return nil, err
			}
			members := get(v, "enum").Items()
			if len(members) == 0 {
				return nil, unsupported(naming.Pointer(pointer, "enum"))
			}
			allowNull := nullableStringEnumNulls(v, pointer)
			for _, member := range members {
				// The reference planner already validated the original vocabulary.
				// Its null token adds no classifier member or nullable inference;
				// value() still uses the actual typed view for every null default.
				if member.Kind() == document.Null && allowNull {
					continue
				}
				if _, ok := member.Text(); !ok {
					return nil, unsupported(naming.Pointer(pointer, "enum"))
				}
			}
			if bare.Kind != Named {
				return nil, unsupported(pointer)
			}
			node.stringEnum = true
			return node, nil
		}
		if role != referencedCollectionOwnDefault && role != referencedScalarOwnDefault {
			return nil, unsupported(naming.Pointer(pointer, "enum"))
		}
		// Native numeric/Boolean enum terminals keep their scalar wire type and
		// the original membership checker. Exact source-default membership runs
		// before rounding in both validation modes; the mapped scalar and the
		// actual backing must still agree after the reference planner's checks.
		if kind != "boolean" && kind != "integer" && kind != "number" {
			return nil, unsupported(naming.Pointer(pointer, "enum"))
		}
		terminalKeys := []string{"type", "nullable", "format", "enum"}
		if a.options.Validation {
			terminalKeys = append(terminalKeys, primitiveBoundKeys...)
		}
		if err := checkKeys(v, terminalKeys, pointer); err != nil {
			return nil, err
		}
		if members := get(v, "enum"); members.Kind() != document.Array || len(members.Items()) == 0 || len(members.Items()) > MaxMembers {
			return nil, diagnostic.At("E_MODEL_LIMIT", "Native enum default exceeds the supported 240-member limit.", naming.Pointer(pointer, "enum"))
		}
		if bare.Kind != Scalar || bare.Serializer != "" {
			return nil, unsupported(pointer)
		}
		scalar, err := scalarMapping(kind, v, pointer)
		if err != nil {
			return nil, err
		}
		if scalar != bare.Name {
			return nil, unsupported(pointer)
		}
		if c := bare.Validation; c != nil && c.MultipleOf != nil && c.Floating {
			return nil, unsupported(naming.Pointer(pointer, "multipleOf"))
		}
		node.native = a
		return node, nil
	}
	if edge == "" && kind == "object" && !has(v, "enum") {
		// A Named terminal may be an ordinary DTO instead of a String enum. The
		// target root must not carry its own default; property schemas keep their
		// own validated defaults through the nested model plan.
		if role != referencedNamedOwnDefault && role != referencedCollectionOwnDefault {
			return nil, unsupported(naming.Pointer(pointer, "properties"))
		}
		if bare.Kind != Named {
			return nil, unsupported(pointer)
		}
		object, err := a.objectDefaultNode(v, pointer, depth, false)
		if err != nil {
			return nil, err
		}
		// The annotation follows the effective consumer/alias nullability, not
		// only the target's own intent.
		object.ty = actual
		return object, nil
	}
	if edge == "" && a.options.Validation {
		keys = append(keys, primitiveBoundKeys...)
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	if edge != "" {
		expected := List
		if edge == "additionalProperties" {
			expected = Map
		}
		if bare.Kind != expected || bare.Element == nil {
			return nil, unsupported(pointer)
		}
		childRole := noReferencedDefaultRole
		if role == referencedCollectionOwnDefault {
			childRole = referencedCollectionOwnDefault
		}
		child, err := a.referencedDefaultSchemaRole(get(v, edge), naming.Pointer(pointer, edge), *bare.Element, depth+1, false, childRole)
		if err != nil {
			return nil, err
		}
		node.element = child
		return node, nil
	}
	if bare.Kind != Scalar || bare.Serializer != "" {
		return nil, unsupported(pointer)
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return nil, err
	}
	switch scalar {
	case "kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double":
	case "kotlin.time.Instant", "kotlinx.datetime.LocalDate":
		// Source role matters independently of coincident provider/type names.
		// Native markers, enums/consts and folds remain outside the keys above.
		format := text(get(v, "format"))
		if role == noReferencedDefaultRole || kind != "string" || format != "date" && format != "date-time" {
			return nil, unsupported(pointer)
		}
	default:
		return nil, unsupported(pointer)
	}
	if scalar != bare.Name {
		return nil, unsupported(pointer)
	}
	if c := bare.Validation; c != nil && c.MultipleOf != nil && c.Floating {
		return nil, unsupported(naming.Pointer(pointer, "multipleOf"))
	}
	return node, nil
}

func (a *analyzer) planReferencedFieldDefaults() *diagnostic.Diagnostic {
	for _, plan := range *a.referencedDefaults {
		node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
		if err != nil {
			return err
		}
		kind := node.ty.WithoutRootNull().Kind
		if kind != List && kind != Map && kind != Scalar && !(kind == Named && node.stringEnum) && !node.object {
			return unsupported(naming.Pointer(plan.Field.Pointer, "default"))
		}
		value := get(plan.Field.Schema, "default")
		at := naming.Pointer(plan.Field.Pointer, "default")
		work, bytes := 0, 0
		if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
			return err
		}
		encoded, err := node.value(value, at)
		if err != nil {
			return err
		}
		if len(encoded) > maxNestedDefaultBytes {
			return diagnostic.At("E_MODEL_LIMIT", "Canonical nested default exceeds the supported 32 KiB JSON byte limit.", at)
		}
		if plan.Model == nil {
			continue
		}
		property := &plan.Model.Properties[plan.Index]
		// Required and Patch annotations are fully checked before suppression.
		if !property.DefaultNull || property.Type.Kind == PatchValue {
			continue
		}
		if value.Kind() == document.Null {
			property.Default = "null"
			continue
		}
		if node.stringEnum {
			// Bind the final actual lexical type in the emitter after relocation.
			property.Default = "`openapi-kt default enum scalar`(" + naming.Literal(encoded) + ")"
			continue
		}
		if node.object {
			// Keep the plain-nominal boundary explicit: alias/Element views and
			// internal chains stay outside this role, and the emitter binds the
			// final actual DTO type after relocation.
			target := property.Type.WithoutRootNull()
			if target.Kind != Named || target.Element != nil {
				return unsupported(naming.Pointer(plan.Field.Pointer, "default"))
			}
			property.Default = "`openapi-kt default model`(" + naming.Literal(encoded) + ")"
			continue
		}
		if kind == Scalar {
			// Source admission/checks and both annotation budgets have already
			// run. Use the actual scalar literal path, without a JSON helper.
			scalar := node.ty.WithoutRootNull().Name
			literal, err := scalarLiteral(scalar, value, at)
			if err != nil {
				return err
			}
			property.Default = scalarDefaultInitializer(scalar, literal)
			continue
		}
		property.Default = "`openapi-kt default tree " + referencedDefaultTypeKey(property.Type.WithoutRootNull()) + "`(" + naming.Literal(encoded) + ")"
	}
	return nil
}

func referencedDefaultTypeKey(ty TypeRef) string {
	switch ty.Kind {
	case Named:
		view := "Alias "
		if ty.ArrayAliasView == NonNullArrayAlias {
			view = "NonNull Alias "
		}
		// Dots are illegal JVM method-name characters even inside backticks.
		return view + strings.ReplaceAll(ty.Name, ".", " ")
	case Nullable:
		return "Nullable " + referencedDefaultTypeKey(*ty.Element)
	case List:
		return "List " + referencedDefaultTypeKey(*ty.Element)
	case Map:
		return "Map " + referencedDefaultTypeKey(*ty.Element)
	default:
		return nestedDefaultTypeKey(ty)
	}
}

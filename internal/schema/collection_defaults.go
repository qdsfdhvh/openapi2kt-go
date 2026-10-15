package schema

import (
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Only direct ordinary DTO collection defaults call this planner. Enum leaves
// retain their original vocabulary/check source, never an injected default or
// copied schema. Nominal/reference leaves keep their guards; direct String
// enums declare the same lexical vocabulary as collection fields without defaults.
func (a *analyzer) collectionDefaultElement(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, err
	}
	if err := object(v, pointer); err != nil {
		return TypeRef{}, err
	}
	kind, nullable, typeErr := schemaType(v, pointer)
	keys := []string{"type", "nullable", "format"}
	if typeErr == nil && slices.Contains([]string{"", "string", "boolean", "integer", "number"}, kind) {
		keys = append(keys, "enum")
	}
	if a.options.Validation {
		keys = append(keys, primitiveBoundKeys...)
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return TypeRef{}, err
	}
	if typeErr != nil {
		return TypeRef{}, typeErr
	}
	if has(v, "enum") && (kind == "" || kind == "string") {
		if has(v, "format") {
			return TypeRef{}, unsupported(naming.Pointer(pointer, "enum"))
		}
		if hasPrimitiveBounds(v) {
			return a.stringEnumCollectionLeaf(v, name, pointer, nested, used, depth)
		}
		enum, err := buildStringEnum(name, v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		if nested != nil {
			if err := reserveNested(used, name, pointer); err != nil {
				return TypeRef{}, err
			}
			a.recordEnumOrigin(enum, v, pointer, nullable)
			*nested = append(*nested, enum)
		}
		ty := TypeRef{Kind: Named, Name: name}
		if nullable {
			ty = ty.Null()
		}
		return ty, nil
	}
	if !slices.Contains([]string{"string", "boolean", "integer", "number"}, kind) {
		return TypeRef{}, unsupported(pointer)
	}
	if has(v, "enum") {
		if values := get(v, "enum"); values.Kind() == document.Array && len(values.Items()) > MaxMembers {
			return TypeRef{}, diagnostic.At("E_MODEL_LIMIT", "Native enum collection default exceeds the supported 240-member limit.", naming.Pointer(pointer, "enum"))
		}
		if a.options.Validation {
			return a.nativeEnumValueType(v, pointer, depth)
		}
		scalar, err := a.nativeEnum(kind, v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		ty := TypeRef{Kind: Scalar, Name: scalar}
		if nullable {
			ty = ty.Null()
		}
		return ty, nil
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if !slices.Contains([]string{"kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double", "kotlin.time.Instant", "kotlinx.datetime.LocalDate"}, scalar) {
		return TypeRef{}, unsupported(pointer)
	}
	checks, err := a.primitiveValidation(v, scalar, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if checks != nil {
		a.validationSources[checks] = pointer
	}
	ty := TypeRef{Kind: Scalar, Name: scalar, Validation: checks}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}

func (a *analyzer) collectionDefaultMemberChecks(value, schema *document.Value, ty TypeRef, schemaPointer, valuePointer string) *diagnostic.Diagnostic {
	if has(schema, "enum") {
		kind, _, err := schemaType(schema, schemaPointer)
		if err != nil {
			return err
		}
		if err := a.enumDefaultMemberAt(schema, value, kind, schemaPointer, valuePointer); err != nil {
			return err
		}
		if !hasPrimitiveBounds(schema) {
			return nil // Boolean membership has no numeric bound arithmetic.
		}
	}
	return validateDefaultChecksAt(value, schema, ty.Validation, ty.Name, schemaPointer, valuePointer)
}

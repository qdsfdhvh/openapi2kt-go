package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Tuple projection may borrow an already accepted builtin or nominal source.
// Validate every original hop before allowing either a typed or raw projection.
func (a *analyzer) tupleScalar(v *document.Value, pointer string, depth int, work *int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	original, source := v, pointer
	start := depth
	for has(v, "$ref") {
		*work++
		if *work > 16384 {
			return tupleScalarSignature{}, diagnostic.At("E_MODEL_LIMIT", "Tuple reference source planning exceeds the supported work limit.", pointer)
		}
		if err := depthCheck(depth, pointer); err != nil {
			return tupleScalarSignature{}, err
		}
		if err := checkKeys(v, a.tupleSourceKeys(v, pointer, []string{"$ref", "nullable"}), pointer); err != nil {
			return tupleScalarSignature{}, err
		}
		if _, _, err := schemaType(v, pointer); err != nil {
			return tupleScalarSignature{}, err
		}
		reference, ok := get(v, "$ref").Text()
		if !ok {
			return tupleScalarSignature{}, invalid("$ref must be a string.", pointer)
		}
		target, p, err := a.defaultSource(reference, pointer)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		if owner := a.componentsByOriginalSchema(target); owner != nil {
			if owner.Pointer != p || reference != "#"+p || a.components[owner.Wire] != owner || a.hierarchySources[target] {
				return tupleScalarSignature{}, schemaReferenceUnsupported(pointer)
			}
		} else if err := a.internalDefaultMemberSource(target, p, reference); err != nil {
			return tupleScalarSignature{}, err
		}
		v, pointer = target, p
		depth++
	}
	if original != v {
		if composedModelCandidate(v) && !a.hierarchySources[v] {
			if err := depthCheck(depth, pointer); err != nil {
				return tupleScalarSignature{}, err
			}
			return a.tupleAllOf(original, source, v, pointer, start)
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		if kind == "object" && directInternalModel(v, pointer) && !a.hierarchySources[v] {
			if err := depthCheck(depth, pointer); err != nil {
				return tupleScalarSignature{}, err
			}
			return a.tupleDTO(original, source, v, pointer, start)
		}
		if kind == "array" || kind == "object" {
			return tupleScalarSignature{}, unsupported(naming.Pointer(source, "$ref"))
		}
		if (kind == "" || kind == "string") && plainStringEnumSchema(v) {
			if err := depthCheck(depth, pointer); err != nil {
				return tupleScalarSignature{}, err
			}
			return a.tupleStringEnum(original, source, v, pointer, start)
		}
		if a.options.Validation && has(v, "enum") && (kind == "integer" || kind == "number" || kind == "boolean") {
			if err := depthCheck(depth, pointer); err != nil {
				return tupleScalarSignature{}, err
			}
			return a.tupleNativeEnum(original, source, v, pointer, start)
		}
	}
	signature, err := tupleScalarOwnDefault(v, pointer, depth, original != v && a.tupleSourceOwnDefault(v, pointer))
	if err != nil {
		if original != v && err.Code == "E_SCHEMA_UNSUPPORTED" && err.Pointer != nil && *err.Pointer == pointer {
			return tupleScalarSignature{}, unsupported(naming.Pointer(source, "$ref"))
		}
		return tupleScalarSignature{}, err
	}
	if original != v {
		target, err := a.resolveSchema(text(get(original, "$ref")), source)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		ty, err := a.referenceType(target, nullableOverride(original), start, source)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		ty = defaultBacking(ty)
		bare := ty.WithoutRootNull()
		mapped, err := scalarMapping(signature.Kind, v, pointer)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		if bare.Kind != Scalar || bare.Name != mapped || bare.Validation != nil || bare.Serializer != "" {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
		signature.Nullable = ty.Kind == Nullable
	}
	return signature, nil
}

// Original lexical vocabulary identity distinguishes nominal enum members from
// plain String and from separate enum declarations with equal wire values.
// The existing borrowed-leaf authority verifies the actual retained declaration,
// even when a later tail or mixed raw projection discards this typed member.
func (a *analyzer) tupleStringEnum(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := checkKeys(terminal, a.tupleSourceKeys(terminal, pointer, []string{"type", "enum", "nullable"}), pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if _, err := buildStringEnumKeysFor("TupleMember", terminal, pointer, false, a); err != nil {
		return tupleScalarSignature{}, err
	}
	ty, err := a.tupleNominalType(original, source, terminal, pointer, depth, false)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: "string", Nullable: ty.Kind == Nullable, EnumSource: terminal}, nil
}

func (a *analyzer) tupleDTO(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := checkKeys(terminal, a.tupleSourceKeys(terminal, pointer, []string{"type", "nullable", "properties", "required", "minProperties", "maxProperties"}), pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	ty, err := a.tupleNominalType(original, source, terminal, pointer, depth, true)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: "object", Nullable: ty.Kind == Nullable, ModelSource: terminal}, nil
}

// A source/type pair is provisional until the shared borrowed-leaf validation
// finds the original unique retained constructor or vocabulary. This also runs
// for typed prefix members later discarded by an explicit tail/raw projection.
func (a *analyzer) tupleNominalType(original *document.Value, source string, terminal *document.Value, pointer string, depth int, dto bool) (TypeRef, *diagnostic.Diagnostic) {
	reference := text(get(original, "$ref"))
	a.recordBorrowedDTOAliasEdges(reference, source)
	if err := a.visitBorrowedSourceAlias(reference, source, depth, dto); err != nil {
		return TypeRef{}, err
	}
	leaf, err := a.resolveSchema("#"+pointer, source)
	if err != nil {
		return TypeRef{}, err
	}
	if leaf.Schema != terminal || leaf.Pointer != pointer {
		return TypeRef{}, schemaReferenceUnsupported(source)
	}
	admitted := false
	if dto && composedModelCandidate(terminal) {
		if a.tupleAllOfSources == nil {
			a.tupleAllOfSources = &tupleAllOfSourceBudget{Seen: map[*document.Value]bool{}}
		}
		admitted, err = a.admitBorrowedAllOfModelBudget(leaf, source, a.tupleAllOfSources)
		if err != nil {
			return TypeRef{}, err
		}
	} else if dto {
		admitted = a.admitBorrowedOrdinaryDTO(leaf, source)
	} else {
		admitted = a.admitBorrowedNominalLeaf(leaf, source)
	}
	if !admitted {
		return TypeRef{}, schemaReferenceUnsupported(source)
	}
	target, err := a.resolveSchema(reference, source)
	if err != nil {
		return TypeRef{}, err
	}
	ty, err := a.referenceType(target, nullableOverride(original), depth, source)
	if err != nil {
		return TypeRef{}, err
	}
	ty = defaultBacking(ty)
	bare := ty.WithoutRootNull()
	if bare.Kind != Named || bare.Element != nil || bare.Serializer != "" {
		return TypeRef{}, schemaReferenceUnsupported(source)
	}
	return ty, nil
}

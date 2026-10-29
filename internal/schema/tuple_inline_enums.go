package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Validate original inline vocabularies before a selected tail or raw carrier
// can discard them. This creates no declaration, check or warning: the actual
// selected element still belongs to the context's ordinary collection planner.
func (a *analyzer) tupleInlineEnum(v *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	keys := []string{"type", "nullable", "enum"}
	if kind == "" || kind == "string" {
		keys = append(keys, "format", "x-native-type")
	}
	if kind == "integer" || kind == "number" || kind == "boolean" {
		keys = append(keys, "format")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	switch kind {
	case "", "string":
		if _, err := buildStringEnumAllowedKeys("TupleMember", v, pointer, keys); err != nil {
			return tupleScalarSignature{}, err
		}
		kind = "string"
	case "integer", "number", "boolean":
		if values := get(v, "enum"); values.Kind() == document.Array && len(values.Items()) > MaxMembers {
			return tupleScalarSignature{}, diagnostic.At("E_MODEL_LIMIT", "Tuple native enum exceeds the supported 240-member limit.", naming.Pointer(pointer, "enum"))
		}
		validation := *a
		validation.warnings = nil
		if _, err := validation.nativeEnumProfile(kind, v, pointer, false, false); err != nil {
			return tupleScalarSignature{}, err
		}
	default:
		return tupleScalarSignature{}, unsupported(pointer)
	}
	// The shared enum source registry captures physical ancestors and all original
	// vocabulary nodes, including sources later discarded by projection.
	if err := a.captureTupleNativeSource(v, pointer, v, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: kind, Format: text(get(v, "format")), HasFormat: has(v, "format"), Nullable: nullable, EnumSource: v}, nil
}

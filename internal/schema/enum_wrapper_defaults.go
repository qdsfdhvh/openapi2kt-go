package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func originalEnumWrapperDefaultSource(v *document.Value) bool {
	if !has(v, "enum") {
		return false
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	return (kind == "" || kind == "string") && plainStringEnumSchema(v) ||
		kind == "integer" || kind == "number" || kind == "boolean"
}

// Source vocabulary and its own annotation retain their original null grammar.
// Only the consuming value plan receives the nearest alias root null intent.
func (a *analyzer) originalEnumWrapperDefaultEncoded(c *component, target *wrapperDefaultSource, validation *analyzer) (string, *diagnostic.Diagnostic) {
	at := naming.Pointer(c.Pointer, "default")
	bound := a.referenceWrapperDefaultTarget(c)
	if !a.componentRoot(c.Schema, c.Pointer) || bound == nil || target.Schema != bound.Schema || target.Pointer != bound.Pointer ||
		!a.componentRoot(target.Schema, target.Pointer) || (target.Nullable == nil) != (bound.Nullable == nil) ||
		target.Nullable != nil && *target.Nullable != *bound.Nullable {
		return "", unsupported(at)
	}
	node, err := validation.nestedDefaultSchema(target.Schema, "Validation", target.Pointer, nil, nil, 0, true)
	if err != nil {
		return "", err
	}
	if target.Nullable != nil {
		node.ty = node.ty.WithoutRootNull()
		if *target.Nullable {
			node.ty = node.ty.Null()
		}
	}
	value := get(c.Schema, "default")
	if _, err := node.value(value, at); err != nil {
		return "", err
	}
	// Value validation may normalize mapped numbers for nested initializers.
	// Factories keep their original JSON tokens and decode through the alias.
	encoded, marshalErr := value.MarshalJSON()
	if marshalErr != nil {
		return "", invalid("A component default must encode to valid JSON.", at)
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	a.warnings = mergeWarnings(a.warnings, validation.warnings)
	return string(encoded), nil
}

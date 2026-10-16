package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// inlineAllOfDefaultElement detects only the consuming inline collection path.
// It grants no reference, object-member or component-root default role. The
// shared type plan must still retain a constructor Model for this exact leaf.
func inlineAllOfDefaultElement(v *document.Value) *document.Value {
	for edges := 0; edges <= MaxSchemaDepth && v != nil; edges++ {
		if has(v, "$ref") {
			return nil
		}
		if edges > 0 && has(v, "allOf") {
			// Primitive compositions keep their original leaf/default route and
			// diagnostics; only provisional ordinary object members join here.
			for _, member := range get(v, "allOf").Items() {
				kind, _, err := schemaType(member, "")
				if err != nil || kind != "" && kind != "object" {
					return nil
				}
			}
			return v
		}
		kind, _, err := schemaType(v, "")
		if err != nil {
			return nil
		}
		switch {
		case kind == "array":
			v = get(v, "items")
		case dictionarySchema(v):
			v = get(v, "additionalProperties")
		default:
			return nil
		}
	}
	return nil
}

// ordinaryAllOfDefaultNode borrows the shared ordinary object shape. Disjoint
// contributions require no cross-contribution constraint/default aggregation;
// every child keeps its original schema node, pointer and source depth.
func (a *analyzer) ordinaryAllOfDefaultNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	wrapper, err := parseReferenceWrapper(v, pointer, true)
	if err != nil {
		return nil, err
	}
	if wrapper != nil {
		return nil, unsupported(naming.Pointer(pointer, "default"))
	}
	shape, err := a.objectShape(v, pointer, depth)
	if err != nil {
		return nil, err
	}
	if len(shape.Fields) == 0 || len(shape.Discarded) != 0 || len(shape.Additional) != 0 || len(shape.CountSources) != 0 {
		return nil, unsupported(naming.Pointer(pointer, "default"))
	}
	node, err := a.objectDefaultShapeNode(shape, pointer, true)
	if err != nil {
		return nil, err
	}
	node.schema = v
	if shape.Nullable != nil && *shape.Nullable {
		node.ty = node.ty.Null()
	}
	return node, nil
}

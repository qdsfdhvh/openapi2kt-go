package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

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

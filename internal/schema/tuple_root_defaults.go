package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Component factories reuse only the direct scalar tuple field grammar. The
// complete original prefix validates before a tail is selected. Reference,
// enum, object, mixed raw and child-default roles remain independently scoped.
func (a *analyzer) scalarTupleComponentDefaultCandidate(c *component) bool {
	if c == nil || !has(c.Schema, "prefixItems") {
		return false
	}
	kind, _, err := schemaType(c.Schema, c.Pointer)
	if err != nil || kind != "array" {
		return false
	}
	prefix := get(c.Schema, "prefixItems")
	if prefix.Kind() != document.Array {
		return false
	}
	for i, member := range prefix.Items() {
		if _, err := tupleScalar(member, naming.Pointer(naming.Pointer(c.Pointer, "prefixItems"), strconv.Itoa(i)), 1); err != nil {
			return false
		}
	}
	if item, exists := c.Schema.Get("items"); exists {
		if _, err := tupleScalar(item, naming.Pointer(c.Pointer, "items"), 1); err != nil {
			return false
		}
	}
	probe := *a
	probe.warnings = nil
	decision, failure := probe.arrayItemsOwnDefault(c.Schema, c.Pointer, 0, true)
	return failure == nil && decision.Projection == nil
}

func (a *analyzer) scalarTupleComponentDefaultNode(c *component) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	const prefix = "/components/schemas/"
	wire, err := naming.DecodePointerSegment(strings.TrimPrefix(c.Pointer, prefix), c.Pointer)
	if err != nil || a.componentSchemas == nil || naming.Pointer("/components/schemas", wire) != c.Pointer || get(a.componentSchemas, wire) != c.Schema {
		return nil, unsupported(naming.Pointer(c.Pointer, "default"))
	}
	decision, err := a.arrayItemsOwnDefault(c.Schema, c.Pointer, 0, true)
	if err != nil {
		return nil, err
	}
	if decision.Projection != nil || has(decision.Selected, "$ref") {
		return nil, unsupported(naming.Pointer(c.Pointer, "prefixItems"))
	}
	element, err := a.nestedDefaultSchema(decision.Selected, c.Name+"Inner", decision.Pointer, nil, nil, 1, false)
	if err != nil {
		return nil, err
	}
	ty, err := a.checkedCollection(TypeRef{Kind: List, Element: &element.ty}, c.Schema, c.Pointer)
	if err != nil {
		return nil, err
	}
	_, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil {
		return nil, err
	}
	if nullable {
		ty = ty.Null()
	}
	return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: ty, element: element}, nil
}

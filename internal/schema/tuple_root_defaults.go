package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Registration admits original scalar/enum/ref prefix syntax. Shared type planning
// validates every source and its actual retained declaration; the final default
// node reuses direct elements, raw projections or actual selected references,
// never a nominal guess.
func (a *analyzer) tupleComponentDefaultCandidate(c *component) bool {
	if c == nil || !has(c.Schema, "prefixItems") {
		return false
	}
	kind, _, err := schemaType(c.Schema, c.Pointer)
	if err != nil || kind != "array" {
		return false
	}
	prefix := get(c.Schema, "prefixItems")
	if prefix.Kind() != document.Array || len(prefix.Items()) == 0 {
		return false
	}
	for i, member := range prefix.Items() {
		if unionNominalCandidate(member) {
			continue
		}
		if tupleNullCandidate(member) || tupleScalarConstCandidate(member) || structuredLiteral(member) {
			continue
		}
		if tupleCollectionCandidate(member) {
			continue
		}
		if directInternalModel(member, "") || composedModelCandidate(member) {
			continue
		}
		if member.Kind() == document.Boolean {
			continue
		}
		if has(member, "enum") {
			continue // Shared source planning validates the complete vocabulary.
		}
		if has(member, "$ref") && checkKeys(member, []string{"$ref", "nullable"}, naming.Pointer(c.Pointer+"/prefixItems", strconv.Itoa(i))) == nil {
			continue
		}
		if _, err := tupleScalar(member, naming.Pointer(naming.Pointer(c.Pointer, "prefixItems"), strconv.Itoa(i)), 1); err != nil {
			return false
		}
	}
	if item, exists := c.Schema.Get("items"); exists && item.Kind() != document.Boolean {
		if unionNominalCandidate(item) {
			return true
		}
		if tupleNullCandidate(item) || tupleScalarConstCandidate(item) || structuredLiteral(item) {
			return true
		}
		if tupleCollectionCandidate(item) {
			return true
		}
		if directInternalModel(item, "") || composedModelCandidate(item) {
			return true
		}
		if has(item, "enum") {
			return true
		}
		if has(item, "$ref") && checkKeys(item, []string{"$ref", "nullable"}, naming.Pointer(c.Pointer, "items")) == nil {
			return true // The final selected nominal/default role is checked later.
		}
		if _, err := tupleScalar(item, naming.Pointer(c.Pointer, "items"), 1); err != nil {
			return false
		}
	}
	return true
}

func (a *analyzer) tupleComponentDefaultNode(c *component) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	const prefix = "/components/schemas/"
	wire, err := naming.DecodePointerSegment(strings.TrimPrefix(c.Pointer, prefix), c.Pointer)
	if err != nil || a.componentSchemas == nil || naming.Pointer("/components/schemas", wire) != c.Pointer || get(a.componentSchemas, wire) != c.Schema {
		return nil, unsupported(naming.Pointer(c.Pointer, "default"))
	}
	decision, err := a.arrayItemsOwnDefault(c.Schema, c.Pointer, 0, true)
	if err != nil {
		return nil, err
	}
	if decision.Projection != nil {
		return a.rawTupleDefaultNode(c.Schema, c.Pointer, decision)
	}
	if a.tupleObjects != nil && a.tupleObjects.Unions[decision.Selected] != nil {
		owner := a.componentsByOriginalSchema(c.Schema)
		if owner == nil || owner.Pointer != c.Pointer {
			return nil, schemaReferenceUnsupported(c.Pointer)
		}
		ty := defaultBacking(owner.AliasType)
		bare := ty.WithoutRootNull()
		if bare.Kind != List || bare.Element == nil {
			return nil, schemaReferenceUnsupported(c.Pointer)
		}
		child, err := a.tupleUnionDefaultElement(decision.Selected, decision.Pointer, *bare.Element, 1)
		if err != nil {
			return nil, err
		}
		return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: ty, element: child, dtoUniqueKeys: a.dtoUniqueKeys}, nil
	}
	if a.tupleObjects != nil && a.tupleObjects.Collections[decision.Selected] {
		owner := a.componentsByOriginalSchema(c.Schema)
		if owner == nil || owner.Pointer != c.Pointer {
			return nil, unsupported(c.Pointer)
		}
		ty := defaultBacking(owner.AliasType)
		bare := ty.WithoutRootNull()
		if bare.Kind != List || bare.Element == nil {
			return nil, unsupported(c.Pointer)
		}
		child, err := a.tupleCollectionDefaultElement(decision.Selected, decision.Pointer, *bare.Element, 1)
		if err != nil {
			return nil, err
		}
		return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: ty, element: child, dtoUniqueKeys: a.dtoUniqueKeys}, nil
	}
	if has(decision.Selected, "$ref") {
		return a.tupleReferenceDefaultNode(c.Schema, c.Pointer, decision, 0)
	}
	if directInternalModel(decision.Selected, decision.Pointer) || composedModelCandidate(decision.Selected) {
		owner := a.componentsByOriginalSchema(c.Schema)
		if owner == nil || owner.Pointer != c.Pointer {
			return nil, unsupported(c.Pointer)
		}
		ty := defaultBacking(owner.AliasType)
		bare := ty.WithoutRootNull()
		if bare.Kind != List || bare.Element == nil {
			return nil, unsupported(c.Pointer)
		}
		child, err := a.tupleInlineDTOObjectNode(decision.Selected, decision.Pointer, *bare.Element, 1)
		if err != nil {
			return nil, err
		}
		return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: ty, element: child, dtoUniqueKeys: a.dtoUniqueKeys}, nil
	}
	if a.booleanFalseRoot(decision.Selected) {
		owner := a.componentsByOriginalSchema(c.Schema)
		if owner == nil || owner.Pointer != c.Pointer {
			return nil, unsupported(c.Pointer)
		}
		ty := defaultBacking(owner.AliasType)
		if ty.WithoutRootNull().Kind != List || ty.WithoutRootNull().Element == nil {
			return nil, unsupported(c.Pointer)
		}
		element := *ty.WithoutRootNull().Element
		return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: ty, element: &nestedDefaultNode{schema: decision.Selected, pointer: decision.Pointer, ty: element, uninhabited: true}}, nil
	}
	var element *nestedDefaultNode
	if metadataStringEnumCandidate(decision.Selected) {
		owner := a.componentsByOriginalSchema(c.Schema)
		if owner == nil || owner.AliasType.WithoutRootNull().Element == nil {
			return nil, schemaReferenceUnsupported(c.Pointer)
		}
		element, err = a.tupleMetadataEnumDefaultElement(decision.Selected, decision.Pointer, *owner.AliasType.WithoutRootNull().Element, 1)
	} else if tupleNullCandidate(decision.Selected) || tupleScalarConstCandidate(decision.Selected) || structuredLiteral(decision.Selected) {
		owner := a.componentsByOriginalSchema(c.Schema)
		if owner == nil || owner.AliasType.WithoutRootNull().Element == nil {
			return nil, schemaReferenceUnsupported(c.Pointer)
		}
		element, err = a.tupleLiteralDefaultNode(decision.Selected, decision.Pointer, *owner.AliasType.WithoutRootNull().Element, 1)
	} else {
		element, err = a.nestedDefaultSchema(decision.Selected, c.Name+"Inner", decision.Pointer, nil, nil, 1, false)
	}
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

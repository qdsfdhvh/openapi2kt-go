package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// A selected tuple reference keeps the actual collection planner's element
// type. Source walking validates literals only; it creates no nominal owner or
// declaration and inherits no target annotation.
func (a *analyzer) tupleReferenceDefaultNode(v *document.Value, pointer string, decision *arrayItemsDecision, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if decision == nil || decision.Projection != nil || !has(v, "prefixItems") || !has(decision.Selected, "$ref") {
		return nil, unsupported(naming.Pointer(pointer, "default"))
	}
	validation := *a
	validation.warnings = nil
	selected, err := validation.arrayItemsOwnDefault(v, pointer, depth, true)
	if err != nil {
		return nil, err
	}
	if selected.Projection != nil || decision.Selected != selected.Selected || decision.Pointer != selected.Pointer {
		return nil, unsupported(decision.Pointer)
	}
	var scratch []Declaration
	elementType, err := a.collectionElementType(decision.Selected, "TupleMember", decision.Pointer, &scratch, map[string]bool{}, depth+1)
	if err != nil {
		return nil, err
	}
	if len(scratch) != 0 {
		return nil, unsupported(decision.Pointer)
	}
	// Actual type planning owns source warnings.
	validation.objectDefaultWork = nil
	element, err := validation.tupleReferenceDefaultElement(decision.Selected, decision.Pointer, elementType, depth+1)
	if err != nil {
		return nil, err
	}
	ty, err := a.checkedArrayItems(elementType, v, pointer, decision)
	if err != nil {
		return nil, err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	if nullable {
		ty = ty.Null()
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: element, dtoUniqueKeys: a.dtoUniqueKeys}, nil
}

func (a *analyzer) tupleReferenceDefaultElement(v *document.Value, pointer string, actual TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if a.tupleSpecials != nil && a.tupleSpecials.Sources[v] != nil {
		return a.tupleSpecialDefaultElement(v, pointer, actual)
	}
	if a.tupleCovers != nil && a.tupleCovers.Sources[v] != nil {
		if !tupleCoverBacking(defaultBacking(actual)) || (defaultBacking(actual).Kind == Nullable) != a.tupleCovers.Sources[v].Nullable {
			return nil, schemaReferenceUnsupported(pointer)
		}
		return &nestedDefaultNode{schema: v, pointer: pointer, ty: defaultBacking(actual), rawJSON: true}, nil
	}
	if a.tupleObjects != nil && a.tupleObjects.Unions[v] != nil {
		return a.tupleUnionDefaultElement(v, pointer, actual, depth)
	}
	original, source := v, pointer
	for {
		var err *diagnostic.Diagnostic
		a, err = a.objectDefaultStep(pointer)
		if err != nil {
			return nil, err
		}
		if err := depthCheck(depth, pointer); err != nil {
			return nil, err
		}
		if !has(v, "$ref") {
			break
		}
		if err := checkKeys(v, a.tupleSourceKeys(v, pointer, []string{"$ref", "nullable"}), pointer); err != nil {
			return nil, err
		}
		target, p, err := a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, err
		}
		v, pointer, depth = target, p, depth+1
	}
	if err := a.captureTupleNativeSource(original, source, v, pointer); err != nil {
		return nil, err
	}
	var node *nestedDefaultNode
	var err *diagnostic.Diagnostic
	if tupleNullCandidate(v) || tupleScalarConstCandidate(v) || structuredLiteral(v) {
		return a.tupleLiteralDefaultNode(v, pointer, actual, depth)
	} else if tupleCollectionCandidate(v) {
		return a.tupleCollectionDefaultElement(original, source, actual, depth)
	} else if a.booleanFalseRoot(v) {
		node = &nestedDefaultNode{schema: v, pointer: pointer, uninhabited: true}
	} else if a.booleanTrueValue(v) {
		node = &nestedDefaultNode{schema: v, pointer: pointer, rawJSON: true}
	} else if composedModelCandidate(v) {
		// The original tuple planner already required an ordinary allOf
		// constructor and its final retained source binding. Preserve its
		// winning/default aggregation and whole-object checks in the value walk.
		node, err = a.ordinaryAllOfDefaultNodeCounts(v, pointer, depth, true)
	} else if objectDefaultCandidateSchema(v) {
		shape, failure := a.modelShape(v, pointer, depth, true)
		if failure != nil {
			return nil, failure
		}
		node, err = a.objectDefaultShapeNode(shape, pointer, true)
		if err == nil {
			node.schema = v
			node.objectCounts, err = a.ordinaryObjectCounts(v, pointer, shape, true, true)
		}
	} else {
		node, err = a.nestedDefaultSchema(v, "TupleMember", pointer, nil, nil, depth, a.tupleSourceOwnDefault(v, pointer))
	}
	if err != nil {
		return nil, err
	}
	node.ty = defaultBacking(actual) // Nearest actual alias/consumer nullable view.
	return node, nil
}

func (a *analyzer) tupleReferenceFieldDefault(v *document.Value, pointer string, decision *arrayItemsDecision, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	node, err := a.tupleReferenceDefaultNode(v, pointer, decision, depth)
	if err != nil {
		return TypeRef{}, "", err
	}
	value, at := get(v, "default"), naming.Pointer(pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return TypeRef{}, "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return TypeRef{}, "", err
	}
	if len(encoded) > maxNestedDefaultBytes {
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical tuple default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	if element := node.ty.WithoutRootNull().Element; element != nil && defaultBacking(*element).WithoutRootNull().Kind == Named && value.Kind() != document.Null {
		// This existing typed marker binds the final actual nominal leaf after
		// lexical linking/hoist/merge, for DTOs as well as String vocabularies.
		return node.ty, "`openapi-kt default enum tree`(" + naming.Literal(encoded) + ")", nil
	}
	if value.Kind() != document.Null && node.element != nil && node.element.rawJSON {
		name := "JsonElement"
		if node.element.ty.Kind == Nullable {
			name = "Nullable JsonElement"
		}
		return node.ty, "`openapi-kt default array " + name + "`(" + naming.Literal(encoded) + ")", nil
	}
	initializer, err := referencedDefaultInitializer(node, node.ty, value, encoded, at)
	return node.ty, initializer, err
}

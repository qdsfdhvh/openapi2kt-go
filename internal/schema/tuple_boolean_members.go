package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type tupleBooleanSelection struct {
	Array, Leaf          *document.Value
	Pointer, LeafPointer string
	Depth                int
}

func (a *analyzer) onlyTupleFalsePeers(ds []Declaration) bool {
	if len(ds) == 0 || a.tupleBooleans == nil {
		return false
	}
	for _, d := range ds {
		helper, ok := d.(*FalseValue)
		if !ok {
			return false
		}
		if _, ok := a.tupleBooleans.Declarations[helper]; !ok {
			return false
		}
	}
	return true
}

func tupleFalseContainerCandidate(v *document.Value) bool {
	for depth := 0; depth <= MaxSchemaDepth && v != nil; depth++ {
		if has(v, "$ref") {
			return false
		}
		if prefix := get(v, "prefixItems"); prefix.Kind() == document.Array {
			for _, member := range prefix.Items() {
				if b, ok := member.Bool(); ok && !b {
					return true
				}
			}
		}
		kind, _, err := schemaType(v, "")
		if err != nil {
			return false
		}
		if kind == "array" {
			v = get(v, "items")
		} else if dictionarySchema(v) {
			v = get(v, "additionalProperties")
		} else {
			return false
		}
	}
	return false
}

// Operation qualification copies container nodes. Preserve source proof for
// those exact new element addresses instead of inferring it from type spelling.
func (a *analyzer) qualifyTupleOperationType(t TypeRef, local, qualified string) TypeRef {
	out := qualifyOperationType(t, local, qualified)
	if a.tupleBooleans != nil || a.tupleBooleanRefs != nil || a.tupleObjects != nil || a.tupleLiterals != nil || a.tupleCovers != nil || a.tupleSpecials != nil {
		old, current := t.Element, out.Element
		seen := map[*TypeRef]bool{}
		for old != nil && current != nil && !seen[old] {
			seen[old] = true
			if a.tupleSpecials != nil {
				if use, ok := a.tupleSpecials.Uses[old]; ok {
					a.recordTupleSpecialUse(current, use.Source)
				}
			}
			if a.tupleCovers != nil {
				if use, ok := a.tupleCovers.Uses[old]; ok {
					a.recordTupleCoverUse(current, use.Source)
				}
			}
			if a.tupleLiterals != nil {
				if use, ok := a.tupleLiterals.Uses[old]; ok {
					a.recordTupleLiteralUse(current, use.Source)
				}
			}
			if a.tupleBooleans != nil {
				if source, ok := a.tupleBooleans.Uses[old]; ok {
					a.tupleBooleans.Uses[current] = source
				}
			}
			if a.tupleBooleanRefs != nil {
				if use, ok := a.tupleBooleanRefs.Uses[old]; ok {
					a.recordTupleBooleanReferenceUse(current, use.Source)
				}
			}
			if a.tupleObjects != nil {
				if use, ok := a.tupleObjects.UnionUses[old]; ok {
					a.recordTupleUnionUse(current, use.Source)
				}
				if use, ok := a.tupleObjects.Uses[old]; ok {
					a.recordTupleInlineAllOfUse(current, use.Source)
				}
				if use, ok := a.tupleObjects.CollectionUses[old]; ok {
					a.recordTupleCollectionUse(current, use.Source)
				}
			}
			old, current = old.Element, current.Element
		}
	}
	return out
}

type tupleBooleanRegistry struct {
	Selections   map[*document.Value]tupleBooleanSelection
	Declarations map[*FalseValue]tupleBooleanSelection
	Names        map[*FalseValue]string
	Uses         map[*TypeRef]tupleBooleanSelection
}

func (a *analyzer) initTupleBooleans() {
	if a.tupleBooleans == nil {
		a.tupleBooleans = &tupleBooleanRegistry{Selections: map[*document.Value]tupleBooleanSelection{}, Declarations: map[*FalseValue]tupleBooleanSelection{}, Names: map[*FalseValue]string{}, Uses: map[*TypeRef]tupleBooleanSelection{}}
	}
}

func (a *analyzer) tupleBooleanMember(v *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if !a.booleanTrueValue(v) && !a.booleanFalseRoot(v) {
		return tupleScalarSignature{}, unsupported(pointer)
	}
	if err := a.captureTupleNativeSource(v, pointer, v, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	kind := "tuple_false"
	if a.booleanTrueValue(v) {
		kind = "tuple_true"
	}
	return tupleScalarSignature{Kind: kind}, nil
}

func (a *analyzer) selectTupleFalse(v *document.Value, pointer string, decision *arrayItemsDecision, depth int) {
	if decision.Projection != nil || !a.booleanFalseRoot(decision.Selected) {
		return
	}
	a.initTupleBooleans()
	a.tupleBooleans.Selections[decision.Selected] = tupleBooleanSelection{v, decision.Selected, pointer, decision.Pointer, depth}
}

// Only a shared tuple decision grants this exact original false value an
// element role. The ordinary caller owns the real nested/peer declaration.
func (a *analyzer) tupleFalseElementType(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if !a.booleanFalseRoot(v) || a.tupleBooleans == nil {
		return nil, nil
	}
	selection, ok := a.tupleBooleans.Selections[v]
	if !ok || selection.LeafPointer != pointer {
		return nil, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if nested == nil {
		return nil, unsupported(pointer)
	}
	if used == nil {
		used = map[string]bool{}
	}
	if err := reserveNested(used, name, pointer); err != nil {
		return nil, err
	}
	d := &FalseValue{ClassName: name}
	*nested = append(*nested, d)
	a.tupleBooleans.Declarations[d] = selection
	a.tupleBooleans.Names[d] = name
	ty := TypeRef{Kind: Named, Name: name}
	return &ty, nil
}

func (a *analyzer) checkedArrayItems(element TypeRef, v *document.Value, pointer string, decision *arrayItemsDecision) (TypeRef, *diagnostic.Diagnostic) {
	element = a.tupleSpecialWork(element, decision)
	element = a.tupleCoverWork(element, decision)
	element = a.tupleBooleanReferenceWork(element, decision)
	element = a.tupleLiteralWork(element, decision)
	element = a.tupleInlineAllOfSourceWork(element, decision)
	ty, err := a.checkedCollection(TypeRef{Kind: List, Element: &element}, v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	a.recordTupleCollectionWork(ty.Element, decision)
	if a.tupleSpecials != nil && decision != nil && decision.Projection == nil && a.tupleSpecials.Sources[decision.Selected] != nil {
		a.recordTupleSpecialUse(ty.Element, decision.Selected)
	}
	if a.tupleCovers != nil && decision != nil && decision.Projection == nil && a.tupleCovers.Sources[decision.Selected] != nil {
		a.recordTupleCoverUse(ty.Element, decision.Selected)
	}
	if a.tupleObjects != nil && decision != nil && decision.Projection == nil && a.tupleObjects.Unions[decision.Selected] != nil {
		a.recordTupleUnionUse(ty.Element, decision.Selected)
	}
	if a.tupleLiterals != nil && decision != nil && decision.Projection == nil {
		if source := a.tupleLiterals.Sources[decision.Selected]; source != nil {
			a.recordTupleLiteralUse(ty.Element, source)
		}
	}
	if a.tupleBooleans != nil && decision != nil && decision.Projection == nil {
		if selection, ok := a.tupleBooleans.Selections[decision.Selected]; ok {
			a.tupleBooleans.Uses[ty.Element] = selection
		}
	}
	if a.tupleBooleanRefs != nil && decision != nil && decision.Projection == nil {
		if source := a.tupleBooleanRefs.Sources[decision.Selected]; source != nil {
			a.recordTupleBooleanReferenceUse(ty.Element, source)
		}
	}
	if a.tupleObjects != nil && decision != nil && decision.Projection == nil && a.tupleObjects.AllOf[decision.Selected] {
		a.recordTupleInlineAllOfUse(ty.Element, decision.Selected)
	}
	if a.tupleObjects != nil && decision != nil && decision.Projection == nil && a.tupleObjects.Collections[decision.Selected] {
		a.recordTupleCollectionUse(ty.Element, decision.Selected)
	}
	return ty, nil
}

func (a *analyzer) tupleFalseDefaultNode(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	var scratch []Declaration
	if nested == nil {
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	element, err := a.collectionElementType(decision.Selected, name+"Inner", decision.Pointer, nested, used, depth+1)
	if err != nil {
		return nil, err
	}
	ty, err := a.checkedArrayItems(element, v, pointer, decision)
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
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: &nestedDefaultNode{schema: decision.Selected, pointer: decision.Pointer, ty: element, uninhabited: true}}, nil
}

func (a *analyzer) tupleFalseFieldDefault(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	node, err := a.tupleFalseDefaultNode(v, pointer, name, decision, nested, used, depth)
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
	if value.Kind() == document.Null {
		return node.ty, "null", nil
	}
	return node.ty, "`openapi-kt default enum tree`(" + naming.Literal(encoded) + ")", nil
}

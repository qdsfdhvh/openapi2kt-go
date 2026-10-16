package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// inlineAllOfDefaultElement detects only an inline collection path. Its caller
// chooses the consuming field or object-property role; this detector grants no
// reference or component-root role. Shared type planning owns the constructor.
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

// allOfDefaultMemberElement scopes a collection property to its original bare
// allOf leaf or element reference. Referenced collection roots stay separate.
func (a *analyzer) allOfDefaultMemberElement(v *document.Value) *document.Value {
	if leaf := inlineAllOfDefaultElement(v); leaf != nil {
		return leaf
	}
	return a.inlineReferencedAllOfElement(v)
}

// inlineAllOfObjectDefaultMember admits only an ordinary inline constructor
// property in an object value walk. The enclosing field's shared type plan has
// already planned its declarations; this walk borrows sources, never declares
// a type or widens reference/collection admission. Collection properties may
// explicitly delegate their original inline leaf to this same constructor gate.
func (a *analyzer) inlineAllOfObjectDefaultMember(v *document.Value, pointer string, depth int, allowOwnDefault bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if !composedModelCandidate(v) || a.hierarchySources[v] ||
		!allowOwnDefault && has(v, "default") ||
		a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) {
		return nil, unsupported(naming.Pointer(pointer, "allOf"))
	}
	node, err := a.ordinaryAllOfDefaultNode(v, pointer, depth)
	if err != nil {
		return nil, err
	}
	// Scalar consts are body properties, so a const-only shape has no
	// constructor and keeps its separate default role even if omitted.
	for _, property := range node.objectFields {
		if !has(property.node.schema, "const") {
			return node, nil
		}
	}
	return nil, unsupported(naming.Pointer(pointer, "allOf"))
}

// referencedAllOfObjectDefaultMember borrows a named or internal allOf terminal
// through finite named/internal reference chains. Type planning owns the actual
// reference and final declaration; this walk retains every original hop and nullable intent.
func (a *analyzer) referencedAllOfObjectDefaultMember(v *document.Value, pointer, reference string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	var nullable *bool
	for {
		if err := depthCheck(depth, pointer); err != nil {
			return nil, err
		}
		owner := a.componentsByOriginalSchema(v)
		if owner == nil && has(v, "$ref") {
			// A transparent internal hop has source identity, not a nominal
			// declaration of its own. Keep its original annotations and location;
			// the terminal still needs the existing constructor binding below.
			if err := a.internalDefaultMemberSource(v, pointer, reference); err != nil {
				return nil, err
			}
		} else if owner == nil {
			// Internal terminals require the existing typed use's original
			// lexical binding. The shared final linker will reject missing,
			// copied or ambiguous declarations before any artifacts can emit.
			if err := a.internalAllOfDefaultMemberSource(v, pointer, reference); err != nil {
				return nil, err
			}
		} else if owner.Pointer != pointer || reference != "#"+owner.Pointer || a.hierarchySources[v] {
			return nil, unsupported(naming.Pointer(pointer, "$ref"))
		}
		if !has(v, "$ref") {
			if !has(v, "allOf") {
				return nil, unsupported(pointer)
			}
			node, err := a.inlineAllOfObjectDefaultMember(v, pointer, depth, false)
			if err != nil {
				return nil, err
			}
			if nullable != nil {
				node.ty = node.ty.WithoutRootNull()
				if *nullable {
					node.ty = node.ty.Null()
				}
			}
			return node, nil
		}
		var err *diagnostic.Diagnostic
		a, err = a.objectDefaultStep(pointer)
		if err != nil {
			return nil, err
		}
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return nil, err
		}
		if _, _, err := schemaType(v, pointer); err != nil {
			return nil, err
		}
		if nullable == nil {
			nullable = nullableOverride(v)
		}
		reference = text(get(v, "$ref"))
		v, pointer, err = a.defaultSource(reference, pointer)
		if err != nil {
			return nil, err
		}
		depth++
	}
}

// internalDefaultMemberSource checks only the original internal source path,
// for reference hops and collection entities. The enclosing type plan already
// admitted the source; no new type is resolved here.
func (a *analyzer) internalDefaultMemberSource(v *document.Value, pointer, reference string) *diagnostic.Diagnostic {
	source, err := a.lookupSchemaReference(reference, pointer)
	if err != nil || reference != "#"+pointer || source.Schema != v || source.Pointer != pointer ||
		source.SourceCatalog == nil || a.components[source.SourceCatalog.Wire] != source.SourceCatalog ||
		a.hierarchySources[source.SourceCatalog.Schema] || a.hierarchySources[v] {
		return unsupported(naming.Pointer(pointer, "$ref"))
	}
	return nil
}

// internalAllOfDefaultMemberSource confirms an original typed terminal without
// resolving a new type or registering a new nominal binding. Source validation
// may precede final linking; successful generation still requires that barrier.
func (a *analyzer) internalAllOfDefaultMemberSource(v *document.Value, pointer, reference string) *diagnostic.Diagnostic {
	binding := a.deferredNominals[v]
	if !composedModelCandidate(v) || binding == nil || binding.Schema != v || binding.Pointer != pointer ||
		reference != "#"+pointer || binding.Owner == nil || binding.Owner.InternalModelTarget != nil ||
		a.components[binding.Owner.Wire] != binding.Owner || a.hierarchySources[binding.Owner.Schema] ||
		a.hierarchySources[v] || a.nominalsLinked && binding.Name == "" {
		return unsupported(naming.Pointer(pointer, "allOf"))
	}
	source, err := a.lookupSchemaReference(reference, pointer)
	if err != nil || source.Schema != v || source.Pointer != pointer || source.SourceCatalog != binding.Owner {
		return unsupported(naming.Pointer(pointer, "allOf"))
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

package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type tupleInlineDTOUse struct {
	Schema     *document.Value
	Pointer    string
	Depth      int
	AllOf      bool
	Collection bool
	Union      bool
}

func (a *analyzer) initTupleInlineDTOs() {
	if a.tupleObjects == nil {
		a.tupleObjects = &tupleInlineDTOState{Seen: map[*document.Value]string{}}
	}
	if a.tupleObjects.AllOf == nil {
		a.tupleObjects.AllOf = map[*document.Value]bool{}
		a.tupleObjects.Uses = map[*TypeRef]tupleInlineAllOfUse{}
		a.tupleObjects.Heights = map[*document.Value]int{}
	}
	if a.tupleObjects.Collections == nil {
		a.tupleObjects.Collections = map[*document.Value]bool{}
		a.tupleObjects.CollectionUses = map[*TypeRef]tupleCollectionUse{}
		a.tupleObjects.CollectionWork = map[*TypeRef]int{}
		a.tupleObjects.CollectionCaptured = map[*document.Value]bool{}
	}
	if a.tupleObjects.Unions == nil {
		a.tupleObjects.Unions = map[*document.Value]*tupleUnionSource{}
		a.tupleObjects.UnionUses = map[*TypeRef]tupleCollectionUse{}
		a.tupleObjects.UnionDeclarations = map[Declaration]tupleUnionDeclaration{}
	}
}

func (a *analyzer) tupleInlineDTOObjectNode(v *document.Value, pointer string, actual TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if composedModelCandidate(v) {
		node, err := a.ordinaryAllOfDefaultNodeCounts(v, pointer, depth, true)
		if err != nil {
			return nil, err
		}
		node.ty = actual
		return node, nil
	}
	shape, err := a.modelShape(v, pointer, depth, true)
	if err != nil {
		return nil, err
	}
	node, err := a.objectDefaultShapeNode(shape, pointer, true)
	if err != nil {
		return nil, err
	}
	node.schema = v
	node.ty = actual
	node.objectCounts, err = a.ordinaryObjectCounts(v, pointer, shape, true, true)
	return node, err
}

func (a *analyzer) tupleInlineDTOFieldDefault(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	if nested == nil {
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	element, err := a.collectionElementType(decision.Selected, name+"Inner", decision.Pointer, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	child, err := a.tupleInlineDTOObjectNode(decision.Selected, decision.Pointer, element, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	ty, err := a.checkedArrayItems(element, v, pointer, decision)
	if err != nil {
		return TypeRef{}, "", err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	if nullable {
		ty = ty.Null()
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: child, dtoUniqueKeys: a.dtoUniqueKeys}
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
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical tuple DTO default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	if value.Kind() == document.Null {
		return ty, "null", nil
	}
	return ty, "`openapi-kt default enum tree`(" + naming.Literal(encoded) + ")", nil
}

type tupleInlineDTOState struct {
	Seen               map[*document.Value]string
	Tasks              []tupleInlineDTOUse
	Validated          bool
	Proof              *tupleNativeSourceRegistry
	AllOf              map[*document.Value]bool
	Uses               map[*TypeRef]tupleInlineAllOfUse
	Selected           []tupleInlineAllOfSelection
	Heights            map[*document.Value]int
	Collections        map[*document.Value]bool
	CollectionUses     map[*TypeRef]tupleCollectionUse
	CollectionUseOrder []*TypeRef
	CollectionWork     map[*TypeRef]int
	CollectionCaptured map[*document.Value]bool
	CollectionSelected []tupleInlineAllOfSelection
	Unions             map[*document.Value]*tupleUnionSource
	UnionUses          map[*TypeRef]tupleCollectionUse
	UnionUseOrder      []*TypeRef
	UnionSelected      []tupleInlineAllOfSelection
	UnionDeclarations  map[Declaration]tupleUnionDeclaration
}

func (a *analyzer) tupleInlineDTO(v *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if err := checkKeys(v, []string{"type", "nullable", "properties", "required", "minProperties", "maxProperties"}, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	if err := a.registerTupleInlineSource(v, pointer, depth, false); err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: "object", Nullable: nullable, ModelSource: v}, nil
}

func (a *analyzer) registerTupleInlineSource(v *document.Value, pointer string, depth int, allOf bool) *diagnostic.Diagnostic {
	a.initTupleInlineDTOs()
	r := a.tupleObjects
	if previous, ok := r.Seen[v]; ok {
		if previous != pointer {
			return schemaReferenceUnsupported(pointer)
		}
	} else {
		if len(r.Tasks) >= 16384 {
			return diagnostic.At("E_MODEL_LIMIT", "Tuple DTO source validation exceeds the supported work budget.", pointer)
		}
		r.Seen[v] = pointer
		r.Tasks = append(r.Tasks, tupleInlineDTOUse{Schema: v, Pointer: pointer, Depth: depth, AllOf: allOf, Collection: r.Collections[v], Union: r.Unions[v] != nil})
	}
	if allOf {
		r.AllOf[v] = true
	}
	if err := a.captureTupleNativeSource(v, pointer, v, pointer); err != nil {
		return err
	}
	if !a.tupleSourceValidation && !r.Validated {
		if _, _, err := analyzeDocument(a.root, a.options, true, &r.Proof); err != nil {
			return err
		}
		r.Validated = true
	}
	return nil
}

func (a *analyzer) validateTupleInlineDTOProof() *diagnostic.Diagnostic {
	if a.tupleObjects == nil || len(a.tupleObjects.Tasks) == 0 || a.tupleSourceValidation {
		return nil
	}
	if !a.tupleObjects.Validated || a.tupleObjects.Proof == nil {
		return schemaReferenceUnsupported(a.tupleObjects.Tasks[0].Pointer)
	}
	r := a.tupleObjects.Proof
	if r.Root != a.root {
		return schemaReferenceUnsupported("/components/schemas")
	}
	for _, use := range a.tupleObjects.Tasks {
		if use.Union {
			fact, ok := r.Unions[use.Schema]
			source := a.tupleObjects.Unions[use.Schema]
			if !ok || source == nil || fact != source.Fact || fact.Pointer != use.Pointer || fact.Height != a.tupleObjects.Heights[use.Schema] {
				return schemaReferenceUnsupported(use.Pointer)
			}
		}
		if use.AllOf {
			if fact, ok := r.Constructors[use.Schema]; !ok || fact.Pointer != use.Pointer || fact.Height != a.tupleObjects.Heights[use.Schema] {
				return schemaReferenceUnsupported(use.Pointer)
			}
		}
		if use.Collection {
			fact, ok := r.Collections[use.Schema]
			if !ok || fact.Pointer != use.Pointer || fact.Height != a.tupleObjects.Heights[use.Schema] {
				return schemaReferenceUnsupported(use.Pointer)
			}
		}
	}
	for _, source := range r.Sources {
		if !tupleNativeNodeUnchanged(r.Nodes[source.Node]) {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	for _, source := range r.Sources {
		if bytes, ok := r.Terminals[source.Node]; ok {
			current, _ := source.Node.MarshalJSON()
			if string(current) != bytes {
				return schemaReferenceUnsupported(source.Pointer)
			}
		}
	}
	return nil
}

// Keep discarded DTO source plans in this isolated branch only. The normal
// planner emits selected DTOs in their real model/operation context. Late
// reference-default and structural checks see a complete local plan here.
func (a *analyzer) attachTupleSourceModels(ds []Declaration, client *Client) ([]Declaration, *diagnostic.Diagnostic) {
	if a.tupleObjects == nil {
		return ds, nil
	}
	planned := map[string]bool{}
	constructors := map[*document.Value][]*Model{}
	names := map[string]bool{}
	var visit func(Declaration)
	visit = func(d Declaration) {
		names[strings.ToLower(d.Name())] = true
		if origin, ok := a.unionOrigins[d]; ok {
			planned[origin.Pointer] = true
		}
		if origin, ok := a.tupleObjects.UnionDeclarations[d]; ok {
			planned[origin.Pointer] = true
		}
		switch d := d.(type) {
		case *Model:
			planned[a.modelPointers[d]] = true
			if source := a.modelOrigins[d]; source != nil && len(d.Properties) != 0 {
				constructors[source] = append(constructors[source], d)
			}
			for _, child := range d.Nested {
				visit(child)
			}
		case *ObjectUnion:
			for _, child := range d.Variants {
				visit(child)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		case *Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf)
			}
		}
	}
	for _, d := range ds {
		visit(d)
	}
	if client != nil {
		for _, d := range client.Models {
			visit(d)
		}
	}
	sequence := 0
	for i := 0; i < len(a.tupleObjects.Tasks); i++ {
		use := a.tupleObjects.Tasks[i]
		if use.Union {
			if !planned[use.Pointer] {
				context := *a
				context.usage, context.fieldUnions, context.warnings = SharedUsage, NativeFieldUnions, nil
				sequence++
				name := "TupleUnionSource" + strconv.Itoa(sequence)
				for names[strings.ToLower(name)] {
					sequence++
					name = "TupleUnionSource" + strconv.Itoa(sequence)
				}
				var scratch []Declaration
				if _, err := context.propertyType(use.Schema, name, use.Pointer, &scratch, map[string]bool{}, use.Depth); err != nil {
					return nil, err
				}
				for _, d := range scratch {
					ds = append(ds, d)
					visit(d)
				}
				planned[use.Pointer] = true
			}
			continue
		}
		if use.Collection {
			context := *a
			context.usage, context.warnings = SharedUsage, nil
			var scratch []Declaration
			sequence++
			name := "TupleCollectionSource" + strconv.Itoa(sequence)
			for names[strings.ToLower(name)] {
				sequence++
				name = "TupleCollectionSource" + strconv.Itoa(sequence)
			}
			ty, err := context.propertyType(use.Schema, name, use.Pointer, &scratch, map[string]bool{}, use.Depth)
			if err != nil {
				return nil, err
			}
			backing := defaultBacking(ty)
			kind := backing.WithoutRootNull().Kind
			if kind != List && kind != Map && kind != JSONObject {
				return nil, unsupported(use.Pointer)
			}
			for _, d := range scratch {
				ds = append(ds, d)
				visit(d)
			}
			if a.tupleNativeSources.Collections == nil {
				a.tupleNativeSources.Collections = map[*document.Value]tupleCollectionFact{}
			}
			height, err := a.tupleCollectionSourceHeight(use.Schema, use.Pointer, use.Depth)
			if err != nil {
				return nil, err
			}
			if err := a.tupleNativeSources.charge(1, 0, use.Pointer); err != nil {
				return nil, err
			}
			a.tupleNativeSources.Collections[use.Schema] = tupleCollectionFact{use.Pointer, backing.Kind == Nullable, kind, height}
			a.tupleObjects.Heights[use.Schema] = height
			continue
		}
		if !planned[use.Pointer] {
			sequence++
			name := "TupleSource" + strconv.Itoa(sequence)
			for names[strings.ToLower(name)] {
				sequence++
				name = "TupleSource" + strconv.Itoa(sequence)
			}
			var scratch []Declaration
			context := *a
			context.usage = SharedUsage
			context.warnings = nil
			if _, err := context.propertyType(use.Schema, name, use.Pointer, &scratch, map[string]bool{}, use.Depth); err != nil {
				return nil, err
			}
			if len(scratch) == 0 {
				return nil, unsupported(use.Pointer)
			}
			for _, d := range scratch {
				ds = append(ds, d)
				visit(d)
			}
		}
		if use.AllOf {
			actual := constructors[use.Schema]
			if len(actual) != 1 || a.modelPointers[actual[0]] != use.Pointer {
				return nil, unsupported(use.Pointer)
			}
			if a.tupleNativeSources.Constructors == nil {
				a.tupleNativeSources.Constructors = map[*document.Value]tupleInlineConstructorFact{}
			}
			if err := a.tupleNativeSources.charge(1, 0, use.Pointer); err != nil {
				return nil, err
			}
			a.tupleNativeSources.Constructors[use.Schema] = tupleInlineConstructorFact{use.Pointer, actual[0].Nullable, a.tupleObjects.Heights[use.Schema]}
		}
	}
	return ds, nil
}

package schema

import (
	"reflect"
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Source validation exports facts only. The actual consumer planner still owns
// inline versus root codecs, normalization, operation projection and names.
type tupleUnionSource struct {
	Original, Terminal       *document.Value
	Pointer, TerminalPointer string
	Fact                     tupleUnionFact
}

type tupleUnionFact struct {
	Pointer   string
	Nullable  bool
	Kind      TypeKind
	Scalar    string
	Height    int
	Product   bool
	Hierarchy bool
}

type tupleUnionDeclaration struct {
	Source   *document.Value
	Pointer  string
	Snapshot Declaration
	Origin   nominalOrigin
	Variants []tupleUnionVariant
}

type tupleUnionVariant struct {
	Actual  *Model
	Schema  *document.Value
	Pointer string
	Name    string
}

func (a *analyzer) tupleProductUnionCandidate(v *document.Value) bool {
	return composedModelCandidate(v) && !a.hierarchySources[v] && a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{})
}

func (a *analyzer) tupleUnionCandidate(v *document.Value) bool {
	return unionNominalCandidate(v) && !has(v, "allOf") || a.tupleProductUnionCandidate(v)
}

func tupleUnionDeclarationSnapshot(d Declaration) Declaration {
	switch d := d.(type) {
	case *PrimitiveUnion:
		copy := *d
		copy.Kinds = slices.Clone(d.Kinds)
		return &copy
	case *ObjectUnion:
		copy := *d
		copy.Common = slices.Clone(d.Common)
		copy.Variants = slices.Clone(d.Variants)
		copy.Tags = slices.Clone(d.Tags)
		copy.Presence = slices.Clone(d.Presence)
		copy.SerialNames = slices.Clone(d.SerialNames)
		copy.TagCombinations = slices.Clone(d.TagCombinations)
		for i, rule := range copy.TagCombinations {
			copy.TagCombinations[i].Matches = slices.Clone(rule.Matches)
			for j, match := range rule.Matches {
				copy.TagCombinations[i].Matches[j].Values = slices.Clone(match.Values)
			}
		}
		return &copy
	case *StringEnum:
		copy := *d
		copy.Members = slices.Clone(d.Members)
		return &copy
	case *Model:
		copy := *d
		copy.Properties = slices.Clone(d.Properties)
		copy.Constants = slices.Clone(d.Constants)
		copy.Nested = slices.Clone(d.Nested)
		return &copy
	case *Hierarchy:
		copy := *d
		copy.SerializeVia = slices.Clone(d.SerializeVia)
		if d.tagBodies != nil {
			copy.tagBodies = map[string]string{}
			for wire, value := range d.tagBodies {
				copy.tagBodies[wire] = value
			}
		}
		copy.Fields = slices.Clone(d.Fields)
		copy.Inherited = slices.Clone(d.Inherited)
		copy.Enums = slices.Clone(d.Enums)
		copy.Tags = slices.Clone(d.Tags)
		copy.OwnFields = slices.Clone(d.OwnFields)
		copy.TagCombinations = slices.Clone(d.TagCombinations)
		for i, rule := range copy.TagCombinations {
			copy.TagCombinations[i].Matches = slices.Clone(rule.Matches)
			for j, match := range rule.Matches {
				copy.TagCombinations[i].Matches[j].Values = slices.Clone(match.Values)
			}
		}
		if d.SerialName != nil {
			value := *d.SerialName
			copy.SerialName = &value
		}
		return &copy
	}
	return nil
}

func tupleUnionDeclarationMatches(d Declaration, saved Declaration) bool {
	switch actual := d.(type) {
	case *ObjectUnion:
		original, ok := saved.(*ObjectUnion)
		if !ok || !slices.Equal(actual.Variants, original.Variants) {
			return false
		}
	case *PrimitiveUnion:
		original, ok := saved.(*PrimitiveUnion)
		if !ok || actual.Object != original.Object {
			return false
		}
	case *Model:
		original, ok := saved.(*Model)
		if !ok || !slices.Equal(actual.Nested, original.Nested) {
			return false
		}
	case *Hierarchy:
		original, ok := saved.(*Hierarchy)
		if !ok || actual.Leaf != original.Leaf || !slices.Equal(actual.Enums, original.Enums) {
			return false
		}
	}
	return reflect.DeepEqual(saved, tupleUnionDeclarationSnapshot(d))
}

func (a *analyzer) recordTupleUnionDeclaration(d Declaration, v *document.Value, pointer string) {
	if a.tupleObjects == nil {
		return
	}
	a.initTupleInlineDTOs()
	if snapshot := tupleUnionDeclarationSnapshot(d); snapshot != nil {
		proof := tupleUnionDeclaration{Source: v, Pointer: pointer, Snapshot: snapshot, Origin: a.unionOrigins[d]}
		if _, ok := d.(*StringEnum); ok {
			proof.Origin = a.enumOrigins[d]
		}
		if h, ok := d.(*Hierarchy); ok {
			proof.Origin = a.hierarchyOrigins[h]
			if h.Leaf != nil {
				proof.Variants = append(proof.Variants, tupleUnionVariant{h.Leaf, a.modelOrigins[h.Leaf], a.modelPointers[h.Leaf], h.Leaf.ClassName})
			}
		}
		if union, ok := d.(*ObjectUnion); ok {
			for _, variant := range union.Variants {
				proof.Variants = append(proof.Variants, tupleUnionVariant{variant, a.modelOrigins[variant], a.modelPointers[variant], variant.ClassName})
			}
		}
		a.tupleObjects.UnionDeclarations[d] = proof
	}
}

func (a *analyzer) tupleUnionSource(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, source); err != nil {
		return tupleScalarSignature{}, err
	}
	if original == terminal && has(original, "default") {
		return tupleScalarSignature{}, unsupported(naming.Pointer(source, "default"))
	}
	a.initTupleInlineDTOs()
	r := a.tupleObjects
	if old := r.Unions[original]; old != nil {
		if old.Pointer != source || old.Terminal != terminal || old.TerminalPointer != pointer {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	} else {
		r.Unions[original] = &tupleUnionSource{Original: original, Terminal: terminal, Pointer: source, TerminalPointer: pointer}
		if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
			return tupleScalarSignature{}, err
		}
		_, refs, err := a.borrowedAllOfSourceBudget(terminal, pointer, source, nil)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		if err := a.captureTupleInlineAllOfCatalogs(refs, source); err != nil {
			return tupleScalarSignature{}, err
		}
		if err := a.captureTupleCollectionGraph(original, source); err != nil {
			return tupleScalarSignature{}, err
		}
	}
	// Existing resolver and borrowed-union admission remain the authority for
	// reference paths. Tuple syntax grants no new internal or composed target.
	if original != terminal {
		reference := text(get(original, "$ref"))
		a.recordBorrowedDTOAliasEdges(reference, source)
		if err := a.visitBorrowedSourceAlias(reference, source, depth, true); err != nil {
			return tupleScalarSignature{}, err
		}
		leaf, err := a.resolveSchema("#"+pointer, source)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		admitted := false
		if a.hierarchySources[terminal] {
			admitted, err = a.admitTupleHierarchyGraph(leaf, source)
			if err != nil {
				return tupleScalarSignature{}, err
			}
		} else if a.tupleProductUnionCandidate(terminal) {
			admitted = a.admitBorrowedProductUnion(leaf, source)
		} else {
			admitted = a.admitBorrowedUnion(leaf, source)
		}
		if leaf.Schema != terminal || leaf.Pointer != pointer || !admitted {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	}
	if err := a.registerTupleInlineSource(original, source, depth, false); err != nil {
		return tupleScalarSignature{}, err
	}
	var fact tupleUnionFact
	if a.tupleSourceValidation {
		context := *a
		context.usage, context.fieldUnions, context.warnings = SharedUsage, NativeFieldUnions, nil
		var scratch []Declaration
		ty, err := context.propertyType(original, "TupleUnionValidation", source, &scratch, map[string]bool{}, depth)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		var height int
		if a.hierarchySources[terminal] {
			height, err = a.tupleHierarchyReferenceHeight(original, source, terminal, pointer, depth)
		} else {
			height, err = a.tupleCollectionSourceHeight(original, source, depth)
		}
		if err != nil {
			return tupleScalarSignature{}, err
		}
		backing := defaultBacking(ty)
		bare := backing.WithoutRootNull()
		fact = tupleUnionFact{Pointer: source, Nullable: backing.Kind == Nullable, Kind: bare.Kind, Height: height, Product: a.tupleProductUnionCandidate(terminal), Hierarchy: a.hierarchySources[terminal]}
		if bare.Kind == Scalar {
			fact.Scalar = bare.Name
		}
		if a.tupleNativeSources.Unions == nil {
			a.tupleNativeSources.Unions = map[*document.Value]tupleUnionFact{}
		}
		if _, exists := a.tupleNativeSources.Unions[original]; !exists {
			if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
				return tupleScalarSignature{}, err
			}
		}
		a.tupleNativeSources.Unions[original] = fact
	} else {
		var ok bool
		fact, ok = r.Proof.Unions[original]
		if !ok || fact.Pointer != source {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	}
	r.Heights[original] = fact.Height
	if saved := r.Unions[original].Fact; saved.Pointer != "" && saved != fact {
		return tupleScalarSignature{}, schemaReferenceUnsupported(source)
	}
	r.Unions[original].Fact = fact
	kind := "union"
	if fact.Hierarchy {
		kind = "hierarchy"
	}
	return tupleScalarSignature{Kind: kind, Nullable: fact.Nullable, UnionSource: terminal}, nil
}

func (a *analyzer) recordTupleUnionUse(t *TypeRef, source *document.Value) {
	r := a.tupleObjects
	if _, exists := r.UnionUses[t]; !exists {
		r.UnionUseOrder = append(r.UnionUseOrder, t)
	}
	r.UnionUses[t] = tupleCollectionUse{Source: source, Pointer: r.Unions[source].Pointer, Snapshots: dtoOriginalGetterTypes(t)}
}

func (a *analyzer) selectTupleUnion(array *document.Value, pointer string, decision *arrayItemsDecision) {
	r := a.tupleObjects
	if r == nil || decision.Projection != nil || r.Unions[decision.Selected] == nil {
		return
	}
	for _, selection := range r.UnionSelected {
		if selection.Array == array {
			return
		}
	}
	r.UnionSelected = append(r.UnionSelected, tupleInlineAllOfSelection{array, decision.Selected, pointer})
}

func (a *analyzer) validateTupleUnionType(t *TypeRef, use tupleCollectionUse, declaration Declaration, count int) *diagnostic.Diagnostic {
	r := a.tupleObjects
	source := r.Unions[use.Source]
	registry := r.Proof
	if a.tupleSourceValidation {
		registry = a.tupleNativeSources
	}
	if source == nil || source.Pointer != use.Pointer || registry == nil || !tupleCollectionUseMatches(t, use) {
		return schemaReferenceUnsupported(use.Pointer)
	}
	fact, proven := registry.Unions[use.Source]
	if !proven || fact != source.Fact || fact.Pointer != use.Pointer || fact.Height != r.Heights[use.Source] {
		return schemaReferenceUnsupported(use.Pointer)
	}
	backing := defaultBacking(*t)
	bare := backing.WithoutRootNull()
	if bare.Kind == Named {
		proof, ok := r.UnionDeclarations[declaration]
		if !ok || count != 1 || proof.Source != source.Terminal || proof.Pointer != source.TerminalPointer || !tupleUnionDeclarationMatches(declaration, proof.Snapshot) {
			return schemaReferenceUnsupported(use.Pointer)
		}
		origin := a.unionOrigins[declaration]
		if h, ok := declaration.(*Hierarchy); ok {
			origin = a.hierarchyOrigins[h]
		}
		if proof.Origin.Schema != nil && (origin != proof.Origin || proof.Origin.Schema != source.Terminal || proof.Origin.Pointer != source.TerminalPointer) {
			return schemaReferenceUnsupported(use.Pointer)
		}
		for _, variant := range proof.Variants {
			if variant.Schema == nil || variant.Actual.ClassName != variant.Name || a.modelOrigins[variant.Actual] != variant.Schema || a.modelPointers[variant.Actual] != variant.Pointer {
				return schemaReferenceUnsupported(use.Pointer)
			}
		}
	} else if bare.Kind != JSON && (bare.Kind != fact.Kind || bare.Kind == Scalar && bare.Name != fact.Scalar) {
		return schemaReferenceUnsupported(use.Pointer)
	}
	return nil
}

func (a *analyzer) onlyTupleUnionPeers(ds []Declaration) bool {
	if len(ds) == 0 || a.tupleObjects == nil {
		return false
	}
	for _, d := range ds {
		proof, ok := a.tupleObjects.UnionDeclarations[d]
		if !ok || a.tupleObjects.Unions[proof.Source] == nil || !tupleUnionDeclarationMatches(d, proof.Snapshot) {
			return false
		}
	}
	return true
}

func tupleUnionContainerCandidate(v *document.Value) bool {
	for depth := 0; depth <= MaxSchemaDepth && v != nil; depth++ {
		if has(v, "$ref") {
			return false
		}
		if prefix := get(v, "prefixItems"); prefix != nil {
			for _, member := range prefix.Items() {
				if unionNominalCandidate(member) {
					return true
				}
			}
			if unionNominalCandidate(get(v, "items")) {
				return true
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

func (a *analyzer) tupleUnionDefaultElement(original *document.Value, pointer string, actual TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	source := a.tupleObjects.Unions[original]
	if source == nil || source.Pointer != pointer {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	ty := defaultBacking(actual)
	bare := ty.WithoutRootNull()
	node := &nestedDefaultNode{schema: source.Terminal, pointer: source.TerminalPointer, ty: ty, native: a, unionDepth: depth}
	if bare.Kind == JSON {
		node.rawJSON = true
		return node, nil
	}
	if bare.Kind == Scalar {
		return node, nil
	}
	if bare.Kind != Named || bare.Element != nil {
		return nil, unsupported(pointer)
	}
	deferred := bare.Name == "" && bare.deferredNominal != nil
	if deferred {
		binding := bare.deferredNominal
		if a.deferredNominals[source.Terminal] != binding || binding.Schema != source.Terminal || binding.Pointer != source.TerminalPointer || binding.Owner == nil || a.components[binding.Owner.Wire] != binding.Owner {
			return nil, schemaReferenceUnsupported(pointer)
		}
	}
	var actualDeclaration Declaration
	for d, origin := range a.tupleObjects.UnionDeclarations {
		if origin.Source != source.Terminal || origin.Pointer != source.TerminalPointer || !deferred && bare.Name != d.Name() && !strings.HasSuffix(bare.Name, "."+d.Name()) {
			continue
		}
		if actualDeclaration != nil || !tupleUnionDeclarationMatches(d, origin.Snapshot) {
			return nil, schemaReferenceUnsupported(pointer)
		}
		actualDeclaration = d
	}
	if actualDeclaration == nil {
		return nil, schemaReferenceUnsupported(pointer)
	}
	switch d := actualDeclaration.(type) {
	case *PrimitiveUnion, *ObjectUnion, *StringEnum, *Hierarchy:
		node.tupleUnion = d
	case *Model:
		// A single non-null object branch plus null retains the original
		// constructor and nullable view, rather than a sealed union wrapper.
		view, err := a.readUnion(source.Terminal, source.TerminalPointer)
		if err != nil {
			return nil, err
		}
		if len(view.Branches) != 1 || !view.HasNull {
			return nil, schemaReferenceUnsupported(pointer)
		}
		branch := view.Branches[0]
		return a.tupleInlineDTOObjectNode(branch.Schema, branch.Pointer, actual, depth+1)
	default:
		return nil, unsupported(pointer)
	}
	return node, nil
}

// Component factories run before the final nominal linker. Only this exact
// actual root declaration may borrow early tuple-default authority; the final
// tuple plan still proves its unique retained occurrence and consumer address.
func (a *analyzer) tupleClosedUnionDefaultAuthorized(source *document.Value, pointer string, union *ObjectUnion) bool {
	r := a.tupleObjects
	if r == nil {
		return false
	}
	proof, ok := r.UnionDeclarations[union]
	if !ok || proof.Source != source || proof.Pointer != pointer || !tupleUnionDeclarationMatches(union, proof.Snapshot) {
		return false
	}
	for _, use := range r.Unions {
		if use.Original != source && use.Terminal == source && use.TerminalPointer == pointer && use.Fact.Pointer == use.Pointer {
			return true
		}
	}
	return false
}

func (a *analyzer) tupleUnionFieldDefault(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
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
	child, err := a.tupleUnionDefaultElement(decision.Selected, decision.Pointer, element, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	return a.tupleElementFieldDefault(v, pointer, element, child, decision)
}

func (a *analyzer) tupleElementFieldDefault(v *document.Value, pointer string, element TypeRef, child *nestedDefaultNode, decision *arrayItemsDecision) (TypeRef, string, *diagnostic.Diagnostic) {
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
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical tuple union default exceeds the supported 32 KiB limit.", at)
	}
	if value.Kind() == document.Null {
		return ty, "null", nil
	}
	if tupleCollectionNominalLeaf(ty) {
		return ty, "`openapi-kt default enum tree`(" + naming.Literal(encoded) + ")", nil
	}
	initializer, err := referencedDefaultInitializer(node, ty, value, encoded, at)
	return ty, initializer, err
}

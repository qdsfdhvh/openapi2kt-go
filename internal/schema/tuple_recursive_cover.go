package schema

import (
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

type tupleCoverSource struct {
	Original, Terminal *document.Value
	Pointer, Target    string
	Root               *component
	Name               string
	Steps              int
	Nullable           bool
}

type tupleCoverRegistry struct {
	Sources    map[*document.Value]*tupleCoverSource
	Order      []*tupleCoverSource
	Uses       map[*TypeRef]tupleCollectionUse
	Selections []tupleInlineAllOfSelection
}

func (a *analyzer) initTupleCovers() {
	if a.tupleCovers == nil {
		a.tupleCovers = &tupleCoverRegistry{Sources: map[*document.Value]*tupleCoverSource{}, Uses: map[*TypeRef]tupleCollectionUse{}}
	}
}

// A complete cover already owns its original graph and JSON codec. Tuple
// admission borrows that authority without replaying it as a nominal union.
func (a *analyzer) tupleRecursiveCover(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	target, err := a.resolveSchema(text(get(original, "$ref")), source)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	if target.Schema != terminal || target.Pointer != pointer || !a.admitBorrowedRecursiveCover(target, source) {
		return tupleScalarSignature{}, schemaReferenceUnsupported(source)
	}
	ty, err := a.referenceType(target, nullableOverride(original), depth, source)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	backing := defaultBacking(ty)
	if !tupleCoverBacking(backing) {
		return tupleScalarSignature{}, schemaReferenceUnsupported(source)
	}
	if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	root := target.CanonicalCover
	// Capture every reachable original named graph node, including graph aliases,
	// into the shared tuple budget. Do not recursively expand a cyclic schema.
	use := a.borrowedRecursiveCoverLeaves[root]
	nodes := []tupleNativeSourcePath{}
	for node, snapshot := range use.Origin.Nodes {
		if c := a.componentsByOriginalSchema(node); c != nil {
			nodes = append(nodes, tupleNativeSourcePath{node, snapshot.Pointer})
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Pointer < nodes[j].Pointer })
	for _, node := range nodes {
		if err := a.captureTupleNativeSource(node.Node, node.Pointer, node.Node, node.Pointer); err != nil {
			return tupleScalarSignature{}, err
		}
	}
	a.initTupleCovers()
	r := a.tupleCovers
	steps := max(1, target.Steps) + use.Origin.Height
	if old := r.Sources[original]; old != nil {
		if old.Terminal != terminal || old.Pointer != source || old.Target != pointer || old.Root != root || old.Name != root.Name || old.Steps != steps || old.Nullable != (backing.Kind == Nullable) {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	} else {
		if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
			return tupleScalarSignature{}, err
		}
		entry := &tupleCoverSource{Original: original, Terminal: terminal, Pointer: source, Target: pointer, Root: root, Name: root.Name, Steps: steps, Nullable: backing.Kind == Nullable}
		r.Sources[original] = entry
		r.Order = append(r.Order, entry)
	}
	return tupleScalarSignature{Kind: "tuple_cover", Nullable: backing.Kind == Nullable, UnionSource: terminal}, nil
}

func tupleCoverBacking(ty TypeRef) bool {
	base := ty.WithoutRootNull()
	return base.Kind == JSON && base.Name == "" && base.Element == nil && base.Serializer == "" && base.Validation == nil && base.Collection == nil && base.deferredNominal == nil && base.ArrayAliasView == OrdinaryAlias
}

func (a *analyzer) tupleCoverElement(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if a.tupleCovers == nil || a.tupleCovers.Sources[v] == nil {
		return nil, nil
	}
	source := a.tupleCovers.Sources[v]
	if source.Pointer != pointer {
		return nil, schemaReferenceUnsupported(pointer)
	}
	target, err := a.resolveSchema(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil, err
	}
	if !a.bindNamedChainRecursiveCover(target, pointer) || target.CanonicalCover != source.Root || target.Schema != source.Terminal || target.Pointer != source.Target || max(1, target.Steps)+a.recursiveJSONHeights[source.Terminal] != source.Steps {
		return nil, schemaReferenceUnsupported(pointer)
	}
	ty, err := a.referenceType(target, nullableOverride(v), depth, pointer)
	if err != nil {
		return nil, err
	}
	ty = defaultBacking(ty)
	if !tupleCoverBacking(ty) || (ty.Kind == Nullable) != source.Nullable {
		return nil, schemaReferenceUnsupported(pointer)
	}
	return &ty, nil
}

func (a *analyzer) tupleCoverWork(element TypeRef, decision *arrayItemsDecision) TypeRef {
	if !a.aliasPlanning || a.tupleCovers == nil || decision == nil {
		return element
	}
	for _, prefix := range decision.Prefix {
		if source := a.tupleCovers.Sources[prefix.Schema]; source != nil {
			element.SchemaSteps = max(element.SchemaSteps, source.Steps)
		}
	}
	return element
}

func (a *analyzer) recordTupleCoverUse(t *TypeRef, source *document.Value) {
	a.tupleCovers.Uses[t] = tupleCollectionUse{Source: source, Pointer: a.tupleCovers.Sources[source].Pointer, Snapshots: dtoOriginalGetterTypes(t), Final: true}
}

func (a *analyzer) selectTupleCover(array *document.Value, pointer string, decision *arrayItemsDecision) {
	if a.tupleCovers == nil || decision.Projection != nil || a.tupleCovers.Sources[decision.Selected] == nil {
		return
	}
	for _, old := range a.tupleCovers.Selections {
		if old.Array == array {
			return
		}
	}
	a.tupleCovers.Selections = append(a.tupleCovers.Selections, tupleInlineAllOfSelection{array, decision.Selected, pointer})
}

func (a *analyzer) validateTupleCovers(plan *Plan, used map[*document.Value]int) *diagnostic.Diagnostic {
	r := a.tupleCovers
	if r == nil {
		return nil
	}
	if err := a.validateBorrowedRecursiveCoverLeaves(plan.Models); err != nil {
		return err
	}
	checked := map[*component]bool{}
	for _, source := range r.Order {
		if r.Sources[source.Original] != source || a.components[source.Root.Wire] != source.Root || source.Root.Schema != source.Terminal || source.Root.Pointer != source.Target || source.Root.Name != source.Name {
			return schemaReferenceUnsupported(source.Pointer)
		}
		if !checked[source.Root] {
			count := 0
			for _, declaration := range plan.Models {
				if declaration.Name() != source.Name {
					continue
				}
				origin, known := a.recursiveCoverOrigins[declaration]
				if !known || origin.Owner != source.Root || origin.Schema != source.Terminal || origin.Pointer != source.Target || !recursiveCoverDeclaration(declaration, origin) {
					return schemaReferenceUnsupported(source.Pointer)
				}
				count++
			}
			if count != 1 {
				return schemaReferenceUnsupported(source.Pointer)
			}
			checked[source.Root] = true
		}
		if _, err := a.tupleCoverElement(source.Original, source.Pointer, 0); err != nil {
			return err
		}
	}
	for _, selection := range r.Selections {
		if r.Sources[selection.Source] == nil || used[selection.Source] == 0 {
			return schemaReferenceUnsupported(selection.Pointer)
		}
	}
	return nil
}

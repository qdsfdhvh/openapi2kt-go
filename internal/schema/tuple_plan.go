package schema

import (
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Check retained tuple classifiers, constructors and element addresses,
// including operation peers. Scratch plans grant no emitted source authority.
func (a *analyzer) validateTuplePlan(plan *Plan) *diagnostic.Diagnostic {
	r := a.tupleBooleans
	hasReferences := a.tupleBooleanRefs != nil && len(a.tupleBooleanRefs.Order) != 0
	hasAllOf := a.tupleObjects != nil && len(a.tupleObjects.AllOf) != 0
	hasCollections := a.tupleObjects != nil && len(a.tupleObjects.Collections) != 0
	hasLiterals := a.tupleLiterals != nil && len(a.tupleLiterals.Order) != 0
	hasUnions := a.tupleObjects != nil && len(a.tupleObjects.Unions) != 0
	hasCovers := a.tupleCovers != nil && len(a.tupleCovers.Order) != 0
	hasSpecials := a.tupleSpecials != nil && len(a.tupleSpecials.Order) != 0
	if (r == nil || len(r.Declarations) == 0) && !hasReferences && !hasAllOf && !hasCollections && !hasLiterals && !hasUnions && !hasCovers && !hasSpecials {
		return nil
	}
	if r == nil {
		a.initTupleBooleans()
		r = a.tupleBooleans
	}
	type symbol struct {
		Declaration Declaration
		Scope       string
		Count       int
	}
	symbols := map[string]symbol{}
	occurrences := map[*FalseValue]int{}
	models := map[*Model]int{}
	type typed struct {
		Type  *TypeRef
		Scope string
	}
	var types []typed
	var visit func(Declaration, string)
	visit = func(d Declaration, scope string) {
		name := scope + "." + d.Name()
		old := symbols[name]
		old.Count++
		old.Declaration = d
		old.Scope = scope
		symbols[name] = old
		switch d := d.(type) {
		case *Model:
			models[d]++
			for i := range d.Properties {
				types = append(types, typed{&d.Properties[i].Type, name})
			}
			for i := range d.Constants {
				types = append(types, typed{&d.Constants[i].Type, name})
			}
			for _, child := range d.Nested {
				visit(child, name)
			}
		case *ValueAlias:
			types = append(types, typed{&d.Type, scope})
		case *ObjectUnion:
			for _, child := range d.Variants {
				visit(child, name)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object, scope)
			}
		case *Hierarchy:
			for i := range d.Fields {
				types = append(types, typed{&d.Fields[i].Type, name})
			}
			if d.Leaf != nil {
				// Hierarchy and Leaf represent one physical classifier. Register
				// its fields and nested declarations without a second symbol.
				models[d.Leaf]++
				for i := range d.Leaf.Properties {
					types = append(types, typed{&d.Leaf.Properties[i].Type, name})
				}
				for i := range d.Leaf.Constants {
					types = append(types, typed{&d.Leaf.Constants[i].Type, name})
				}
				for _, child := range d.Leaf.Nested {
					visit(child, name)
				}
			}
		case *SSEEvents:
			for i := range d.Variants {
				if d.Variants[i].Data != nil {
					types = append(types, typed{d.Variants[i].Data, scope})
				}
			}
		case *FalseValue:
			occurrences[d]++
		}
	}
	for _, d := range plan.Models {
		visit(d, a.options.PackageName+".models")
	}
	if plan.Client != nil {
		for _, d := range plan.Client.Models {
			visit(d, a.options.PackageName+".api.models")
		}
		for gi := range plan.Client.Groups {
			for oi := range plan.Client.Groups[gi].Operations {
				op := &plan.Client.Groups[gi].Operations[oi]
				scope := a.options.PackageName + ".api.models"
				types = append(types, typed{&op.Response, scope})
				for pi := range op.Parameters {
					types = append(types, typed{&op.Parameters[pi].Type, scope})
				}
				if op.Body != nil {
					types = append(types, typed{&op.Body.Type, scope})
				}
				for ei := range op.Errors {
					types = append(types, typed{op.Errors[ei].Type, scope})
				}
			}
		}
	}
	resolve := func(name, scope string) symbol {
		if s, ok := symbols[name]; ok {
			return s
		}
		for strings.HasPrefix(scope, a.options.PackageName+".") {
			if s, ok := symbols[scope+"."+name]; ok {
				return s
			}
			i := strings.LastIndex(scope, ".")
			if i < 0 {
				break
			}
			scope = scope[:i]
		}
		return symbol{}
	}
	active := map[*TypeRef]bool{}
	used := map[*FalseValue]int{}
	referenceUses := map[*tupleBooleanReferenceSource]int{}
	allOfUses := map[*document.Value]int{}
	collectionUses := map[*document.Value]int{}
	literalUses := map[*document.Value]int{}
	unionUses := map[*document.Value]int{}
	coverUses := map[*document.Value]int{}
	specialUses := map[*document.Value]int{}
	var check func(*TypeRef, string) *diagnostic.Diagnostic
	check = func(t *TypeRef, scope string) *diagnostic.Diagnostic {
		if t == nil || active[t] {
			return nil
		}
		active[t] = true
		if hasSpecials {
			if use, ok := a.tupleSpecials.Uses[t]; ok {
				source := a.tupleSpecials.Sources[use.Source]
				if source == nil || !tupleCollectionUseMatches(t, use) || falsePropertySignature(Property{Type: defaultBacking(*t)}) != source.Signature {
					return schemaReferenceUnsupported(use.Pointer)
				}
				specialUses[use.Source]++
			}
		}
		if hasCovers {
			if use, ok := a.tupleCovers.Uses[t]; ok {
				source := a.tupleCovers.Sources[use.Source]
				backing := defaultBacking(*t)
				if source == nil || !tupleCollectionUseMatches(t, use) || !tupleCoverBacking(backing) || (backing.Kind == Nullable) != source.Nullable {
					return schemaReferenceUnsupported(use.Pointer)
				}
				coverUses[use.Source]++
			}
		}
		if hasUnions {
			if use, ok := a.tupleObjects.UnionUses[t]; ok {
				backing := defaultBacking(*t).WithoutRootNull()
				s := resolve(backing.Name, scope)
				if err := a.validateTupleUnionType(t, use, s.Declaration, s.Count); err != nil {
					return err
				}
				unionUses[use.Source]++
			}
		}
		if hasLiterals {
			if use, ok := a.tupleLiterals.Uses[t]; ok {
				if len(use.Snapshots) == 0 || use.Snapshots[0].Node != t {
					return schemaReferenceUnsupported(use.Source.Pointer)
				}
				for _, saved := range use.Snapshots {
					value := *saved.Node
					value.SchemaSteps = 0
					if value != saved.Value || namedCollectionChecks(value) != saved.Checks {
						return schemaReferenceUnsupported(use.Source.Pointer)
					}
				}
				backing := defaultBacking(*t)
				bare := backing.WithoutRootNull()
				if bare.Kind != Scalar || bare.Name != use.Source.Scalar || bare.Validation != nil || bare.Collection != nil || bare.Serializer != "" || (backing.Kind == Nullable) != use.Source.Nullable {
					return schemaReferenceUnsupported(use.Source.Pointer)
				}
				for node := t; node != nil; node = node.Element {
					if node.Kind == Named {
						s := resolve(node.Name, scope)
						alias, scalar := s.Declaration.(*ScalarAlias)
						if !scalar || s.Count != 1 || alias.Scalar != use.Source.Scalar {
							return schemaReferenceUnsupported(use.Source.Pointer)
						}
					}
				}
				literalUses[use.Source.Original]++
			}
		}
		if hasCollections {
			if use, ok := a.tupleObjects.CollectionUses[t]; ok {
				fact, proven := a.tupleCollectionFact(use.Source)
				backing := defaultBacking(*t)
				if !proven || fact.Pointer != use.Pointer || fact.Nullable != (backing.Kind == Nullable) || fact.Kind != backing.WithoutRootNull().Kind || !tupleCollectionUseMatches(t, use) {
					return schemaReferenceUnsupported(use.Pointer)
				}
				if fact.Kind == JSONObject {
					for _, saved := range use.Snapshots {
						if saved.Value.Kind != Named {
							continue
						}
						s := resolve(saved.Value.Name, scope)
						alias, ok := s.Declaration.(*ValueAlias)
						if !ok || s.Count != 1 || defaultBacking(alias.Type).WithoutRootNull().Kind != JSONObject {
							return schemaReferenceUnsupported(use.Pointer)
						}
					}
				}
				collectionUses[use.Source]++
			}
		}
		if hasAllOf {
			if use, ok := a.tupleObjects.Uses[t]; ok {
				bare := t.WithoutRootNull()
				s := resolve(bare.Name, scope)
				model, constructor := s.Declaration.(*Model)
				fact, proven := a.tupleInlineAllOfFact(use.Source)
				if !tupleInlineAllOfUseMatches(t, use) || bare.Kind != Named || bare.Element != nil || !constructor || s.Count != 1 || models[model] != 1 || len(model.Properties) == 0 || a.modelOrigins[model] != use.Source || a.modelPointers[model] != use.Pointer || !proven || fact.Pointer != use.Pointer || fact.Height != a.tupleObjects.Heights[use.Source] || fact.Nullable != (t.Kind == Nullable) || fact.Nullable != model.Nullable {
					return schemaReferenceUnsupported(use.Pointer)
				}
				allOfUses[use.Source]++
				return nil
			}
		}
		s := symbol{}
		if t.Kind == Named {
			s = resolve(t.Name, scope)
		}
		if hasReferences {
			if use, ok := a.tupleBooleanRefs.Uses[t]; ok {
				if !tupleBooleanReferenceUseMatches(t, use) {
					return schemaReferenceUnsupported(use.Source.Pointer)
				}
				referenceUses[use.Source]++
			}
		}
		if source, ok := r.Uses[t]; ok {
			helper, isFalse := s.Declaration.(*FalseValue)
			if !isFalse || s.Count != 1 || occurrences[helper] != 1 || r.Declarations[helper] != source || helper.ClassName != r.Names[helper] || helper.AbsentSerializer || t.Kind != Named || t.Element != nil || t.Serializer != "" || t.Validation != nil || t.Collection != nil || t.ArrayAliasView != OrdinaryAlias {
				return schemaReferenceUnsupported(source.LeafPointer)
			}
			validation := *a
			validation.warnings = nil
			decision, err := validation.arrayItemsOwnDefault(source.Array, source.Pointer, source.Depth, true)
			if err != nil {
				return err
			}
			if decision.Projection != nil || decision.Selected != source.Leaf || decision.Pointer != source.LeafPointer || !a.booleanFalseRoot(source.Leaf) {
				return schemaReferenceUnsupported(source.LeafPointer)
			}
			used[helper]++
		} else if helper, ok := s.Declaration.(*FalseValue); ok {
			if source, owned := r.Declarations[helper]; owned {
				return schemaReferenceUnsupported(source.LeafPointer)
			}
		}
		if t.Kind == Named && t.Element != nil && s.Count == 1 {
			scope = s.Scope
		}
		return check(t.Element, scope)
	}
	for _, t := range types {
		if err := check(t.Type, t.Scope); err != nil {
			return err
		}
	}
	names := make([]string, 0, len(symbols))
	for name := range symbols {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		s := symbols[name]
		if d, ok := s.Declaration.(*FalseValue); ok {
			if source, owned := r.Declarations[d]; owned && (s.Count != 1 || used[d] == 0) {
				return schemaReferenceUnsupported(source.LeafPointer)
			}
		}
	}
	if err := a.validateTupleBooleanReferenceSources(plan.Models, referenceUses); err != nil {
		return err
	}
	if hasAllOf {
		for _, selection := range a.tupleObjects.Selected {
			if allOfUses[selection.Source] == 0 {
				return schemaReferenceUnsupported(selection.Pointer)
			}
		}
	}
	if hasUnions {
		for _, selection := range a.tupleObjects.UnionSelected {
			if unionUses[selection.Source] == 0 {
				return schemaReferenceUnsupported(selection.Pointer)
			}
		}
	}
	if hasCollections {
		for _, selection := range a.tupleObjects.CollectionSelected {
			if collectionUses[selection.Source] == 0 {
				return schemaReferenceUnsupported(selection.Pointer)
			}
		}
	}
	if hasLiterals {
		for _, source := range a.tupleLiterals.Order {
			if a.tupleLiterals.Sources[source.Original] != source {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if source.StructuredKey != "" && a.tupleLiterals.StructuredKeys[source.Terminal] != source.StructuredKey {
				return schemaReferenceUnsupported(source.Pointer)
			}
			for _, binding := range source.Bindings {
				c := binding.Owner
				if a.components[c.Wire] != c || c.Name != binding.Name || c.Scalar != binding.Scalar || c.BindingTarget != binding.BindingTarget || c.ScalarTarget != binding.ScalarTarget || c.AliasHeight != binding.AliasHeight || falseNullableIntent(c) != binding.NullableIntent {
					return schemaReferenceUnsupported(source.Pointer)
				}
				s := symbols[a.options.PackageName+".models."+binding.Name]
				valid := false
				switch alias := s.Declaration.(type) {
				case *ScalarAlias:
					valid = alias.Scalar == source.Scalar
				case *ValueAlias:
					backing := defaultBacking(alias.Type)
					bare := backing.WithoutRootNull()
					valid = bare.Kind == Scalar && bare.Name == source.Scalar && bare.Validation == nil && bare.Serializer == ""
				}
				if !valid || s.Count != 1 {
					return schemaReferenceUnsupported(source.Pointer)
				}
			}
		}
		for _, selection := range a.tupleLiterals.Selections {
			if literalUses[selection.Source] == 0 {
				return schemaReferenceUnsupported(selection.Pointer)
			}
		}
	}
	if err := a.validateTupleHierarchyGraphPlan(plan); err != nil {
		return err
	}
	if err := a.validateTupleCovers(plan, coverUses); err != nil {
		return err
	}
	if err := a.validateTupleSpecials(plan, specialUses); err != nil {
		return err
	}
	return a.validateTupleNativeSources()
}

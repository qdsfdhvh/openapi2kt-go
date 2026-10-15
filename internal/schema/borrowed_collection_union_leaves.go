package schema

import (
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// A borrowed container reuses the original retained union, never owns variants.
// Syntax permits provisional binding; original occurrence/origin grants the API.
type borrowedUnionLeafUse struct {
	Owner                        *component
	OwnerSchema, Schema          *document.Value
	Pointer, Source, Bytes, Name string
	Nodes                        []borrowedAllOfNode
	Refs                         []borrowedAllOfShapeRef
	Actual                       Declaration
	Origin                       nominalOrigin
	NullVariant                  bool
	TagField                     string
	Kinds                        []PrimitiveKind
	Object                       *Model
	TagCombinations              []ObjectTagRule
	Variants                     []*Model
	Tags, Presence               []DispatchRule
	SerialNames                  []string
	Final                        bool
	Product                      bool
}

func (a *analyzer) borrowedUnionCandidate(v *document.Value, product bool) bool {
	if product {
		return composedModelCandidate(v) && a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{})
	}
	return unionNominalCandidate(v) && !has(v, "allOf")
}

func (a *analyzer) admitBorrowedUnion(target *resolvedSchema, source string) bool {
	return a.admitBorrowedUnionRole(target, source, false)
}

func (a *analyzer) admitBorrowedProductUnion(target *resolvedSchema, source string) bool {
	return a.admitBorrowedUnionRole(target, source, true)
}

func (a *analyzer) admitBorrowedUnionRole(target *resolvedSchema, source string, product bool) bool {
	if !a.borrowedUnionCandidate(target.Schema, product) {
		return false
	}
	bound := false
	if target.Catalog != nil {
		bound = a.bindNamedChainModel(target, source)
	} else {
		bound = a.bindInternalModel(target, source)
	}
	if !bound {
		return false
	}
	if a.borrowedUnionLeaves == nil {
		a.borrowedUnionLeaves = map[*document.Value]*borrowedUnionLeafUse{}
	}
	if old := a.borrowedUnionLeaves[target.Schema]; old != nil {
		if old.Product != product {
			return false
		}
		if source < old.Source {
			old.Source = source
		}
		return true
	}
	owner := target.SourceCatalog
	if owner == nil && target.Deferred != nil {
		owner = target.Deferred.Owner
	}
	if owner == nil {
		if binding := a.deferredNominals[target.Schema]; binding != nil {
			owner = binding.Owner
		}
	}
	if owner == nil {
		owner = a.componentsByOriginalSchema(target.Schema)
	}
	if owner == nil {
		return false
	}
	bytes, _ := target.Schema.MarshalJSON()
	use := &borrowedUnionLeafUse{Owner: owner, OwnerSchema: owner.Schema, Schema: target.Schema, Pointer: target.Pointer, Source: source, Bytes: string(bytes), Product: product}
	nodes, refs, err := a.borrowedUnionSource(target.Schema, target.Pointer)
	if err != nil {
		return false
	}
	use.Nodes, use.Refs = nodes, refs
	if a.nominalsLinked {
		use.Name, use.Final = a.physicalDTOName(target.Schema), true
	}
	a.borrowedUnionLeaves[target.Schema] = use
	return true
}

func (a *analyzer) borrowedUnionSource(v *document.Value, pointer string) ([]borrowedAllOfNode, []borrowedAllOfShapeRef, *diagnostic.Diagnostic) {
	nodes := []borrowedAllOfNode{}
	refs := []borrowedAllOfShapeRef{}
	seen := map[*document.Value]bool{}
	var walk func(*document.Value, string, int, string) *diagnostic.Diagnostic
	walk = func(v *document.Value, p string, depth int, role string) *diagnostic.Diagnostic {
		if err := depthCheck(depth, p); err != nil {
			return err
		}
		if v == nil || seen[v] {
			return nil
		}
		if len(seen) >= 16384 {
			return diagnostic.At("E_SCHEMA_LIMIT", "Original union source exceeds the supported work budget.", p)
		}
		seen[v] = true
		nodes = append(nodes, borrowedAllOfNode{v, append([]document.Member{}, v.Members()...), append([]*document.Value{}, v.Items()...)})
		if role == "shape" && has(v, "$ref") {
			reference := text(get(v, "$ref"))
			declared, err := a.resolveDeclared(reference, p)
			if err != nil {
				return err
			}
			if a.borrowedEnumAliasVisit != nil && declared.Reference != "" {
				if err := a.borrowedEnumAliasVisit(declared, depth+1); err != nil {
					return err
				}
			}
			owner := declared
			if declared.BindingTarget != nil {
				owner = declared.BindingTarget
			}
			chain, aliases := declared, map[*component]bool{}
			for {
				if aliases[chain] || len(aliases) >= MaxSchemaDepth {
					return schemaReferenceUnsupported(p)
				}
				aliases[chain] = true
				refs = append(refs, borrowedAllOfShapeRef{chain, chain.Schema, chain.Pointer, chain.Reference, chain.AliasHeight})
				if chain.Reference == "" {
					break
				}
				chain, err = a.resolveDeclared(chain.Reference, p)
				if err != nil {
					return err
				}
			}
			if err := walk(owner.Schema, owner.Pointer, depth+1+declared.AliasHeight, "shape"); err != nil {
				return err
			}
		}
		for _, member := range v.Members() {
			if role != "properties" && slices.Contains(metadata, member.Name) {
				continue
			}
			at := naming.Pointer(p, member.Name)
			childRole := "schema"
			if role != "properties" {
				if member.Name == "oneOf" || member.Name == "anyOf" || member.Name == "allOf" {
					childRole = "branches"
				} else if member.Name == "properties" {
					childRole = "properties"
				}
			}
			if err := walk(member.Value, at, depth, childRole); err != nil {
				return err
			}
		}
		// Branch arrays are visited above with shape authority. Other semantic
		// arrays retain their item identities without becoming shape refs.
		if v.Kind() == document.Array {
			for i, item := range v.Items() {
				childRole, childDepth := "schema", depth
				if role == "branches" {
					childRole, childDepth = "shape", depth+1
				}
				if err := walk(item, naming.Pointer(p, strconv.Itoa(i)), childDepth, childRole); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(v, pointer, 0, "shape"); err != nil {
		return nil, nil, err
	}
	return nodes, refs, nil
}

func (a *analyzer) finalizeBorrowedUnionLeaves() {
	for _, use := range a.borrowedUnionLeaves {
		use.Name, use.Final = a.physicalDTOName(use.Schema), true
	}
}

func (a *analyzer) validateBorrowedUnionLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedUnionLeaves) == 0 {
		return nil
	}
	type occurrence struct {
		Union  Declaration
		Owner  *component
		Name   string
		Origin nominalOrigin
	}
	matches := map[*document.Value][]occurrence{}
	var visit func(Declaration, *component, string)
	visit = func(d Declaration, owner *component, parent string) {
		if origin, ok := a.unionOrigins[d]; ok && owner != nil && (origin.Pointer == owner.Pointer && origin.Schema == owner.Schema || strings.HasPrefix(origin.Pointer, owner.Pointer+"/")) {
			matches[origin.Schema] = append(matches[origin.Schema], occurrence{d, owner, parent + "." + d.Name(), origin})
		}
		switch x := d.(type) {
		case *Model:
			for _, child := range x.Nested {
				visit(child, owner, parent+"."+x.ClassName)
			}
		case *ObjectUnion:
			for _, child := range x.Variants {
				visit(child, owner, parent+"."+x.ClassName)
			}
		case *PrimitiveUnion:
			if x.Object != nil {
				visit(x.Object, owner, parent)
			}
		case *Hierarchy:
			if x.Leaf != nil {
				visit(x.Leaf, owner, parent)
			}
		}
	}
	byName := map[string]*component{}
	for _, c := range a.components {
		byName[c.Name] = c
	}
	for _, d := range ds {
		owner := byName[d.Name()]
		if owner == nil {
			owner = a.declarationOwners[d]
		}
		visit(d, owner, a.options.PackageName+".models")
	}
	uses := []*borrowedUnionLeafUse{}
	for _, use := range a.borrowedUnionLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		owner := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if owner == nil || a.components[owner.Wire] != owner || owner.Schema != use.OwnerSchema || string(bytes) != use.Bytes || !a.borrowedUnionCandidate(use.Schema, use.Product) {
			return schemaReferenceUnsupported(use.Source)
		}
		found, err := a.lookupSchemaReference("#"+use.Pointer, use.Source)
		if err != nil || found.Schema != use.Schema || found.Pointer != use.Pointer || use.Pointer != owner.Pointer && found.SourceCatalog != owner {
			return schemaReferenceUnsupported(use.Source)
		}
		nodes, refs, sourceErr := a.borrowedUnionSource(use.Schema, use.Pointer)
		if sourceErr != nil || len(nodes) != len(use.Nodes) || !slices.Equal(refs, use.Refs) {
			return schemaReferenceUnsupported(use.Source)
		}
		for i, original := range use.Nodes {
			if nodes[i].Node != original.Node || !slices.Equal(original.Node.Members(), original.Members) || !slices.Equal(original.Node.Items(), original.Items) {
				return schemaReferenceUnsupported(use.Source)
			}
			for _, member := range original.Members {
				if get(original.Node, member.Name) != member.Value {
					return schemaReferenceUnsupported(use.Source)
				}
			}
		}
		all := matches[use.Schema]
		if len(all) != 1 || all[0].Owner != owner || all[0].Origin.Pointer != use.Pointer || use.Final && all[0].Name != use.Name || use.Actual != nil && (use.Actual != all[0].Union || use.Origin != all[0].Origin) {
			return schemaReferenceUnsupported(use.Source)
		}
		switch x := all[0].Union.(type) {
		case *ObjectUnion:
			if use.Actual != nil && (x.NullVariant != use.NullVariant || x.TagField != use.TagField || !slices.Equal(x.Variants, use.Variants) || !slices.Equal(x.Tags, use.Tags) || !slices.Equal(x.Presence, use.Presence) || !reflect.DeepEqual(x.SerialNames, use.SerialNames) || !reflect.DeepEqual(x.TagCombinations, use.TagCombinations)) {
				return schemaReferenceUnsupported(use.Source)
			}
			use.NullVariant, use.TagField, use.Variants, use.Tags, use.Presence, use.SerialNames = x.NullVariant, x.TagField, slices.Clone(x.Variants), slices.Clone(x.Tags), slices.Clone(x.Presence), slices.Clone(x.SerialNames)
			use.TagCombinations = slices.Clone(x.TagCombinations)
			for i, rule := range use.TagCombinations {
				use.TagCombinations[i].Matches = slices.Clone(rule.Matches)
				for j, match := range rule.Matches {
					use.TagCombinations[i].Matches[j].Values = slices.Clone(match.Values)
				}
			}
		case *PrimitiveUnion:
			if use.Product {
				return schemaReferenceUnsupported(use.Source)
			}
			if use.Actual != nil && (x.NullVariant != use.NullVariant || x.Object != use.Object || !slices.Equal(x.Kinds, use.Kinds)) {
				return schemaReferenceUnsupported(use.Source)
			}
			use.NullVariant, use.Kinds, use.Object = x.NullVariant, slices.Clone(x.Kinds), x.Object
		default:
			return schemaReferenceUnsupported(use.Source)
		}
		use.Actual, use.Origin = all[0].Union, all[0].Origin
	}
	return nil
}

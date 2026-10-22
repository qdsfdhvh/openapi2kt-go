package schema

import (
	"sort"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Borrowed containers do not own their enum leaf. Its original retained
// vocabulary declaration, lexical catalog and source node authorize the codec.
type borrowedEnumLeafUse struct {
	Owner                           *component
	OwnerSchema, Schema, Vocabulary *document.Value
	Members                         []*document.Value
	SchemaMembers                   []document.Member
	Pointer, Source, Bytes, Name    string
	Actual                          *StringEnum
	Origin                          nominalOrigin
	Final                           bool
}

// Forward nominal-source traversal shares the existing catalog DFS before a
// consumer caches source metadata. Only already admitted terminal profiles
// trigger a visit; traversing DTO fields is not part of alias resolution.
func (a *analyzer) visitBorrowedEnumAlias(reference, source string, depth int) *diagnostic.Diagnostic {
	return a.visitBorrowedSourceAlias(reference, source, depth, false)
}

func (a *analyzer) visitBorrowedSourceAlias(reference, source string, depth int, ordinaryDTO bool) *diagnostic.Diagnostic {
	// Nullable union members also need ordinary forward DTO aliases resolved.
	// Existing borrowed-container callers keep their terminal/diagnostic policy.
	if a.borrowedEnumAliasVisit == nil {
		return nil
	}
	declared, err := a.resolveDeclared(reference, source)
	if err != nil || declared.Reference == "" {
		return nil
	}
	ref, seen := reference, map[*document.Value]bool{}
	for steps := 0; steps < MaxSchemaDepth; steps++ {
		target, err := a.lookupSchemaReference(ref, source)
		if err != nil || seen[target.Schema] {
			return nil
		}
		seen[target.Schema] = true
		kind, _, _ := schemaType(target.Schema, target.Pointer)
		internalNativeEnum := a.options.Validation && target.Catalog == nil && has(target.Schema, "enum") && (kind == "integer" || kind == "number" || kind == "boolean")
		if internalNativeEnum || enumNominalCandidate(target.Schema) || ordinaryDTO && directInternalModel(target.Schema, target.Pointer) || composedModelCandidate(target.Schema) || unionNominalCandidate(target.Schema) || a.hierarchySources[target.Schema] || target.Catalog != nil && (builtinPrimitiveBinding(target.Catalog) || a.options.Validation && nativeEnumBinding(target.Catalog) || temporalBinding(target.Catalog) || bigintBinding(target.Catalog) || bytesBinding(target.Catalog) || callerNativeBinding(target.Catalog) || a.rawTrueCandidate(target.Catalog) || target.Catalog.RecursiveJSON || target.Catalog.ValueClass && target.Catalog.Schema == target.Schema) {
			return a.borrowedEnumAliasVisit(declared, depth)
		}
		if !has(target.Schema, "$ref") {
			return nil
		}
		ref = text(get(target.Schema, "$ref"))
	}
	return nil
}

func (a *analyzer) admitBorrowedNominalLeaf(target *resolvedSchema, source string) bool {
	// A native enum bridge is transparent, but its existing borrowed root
	// origin/actual export still authorizes the linked scalar and checks.
	if a.options.Validation && a.bindNamedBorrowedScalar(target) && target.BorrowedScalar.Vocabulary != nil {
		return true
	}
	if a.bindNamedChainNativeEnum(target, source) {
		return true
	}
	if a.admitBorrowedRecursiveCover(target, source) || a.admitBorrowedRawTrue(target, source) || a.admitBorrowedCallerNative(target, source) || a.admitBorrowedBytes(target, source) || a.admitBorrowedBigint(target, source) || a.admitBorrowedTemporal(target, source) || a.admitBorrowedBuiltin(target, source) || a.admitBorrowedHierarchy(target, source) || a.admitBorrowedOrdinaryDTO(target, source) || a.admitBorrowedValueClass(target, source) || a.admitBorrowedProductUnion(target, source) || a.admitBorrowedAllOfModel(target, source) || a.admitBorrowedUnion(target, source) {
		return true
	}
	if !enumNominalCandidate(target.Schema) {
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
	if a.borrowedEnumLeaves == nil {
		a.borrowedEnumLeaves = map[*document.Value]*borrowedEnumLeafUse{}
	}
	if old := a.borrowedEnumLeaves[target.Schema]; old != nil {
		if source < old.Source {
			old.Source = source
		}
		return true
	}
	owner := target.SourceCatalog
	if owner == nil && target.Deferred != nil {
		owner = target.Deferred.Owner
	}
	if owner == nil && target.Catalog != nil && target.Catalog.InternalModelTarget != nil {
		owner = target.Catalog.InternalModelTarget.SourceCatalog
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
	use := &borrowedEnumLeafUse{Owner: owner, OwnerSchema: owner.Schema, Schema: target.Schema, Pointer: target.Pointer, Source: source, Bytes: string(bytes), Vocabulary: get(target.Schema, "enum"), Members: append([]*document.Value{}, get(target.Schema, "enum").Items()...), SchemaMembers: append([]document.Member{}, target.Schema.Members()...)}
	if a.nominalsLinked {
		use.Name, use.Final = a.physicalDTOName(target.Schema), true
	}
	a.borrowedEnumLeaves[target.Schema] = use
	return true
}

func (a *analyzer) finalizeBorrowedEnumLeaves() {
	for _, use := range a.borrowedEnumLeaves {
		use.Name, use.Final = a.physicalDTOName(use.Schema), true
	}
}

func (a *analyzer) validateBorrowedEnumLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedEnumLeaves) == 0 {
		return nil
	}
	type occurrence struct {
		Enum  *StringEnum
		Owner *component
		Name  string
	}
	matches := map[*document.Value][]occurrence{}
	var visit func(Declaration, *component, string)
	visit = func(d Declaration, owner *component, parent string) {
		switch x := d.(type) {
		case *StringEnum:
			origin, ok := a.enumOrigins[x]
			if ok && owner != nil && (origin.Pointer == owner.Pointer && origin.Schema == owner.Schema || strings.HasPrefix(origin.Pointer, owner.Pointer+"/")) {
				matches[origin.Schema] = append(matches[origin.Schema], occurrence{x, owner, parent + "." + x.ClassName})
			}
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
	uses := []*borrowedEnumLeafUse{}
	for _, use := range a.borrowedEnumLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		owner := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if owner == nil || a.components[owner.Wire] != owner || owner.Schema != use.OwnerSchema || string(bytes) != use.Bytes || !enumNominalCandidate(use.Schema) {
			return schemaReferenceUnsupported(use.Source)
		}
		found, err := a.lookupSchemaReference("#"+use.Pointer, use.Source)
		if err != nil || found.Schema != use.Schema || found.Pointer != use.Pointer || use.Pointer != owner.Pointer && found.SourceCatalog != owner {
			return schemaReferenceUnsupported(use.Source)
		}
		members := use.Vocabulary.Items()
		if get(use.Schema, "enum") != use.Vocabulary || len(members) != len(use.Members) || len(use.Schema.Members()) != len(use.SchemaMembers) {
			return schemaReferenceUnsupported(use.Source)
		}
		for i, member := range members {
			if member != use.Members[i] {
				return schemaReferenceUnsupported(use.Source)
			}
		}
		for i, member := range use.Schema.Members() {
			if member != use.SchemaMembers[i] {
				return schemaReferenceUnsupported(use.Source)
			}
		}
		all := matches[use.Schema]
		if len(all) != 1 || all[0].Owner != owner || a.enumOrigins[all[0].Enum].Pointer != use.Pointer || use.Final && all[0].Name != use.Name || use.Actual != nil && all[0].Enum != use.Actual {
			return schemaReferenceUnsupported(use.Source)
		}
		actual := all[0].Enum
		origin := a.enumOrigins[actual]
		if use.Actual != nil && use.Origin != origin {
			return schemaReferenceUnsupported(use.Source)
		}
		expected, invalid := buildStringEnum(actual.ClassName, use.Schema, use.Pointer)
		if invalid != nil {
			return invalid
		}
		if len(actual.Members) != len(expected.Members) {
			return schemaReferenceUnsupported(use.Source)
		}
		for i, member := range actual.Members {
			if member != expected.Members[i] {
				return schemaReferenceUnsupported(use.Source)
			}
		}
		use.Actual, use.Origin = actual, origin
	}
	return nil
}

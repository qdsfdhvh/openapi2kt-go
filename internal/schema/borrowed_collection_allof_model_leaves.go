package schema

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Original allOf syntax grants only provisional typed admission. A unique
// retained constructor Model under the original lexical owner grants the API.
type borrowedAllOfNode struct {
	Node    *document.Value
	Members []document.Member
	Items   []*document.Value
}
type borrowedAllOfShapeRef struct {
	Owner              *component
	Schema             *document.Value
	Pointer, Reference string
	Height             int
}
type borrowedAllOfLeafUse struct {
	Owner                        *component
	OwnerSchema, Schema          *document.Value
	Pointer, Source, Bytes, Name string
	Nodes                        []borrowedAllOfNode
	Refs                         []borrowedAllOfShapeRef
	Actual                       *Model
	Nullable, Final              bool
}

func (a *analyzer) admitBorrowedAllOfModel(target *resolvedSchema, source string) bool {
	admitted, _ := a.admitBorrowedAllOfModelBudget(target, source, nil)
	return admitted
}

func (a *analyzer) admitBorrowedAllOfModelBudget(target *resolvedSchema, source string, budget *tupleAllOfSourceBudget) (bool, *diagnostic.Diagnostic) {
	if !composedModelCandidate(target.Schema) {
		return false, nil
	}
	bound := false
	if target.Catalog != nil {
		bound = a.bindNamedChainModel(target, source)
	} else {
		bound = a.bindInternalModel(target, source)
	}
	if !bound {
		return false, nil
	}
	if a.borrowedAllOfLeaves == nil {
		a.borrowedAllOfLeaves = map[*document.Value]*borrowedAllOfLeafUse{}
	}
	if old := a.borrowedAllOfLeaves[target.Schema]; old != nil {
		if budget != nil && !budget.Seen[target.Schema] {
			if err := budget.charge(0, len(old.Bytes), source); err != nil {
				return false, err
			}
			for _, node := range old.Nodes {
				if err := budget.charge(1+len(node.Members)+len(node.Items), 0, source); err != nil {
					return false, err
				}
			}
			if err := budget.charge(len(old.Refs), 0, source); err != nil {
				return false, err
			}
			budget.Seen[target.Schema] = true
		}
		if source < old.Source {
			old.Source = source
		}
		return true, nil
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
		return false, nil
	}
	bytes, _ := target.Schema.MarshalJSON()
	if err := budget.charge(0, len(bytes), source); err != nil {
		return false, err
	}
	use := &borrowedAllOfLeafUse{Owner: owner, OwnerSchema: owner.Schema, Schema: target.Schema, Pointer: target.Pointer, Source: source, Bytes: string(bytes)}
	nodes, refs, err := a.borrowedAllOfSourceBudget(target.Schema, target.Pointer, source, budget)
	if err != nil {
		return false, err
	}
	use.Nodes, use.Refs = nodes, refs
	if a.nominalsLinked {
		use.Name, use.Final = a.physicalDTOName(target.Schema), true
	}
	a.borrowedAllOfLeaves[target.Schema] = use
	if budget != nil {
		budget.Seen[target.Schema] = true
	}
	return true, nil
}

// Retain original semantic node identities. Descriptive metadata stays opaque;
// only allOf shape refs follow catalog targets, never nominal property refs.
func (a *analyzer) borrowedAllOfSource(v *document.Value, pointer string) ([]borrowedAllOfNode, []borrowedAllOfShapeRef, *diagnostic.Diagnostic) {
	return a.borrowedAllOfSourceBudget(v, pointer, pointer, nil)
}

func (a *analyzer) borrowedAllOfSourceBudget(v *document.Value, pointer, source string, budget originalSourceCaptureBudget) ([]borrowedAllOfNode, []borrowedAllOfShapeRef, *diagnostic.Diagnostic) {
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
			return diagnostic.At("E_SCHEMA_LIMIT", "Original allOf source exceeds the supported work budget.", p)
		}
		seen[v] = true
		if budget != nil {
			if err := budget.charge(1+len(v.Members())+len(v.Items()), 0, source); err != nil {
				return err
			}
		}
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
				if budget != nil {
					if err := budget.charge(1, 0, source); err != nil {
						return err
					}
				}
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
				if member.Name == "allOf" {
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

func (a *analyzer) finalizeBorrowedAllOfLeaves() {
	for _, use := range a.borrowedAllOfLeaves {
		use.Name, use.Final = a.physicalDTOName(use.Schema), true
	}
}

func (a *analyzer) validateBorrowedAllOfLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedAllOfLeaves) == 0 {
		return nil
	}
	type occurrence struct {
		Model *Model
		Owner *component
		Name  string
	}
	matches := map[*document.Value][]occurrence{}
	var visit func(Declaration, *component, string)
	visit = func(d Declaration, owner *component, parent string) {
		switch x := d.(type) {
		case *Model:
			name := parent + "." + x.ClassName
			v, p := a.modelOrigins[x], a.modelPointers[x]
			if owner != nil && v != nil && len(x.Properties) > 0 && (p == owner.Pointer && v == owner.Schema || strings.HasPrefix(p, owner.Pointer+"/")) {
				matches[v] = append(matches[v], occurrence{x, owner, name})
			}
			for _, child := range x.Nested {
				visit(child, owner, name)
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
	uses := []*borrowedAllOfLeafUse{}
	for _, use := range a.borrowedAllOfLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		owner := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if owner == nil || a.components[owner.Wire] != owner || owner.Schema != use.OwnerSchema || string(bytes) != use.Bytes || !composedModelCandidate(use.Schema) {
			return schemaReferenceUnsupported(use.Source)
		}
		found, err := a.lookupSchemaReference("#"+use.Pointer, use.Source)
		if err != nil || found.Schema != use.Schema || found.Pointer != use.Pointer || use.Pointer != owner.Pointer && found.SourceCatalog != owner {
			return schemaReferenceUnsupported(use.Source)
		}
		nodes, refs, sourceErr := a.borrowedAllOfSource(use.Schema, use.Pointer)
		if sourceErr != nil || len(nodes) != len(use.Nodes) || len(refs) != len(use.Refs) {
			return schemaReferenceUnsupported(use.Source)
		}
		for i, ref := range refs {
			if ref != use.Refs[i] {
				return schemaReferenceUnsupported(use.Source)
			}
		}
		for i, original := range use.Nodes {
			if nodes[i].Node != original.Node {
				return schemaReferenceUnsupported(use.Source)
			}
			members, items := original.Node.Members(), original.Node.Items()
			if len(members) != len(original.Members) || len(items) != len(original.Items) {
				return schemaReferenceUnsupported(use.Source)
			}
			for i, member := range members {
				if member != original.Members[i] || get(original.Node, member.Name) != member.Value {
					return schemaReferenceUnsupported(use.Source)
				}
			}
			for i, item := range items {
				if item != original.Items[i] {
					return schemaReferenceUnsupported(use.Source)
				}
			}
		}
		all := matches[use.Schema]
		if len(all) != 1 || all[0].Owner != owner || a.modelPointers[all[0].Model] != use.Pointer || use.Final && all[0].Name != use.Name || use.Actual != nil && (use.Actual != all[0].Model || use.Nullable != all[0].Model.Nullable) {
			return schemaReferenceUnsupported(use.Source)
		}
		use.Actual, use.Nullable = all[0].Model, all[0].Model.Nullable
	}
	return nil
}

package schema

import (
	"encoding/json"
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Borrowed hierarchy leaves keep actual named classifier and lineage authority.
// A branch's object syntax never grants ordinary DTO or inline owner permission.
type borrowedHierarchyDeclaration struct {
	Actual                   *Hierarchy
	Owner                    *component
	Schema                   *document.Value
	Pointer, Bytes, Semantic string
	Nodes                    []borrowedAllOfNode
	Refs                     []borrowedAllOfShapeRef
	Leaf                     *Model
	Enums                    []*StringEnum
}

type borrowedHierarchyLeafUse struct {
	Owner                        *component
	Schema                       *document.Value
	Pointer, Source, Bytes, Name string
	Nodes                        []borrowedAllOfNode
	Refs                         []borrowedAllOfShapeRef
	Lineage                      []borrowedHierarchyDeclaration
	Final                        bool
}

func (a *analyzer) admitBorrowedHierarchy(target *resolvedSchema, source string) bool {
	c := target.Catalog
	if c == nil || c.Schema != target.Schema || !a.hierarchySources[target.Schema] || c.Reference != "" {
		return false
	}
	if !a.bindNamedChainModel(target, source) {
		return false
	}
	if a.borrowedHierarchyLeaves == nil {
		a.borrowedHierarchyLeaves = map[*document.Value]*borrowedHierarchyLeafUse{}
	}
	if old := a.borrowedHierarchyLeaves[target.Schema]; old != nil {
		if source < old.Source {
			old.Source = source
		}
		return true
	}
	nodes, refs, err := a.borrowedUnionSource(target.Schema, target.Pointer)
	if err != nil {
		return false
	}
	bytes, _ := target.Schema.MarshalJSON()
	use := &borrowedHierarchyLeafUse{Owner: c, Schema: target.Schema, Pointer: target.Pointer, Source: source, Bytes: string(bytes), Nodes: nodes, Refs: refs}
	if a.nominalsLinked {
		use.Name, use.Final = a.physicalDTOName(target.Schema), true
	}
	a.borrowedHierarchyLeaves[target.Schema] = use
	return true
}

func (a *analyzer) finalizeBorrowedHierarchyLeaves() {
	for _, use := range a.borrowedHierarchyLeaves {
		use.Name, use.Final = a.physicalDTOName(use.Schema), true
	}
}

func sameBorrowedHierarchyNodes(nodes, old []borrowedAllOfNode) bool {
	if len(nodes) != len(old) {
		return false
	}
	for i, node := range nodes {
		original := old[i]
		if node.Node != original.Node || !slices.Equal(node.Node.Members(), original.Members) || !slices.Equal(node.Node.Items(), original.Items) {
			return false
		}
		for _, member := range original.Members {
			if get(original.Node, member.Name) != member.Value {
				return false
			}
		}
	}
	return true
}

func (a *analyzer) validateBorrowedHierarchyLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedHierarchyLeaves) == 0 {
		return nil
	}
	actual := map[string][]*Hierarchy{}
	owners := map[string]*component{}
	for _, c := range a.components {
		if a.hierarchySources[c.Schema] && c.Reference == "" {
			owners[c.Name] = c
		}
	}
	for _, d := range ds {
		if h, ok := d.(*Hierarchy); ok {
			actual[h.Name()] = append(actual[h.Name()], h)
		}
	}
	uses := []*borrowedHierarchyLeafUse{}
	for _, use := range a.borrowedHierarchyLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		c := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if c == nil || a.components[c.Wire] != c || c.Schema != use.Schema || c.Pointer != use.Pointer || string(bytes) != use.Bytes || !a.hierarchySources[use.Schema] || c.Reference != "" {
			return schemaReferenceUnsupported(use.Source)
		}
		found, err := a.lookupSchemaReference("#"+use.Pointer, use.Source)
		if err != nil || found.Schema != use.Schema || found.Catalog != c || found.Pointer != use.Pointer {
			return schemaReferenceUnsupported(use.Source)
		}
		nodes, refs, err := a.borrowedUnionSource(use.Schema, use.Pointer)
		if err != nil || !sameBorrowedHierarchyNodes(nodes, use.Nodes) || !slices.Equal(refs, use.Refs) {
			return schemaReferenceUnsupported(use.Source)
		}
		lineage := []borrowedHierarchyDeclaration{}
		name := c.Name
		seen := map[string]bool{}
		for name != "" {
			if seen[name] || len(seen) >= MaxSchemaDepth || len(actual[name]) != 1 {
				return schemaReferenceUnsupported(use.Source)
			}
			seen[name] = true
			h, owner := actual[name][0], owners[name]
			if owner == nil || a.components[owner.Wire] != owner || !a.hierarchySources[owner.Schema] {
				return schemaReferenceUnsupported(use.Source)
			}
			origin, retained := a.hierarchyOrigins[h]
			if !retained || origin.Schema != owner.Schema || origin.Pointer != owner.Pointer || origin.Nullable != owner.NullableHierarchy {
				return schemaReferenceUnsupported(use.Source)
			}
			if len(lineage) == 0 && use.Final && a.options.PackageName+".models."+h.ClassName != use.Name {
				return schemaReferenceUnsupported(use.Source)
			}
			semantic, e := json.Marshal(h)
			if e != nil {
				return schemaReferenceUnsupported(use.Source)
			}
			bytes, _ := owner.Schema.MarshalJSON()
			nodes, refs, err := a.borrowedUnionSource(owner.Schema, owner.Pointer)
			if err != nil {
				return err
			}
			entry := borrowedHierarchyDeclaration{Actual: h, Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer, Bytes: string(bytes), Semantic: string(semantic), Nodes: nodes, Refs: refs, Leaf: h.Leaf, Enums: slices.Clone(h.Enums)}
			if use.Lineage != nil {
				i := len(lineage)
				if i >= len(use.Lineage) {
					return schemaReferenceUnsupported(use.Source)
				}
				old := use.Lineage[i]
				if entry.Actual != old.Actual || entry.Owner != old.Owner || entry.Schema != old.Schema || entry.Pointer != old.Pointer || entry.Bytes != old.Bytes || entry.Semantic != old.Semantic || entry.Leaf != old.Leaf || !slices.Equal(entry.Enums, old.Enums) || !slices.Equal(entry.Refs, old.Refs) || !sameBorrowedHierarchyNodes(entry.Nodes, old.Nodes) {
					return schemaReferenceUnsupported(use.Source)
				}
			}
			lineage = append(lineage, entry)
			if h.Parent == "" {
				break
			}
			prefix := a.options.PackageName + ".models."
			if len(h.Parent) <= len(prefix) || h.Parent[:len(prefix)] != prefix {
				return schemaReferenceUnsupported(use.Source)
			}
			name = h.Parent[len(prefix):]
		}
		if use.Lineage != nil && len(lineage) != len(use.Lineage) {
			return schemaReferenceUnsupported(use.Source)
		}
		use.Lineage = lineage
	}
	return nil
}

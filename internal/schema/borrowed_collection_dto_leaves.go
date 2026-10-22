package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"sort"
	"strings"
)

// Collection admission is provisional; only one original actual constructor
// declaration at the original lexical scope authorizes a nominal leaf.
type borrowedDTOAliasEdge struct {
	Owner                  *component
	Schema                 *document.Value
	Pointer, Bytes, Source string
	Actual                 *ValueAlias
	Height                 int
	Final                  bool
}

func (a *analyzer) recordBorrowedDTOAliasEdges(reference, source string) {
	seen := map[*component]bool{}
	for len(seen) < MaxSchemaDepth && strings.HasPrefix(reference, "#/components/schemas/") && !strings.Contains(strings.TrimPrefix(reference, "#/components/schemas/"), "/") {
		c, err := a.resolveDeclared(reference, source)
		if err != nil || c.Reference == "" || seen[c] {
			return
		}
		seen[c] = true
		if a.borrowedDTOAliasEdges == nil {
			a.borrowedDTOAliasEdges = map[*component]*borrowedDTOAliasEdge{}
		}
		if a.borrowedDTOAliasEdges[c] == nil {
			bytes, _ := c.Schema.MarshalJSON()
			a.borrowedDTOAliasEdges[c] = &borrowedDTOAliasEdge{Owner: c, Schema: c.Schema, Pointer: c.Pointer, Bytes: string(bytes), Source: source}
		}
		reference = c.Reference
	}
}

func (a *analyzer) finalizeBorrowedDTOAliasEdges() {
	for _, edge := range a.borrowedDTOAliasEdges {
		edge.Height = edge.Owner.AliasHeight
		edge.Final = true
	}
}

type borrowedDTOLeafUse struct {
	Schema                       *document.Value
	Owner                        *component
	OwnerSchema                  *document.Value
	Pointer, Source, Name, Bytes string
	Actual                       *Model
}

func (a *analyzer) recordBorrowedDTOLeaf(target *resolvedSchema, source string) {
	if a.borrowedDTOLeaves == nil {
		a.borrowedDTOLeaves = map[*document.Value]*borrowedDTOLeafUse{}
	}
	if old := a.borrowedDTOLeaves[target.Schema]; old != nil {
		if source < old.Source {
			old.Source = source
		}
		return
	}
	owner := target.SourceCatalog
	if owner == nil {
		owner = target.Catalog
	}
	if owner != nil && owner.Schema != target.Schema {
		if named := a.componentsByOriginalSchema(target.Schema); named != nil {
			owner = named
		}
	}
	bytes, _ := target.Schema.MarshalJSON()
	name := a.physicalDTOName(target.Schema)
	a.borrowedDTOLeaves[target.Schema] = &borrowedDTOLeafUse{Schema: target.Schema, Owner: owner, OwnerSchema: owner.Schema, Pointer: target.Pointer, Source: source, Name: name, Bytes: string(bytes)}
}
func (a *analyzer) componentsByOriginalSchema(v *document.Value) *component {
	for _, c := range a.components {
		if c.Schema == v {
			return c
		}
	}
	return nil
}
func (a *analyzer) validateBorrowedDTOLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedDTOAliasEdges) == 0 && len(a.borrowedDTOLeaves) == 0 {
		return nil
	}
	edges := []*borrowedDTOAliasEdge{}
	for _, edge := range a.borrowedDTOAliasEdges {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Source < edges[j].Source || edges[i].Source == edges[j].Source && edges[i].Pointer < edges[j].Pointer
	})
	for _, edge := range edges {
		c := edge.Owner
		bytes, _ := c.Schema.MarshalJSON()
		if a.components[c.Wire] != c || c.Schema != edge.Schema || c.Pointer != edge.Pointer || string(bytes) != edge.Bytes || edge.Final && c.AliasHeight != edge.Height {
			return schemaReferenceUnsupported(edge.Source)
		}
		var actual *ValueAlias
		count := 0
		for _, d := range ds {
			if alias, ok := d.(*ValueAlias); ok && alias.ClassName == c.Name {
				actual = alias
				count++
			}
		}
		if count != 1 || edge.Actual != nil && edge.Actual != actual {
			return schemaReferenceUnsupported(edge.Source)
		}
		edge.Actual = actual
	}
	if len(a.borrowedDTOLeaves) == 0 {
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
			v := a.modelOrigins[x]
			p := a.modelPointers[x]
			if v != nil && len(x.Properties) > 0 && owner != nil && (p == owner.Pointer && owner.Schema == v || strings.HasPrefix(p, owner.Pointer+"/")) {
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
	uses := []*borrowedDTOLeafUse{}
	for _, use := range a.borrowedDTOLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		bytes, _ := use.Schema.MarshalJSON()
		if use.Owner == nil || a.components[use.Owner.Wire] != use.Owner || use.Owner.Schema != use.OwnerSchema || !directInternalModel(use.Schema, use.Pointer) || string(bytes) != use.Bytes {
			return schemaReferenceUnsupported(use.Source)
		}
		if use.Pointer == use.Owner.Pointer {
			if use.Owner.Schema != use.Schema {
				return schemaReferenceUnsupported(use.Source)
			}
		} else {
			found, err := a.lookupSchemaReference("#"+use.Pointer, use.Source)
			if err != nil || found.Schema != use.Schema || found.SourceCatalog != use.Owner {
				return schemaReferenceUnsupported(use.Source)
			}
		}
		all := matches[use.Schema]
		if len(all) != 1 || all[0].Owner != use.Owner || a.modelPointers[all[0].Model] != use.Pointer {
			return schemaReferenceUnsupported(use.Source)
		}
		if use.Name == "" {
			// A composed lexical owner may be built after its first union use.
			// Only that original deferred source and one retained declaration
			// can supply the physical name; borrowed variant copies cannot.
			binding := a.deferredNominals[use.Schema]
			if binding == nil || binding.Schema != use.Schema || binding.Owner != use.Owner || binding.Pointer != use.Pointer {
				return schemaReferenceUnsupported(use.Source)
			}
			use.Name = all[0].Name
		}
		if all[0].Name != use.Name {
			return schemaReferenceUnsupported(use.Source)
		}
		if use.Actual != nil && use.Actual != all[0].Model {
			return schemaReferenceUnsupported(use.Source)
		}
		use.Actual = all[0].Model
	}
	return nil
}

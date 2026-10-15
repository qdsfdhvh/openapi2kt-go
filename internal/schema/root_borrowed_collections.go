package schema

import (
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// A borrowed tree has an original lexical owner and no catalog or classifier
// identity of its own. Root exports share its check objects and finite backing.
type borrowedCollectionNode struct {
	Node                      *document.Value
	Reference, Pointer, Bytes string
}
type borrowedCollectionSource struct {
	Owner               *component
	OwnerSchema, Schema *document.Value
	Pointer, Reference  string
	Type                TypeRef
	Graph               []borrowedCollectionNode
	OrderedParents      []borrowedCollectionParent
	Types               []namedCollectionSourceType
}
type borrowedCollectionParent struct {
	Parent  *document.Value
	Members []document.Member
	Items   []*document.Value
}

type borrowedCollectionRoot struct {
	Root                          *component
	Schema                        *document.Value
	Pointer                       string
	Source                        *borrowedCollectionSource
	AliasHeight, CollectionHeight int
	Nullable                      bool
}

func (a *analyzer) bindBorrowedRootCollection(target *resolvedSchema, source string) *diagnostic.Diagnostic {
	if target.BorrowedCollection != nil || target.Catalog != nil || target.SourceCatalog == nil || target.ModelName != "" || target.Deferred != nil {
		return nil
	}
	kind, _, err := schemaType(target.Schema, target.Pointer)
	if err != nil {
		return err
	}
	if kind != "array" && kind != "object" {
		return nil
	}
	if err := a.admitInternalSchema(target.Schema, target.Pointer, source); err != nil {
		return err
	}
	if old := a.borrowedCollections[target.Schema]; old != nil {
		target.BorrowedCollection = old
		return nil
	}
	leaf := *target
	leaf.Nullable = nil
	leaf.Steps = 0
	planning := a.aliasPlanning
	a.aliasPlanning = true
	ty, err := a.referenceType(&leaf, nil, 0, source)
	a.aliasPlanning = planning
	if err != nil {
		return err
	}
	if bare(ty).Kind != List && bare(ty).Kind != Map && bare(ty).Kind != JSONObject {
		return schemaReferenceUnsupported(source)
	}
	// The root/consumer owns its incoming edge; only original nested source
	// work belongs to this backing. Nullable wrapping must not reset it.
	ty.SchemaSteps--
	origin := &borrowedCollectionSource{Owner: target.SourceCatalog, OwnerSchema: target.SourceCatalog.Schema, Schema: target.Schema, Pointer: target.Pointer, Reference: "#" + target.Pointer, Type: ty}
	graph, err := a.borrowedCollectionGraph(origin, source)
	if err != nil {
		return err
	}
	origin.Graph = graph
	// Ordered source views and keyed lookup are independent document surfaces;
	// retain both so identical-byte replacements cannot bypass identity checks.
	seenParents := map[*document.Value]bool{}
	for _, node := range graph {
		sourceNode, e := a.lookupSchemaReference(node.Reference, source)
		if e != nil {
			return schemaReferenceUnsupported(source)
		}
		owner := sourceNode.SourceCatalog
		if owner == nil {
			owner = sourceNode.Catalog
		}
		if owner == nil {
			return schemaReferenceUnsupported(source)
		}
		v := owner.Schema
		segments := strings.Split(strings.TrimPrefix(node.Pointer, owner.Pointer+"/"), "/")
		if node.Pointer == owner.Pointer {
			continue
		}
		for _, segment := range segments {
			if !seenParents[v] {
				seenParents[v] = true
				origin.OrderedParents = append(origin.OrderedParents, borrowedCollectionParent{v, append([]document.Member{}, v.Members()...), append([]*document.Value{}, v.Items()...)})
			}
			key, e := naming.DecodePointerSegment(segment, source)
			if e != nil {
				return e
			}
			if v.Kind() == document.Array {
				index, e := strconv.Atoi(key)
				if e != nil || index < 0 || index >= len(v.Items()) {
					return schemaReferenceUnsupported(source)
				}
				v = v.Items()[index]
			} else {
				v = get(v, key)
			}
		}
	}
	a.borrowedCollections[target.Schema] = origin
	target.BorrowedCollection = origin
	return nil
}
func (a *analyzer) bindNamedBorrowedCollection(target *resolvedSchema) bool {
	return target.Catalog != nil && target.BorrowedCollection != nil && target.Schema == target.BorrowedCollection.Schema && target.Pointer == target.BorrowedCollection.Pointer
}
func (a *analyzer) borrowedCollectionGraph(origin *borrowedCollectionSource, source string) ([]borrowedCollectionNode, *diagnostic.Diagnostic) {
	var nodes []borrowedCollectionNode
	active := map[*document.Value]bool{}
	var visit func(*document.Value, string, int) *diagnostic.Diagnostic
	visit = func(v *document.Value, p string, depth int) *diagnostic.Diagnostic {
		if err := depthCheck(depth, source); err != nil {
			return err
		}
		if active[v] {
			return schemaReferenceUnsupported(source)
		}
		active[v] = true
		defer delete(active, v)
		reference := "#" + p
		found, err := a.lookupSchemaReference(reference, source)
		if err != nil || found.Schema != v || found.Pointer != p {
			return schemaReferenceUnsupported(source)
		}
		nominal := a.admitBorrowedNominalLeaf(found, source)
		if found.Catalog != nil && !nominal {
			return schemaReferenceUnsupported(source)
		}
		bytes, _ := v.MarshalJSON()
		nodes = append(nodes, borrowedCollectionNode{v, reference, p, string(bytes)})
		if nominal {
			return nil
		}
		if has(v, "$ref") {
			next, err := a.lookupSchemaReference(text(get(v, "$ref")), p+"/$ref")
			if err != nil {
				return err
			}
			if next.Catalog != nil && !a.admitBorrowedNominalLeaf(next, p+"/$ref") {
				return schemaReferenceUnsupported(p + "/$ref")
			}
			return visit(next.Schema, next.Pointer, depth+max(1, next.Steps))
		}
		kind, _, err := schemaType(v, p)
		if err != nil {
			return err
		}
		if kind == "array" {
			return visit(get(v, "items"), naming.Pointer(p, "items"), depth+1)
		}
		if kind == "object" {
			return visit(get(v, "additionalProperties"), naming.Pointer(p, "additionalProperties"), depth+1)
		}
		return nil
	}
	if err := visit(origin.Schema, origin.Pointer, 0); err != nil {
		return nil, err
	}
	return nodes, nil
}
func (a *analyzer) recordBorrowedCollectionRoot(d Declaration, c *component) {
	if c.InternalModelTarget == nil || c.InternalModelTarget.BorrowedCollection == nil {
		return
	}
	a.borrowedCollectionRoots[d] = borrowedCollectionRoot{Root: c, Schema: c.Schema, Pointer: c.Pointer, Source: c.InternalModelTarget.BorrowedCollection}
}
func (a *analyzer) finalizeBorrowedCollectionSources() {
	for d, origin := range a.borrowedCollectionRoots {
		origin.AliasHeight, origin.CollectionHeight = origin.Root.AliasHeight, origin.Root.CollectionHeight
		origin.Nullable = origin.Root.BindingNullable != nil && *origin.Root.BindingNullable
		a.borrowedCollectionRoots[d] = origin
	}
	for _, origin := range a.borrowedCollections {
		seen := map[*TypeRef]bool{}
		var record func(*TypeRef)
		record = func(node *TypeRef) {
			if node == nil || seen[node] {
				return
			}
			seen[node] = true
			value := *node
			value.SchemaSteps = 0
			origin.Types = append(origin.Types, namedCollectionSourceType{node, value, namedCollectionChecks(value)})
			record(node.Element)
		}
		record(&origin.Type)
	}
}
func (a *analyzer) validateBorrowedCollectionBindings(ds []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	roots := []*component{}
	for _, c := range a.components {
		if c.InternalModelTarget != nil && c.InternalModelTarget.BorrowedCollection != nil {
			roots = append(roots, c)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Pointer < roots[j].Pointer })
	for _, c := range roots {
		origin := c.InternalModelTarget.BorrowedCollection
		source := c.Pointer + "/$ref"
		found, err := a.lookupSchemaReference(origin.Reference, source)
		if err != nil || found.Catalog != nil || found.Schema != origin.Schema || found.Pointer != origin.Pointer || found.SourceCatalog != origin.Owner || a.components[origin.Owner.Wire] != origin.Owner || origin.Owner.Schema != origin.OwnerSchema {
			return schemaReferenceUnsupported(source)
		}
		if err := a.admitInternalSchema(origin.Schema, origin.Pointer, source); err != nil {
			return err
		}
		graph, err := a.borrowedCollectionGraph(origin, source)
		if err != nil {
			return err
		}
		if len(graph) != len(origin.Graph) {
			return schemaReferenceUnsupported(source)
		}
		for i, node := range graph {
			if node != origin.Graph[i] {
				return schemaReferenceUnsupported(source)
			}
		}
		for _, parent := range origin.OrderedParents {
			members := parent.Parent.Members()
			if len(members) != len(parent.Members) {
				return schemaReferenceUnsupported(source)
			}
			for i, member := range members {
				if member != parent.Members[i] {
					return schemaReferenceUnsupported(source)
				}
			}
			items := parent.Parent.Items()
			if len(items) != len(parent.Items) {
				return schemaReferenceUnsupported(source)
			}
			for i, item := range items {
				if item != parent.Items[i] {
					return schemaReferenceUnsupported(source)
				}
			}
		}
		for _, node := range origin.Types {
			value := *node.Node
			value.SchemaSteps = 0
			if value != node.Value || namedCollectionChecks(value) != node.Checks {
				return schemaReferenceUnsupported(source)
			}
		}
		matches := 0
		for d, recorded := range a.borrowedCollectionRoots {
			alias, ok := d.(*ValueAlias)
			if !ok || actual[d] == 0 || recorded.Root != c || recorded.Schema != c.Schema || recorded.Pointer != c.Pointer || recorded.Source != origin || alias.ClassName != c.Name || recorded.AliasHeight != c.AliasHeight || recorded.CollectionHeight != c.CollectionHeight || recorded.Nullable != (c.BindingNullable != nil && *c.BindingNullable) {
				continue
			}
			observed, expected := bare(alias.Type), bare(a.componentType(c))
			observed.SchemaSteps, expected.SchemaSteps = 0, 0
			if alias.Type.Kind == a.componentType(c).Kind && observed == expected {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(source)
		}
	}
	return nil
}

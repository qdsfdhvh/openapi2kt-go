package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"slices"
	"sort"
	"strconv"

	"openapi-kt/internal/naming"
)

// The original complete graph, not JSON spelling or just the entry root,
// grants cover authority. Incoming reference hops remain a separate segment.
type recursiveCoverSourceNode struct {
	Pointer, SourceBytes string
	Members              []document.Member
	Items                []*document.Value
}
type recursiveCoverOrigin struct {
	Owner   *component
	Schema  *document.Value
	Pointer string
	Height  int
	Nodes   map[*document.Value]recursiveCoverSourceNode
}

func (a *analyzer) recursiveCoverSnapshot(c *component) (map[*document.Value]recursiveCoverSourceNode, int, bool) {
	if c == nil || !c.RecursiveJSON || c.Reference != "" || a.components[c.Wire] != c || get(get(get(a.root, "components"), "schemas"), c.Wire) != c.Schema {
		return nil, 0, false
	}
	// Do not let an old successful height cache hide a changed reachable source.
	fresh := analyzer{root: a.root}
	height, err := fresh.recursiveJSONRoot(c.Schema, c.Pointer)
	if err != nil || height == 0 {
		return nil, 0, false
	}
	nodes := map[*document.Value]recursiveCoverSourceNode{}
	valid := true
	record := func(node *document.Value, pointer string) {
		bytes, _ := node.MarshalJSON()
		members := append([]document.Member{}, node.Members()...)
		for _, member := range members {
			valid = valid && get(node, member.Name) == member.Value
		}
		nodes[node] = recursiveCoverSourceNode{pointer, string(bytes), members, append([]*document.Value{}, node.Items()...)}
	}
	queue := []jsonCoverEdge{{c.Schema, c.Pointer}}
	for len(queue) > 0 {
		edge := queue[0]
		queue = queue[1:]
		if _, ok := nodes[edge.schema]; ok {
			continue
		}
		weight, edges, err := fresh.recursiveJSONEdges(edge.schema, edge.pointer, 0)
		if err != nil || weight == 0 {
			return nil, 0, false
		}
		record(edge.schema, edge.pointer)
		if weight == 2 {
			key := "oneOf"
			if !has(edge.schema, key) {
				key = "anyOf"
			}
			branches := get(edge.schema, key)
			branchPointer := naming.Pointer(edge.pointer, key)
			record(branches, branchPointer)
			for i, branch := range branches.Items() {
				p := naming.Pointer(branchPointer, strconv.Itoa(i))
				record(branch, p)
				record(get(branch, "type"), naming.Pointer(p, "type"))
				for _, key := range []string{"items", "additionalProperties"} {
					if child := get(branch, key); child != nil {
						record(child, naming.Pointer(p, key))
						record(get(child, "$ref"), naming.Pointer(naming.Pointer(p, key), "$ref"))
						if intent := get(child, "nullable"); intent != nil {
							record(intent, naming.Pointer(naming.Pointer(p, key), "nullable"))
						}
					}
				}
			}
		} else {
			record(get(edge.schema, "$ref"), naming.Pointer(edge.pointer, "$ref"))
			if intent := get(edge.schema, "nullable"); intent != nil {
				record(intent, naming.Pointer(edge.pointer, "nullable"))
			}
		}
		queue = append(queue, edges...)
	}
	return nodes, height, valid
}

func (a *analyzer) recordRecursiveCoverOrigin(d Declaration, c *component) {
	nodes, height, ok := a.recursiveCoverSnapshot(c)
	if !ok {
		return
	}
	a.recursiveCoverOrigins[d] = recursiveCoverOrigin{c, c.Schema, c.Pointer, height, nodes}
}
func (a *analyzer) recursiveCoverSourceMatches(origin recursiveCoverOrigin) bool {
	if origin.Owner.Schema != origin.Schema || origin.Owner.Pointer != origin.Pointer || a.recursiveJSONHeights[origin.Schema] != origin.Height || origin.Owner.AliasType.Kind != JSON {
		return false
	}
	nodes, height, ok := a.recursiveCoverSnapshot(origin.Owner)
	if !ok || height != origin.Height || len(nodes) != len(origin.Nodes) {
		return false
	}
	for node, source := range origin.Nodes {
		if current, ok := nodes[node]; !ok || current.Pointer != source.Pointer || current.SourceBytes != source.SourceBytes || !slices.Equal(current.Members, source.Members) || !slices.Equal(current.Items, source.Items) {
			return false
		}
	}
	return true
}
func recursiveCoverDeclaration(d Declaration, origin recursiveCoverOrigin) bool {
	alias, ok := d.(*ValueAlias)
	return ok && d.Name() == origin.Owner.Name && alias.Type.Kind == JSON && alias.Type.Name == "" && alias.Type.Element == nil && alias.Type.Serializer == "" && alias.Type.Validation == nil && alias.Type.Collection == nil && alias.Type.deferredNominal == nil && alias.Type.ArrayAliasView == OrdinaryAlias && (alias.Type.SchemaSteps == 0 || alias.Type.SchemaSteps == origin.Height)
}
func (a *analyzer) bindNamedChainRecursiveCover(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	c := target.CanonicalCover
	if c == nil {
		c = target.Catalog
		if c.BindingTarget != nil {
			c = c.BindingTarget
		}
	}
	if !c.RecursiveJSON || target.Schema != c.Schema || target.Pointer != c.Pointer {
		return false
	}
	if a.nominalsLinked {
		found := false
		for d, origin := range a.recursiveCoverOrigins {
			if origin.Owner == c && a.recursiveCoverSourceMatches(origin) && recursiveCoverDeclaration(d, origin) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	target.CanonicalCover = c
	if previous, ok := a.recursiveCoverUses[c]; !ok || source < previous {
		a.recursiveCoverUses[c] = source
	}
	return true
}
func (a *analyzer) validateRecursiveCoverBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	used := []*component{}
	for c := range a.recursiveCoverUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.recursiveCoverUses[used[i]] < a.recursiveCoverUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.recursiveCoverOrigins {
			if actual[d] != 0 && origin.Owner == c && a.recursiveCoverSourceMatches(origin) && recursiveCoverDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.recursiveCoverUses[c])
		}
	}
	return nil
}

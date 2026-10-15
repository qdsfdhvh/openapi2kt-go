package schema

import (
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
)

// Borrowed containers retain the original complete graph and actual JSON export.
// Incoming reference work remains separate from the cover's SCC suffix height.
type borrowedRecursiveCoverLeafUse struct {
	Origin recursiveCoverOrigin
	Source string
	Nodes  []borrowedAllOfNode
}

func (a *analyzer) admitBorrowedRecursiveCover(target *resolvedSchema, source string) bool {
	if !a.bindNamedChainRecursiveCover(target, source) {
		return false
	}
	c := target.CanonicalCover
	if a.borrowedRecursiveCoverLeaves == nil {
		a.borrowedRecursiveCoverLeaves = map[*component]*borrowedRecursiveCoverLeafUse{}
	}
	if old := a.borrowedRecursiveCoverLeaves[c]; old != nil {
		if source < old.Source {
			old.Source = source
		}
		return true
	}
	nodes, height, ok := a.recursiveCoverSnapshot(c)
	if !ok {
		return false
	}
	use := &borrowedRecursiveCoverLeafUse{Origin: recursiveCoverOrigin{c, c.Schema, c.Pointer, height, nodes}, Source: source}
	for node := range nodes {
		use.Nodes = append(use.Nodes, borrowedAllOfNode{node, slices.Clone(node.Members()), slices.Clone(node.Items())})
	}
	a.borrowedRecursiveCoverLeaves[c] = use
	return true
}

func (a *analyzer) validateBorrowedRecursiveCoverLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedRecursiveCoverLeaves) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	uses := []*borrowedRecursiveCoverLeafUse{}
	for _, use := range a.borrowedRecursiveCoverLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		if !a.recursiveCoverSourceMatches(use.Origin) {
			return schemaReferenceUnsupported(use.Source)
		}
		for _, node := range use.Nodes {
			if !slices.Equal(node.Node.Members(), node.Members) || !slices.Equal(node.Node.Items(), node.Items) {
				return schemaReferenceUnsupported(use.Source)
			}
			for _, member := range node.Members {
				if get(node.Node, member.Name) != member.Value {
					return schemaReferenceUnsupported(use.Source)
				}
			}
		}
		matches := 0
		for d, origin := range a.recursiveCoverOrigins {
			if actual[d] > 0 && origin.Owner == use.Origin.Owner && origin.Schema == use.Origin.Schema && origin.Pointer == use.Origin.Pointer && origin.Height == use.Origin.Height && a.recursiveCoverSourceMatches(origin) && recursiveCoverDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(use.Source)
		}
	}
	return nil
}

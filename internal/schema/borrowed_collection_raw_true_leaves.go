package schema

import (
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// A transparent named raw/true leaf retains its original JSON source role, no nominal
// classifier barrier. The actual empty Model or literal-true ValueAlias, never type spelling, authorizes its codec.
type borrowedRawTrueLeafUse struct {
	Owner                          *component
	Schema                         *document.Value
	Pointer, Source, Bytes, Scalar string
	Nodes                          []borrowedAllOfNode
}

func (a *analyzer) admitBorrowedRawTrue(target *resolvedSchema, source string) bool {
	if !a.bindNamedChainRawTrue(target, source) {
		return false
	}
	owner := target.CanonicalJSON
	if owner == nil {
		return false
	}
	if a.borrowedRawTrueLeaves == nil {
		a.borrowedRawTrueLeaves = map[*component]*borrowedRawTrueLeafUse{}
	}
	if old := a.borrowedRawTrueLeaves[owner]; old != nil {
		if source < old.Source {
			old.Source = source
		}
		return true
	}
	nodes, _, err := a.borrowedAllOfSource(owner.Schema, owner.Pointer)
	if err != nil {
		return false
	}
	bytes, _ := owner.Schema.MarshalJSON()
	use := &borrowedRawTrueLeafUse{Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer, Source: source, Bytes: string(bytes), Scalar: owner.Scalar, Nodes: nodes}
	a.borrowedRawTrueLeaves[owner] = use
	return true
}

func (a *analyzer) validateBorrowedRawTrueLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedRawTrueLeaves) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	uses := []*borrowedRawTrueLeafUse{}
	for _, use := range a.borrowedRawTrueLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		c := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if c == nil || a.components[c.Wire] != c || c.Schema != use.Schema || c.Pointer != use.Pointer || c.Scalar != use.Scalar || string(bytes) != use.Bytes || !a.rawTrueCandidate(c) {
			return schemaReferenceUnsupported(use.Source)
		}
		nodes, _, err := a.borrowedAllOfSource(use.Schema, use.Pointer)
		if err != nil || len(nodes) != len(use.Nodes) {
			return schemaReferenceUnsupported(use.Source)
		}
		for i, node := range nodes {
			old := use.Nodes[i]
			if node.Node != old.Node || !slices.Equal(node.Node.Members(), old.Members) || !slices.Equal(node.Node.Items(), old.Items) {
				return schemaReferenceUnsupported(use.Source)
			}
			for _, member := range old.Members {
				if get(old.Node, member.Name) != member.Value {
					return schemaReferenceUnsupported(use.Source)
				}
			}
		}
		matches := 0
		for d, origin := range a.rawTrueOrigins {
			if actual[d] > 0 && origin.Owner == c && origin.Schema == use.Schema && origin.Pointer == use.Pointer && a.rawTrueSourceMatches(origin) && a.rawTrueDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(use.Source)
		}
	}
	return nil
}

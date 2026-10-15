package schema

import (
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// A transparent named temporal leaf has source/check authority, no nominal
// classifier barrier. The actual original scalar export authorizes its codec.
type borrowedTemporalLeafUse struct {
	Owner                          *component
	Schema                         *document.Value
	Pointer, Source, Bytes, Scalar string
	Nodes                          []borrowedAllOfNode
}

func (a *analyzer) admitBorrowedTemporal(target *resolvedSchema, source string) bool {
	if !a.bindNamedChainTemporal(target, source) {
		return false
	}
	owner := target.CanonicalTemporal
	if owner == nil {
		return false
	}
	if a.borrowedTemporalLeaves == nil {
		a.borrowedTemporalLeaves = map[*component]*borrowedTemporalLeafUse{}
	}
	if old := a.borrowedTemporalLeaves[owner]; old != nil {
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
	use := &borrowedTemporalLeafUse{Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer, Source: source, Bytes: string(bytes), Scalar: owner.Scalar, Nodes: nodes}
	a.borrowedTemporalLeaves[owner] = use
	return true
}

func (a *analyzer) validateBorrowedTemporalLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedTemporalLeaves) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	uses := []*borrowedTemporalLeafUse{}
	for _, use := range a.borrowedTemporalLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		c := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if c == nil || a.components[c.Wire] != c || c.Schema != use.Schema || c.Pointer != use.Pointer || c.Scalar != use.Scalar || string(bytes) != use.Bytes || !temporalBinding(c) {
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
		for d, origin := range a.temporalOrigins {
			if actual[d] > 0 && origin.Owner == c && origin.Schema == use.Schema && origin.Pointer == use.Pointer && origin.Scalar == use.Scalar && temporalDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(use.Source)
		}
	}
	return nil
}

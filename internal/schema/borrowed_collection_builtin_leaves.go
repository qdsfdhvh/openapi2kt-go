package schema

import (
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// A transparent named builtin leaf has source/check authority, no nominal
// classifier barrier. The actual original scalar export authorizes its codec.
type borrowedBuiltinLeafUse struct {
	Owner                          *component
	Schema                         *document.Value
	Pointer, Source, Bytes, Scalar string
	Nodes                          []borrowedAllOfNode
	Validation                     *PrimitiveValidation
	Checks                         string
}

func (a *analyzer) admitBorrowedBuiltin(target *resolvedSchema, source string) bool {
	if !a.bindNamedChainBuiltin(target, source) {
		return false
	}
	owner := target.CanonicalBuiltin
	if owner == nil {
		return false
	}
	if a.borrowedBuiltinLeaves == nil {
		a.borrowedBuiltinLeaves = map[*component]*borrowedBuiltinLeafUse{}
	}
	if old := a.borrowedBuiltinLeaves[owner]; old != nil {
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
	use := &borrowedBuiltinLeafUse{Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer, Source: source, Bytes: string(bytes), Scalar: owner.Scalar, Nodes: nodes, Validation: owner.Validation}
	use.Checks = namedCollectionChecks(TypeRef{Validation: owner.Validation})
	a.borrowedBuiltinLeaves[owner] = use
	return true
}

func (a *analyzer) validateBorrowedBuiltinLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedBuiltinLeaves) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	uses := []*borrowedBuiltinLeafUse{}
	for _, use := range a.borrowedBuiltinLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		c := use.Owner
		bytes, _ := use.Schema.MarshalJSON()
		if c == nil || a.components[c.Wire] != c || c.Schema != use.Schema || c.Pointer != use.Pointer || c.Scalar != use.Scalar || string(bytes) != use.Bytes || !builtinPrimitiveBinding(c) || c.Validation != use.Validation || namedCollectionChecks(TypeRef{Validation: c.Validation}) != use.Checks {
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
		for d, origin := range a.builtinOrigins {
			if actual[d] > 0 && origin.Owner == c && origin.Schema == use.Schema && origin.Pointer == use.Pointer && origin.Validation == use.Validation && origin.Scalar == use.Scalar && builtinDeclaration(d, origin) {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(use.Source)
		}
	}
	return nil
}

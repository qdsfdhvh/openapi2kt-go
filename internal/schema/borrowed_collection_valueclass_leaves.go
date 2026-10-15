package schema

import (
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// A borrowed container does not turn its existing named wrapper into a generic
// Named type. The original physical declaration supplies InlineValue layout,
// constructor checks and canonical serializer.
type borrowedValueClassLeafUse struct {
	Owner                          *component
	Schema                         *document.Value
	Pointer, Source, Bytes, Scalar string
	SchemaMembers                  []document.Member
	Validation                     *PrimitiveValidation
	Checks                         string
	Origin                         nominalOrigin
	Actual                         *PrimitiveValueClass
}

func (a *analyzer) admitBorrowedValueClass(target *resolvedSchema, source string) bool {
	owner := target.PhysicalValueClass
	if owner == nil && target.Catalog != nil && target.Catalog.ValueClass {
		owner = target.Catalog
	}
	if owner == nil || owner.Schema != target.Schema || !owner.ValueClass {
		return false
	}
	if !a.bindNamedChainModel(target, source) {
		return false
	}
	if a.borrowedValueClassLeaves == nil {
		a.borrowedValueClassLeaves = map[*component]*borrowedValueClassLeafUse{}
	}
	if old := a.borrowedValueClassLeaves[owner]; old != nil {
		if source < old.Source {
			old.Source = source
		}
		return true
	}
	bytes, _ := owner.Schema.MarshalJSON()
	a.borrowedValueClassLeaves[owner] = &borrowedValueClassLeafUse{
		Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer, Source: source,
		Bytes: string(bytes), Scalar: owner.Scalar, Validation: owner.Validation,
		Checks:        namedCollectionChecks(TypeRef{Validation: owner.Validation}),
		SchemaMembers: append([]document.Member{}, owner.Schema.Members()...),
		Origin:        nominalOrigin{owner.Schema, owner.Pointer, owner.NullableValueClass},
	}
	return true
}

func (a *analyzer) validateBorrowedValueClassLeaves(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.borrowedValueClassLeaves) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	for _, d := range ds {
		actual[d]++
	}
	uses := []*borrowedValueClassLeafUse{}
	for _, use := range a.borrowedValueClassLeaves {
		uses = append(uses, use)
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Source < uses[j].Source })
	for _, use := range uses {
		c := use.Owner
		bytes, _ := c.Schema.MarshalJSON()
		if a.components[c.Wire] != c || c.Schema != use.Schema || c.Pointer != use.Pointer || string(bytes) != use.Bytes || !c.ValueClass || c.Scalar != use.Scalar || c.Validation != use.Validation || namedCollectionChecks(TypeRef{Validation: c.Validation}) != use.Checks || c.NullableValueClass != use.Origin.Nullable {
			return schemaReferenceUnsupported(use.Source)
		}
		kind, _, err := schemaType(c.Schema, c.Pointer)
		if err != nil {
			return err
		}
		scalar, enabled, err := valueClassScalar(c.Schema, kind, c.Pointer)
		if err != nil || !enabled || scalar != use.Scalar {
			return schemaReferenceUnsupported(use.Source)
		}
		members := c.Schema.Members()
		if len(members) != len(use.SchemaMembers) {
			return schemaReferenceUnsupported(use.Source)
		}
		for i, member := range members {
			if member != use.SchemaMembers[i] || get(c.Schema, member.Name) != member.Value {
				return schemaReferenceUnsupported(use.Source)
			}
		}
		var retained *PrimitiveValueClass
		count := 0
		for d, origin := range a.valueClassOrigins {
			wrapper, ok := d.(*PrimitiveValueClass)
			if !ok || actual[d] == 0 || origin.Schema != use.Schema || origin != use.Origin || wrapper.ClassName != c.Name || wrapper.Scalar != use.Scalar || wrapper.Validation != use.Validation || namedCollectionChecks(TypeRef{Validation: wrapper.Validation}) != use.Checks {
				continue
			}
			retained = wrapper
			count += actual[d]
		}
		if count != 1 || use.Actual != nil && use.Actual != retained {
			return schemaReferenceUnsupported(use.Source)
		}
		use.Actual = retained
	}
	return nil
}

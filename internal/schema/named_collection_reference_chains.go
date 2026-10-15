package schema

import (
	"encoding/json"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"sort"
)

// Named collection backing stays linked until forward alias plans and original
// peers are complete. New views retain that address rather than copying a tree.
type namedCollectionSourceType struct {
	Node   *TypeRef
	Value  TypeRef
	Checks string
}
type namedCollectionOrigin struct {
	Owner                *component
	Schema               *document.Value
	Pointer, SourceBytes string
	Backing              TypeRef
	Types                []namedCollectionSourceType
	Final                bool
	Peers                []Declaration
}

func namedCollectionCandidate(c *component) bool {
	return c != nil && c.Reference == "" && c.Scalar == "" && !c.ValueClass && !c.BooleanRoot && !c.RecursiveJSON && collectionAliasSchema(c.Schema)
}
func (a *analyzer) recordNamedCollectionOrigin(d Declaration, c *component) {
	if !namedCollectionCandidate(c) {
		return
	}
	bytes, _ := c.Schema.MarshalJSON()
	var types []namedCollectionSourceType
	seen := map[*TypeRef]bool{}
	var record func(*TypeRef)
	record = func(node *TypeRef) {
		if node == nil || seen[node] {
			return
		}
		seen[node] = true
		value := *node
		value.SchemaSteps = 0
		types = append(types, namedCollectionSourceType{node, value, namedCollectionChecks(value)})
		record(node.Element)
	}
	record(&c.AliasType)
	var peers []Declaration
	for peer, owner := range a.declarationOwners {
		if owner == c {
			peers = append(peers, peer)
		}
	}
	a.namedCollectionOrigins[d] = namedCollectionOrigin{c, c.Schema, c.Pointer, string(bytes), c.AliasType, types, false, peers}
}
func namedCollectionChecks(value TypeRef) string {
	checks, _ := json.Marshal(struct {
		Primitive  *PrimitiveValidation
		Collection *CollectionValidation
	}{value.Validation, value.Collection})
	return string(checks)
}
func (a *analyzer) finalizeNamedCollectionOrigins() {
	for d, origin := range a.namedCollectionOrigins {
		a.recordNamedCollectionOrigin(d, origin.Owner)
		origin = a.namedCollectionOrigins[d]
		origin.Final = true
		a.namedCollectionOrigins[d] = origin
		// New named Map views need the same bounded constructor check calls
		// as borrowed trees and root arrays. Linking is complete here; retain
		// original check objects and mark only this newly admitted provenance.
		if _, used := a.namedCollectionUses[origin.Owner]; used {
			a.hintInternalCollectionLeaves(origin.Owner.AliasType)
		}
	}
}
func (a *analyzer) bindNamedChainCollection(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	c := target.CanonicalCollection
	if c == nil {
		c = target.Catalog
		if c.BindingTarget != nil {
			c = c.BindingTarget
		}
	}
	if !namedCollectionCandidate(c) || target.Schema != c.Schema || target.Pointer != c.Pointer {
		return false
	}
	target.CanonicalCollection = c
	target.CollectionBinding = true
	if previous, ok := a.namedCollectionUses[c]; !ok || source < previous {
		a.namedCollectionUses[c] = source
	}
	return true
}
func (a *analyzer) namedCollectionType(c *component) TypeRef {
	// NonNullArrayAlias renders the original underlying view with canonical peer
	// qualification. The flag applies to typed Map/JsonObject bindings as well.
	ty := TypeRef{Kind: Named, Name: a.options.PackageName + ".models." + c.Name, Element: &c.AliasType, ArrayAliasView: NonNullArrayAlias}
	_, nullable, _ := schemaType(c.Schema, c.Pointer)
	if nullable {
		ty = ty.Null()
	}
	return ty
}
func (a *analyzer) validateNamedCollectionBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	used := []*component{}
	for c := range a.namedCollectionUses {
		used = append(used, c)
	}
	sort.Slice(used, func(i, j int) bool { return a.namedCollectionUses[used[i]] < a.namedCollectionUses[used[j]] })
	for _, c := range used {
		matches := 0
		for d, origin := range a.namedCollectionOrigins {
			alias, ok := d.(*ValueAlias)
			if !ok || actual[d] == 0 || origin.Owner != c || origin.Schema != c.Schema || origin.Pointer != c.Pointer || a.components[c.Wire] != c || !namedCollectionCandidate(c) || alias.ClassName != c.Name {
				continue
			}
			bytes, _ := c.Schema.MarshalJSON()
			// Root lowering work is transient; every pointer/check/peer binding stays
			// the original object. Final source budget uses the linked alias height.
			expected, backing, observed := origin.Backing, c.AliasType, alias.Type
			expected.SchemaSteps, backing.SchemaSteps, observed.SchemaSteps = 0, 0, 0
			unchanged := true
			for _, peer := range origin.Peers {
				unchanged = unchanged && actual[peer] == 1 && a.declarationOwners[peer] == c
			}
			for _, source := range origin.Types {
				if !origin.Final {
					continue
				}
				current := *source.Node
				current.SchemaSteps = 0
				unchanged = unchanged && current == source.Value && namedCollectionChecks(current) == source.Checks
			}
			if unchanged && string(bytes) == origin.SourceBytes && backing == expected && observed == expected {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(a.namedCollectionUses[c])
		}
		// First HTTP/union uses may be discovered after forward finalization.
		// Only stable, cycle-checked original graphs can supply emitter hints.
		for _, origin := range a.namedCollectionOrigins {
			if origin.Owner == c && origin.Final {
				a.hintInternalCollectionLeaves(c.AliasType)
				break
			}
		}
	}
	return nil
}

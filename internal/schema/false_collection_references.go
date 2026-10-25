package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"slices"
	"strings"
)

// A finite inline collection root grants permission only to its original leaf.
// Referenced collection backings, tuple projection and scratch DTOs are separate.
type falseCollectionSource struct {
	Owner, Root *component
	Leaf        *document.Value
	Pointer     string
	Height      int
}

func (a *analyzer) falseCollectionRootSource(c *component) *falseCollectionSource {
	v, p := c.Schema, c.Pointer
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || has(v, "default") && !(depth == 0 && a.componentDefaultSchemas[v]) || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") || has(v, "prefixItems") {
			return nil
		}
		if has(v, "$ref") {
			declared, err := a.resolveDeclared(text(get(v, "$ref")), p)
			if err != nil {
				return nil
			}
			root := declared
			if root.BindingTarget != nil {
				root = root.BindingTarget
			}
			if !root.FalseRoot {
				return nil
			}
			return &falseCollectionSource{Owner: c, Root: root, Leaf: v, Pointer: p, Height: depth + 1 + declared.AliasHeight}
		}
		kind, _, err := schemaType(v, p)
		if err != nil {
			return nil
		}
		if kind == "array" {
			v = get(v, "items")
			p = naming.Pointer(p, "items")
		} else if kind == "object" && dictionarySchema(v) {
			v = get(v, "additionalProperties")
			p = naming.Pointer(p, "additionalProperties")
		} else {
			return nil
		}
	}
	return nil
}
func (a *analyzer) falseCollectionElementType(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	source := a.falseCollectionPlanning
	if source == nil || source.Leaf != v || source.Pointer != pointer || a.usage != SharedUsage || !a.aliasPlanning {
		return a.falseInlineCollectionElementType(v, pointer, depth)
	}
	if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
		return nil, err
	}
	if _, _, err := schemaType(v, pointer); err != nil {
		return nil, err
	}
	target, err := a.resolveSchema(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil, err
	}
	ty, err := a.referenceTypeRole(target, nullableOverride(v), depth, pointer, true)
	return &ty, err
}

// Source capture and actual alias/backing proof survive semantic presentation.
type falseCollectionOrigin struct {
	Source    falseCollectionSource
	Nodes     []borrowedAllOfNode
	Types     []namedCollectionSourceType
	Signature string
	Name      string
	Backing   []namedCollectionSourceType
}

func (a *analyzer) recordFalseCollectionOrigin(alias *ValueAlias, source falseCollectionSource) *diagnostic.Diagnostic {
	if err := a.captureFalseRootSource(source.Root); err != nil {
		return err
	}
	nodes, _, err := a.borrowedAllOfSourceBudget(source.Owner.Schema, source.Owner.Pointer, source.Pointer, falseFieldCaptureBudget{a.falseRootSources})
	if err != nil {
		return err
	}
	a.falseCollectionOrigins[alias] = falseCollectionOrigin{Source: source, Nodes: nodes, Types: dtoOriginalGetterTypes(&alias.Type), Signature: falsePropertySignature(Property{Type: alias.Type}), Name: source.Owner.Name, Backing: dtoOriginalGetterTypes(&source.Owner.AliasType)}
	return nil
}
func (a *analyzer) validateFalseCollectionBindings(ds []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	aliases := make([]*ValueAlias, 0, len(a.falseCollectionOrigins))
	for alias := range a.falseCollectionOrigins {
		aliases = append(aliases, alias)
	}
	slices.SortFunc(aliases, func(x, y *ValueAlias) int {
		return strings.Compare(a.falseCollectionOrigins[x].Source.Pointer, a.falseCollectionOrigins[y].Source.Pointer)
	})
	for _, alias := range aliases {
		origin := a.falseCollectionOrigins[alias]
		source := origin.Source
		current := a.falseCollectionRootSource(source.Owner)
		if current == nil || *current != source || actual[alias] != 1 || names[origin.Name] != 1 || alias.ClassName != origin.Name || source.Owner.Name != origin.Name || source.Owner.CollectionHeight != source.Height || !a.falseCollectionFactoryMatches(alias, source.Owner) || a.components[source.Owner.Wire] != source.Owner || !a.componentRoot(source.Owner.Schema, source.Owner.Pointer) || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) || len(origin.Types) == 0 {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, snapshot := range append(slices.Clone(origin.Types), origin.Backing...) {
			value := *snapshot.Node
			value.SchemaSteps = 0
			if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
				return schemaReferenceUnsupported(source.Pointer)
			}
		}
		if falsePropertySignature(Property{Type: alias.Type}) != origin.Signature {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	return nil
}

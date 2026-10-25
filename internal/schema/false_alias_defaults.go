package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Only an original named alias with effective nullable intent owns this role.
// The extra provisional hop leaves the actual alias resolver's depth48 guard
// responsible for the first over-budget candidate; no such alias can emit.
func (a *analyzer) falseAliasOwnDefaultCandidate(c *component) bool {
	if c == nil || !has(c.Schema, "default") || !has(c.Schema, "$ref") {
		return false
	}
	pointer, original, ok := a.componentSchemaAtReference("#" + c.Pointer)
	if !ok || pointer != c.Pointer || original != c.Schema {
		return false
	}
	v, p := c.Schema, c.Pointer
	var intent *bool
	seen := map[*document.Value]bool{}
	for steps := 0; steps <= MaxSchemaDepth; steps++ {
		if seen[v] {
			return false
		}
		seen[v] = true
		if checkKeys(v, []string{"$ref", "nullable", "default"}, p) != nil {
			return false
		}
		if _, _, err := schemaType(v, p); err != nil {
			return false
		}
		if intent == nil {
			intent = nullableOverride(v)
		}
		reference, ok := get(v, "$ref").Text()
		if !ok {
			return false
		}
		next, target, ok := a.componentSchemaAtReference(reference)
		if !ok {
			return false
		}
		if a.booleanFalseRoot(target) {
			return intent != nil && *intent
		}
		v, p = target, next
	}
	return false
}
func (a *analyzer) falseAliasDefaultEncoded(c *component) (string, *diagnostic.Diagnostic) {
	if c.BindingTarget == nil || !c.BindingTarget.FalseRoot || c.BindingNullable == nil || !*c.BindingNullable {
		return "", schemaReferenceUnsupported(c.Pointer)
	}
	node := nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: a.componentType(c), uninhabited: true}
	return node.value(get(c.Schema, "default"), c.Pointer+"/default")
}
func (a *analyzer) recordFalseAliasFactory(alias *ValueAlias, c *component, encoded string) *diagnostic.Diagnostic {
	if !a.falseAliasOwnDefaultCandidate(c) {
		return nil
	}
	value := get(c.Schema, "default")
	if value.Kind() != document.Null || encoded != "null" {
		return schemaReferenceUnsupported(c.Pointer + "/default")
	}
	if _, ok := a.falseAliasOrigins[alias]; !ok {
		return schemaReferenceUnsupported(c.Pointer)
	}
	if err := a.falseRootSources.charge(1, c.Pointer+"/default"); err != nil {
		return err
	}
	a.falseAliasFactories[alias] = falseCollectionFactoryOrigin{Owner: c, Schema: c.Schema, Value: value, Encoded: encoded, Nodes: []borrowedAllOfNode{{Node: value}}}
	return nil
}
func (a *analyzer) falseAliasFactoryMatches(alias *ValueAlias, c *component) bool {
	origin, ok := a.falseAliasFactories[alias]
	if !ok {
		return alias.ComponentDefault == "" && !has(c.Schema, "default")
	}
	return origin.Owner == c && origin.Schema == c.Schema && origin.Value == get(c.Schema, "default") && origin.Value.Kind() == document.Null && origin.Encoded == "null" && alias.ComponentDefault == origin.Encoded && sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes)
}

package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"slices"
)

// The pre-catalog predicate borrows original named sources; it grants only a
// component's own annotation, never field/child/wrapper permission.
type falseCollectionDefaultSource struct {
	Schema, Leaf         *document.Value
	Pointer, LeafPointer string
}

func (a *analyzer) falseCollectionDefaultProfile(v *document.Value, p string) *falseCollectionDefaultSource {
	owner, at := v, p
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || depth > 0 && has(v, "default") || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") || has(v, "prefixItems") {
			return nil
		}
		if has(v, "$ref") {
			if checkKeys(v, []string{"$ref", "nullable"}, p) != nil {
				return nil
			}
			if _, _, err := schemaType(v, p); err != nil {
				return nil
			}
			leaf, location := v, p
			seen := map[*document.Value]bool{}
			for steps := depth + 1; steps <= MaxSchemaDepth; steps++ {
				reference, ok := get(v, "$ref").Text()
				if !ok {
					return nil
				}
				next, target, ok := a.componentSchemaAtReference(reference)
				if !ok || seen[target] {
					return nil
				}
				seen[target] = true
				if a.booleanFalseRoot(target) {
					return &falseCollectionDefaultSource{owner, leaf, at, location}
				}
				if checkKeys(target, []string{"$ref", "nullable", "default"}, next) != nil || !has(target, "$ref") {
					return nil
				}
				if _, _, err := schemaType(target, next); err != nil {
					return nil
				}
				v = target
			}
			return nil
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
func (a *analyzer) falseCollectionOwnDefaultCandidate(c *component) bool {
	return c != nil && has(c.Schema, "default") && a.falseCollectionDefaultProfile(c.Schema, c.Pointer) != nil
}
func (a *analyzer) falseCollectionDefaultLeaf(v *document.Value, p string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	source := a.falseCollectionDefaultPlanning
	if source == nil || source.Leaf != v || source.LeafPointer != p {
		return nil, nil
	}
	if err := checkKeys(v, []string{"$ref", "nullable"}, p); err != nil {
		return nil, err
	}
	if _, _, err := schemaType(v, p); err != nil {
		return nil, err
	}
	target, err := a.resolveSchema(text(get(v, "$ref")), p)
	if err != nil {
		return nil, err
	}
	ty, err := a.referenceTypeRole(target, nullableOverride(v), depth, p, true)
	if err != nil {
		return nil, err
	}
	return &nestedDefaultNode{schema: v, pointer: p, ty: ty, uninhabited: true}, nil
}

type falseCollectionFactoryOrigin struct {
	Owner         *component
	Schema, Value *document.Value
	Encoded       string
	Nodes         []borrowedAllOfNode
}

func (a *analyzer) recordFalseCollectionFactory(alias *ValueAlias, c *component, encoded string) *diagnostic.Diagnostic {
	if !a.falseCollectionOwnDefaultCandidate(c) {
		return nil
	}
	typed, ok := a.falseCollectionOrigins[alias]
	if !ok {
		return schemaReferenceUnsupported(c.Pointer)
	}
	origin, err := a.captureFalseFactoryLiteral(c, encoded, typed.Nodes)
	if err != nil {
		return err
	}
	a.falseCollectionFactories[alias] = origin
	return nil
}

// Literal JSON keys are data, including names ignored as schema metadata.
// Reuse existing source snapshots and charge only additional original nodes.
func (a *analyzer) captureFalseFactoryLiteral(c *component, encoded string, knownNodes []borrowedAllOfNode) (falseCollectionFactoryOrigin, *diagnostic.Diagnostic) {
	return a.captureFalseDefaultLiteral(c, c.Schema, c.Pointer, get(c.Schema, "default"), encoded, knownNodes)
}

func (a *analyzer) captureFalseDefaultLiteral(owner *component, schema *document.Value, pointer string, literal *document.Value, encoded string, knownNodes []borrowedAllOfNode) (falseCollectionFactoryOrigin, *diagnostic.Diagnostic) {
	known := map[*document.Value]bool{}
	for _, node := range knownNodes {
		known[node.Node] = true
	}
	origin := falseCollectionFactoryOrigin{Owner: owner, Schema: schema, Value: literal, Encoded: encoded}
	seen := map[*document.Value]bool{}
	queue := []*document.Value{origin.Value}
	for len(queue) > 0 {
		value := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if value == nil || seen[value] {
			continue
		}
		seen[value] = true
		if len(seen) > maxNestedDefaultNodes {
			return falseCollectionFactoryOrigin{}, diagnostic.At("E_MODEL_LIMIT", "False collection default exceeds the supported value budget.", pointer+"/default")
		}
		if !known[value] {
			if err := a.falseRootSources.charge(1+len(value.Members())+len(value.Items()), pointer+"/default"); err != nil {
				return falseCollectionFactoryOrigin{}, err
			}
		}
		origin.Nodes = append(origin.Nodes, borrowedAllOfNode{value, slices.Clone(value.Members()), slices.Clone(value.Items())})
		for _, m := range value.Members() {
			queue = append(queue, m.Value)
		}
		queue = append(queue, value.Items()...)
	}
	return origin, nil
}

func (a *analyzer) falseCollectionFactoryMatches(alias *ValueAlias, c *component) bool {
	origin, ok := a.falseCollectionFactories[alias]
	if !ok {
		return alias.ComponentDefault == "" && !has(c.Schema, "default")
	}
	return origin.Owner == c && origin.Schema == c.Schema && origin.Value == get(c.Schema, "default") && alias.ComponentDefault == origin.Encoded && sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes)
}

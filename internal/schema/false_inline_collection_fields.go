package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"slices"
	"strings"
)

// Field collection permission names only an original terminal reference. It
// neither creates a component nor changes named collection alias planning.
type falseInlineCollectionSource struct {
	Root       *component
	Leaf       *document.Value
	Pointer    string
	Height     int
	OwnDefault bool
}

func (a *analyzer) falseInlineCollectionSource(v *document.Value, p string) *falseInlineCollectionSource {
	return a.falseInlineCollectionSourceRole(v, p, false)
}
func (a *analyzer) falseInlineCollectionSourceRole(v *document.Value, p string, own bool) *falseInlineCollectionSource {
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || has(v, "default") && !(own && depth == 0) || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") || has(v, "prefixItems") {
			return nil
		}
		if has(v, "$ref") {
			if depth == 0 {
				return nil
			}
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
			return &falseInlineCollectionSource{root, v, p, depth + 1 + declared.AliasHeight, own}
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
func (a *analyzer) falseInlineCollectionFieldSource(f field, owner *document.Value, pointer string, ordinary bool, common []Property, typed map[string]typedLeafField) *falseInlineCollectionSource {
	if !a.componentRoot(owner, pointer) || !ordinary || a.usage != SharedUsage || common != nil || len(typed) != 0 || has(owner, "default") || has(owner, "allOf") || a.hierarchySources[owner] || has(f.Schema, "readOnly") || has(f.Schema, "writeOnly") || get(get(owner, "properties"), f.Wire) != f.Schema || f.Pointer != naming.Pointer(pointer+"/properties", f.Wire) {
		return nil
	}
	return a.falseInlineCollectionSourceRole(f.Schema, f.Pointer, has(f.Schema, "default"))
}
func (a *analyzer) falseInlineCollectionElementType(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	source := a.falseInlineCollectionPlanning
	if source == nil || source.Leaf != v || source.Pointer != pointer || a.usage != SharedUsage {
		return nil, nil
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

type falseInlineCollectionPlan struct {
	Index   int
	Field   field
	Source  falseInlineCollectionSource
	Encoded string
}
type falseInlineCollectionFieldSnapshot struct {
	Plan      falseInlineCollectionPlan
	Property  *Property
	Signature string
	Types     []namedCollectionSourceType
	Literal   *falseCollectionFactoryOrigin
}
type falseInlineCollectionOrigin struct {
	Schema         *document.Value
	Pointer, Name  string
	Nodes          []borrowedAllOfNode
	Fields         []falseInlineCollectionFieldSnapshot
	Count          *CollectionValidation
	CountSignature string
}

func (a *analyzer) recordFalseInlineCollectionOrigin(model *Model, v *document.Value, pointer string, plans []falseInlineCollectionPlan) *diagnostic.Diagnostic {
	a.falseInlineCollectionUses[model] = pointer
	for _, plan := range plans {
		if err := a.captureFalseRootSource(plan.Source.Root); err != nil {
			return err
		}
	}
	nodes, _, err := a.borrowedAllOfSourceBudget(v, pointer, pointer, falseFieldCaptureBudget{a.falseRootSources})
	if err != nil {
		return err
	}
	origin := falseInlineCollectionOrigin{Schema: v, Pointer: pointer, Name: model.ClassName, Nodes: nodes, Count: model.PropertyCount, CountSignature: namedCollectionChecks(TypeRef{Collection: model.PropertyCount})}
	for _, plan := range plans {
		property := &model.Properties[plan.Index]
		snapshot := falseInlineCollectionFieldSnapshot{Plan: plan, Property: property, Signature: falsePropertySignature(*property), Types: dtoOriginalGetterTypes(&property.Type)}
		if plan.Source.OwnDefault {
			owner := a.componentsByOriginalSchema(v)
			if owner == nil {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			literal, err := a.captureFalseDefaultLiteral(owner, plan.Field.Schema, plan.Field.Pointer, get(plan.Field.Schema, "default"), plan.Encoded, nodes)
			if err != nil {
				return err
			}
			snapshot.Literal = &literal
		}
		origin.Fields = append(origin.Fields, snapshot)
	}
	a.falseInlineCollectionOrigins[model] = origin
	return nil
}
func (a *analyzer) validateFalseInlineCollectionBindings(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.falseInlineCollectionOrigins) == 0 && len(a.falseInlineCollectionUses) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	models := make([]*Model, 0, len(a.falseInlineCollectionUses))
	for model := range a.falseInlineCollectionUses {
		models = append(models, model)
	}
	slices.SortFunc(models, func(x, y *Model) int {
		return strings.Compare(a.falseInlineCollectionUses[x], a.falseInlineCollectionUses[y])
	})
	for _, model := range models {
		origin, ok := a.falseInlineCollectionOrigins[model]
		if !ok || origin.Pointer != a.falseInlineCollectionUses[model] || actual[model] != 1 || names[origin.Name] != 1 || model.ClassName != origin.Name || a.modelOrigins[model] != origin.Schema || a.modelPointers[model] != origin.Pointer || !a.componentRoot(origin.Schema, origin.Pointer) || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) || model.PropertyCount != origin.Count || namedCollectionChecks(TypeRef{Collection: model.PropertyCount}) != origin.CountSignature || len(origin.Fields) == 0 {
			return schemaReferenceUnsupported(a.falseInlineCollectionUses[model])
		}
		for _, snapshot := range origin.Fields {
			plan := snapshot.Plan
			current := a.falseInlineCollectionSourceRole(plan.Field.Schema, plan.Field.Pointer, plan.Source.OwnDefault)
			if current == nil || *current != plan.Source || get(get(origin.Schema, "properties"), plan.Field.Wire) != plan.Field.Schema || plan.Field.Pointer != naming.Pointer(origin.Pointer+"/properties", plan.Field.Wire) || plan.Index < 0 || plan.Index >= len(model.Properties) {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			property := &model.Properties[plan.Index]
			if property != snapshot.Property || len(snapshot.Types) == 0 {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			for _, ty := range snapshot.Types {
				value := *ty.Node
				value.SchemaSteps = 0
				if value != ty.Value || namedCollectionChecks(value) != ty.Checks {
					return schemaReferenceUnsupported(plan.Field.Pointer)
				}
			}
			if falsePropertySignature(*property) != snapshot.Signature {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			if plan.Source.OwnDefault {
				literal := snapshot.Literal
				if literal == nil || literal.Owner != a.componentsByOriginalSchema(origin.Schema) || literal.Schema != plan.Field.Schema || literal.Value != get(plan.Field.Schema, "default") || literal.Encoded != plan.Encoded || !sameBorrowedHierarchyNodes(literal.Nodes, literal.Nodes) {
					return schemaReferenceUnsupported(plan.Field.Pointer + "/default")
				}
			} else if snapshot.Literal != nil || plan.Encoded != "" {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}

		}
	}
	if len(a.falseInlineCollectionOrigins) != len(models) {
		return schemaReferenceUnsupported("/components/schemas")
	}
	return nil
}

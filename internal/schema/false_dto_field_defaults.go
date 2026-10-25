package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"slices"
	"strings"
)

// A field annotation borrows the original retained target, never a scratch DTO
// or a target annotation. Its scope exists only during deferred literal work.
type falseDTOFieldDefaultScope struct {
	Plan         referencedFieldDefault
	Target       *Model
	TargetSchema *document.Value
	Used         bool
	CountsUsed   bool
}

func (a *analyzer) falseDTOFieldScope(plan referencedFieldDefault) *falseDTOFieldDefaultScope {
	if plan.Model == nil || (plan.Index < 0 || plan.Index >= len(plan.Model.Properties)) || !has(plan.Field.Schema, "$ref") {
		return nil
	}
	owner := a.modelOrigins[plan.Model]
	pointer := a.modelPointers[plan.Model]
	if !a.componentRoot(owner, pointer) || has(owner, "default") || has(owner, "allOf") || a.hierarchySources[owner] || get(get(owner, "properties"), plan.Field.Wire) != plan.Field.Schema || plan.Field.Pointer != naming.Pointer(pointer+"/properties", plan.Field.Wire) {
		return nil
	}
	c := a.componentsByOriginalSchema(owner)
	if c == nil || c.Name != plan.Model.ClassName || plan.Model.Properties[plan.Index].WireName != plan.Field.Wire {
		return nil
	}
	declared, err := a.resolveDeclared(text(get(plan.Field.Schema, "$ref")), plan.Field.Pointer)
	if err != nil {
		return nil
	}
	root := declared
	if root.BindingTarget != nil {
		root = root.BindingTarget
	}
	if has(root.Schema, "allOf") || a.hierarchySources[root.Schema] || !objectDefaultCandidateSchema(root.Schema) {
		return nil
	}
	var target *Model
	for model, source := range a.falseFieldOrigins {
		if source.Schema != root.Schema || source.Pointer != root.Pointer || source.Name != root.Name {
			continue
		}
		if target != nil {
			return nil
		}
		target = model
	}
	for model, source := range a.literalFalseFieldOrigins {
		if source.Schema != root.Schema || source.Pointer != root.Pointer || source.Name != root.Name {
			continue
		}
		if target != nil && target != model {
			return nil
		}
		target = model
	}
	if target == nil || !a.falseDTOTargetFactoriesMatch(target) {
		return nil
	}
	return &falseDTOFieldDefaultScope{Plan: plan, Target: target, TargetSchema: root.Schema}
}

// Every retained false getter role validates its own root annotation; borrowing
// a field initializer neither inherits nor replaces those independent factories.
func (a *analyzer) falseDTOTargetFactoriesMatch(target *Model) bool {
	found := false
	if source, ok := a.falseFieldOrigins[target]; ok {
		found = true
		if !a.falseObjectFactoryMatches(target, source) {
			return false
		}
	}
	if source, ok := a.literalFalseFieldOrigins[target]; ok {
		found = true
		if !a.literalFalseObjectFactoryMatches(target, source) {
			return false
		}
	}
	return found
}

type falseDTOFieldDefaultKey struct {
	Model *Model
	Index int
}
type falseDTOFieldDefaultOrigin struct {
	Scope                                     falseDTOFieldDefaultScope
	ConsumerSource                            *document.Value
	ConsumerPointer, ConsumerName, TargetName string
	Property                                  *Property
	Signature                                 string
	Types                                     []namedCollectionSourceType
	Nodes                                     []borrowedAllOfNode
	Literal                                   falseCollectionFactoryOrigin
	Count                                     *CollectionValidation
	CountSignature                            string
}

func (a *analyzer) recordFalseDTOFieldDefault(scope falseDTOFieldDefaultScope, encoded string) *diagnostic.Diagnostic {
	plan := scope.Plan
	source := a.modelOrigins[plan.Model]
	pointer := a.modelPointers[plan.Model]
	owner := a.componentsByOriginalSchema(source)
	if owner == nil || !scope.Used {
		return schemaReferenceUnsupported(plan.Field.Pointer)
	}
	// The original consumer is captured once, with all its fields charged to
	// the same cumulative source budget. Each field retains that immutable
	// snapshot and captures its complete literal separately below.
	nodes, captured := a.falseDTOConsumerSources[plan.Model]
	if !captured {
		var err *diagnostic.Diagnostic
		nodes, _, err = a.borrowedAllOfSourceBudget(source, pointer, plan.Field.Pointer, falseFieldCaptureBudget{a.falseRootSources})
		if err != nil {
			return err
		}
		a.falseDTOConsumerSources[plan.Model] = nodes
	} else if !sameBorrowedHierarchyNodes(nodes, nodes) {
		return schemaReferenceUnsupported(plan.Field.Pointer)
	}
	literal, err := a.captureFalseDefaultLiteral(owner, plan.Field.Schema, plan.Field.Pointer, get(plan.Field.Schema, "default"), encoded, nodes)
	if err != nil {
		return err
	}
	property := &plan.Model.Properties[plan.Index]
	a.falseDTOFieldDefaults[falseDTOFieldDefaultKey{plan.Model, plan.Index}] = falseDTOFieldDefaultOrigin{Scope: scope, ConsumerSource: source, ConsumerPointer: pointer, ConsumerName: plan.Model.ClassName, TargetName: scope.Target.ClassName, Property: property, Signature: falsePropertySignature(*property), Types: dtoOriginalGetterTypes(&property.Type), Nodes: nodes, Literal: literal}

	if scope.CountsUsed {
		key := falseDTOFieldDefaultKey{plan.Model, plan.Index}
		origin := a.falseDTOFieldDefaults[key]
		origin.Count = scope.Target.PropertyCount
		origin.CountSignature = namedCollectionChecks(TypeRef{Collection: origin.Count})
		a.falseDTOFieldDefaults[key] = origin
	}
	return nil
}
func (a *analyzer) validateFalseDTOFieldDefaults(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.falseDTOFieldDefaults) == 0 && len(a.falseDTOFieldDefaultUses) == 0 {
		return nil
	}
	if err := a.validateFalseFieldBindings(ds); err != nil {
		return err
	}
	if err := a.validateLiteralFalseFields(ds); err != nil {
		return err
	}
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	keys := make([]falseDTOFieldDefaultKey, 0, len(a.falseDTOFieldDefaultUses))
	for key := range a.falseDTOFieldDefaultUses {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(x, y falseDTOFieldDefaultKey) int {
		return strings.Compare(a.falseDTOFieldDefaultUses[x], a.falseDTOFieldDefaultUses[y])
	})
	if len(a.falseDTOFieldDefaults) != len(keys) {
		for _, key := range keys {
			if _, ok := a.falseDTOFieldDefaults[key]; !ok {
				return schemaReferenceUnsupported(a.falseDTOFieldDefaultUses[key])
			}
		}
		return schemaReferenceUnsupported("")
	}
	for _, key := range keys {
		origin, ok := a.falseDTOFieldDefaults[key]
		if !ok || origin.Scope.Plan.Model != key.Model || origin.Scope.Plan.Index != key.Index || origin.Scope.Plan.Field.Pointer != a.falseDTOFieldDefaultUses[key] {
			return schemaReferenceUnsupported(a.falseDTOFieldDefaultUses[key])
		}
		scope := origin.Scope
		plan := scope.Plan
		current := a.falseDTOFieldScope(plan)
		if current == nil || current.Target != scope.Target || current.TargetSchema != scope.TargetSchema || actual[plan.Model] != 1 || actual[scope.Target] != 1 || names[origin.ConsumerName] != 1 || names[origin.TargetName] != 1 || plan.Model.ClassName != origin.ConsumerName || scope.Target.ClassName != origin.TargetName || a.modelOrigins[plan.Model] != origin.ConsumerSource || a.modelPointers[plan.Model] != origin.ConsumerPointer || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) || plan.Index >= len(plan.Model.Properties) {
			return schemaReferenceUnsupported(plan.Field.Pointer)
		}

		if scope.CountsUsed && (scope.Target.PropertyCount != origin.Count || origin.Count == nil || namedCollectionChecks(TypeRef{Collection: origin.Count}) != origin.CountSignature) {
			return schemaReferenceUnsupported(plan.Field.Pointer)
		}
		property := &plan.Model.Properties[plan.Index]
		if property != origin.Property || len(origin.Types) == 0 {
			return schemaReferenceUnsupported(plan.Field.Pointer)
		}
		for _, snapshot := range origin.Types {
			value := *snapshot.Node
			value.SchemaSteps = 0
			if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
		}
		if falsePropertySignature(*property) != origin.Signature {
			return schemaReferenceUnsupported(plan.Field.Pointer)
		}
		literal := origin.Literal
		if literal.Owner != a.componentsByOriginalSchema(origin.ConsumerSource) || literal.Schema != plan.Field.Schema || literal.Value != get(plan.Field.Schema, "default") || !sameBorrowedHierarchyNodes(literal.Nodes, literal.Nodes) {
			return schemaReferenceUnsupported(plan.Field.Pointer + "/default")
		}
	}
	return nil
}

// This permission belongs only to the exact retained target of a deferred
// consuming field default. Component-owned default permissions stay separate.
func (a *analyzer) falseDTOFieldCounts(v *document.Value, pointer string) (bool, *diagnostic.Diagnostic) {
	scope := a.falseDTOFieldDefaultScope
	if scope == nil || !a.options.Validation || scope.TargetSchema != v || a.modelOrigins[scope.Target] != v || a.modelPointers[scope.Target] != pointer || !a.componentRoot(v, pointer) || (!has(v, "minProperties") && !has(v, "maxProperties")) {
		return false, nil
	}
	parsed, err := a.checkedCollection(TypeRef{Kind: JSONObject}, v, pointer)
	if err != nil {
		return false, err
	}
	if parsed.Collection == nil || scope.Target.PropertyCount == nil || namedCollectionChecks(parsed) != namedCollectionChecks(TypeRef{Kind: JSONObject, Collection: scope.Target.PropertyCount}) {
		return false, schemaReferenceUnsupported(pointer)
	}
	scope.CountsUsed = true
	return true, nil
}

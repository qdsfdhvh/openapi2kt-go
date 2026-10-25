package schema

import (
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// This edge belongs to an owner's root annotation. It is not a deferred
// property annotation and never grants the target an own-default permission.
type falseDTOOwnerDefaultEdge struct {
	Owner, Target    *Model
	TargetComponent  *component
	Index            int
	Field            *document.Value
	Pointer          string
	Property         *Property
	Signature        string
	Types            []namedCollectionSourceType
	Used, CountsUsed bool
	Count            *CollectionValidation
	CountSignature   string
	Aliases          []falseDTOOwnerAliasStep
}

type falseDTOOwnerDefaultOrigin struct {
	Owner   *component
	Name    string
	Pointer string
	Schema  *document.Value
	Nodes   []borrowedAllOfNode
	Edges   map[int]*falseDTOOwnerDefaultEdge
	Factory *falseCollectionFactoryOrigin
}

func (a *analyzer) falseDTOOwnerEdge(v *document.Value, pointer string) *falseDTOOwnerDefaultEdge {
	root := a.componentDefaultCountRoot
	if root == nil || len(a.falseFieldOrigins)+len(a.literalFalseFieldOrigins) == 0 || a.falseDTOOwnerDefaultScope != nil || a.usage != SharedUsage || !a.componentDefaultSchemas[root] || !has(root, "default") || has(root, "allOf") || a.hierarchySources[root] || has(v, "default") || !has(v, "$ref") {
		return nil
	}
	owner := a.componentsByOriginalSchema(root)
	if owner == nil || a.components[owner.Wire] != owner || !a.componentRoot(root, owner.Pointer) {
		return nil
	}
	var model *Model
	index := -1
	for candidate, source := range a.modelOrigins {
		if source != root || candidate.ClassName != owner.Name || a.modelPointers[candidate] != owner.Pointer {
			continue
		}
		for i, p := range candidate.Properties {
			if get(get(root, "properties"), p.WireName) == v && pointer == naming.Pointer(owner.Pointer+"/properties", p.WireName) {
				if model != nil {
					return nil
				}
				model, index = candidate, i
			}
		}
	}
	if model == nil {
		return nil
	}
	declared, err := a.resolveDeclared(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil
	}
	targetComponent, aliases, ok := a.falseDTOOwnerAliasTarget(declared, pointer)
	if !ok || targetComponent == nil || has(targetComponent.Schema, "allOf") || a.hierarchySources[targetComponent.Schema] || !objectDefaultCandidateSchema(targetComponent.Schema) {
		return nil
	}
	var target *Model
	consider := func(candidate *Model) bool {
		if a.modelOrigins[candidate] != targetComponent.Schema || a.modelPointers[candidate] != targetComponent.Pointer || candidate.ClassName != targetComponent.Name {
			return true
		}
		if target != nil && target != candidate {
			return false
		}
		target = candidate
		return true
	}
	for candidate := range a.falseFieldOrigins {
		if !consider(candidate) {
			return nil
		}
	}
	for candidate := range a.literalFalseFieldOrigins {
		if !consider(candidate) {
			return nil
		}
	}
	if target == nil || target == model {
		return nil
	}
	p := &model.Properties[index]
	ty := p.Type.WithoutRootNull()
	if ty.Kind != Named || ty.Element != nil || ty.Name != a.options.PackageName+".models."+targetComponent.Name {
		return nil
	}
	return &falseDTOOwnerDefaultEdge{Owner: model, Target: target, TargetComponent: targetComponent, Index: index, Field: v, Pointer: pointer, Property: p, Signature: falsePropertySignature(*p), Types: dtoOriginalGetterTypes(&p.Type), Aliases: aliases}
}

func (a *analyzer) falseDTOOwnerReferenceNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	edge := a.falseDTOOwnerEdge(v, pointer)
	if edge == nil {
		return nil, nil
	}
	if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
		return nil, err
	}
	for i := range edge.Aliases {
		step := &edge.Aliases[i]
		nodes, captured := a.falseDTOOwnerAliasSources[step.State.Component]
		if !captured {
			if err := a.captureFalseRootSource(step.State.Component); err != nil {
				return nil, err
			}
			var err *diagnostic.Diagnostic
			nodes, _, err = a.borrowedAllOfSourceBudget(step.State.Schema, step.State.Pointer, pointer, falseFieldCaptureBudget{a.falseRootSources})
			if err != nil {
				return nil, err
			}
			a.falseDTOOwnerAliasSources[step.State.Component] = nodes
		} else if len(nodes) == 0 || nodes[0].Node != step.State.Schema || !sameBorrowedHierarchyNodes(nodes, nodes) {
			return nil, schemaReferenceUnsupported(pointer)
		}
		step.Nodes = nodes
	}
	validation := *a
	validation.falseDTOOwnerDefaultScope = edge
	// The target's annotation will be checked independently in the component
	// factory loop. Its name may sort after this owner, so defer factory proof.
	node, err := validation.objectDefaultNode(edge.TargetComponent.Schema, edge.TargetComponent.Pointer, depth+len(edge.Aliases), has(edge.TargetComponent.Schema, "default"))
	if err != nil {
		return nil, err
	}
	if !edge.Used {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if len(edge.Aliases) != 0 && edge.Aliases[0].State.HasNullable {
		node.ty = node.ty.WithoutRootNull()
		if edge.Aliases[0].State.Nullable {
			node.ty = node.ty.Null()
		}
	}
	if has(v, "nullable") {
		_, nullable, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		node.ty = node.ty.WithoutRootNull()
		if nullable {
			node.ty = node.ty.Null()
		}
	}
	origin := a.falseDTOOwnerDefaults[edge.Owner]
	if origin == nil {
		owner := a.componentsByOriginalSchema(a.modelOrigins[edge.Owner])
		if owner == nil {
			return nil, schemaReferenceUnsupported(pointer)
		}
		if err := a.captureFalseRootSource(owner); err != nil {
			return nil, err
		}
		nodes, _, err := a.borrowedAllOfSourceBudget(owner.Schema, owner.Pointer, pointer, falseFieldCaptureBudget{a.falseRootSources})
		if err != nil {
			return nil, err
		}
		origin = &falseDTOOwnerDefaultOrigin{Owner: owner, Name: owner.Name, Pointer: owner.Pointer, Schema: owner.Schema, Nodes: nodes, Edges: map[int]*falseDTOOwnerDefaultEdge{}}
		a.falseDTOOwnerDefaults[edge.Owner] = origin
	}
	if origin.Edges[edge.Index] != nil {
		return nil, schemaReferenceUnsupported(pointer)
	}
	origin.Edges[edge.Index] = edge
	a.falseDTOOwnerDefaultUses[falseDTOFieldDefaultKey{edge.Owner, edge.Index}] = pointer
	return node, nil
}

func (a *analyzer) falseDTOOwnerCounts(v *document.Value, pointer string) (bool, *diagnostic.Diagnostic) {
	edge := a.falseDTOOwnerDefaultScope
	if edge == nil || !a.options.Validation || edge.TargetComponent.Schema != v || edge.TargetComponent.Pointer != pointer || (!has(v, "minProperties") && !has(v, "maxProperties")) {
		return false, nil
	}
	parsed, err := a.checkedCollection(TypeRef{Kind: JSONObject}, v, pointer)
	if err != nil {
		return false, err
	}
	if parsed.Collection == nil || edge.Target.PropertyCount == nil || namedCollectionChecks(parsed) != namedCollectionChecks(TypeRef{Collection: edge.Target.PropertyCount}) {
		return false, schemaReferenceUnsupported(pointer)
	}
	edge.CountsUsed = true
	edge.Count = edge.Target.PropertyCount
	edge.CountSignature = namedCollectionChecks(TypeRef{Collection: edge.Count})
	return true, nil
}

func (a *analyzer) recordFalseDTOOwnerFactory(model *Model, c *component, encoded string) *diagnostic.Diagnostic {
	origin := a.falseDTOOwnerDefaults[model]
	if origin == nil {
		return nil
	}
	if origin.Owner != c || origin.Schema != c.Schema || origin.Name != model.ClassName {
		return schemaReferenceUnsupported(c.Pointer)
	}
	literal, err := a.captureFalseFactoryLiteral(c, encoded, origin.Nodes)
	if err != nil {
		return err
	}
	origin.Factory = &literal
	return nil
}

func (a *analyzer) validateFalseDTOOwnerDefaults(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.falseDTOOwnerDefaults) == 0 && len(a.falseDTOOwnerDefaultUses) == 0 {
		return nil
	}
	if err := a.validateLiteralFalseFields(ds); err != nil {
		return err
	}
	if err := a.validateFalseFieldBindings(ds); err != nil {
		return err
	}
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	keys := make([]falseDTOFieldDefaultKey, 0, len(a.falseDTOOwnerDefaultUses))
	for key := range a.falseDTOOwnerDefaultUses {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(x, y falseDTOFieldDefaultKey) int {
		return strings.Compare(a.falseDTOOwnerDefaultUses[x], a.falseDTOOwnerDefaultUses[y])
	})
	seen := map[*Model]bool{}
	for _, key := range keys {
		pointer := a.falseDTOOwnerDefaultUses[key]
		origin := a.falseDTOOwnerDefaults[key.Model]
		if origin == nil || origin.Owner == nil || a.components[origin.Owner.Wire] != origin.Owner || origin.Owner.Schema != origin.Schema || origin.Owner.Name != origin.Name || origin.Owner.Pointer != origin.Pointer || a.modelOrigins[key.Model] != origin.Schema || a.modelPointers[key.Model] != origin.Pointer || key.Model.ClassName != origin.Name || actual[key.Model] != 1 || names[origin.Name] != 1 || len(origin.Nodes) == 0 || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) {
			return schemaReferenceUnsupported(pointer)
		}
		factory := origin.Factory
		if factory == nil || factory.Owner != origin.Owner || factory.Schema != origin.Schema || factory.Value != get(origin.Schema, "default") || key.Model.ComponentDefault != factory.Encoded || factory.Encoded == "" || len(factory.Nodes) == 0 || !sameBorrowedHierarchyNodes(factory.Nodes, factory.Nodes) {
			return schemaReferenceUnsupported(origin.Pointer + "/default")
		}
		edge := origin.Edges[key.Index]
		if edge == nil || edge.Owner != key.Model || edge.Index != key.Index || edge.Pointer != pointer || !edge.Used || edge.Target == nil || edge.TargetComponent == nil || actual[edge.Target] != 1 || names[edge.TargetComponent.Name] != 1 || edge.Target.ClassName != edge.TargetComponent.Name || a.modelOrigins[edge.Target] != edge.TargetComponent.Schema || a.modelPointers[edge.Target] != edge.TargetComponent.Pointer || key.Index < 0 || key.Index >= len(key.Model.Properties) || !a.falseDTOTargetFactoriesMatch(edge.Target) {
			return schemaReferenceUnsupported(pointer)
		}
		validation := *a
		validation.componentDefaultCountRoot = origin.Schema
		validation.falseDTOOwnerDefaultScope = nil
		current := validation.falseDTOOwnerEdge(edge.Field, edge.Pointer)
		if current == nil || current.Owner != edge.Owner || current.Target != edge.Target || current.TargetComponent != edge.TargetComponent || current.Index != edge.Index || &key.Model.Properties[key.Index] != edge.Property || falsePropertySignature(*edge.Property) != edge.Signature || len(edge.Types) == 0 || !sameFalseDTOOwnerAliasPath(edge.Aliases, current.Aliases) {
			return schemaReferenceUnsupported(pointer)
		}
		for _, snapshot := range edge.Types {
			if snapshot.Node == nil {
				return schemaReferenceUnsupported(pointer)
			}
			value := *snapshot.Node
			value.SchemaSteps = 0
			if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
				return schemaReferenceUnsupported(pointer)
			}
		}
		expectCounts := a.options.Validation && (has(edge.TargetComponent.Schema, "minProperties") || has(edge.TargetComponent.Schema, "maxProperties"))
		if edge.CountsUsed != expectCounts || edge.CountsUsed && (edge.Count == nil || edge.Target.PropertyCount != edge.Count || namedCollectionChecks(TypeRef{Collection: edge.Count}) != edge.CountSignature) {
			return schemaReferenceUnsupported(pointer)
		}
		seen[key.Model] = true
	}
	if len(seen) != len(a.falseDTOOwnerDefaults) {
		return schemaReferenceUnsupported("/components/schemas")
	}
	for owner, origin := range a.falseDTOOwnerDefaults {
		for index, edge := range origin.Edges {
			if edge == nil || a.falseDTOOwnerDefaultUses[falseDTOFieldDefaultKey{owner, index}] != edge.Pointer {
				return schemaReferenceUnsupported(origin.Pointer)
			}
		}
	}
	return nil
}

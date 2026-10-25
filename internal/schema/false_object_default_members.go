package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Only an actual original named object's own default may reuse the constructor
// field's false-source authority. Presence wrapping is not literal nullability.
func (a *analyzer) falseObjectDefaultMemberNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	rootSource := a.componentDefaultCountRoot
	ownerScope := a.falseDTOOwnerDefaultScope
	fieldScope := a.falseDTOFieldDefaultScope
	own := rootSource != nil
	if ownerScope != nil {
		rootSource, own = ownerScope.TargetComponent.Schema, false
	}
	if rootSource == nil && fieldScope != nil {
		rootSource = fieldScope.TargetSchema
	}
	if rootSource == nil || a.usage != SharedUsage {
		return nil, nil
	}
	var model *Model
	var origin falseFieldOrigin
	var field falseFieldSnapshot
	for candidate, source := range a.falseFieldOrigins {
		if source.Schema != rootSource {
			continue
		}
		for _, snapshot := range source.Fields {
			if snapshot.Plan.Field.Schema != v || snapshot.Plan.Field.Pointer != pointer {
				continue
			}
			if model != nil {
				return nil, schemaReferenceUnsupported(pointer)
			}
			model, origin, field = candidate, source, snapshot
		}
	}
	if model == nil {
		return nil, nil
	}
	if own && (!a.componentDefaultSchemas[origin.Schema] || !has(origin.Schema, "default")) || ownerScope != nil && ownerScope.Target != model || !own && ownerScope == nil && (fieldScope == nil || fieldScope.Target != model || has(origin.Schema, "default") && !a.falseObjectFactoryMatches(model, origin)) || !a.componentRoot(origin.Schema, origin.Pointer) || model.ClassName != origin.Name || a.modelOrigins[model] != origin.Schema || a.modelPointers[model] != origin.Pointer || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) || field.Plan.Index >= len(model.Properties) {
		return nil, schemaReferenceUnsupported(pointer)
	}
	property := &model.Properties[field.Plan.Index]
	if property != field.Property || len(field.Types) == 0 {
		return nil, schemaReferenceUnsupported(pointer)
	}
	for _, snapshot := range field.Types {
		value := *snapshot.Node
		value.SchemaSteps = 0
		if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
			return nil, schemaReferenceUnsupported(pointer)
		}
	}
	if falsePropertySignature(*property) != field.Signature {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if err := checkKeys(v, []string{"$ref", "nullable", "default"}, pointer); err != nil {
		return nil, err
	}
	target, err := a.resolveSchema(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil, err
	}
	ty, err := a.referenceTypeRole(target, nullableOverride(v), depth, pointer, true)
	if err != nil {
		return nil, err
	}
	if ownerScope != nil {
		ownerScope.Used = true
	} else if own {
		a.falseObjectDefaultUses[model] = true
	} else {
		fieldScope.Used = true
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, uninhabited: true}, nil
}

func (a *analyzer) recordFalseObjectFactory(model *Model, c *component, encoded string) *diagnostic.Diagnostic {
	if !a.falseObjectDefaultUses[model] {
		return nil
	}
	typed, ok := a.falseFieldOrigins[model]
	if !ok || typed.Schema != c.Schema || typed.Pointer != c.Pointer || typed.Name != c.Name {
		return schemaReferenceUnsupported(c.Pointer)
	}
	origin, err := a.captureFalseFactoryLiteral(c, encoded, typed.Nodes)
	if err != nil {
		return err
	}
	a.falseObjectFactories[model] = origin
	return nil
}
func (a *analyzer) falseObjectFactoryMatches(model *Model, source falseFieldOrigin) bool {
	origin, ok := a.falseObjectFactories[model]
	if !has(source.Schema, "default") {
		return !ok && !a.falseObjectDefaultUses[model] && model.ComponentDefault == ""
	}
	if !ok || !a.falseObjectDefaultUses[model] || origin.Owner == nil {
		return false
	}
	c := origin.Owner
	return a.components[c.Wire] == c && c.Schema == source.Schema && c.Pointer == source.Pointer && c.Name == source.Name && origin.Schema == source.Schema && origin.Value == get(source.Schema, "default") && model.ComponentDefault == origin.Encoded && sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes)
}

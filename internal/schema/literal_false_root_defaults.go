package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// An actual false field may validate its owner's root annotation or the exact
// retained target of a deferred consuming field default. Optional absence
// carriers never confer schema nullability.
func (a *analyzer) literalFalseObjectDefaultNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	root := a.componentDefaultCountRoot
	own := root != nil
	ownerScope := a.falseDTOOwnerDefaultScope
	if ownerScope != nil {
		root, own = ownerScope.TargetComponent.Schema, false
	}
	fieldScope := a.falseDTOFieldDefaultScope
	if root == nil && fieldScope != nil {
		root = fieldScope.TargetSchema
	}
	if root == nil || a.usage != SharedUsage || !a.booleanFalseRoot(v) || own && (!a.componentDefaultSchemas[root] || !has(root, "default")) {
		return nil, nil
	}
	var model *Model
	var origin literalFalseFieldOrigin
	var field literalFalseFieldSnapshot
	for candidate, source := range a.literalFalseFieldOrigins {
		if source.Schema != root {
			continue
		}
		for _, snapshot := range source.Fields {
			direct := snapshot.Plan.Collection.Leaf == nil && snapshot.Plan.Field.Schema == v && snapshot.Plan.Field.Pointer == pointer
			consuming := fieldScope != nil && fieldScope.Plan.Model != nil && fieldScope.Target == candidate
			borrowed := ownerScope != nil && ownerScope.Target == candidate
			collection := (own || consuming || borrowed) && snapshot.Plan.Collection.Leaf == v && snapshot.Plan.Collection.Pointer == pointer
			if !direct && !collection {
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
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	c := origin.Owner
	if c == nil || a.components[c.Wire] != c || c.Schema != origin.Schema || c.Pointer != origin.Pointer || c.Name != origin.Name || model.ClassName != origin.Name || a.modelOrigins[model] != origin.Schema || a.modelPointers[model] != origin.Pointer || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) || field.Plan.Index < 0 || field.Plan.Index >= len(model.Properties) {
		return nil, schemaReferenceUnsupported(pointer)
	}
	p := &model.Properties[field.Plan.Index]
	if p != field.Property || len(field.Types) == 0 {
		return nil, schemaReferenceUnsupported(pointer)
	}
	for _, snapshot := range field.Types {
		ty := *snapshot.Node
		ty.SchemaSteps = 0
		if ty != snapshot.Value || namedCollectionChecks(ty) != snapshot.Checks {
			return nil, schemaReferenceUnsupported(pointer)
		}
	}
	if falsePropertySignature(*p) != field.Signature {
		return nil, schemaReferenceUnsupported(pointer)
	}
	ty := p.Type.WithoutRootNull()
	if field.Plan.Collection.Leaf != nil {
		// The root annotation borrows this actual field's codec, never a new
		// declaration or a nullable absence carrier. Consuming annotations
		// require the existing deferred field plan and this exact target.
		for ty.Kind == Nullable || ty.Kind == List || ty.Kind == Map {
			if ty.Element == nil {
				return nil, schemaReferenceUnsupported(pointer)
			}
			ty = *ty.Element
		}
		if ty.Kind != Named || ty.Element != nil || origin.Helper == nil || ty.Name != a.options.PackageName+".models."+origin.Name+"."+origin.HelperName {
			return nil, schemaReferenceUnsupported(pointer)
		}
	}
	if ownerScope != nil {
		if ownerScope.Target != model {
			return nil, schemaReferenceUnsupported(pointer)
		}
		ownerScope.Used = true
	} else if own {
		if origin.Default == nil || get(origin.Schema, "default") != origin.Default {
			return nil, schemaReferenceUnsupported(pointer)
		}
		a.literalFalseObjectDefaultUses[model] = true
	} else {
		if fieldScope == nil || fieldScope.Target != model || has(origin.Schema, "default") && !a.literalFalseObjectFactoryMatches(model, origin) {
			return nil, schemaReferenceUnsupported(pointer)
		}
		fieldScope.Used = true
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, uninhabited: true}, nil
}
func (a *analyzer) recordLiteralFalseObjectFactory(model *Model, c *component, encoded string) *diagnostic.Diagnostic {
	source, ok := a.literalFalseFieldOrigins[model]
	if !ok {
		return nil
	}
	if source.Default == nil || source.Default != get(c.Schema, "default") || source.Owner != c || !a.literalFalseObjectDefaultUses[model] {
		return schemaReferenceUnsupported(c.Pointer)
	}
	origin, err := a.captureFalseFactoryLiteral(c, encoded, source.Nodes)
	if err != nil {
		return err
	}
	a.literalFalseObjectFactories[model] = origin
	return nil
}
func (a *analyzer) literalFalseObjectFactoryMatches(model *Model, source literalFalseFieldOrigin) bool {
	origin, ok := a.literalFalseObjectFactories[model]
	if source.Default == nil {
		return get(source.Schema, "default") == nil && !ok && !a.literalFalseObjectDefaultUses[model] && model.ComponentDefault == ""
	}
	c := source.Owner
	return ok && a.literalFalseObjectDefaultUses[model] && c != nil && a.components[c.Wire] == c && origin.Owner == c && c.Schema == source.Schema && c.Name == source.Name && c.Pointer == source.Pointer && get(source.Schema, "default") == source.Default && origin.Schema == source.Schema && origin.Value == source.Default && model.ComponentDefault == origin.Encoded && sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes)
}

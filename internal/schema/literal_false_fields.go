package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"slices"
	"strconv"
	"strings"
)

func (a *analyzer) literalFalseField(f field, owner *document.Value, pointer string, ordinary bool, common []Property, typed map[string]typedLeafField) bool {
	return a.booleanFalseRoot(f.Schema) && a.usage == SharedUsage && ordinary && common == nil && len(typed) == 0 && a.componentRoot(owner, pointer) && !has(owner, "allOf") && !a.hierarchySources[owner] && get(get(owner, "properties"), f.Wire) == f.Schema && f.Pointer == naming.Pointer(pointer+"/properties", f.Wire)
}

// One zero-domain literal codec belongs to the actual model. It never creates
// a component; each source field retains an independent original-node proof.
func (a *analyzer) literalFalseHelper(model *Model, fields []field, used map[string]bool, pointer string) (*FalseValue, *diagnostic.Diagnostic) {
	natural := map[string]bool{}
	for _, f := range fields {
		if a.booleanFalseRoot(f.Schema) {
			continue
		}
		if name, err := naming.ClassName(f.Wire, f.Pointer); err == nil {
			natural[strings.ToLower(name)] = true
		}
	}
	name := "FalseValue"
	for index := 2; used[strings.ToLower(name)] || natural[strings.ToLower(name)]; index++ {
		name = "FalseValue" + strconv.Itoa(index)
	}
	if err := reserveNested(used, name, pointer); err != nil {
		return nil, err
	}
	value := &FalseValue{ClassName: name}
	model.Nested = append(model.Nested, value)
	return value, nil
}

type literalFalseFieldPlan struct {
	Index      int
	Field      field
	Collection literalFalseCollectionSource
	Encoded    string
}
type literalFalseFieldSnapshot struct {
	Plan      literalFalseFieldPlan
	Property  *Property
	Signature string
	Types     []namedCollectionSourceType
	Literal   *falseCollectionFactoryOrigin
}
type literalFalseFieldOrigin struct {
	Owner                     *component
	Schema                    *document.Value
	Default                   *document.Value
	Pointer, Name, HelperName string
	Helper                    *FalseValue
	Absent                    bool
	Nullable                  bool
	Nodes                     []borrowedAllOfNode
	Nested                    []Declaration
	Fields                    []literalFalseFieldSnapshot
	Count                     *CollectionValidation
	CountSignature            string
}

func (a *analyzer) recordLiteralFalseFields(model *Model, v *document.Value, pointer string, helper *FalseValue, plans []literalFalseFieldPlan) *diagnostic.Diagnostic {
	a.literalFalseFieldUses[model] = pointer
	owner := a.componentsByOriginalSchema(v)
	if owner == nil || owner.Name != model.ClassName || owner.Pointer != pointer || a.components[owner.Wire] != owner || helper == nil || len(plans) == 0 {
		return schemaReferenceUnsupported(pointer)
	}
	// This is the real parent component's document capture, not false type admission.
	if err := a.captureFalseRootSource(owner); err != nil {
		return err
	}
	nodes, _, err := a.borrowedAllOfSourceBudget(v, pointer, pointer, falseFieldCaptureBudget{a.falseRootSources})
	if err != nil {
		return err
	}
	origin := literalFalseFieldOrigin{Owner: owner, Schema: v, Default: get(v, "default"), Pointer: pointer, Name: model.ClassName, HelperName: helper.ClassName, Helper: helper, Absent: helper.AbsentSerializer, Nullable: model.Nullable, Nodes: nodes, Nested: slices.Clone(model.Nested), Count: model.PropertyCount, CountSignature: namedCollectionChecks(TypeRef{Collection: model.PropertyCount})}
	for _, plan := range plans {
		p := &model.Properties[plan.Index]
		snapshot := literalFalseFieldSnapshot{Plan: plan, Property: p, Signature: falsePropertySignature(*p), Types: dtoOriginalGetterTypes(&p.Type)}
		if plan.Collection.OwnDefault {
			literal, err := a.captureFalseDefaultLiteral(owner, plan.Field.Schema, plan.Field.Pointer, get(plan.Field.Schema, "default"), plan.Encoded, nodes)
			if err != nil {
				return err
			}
			snapshot.Literal = &literal
		}
		origin.Fields = append(origin.Fields, snapshot)
	}
	a.literalFalseFieldOrigins[model] = origin
	return nil
}
func (a *analyzer) validateLiteralFalseFields(ds []Declaration) *diagnostic.Diagnostic {
	if len(a.literalFalseFieldOrigins) == 0 && len(a.literalFalseFieldUses) == 0 {
		return nil
	}
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	models := make([]*Model, 0, len(a.literalFalseFieldUses))
	for model := range a.literalFalseFieldUses {
		models = append(models, model)
	}
	slices.SortFunc(models, func(x, y *Model) int { return strings.Compare(a.literalFalseFieldUses[x], a.literalFalseFieldUses[y]) })
	for _, model := range models {
		origin, ok := a.literalFalseFieldOrigins[model]
		if !ok || origin.Owner == nil || a.components[origin.Owner.Wire] != origin.Owner || origin.Owner.Schema != origin.Schema || origin.Owner.Pointer != origin.Pointer || origin.Owner.Name != origin.Name || origin.Pointer != a.literalFalseFieldUses[model] || actual[model] != 1 || names[origin.Name] != 1 || model.ClassName != origin.Name || a.modelOrigins[model] != origin.Schema || a.modelPointers[model] != origin.Pointer || !a.componentRoot(origin.Schema, origin.Pointer) || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) || model.Nullable != origin.Nullable || !a.literalFalseObjectFactoryMatches(model, origin) || !slices.Equal(origin.Nested, model.Nested) || origin.Helper == nil || origin.Helper.ClassName != origin.HelperName || origin.Helper.AbsentSerializer != origin.Absent || model.PropertyCount != origin.Count || namedCollectionChecks(TypeRef{Collection: model.PropertyCount}) != origin.CountSignature || len(origin.Fields) == 0 {
			return schemaReferenceUnsupported(a.literalFalseFieldUses[model])
		}
		helperCount, helperNames := 0, 0
		for _, d := range model.Nested {
			if d == origin.Helper {
				helperCount++
			}
			if d.Name() == origin.HelperName {
				helperNames++
			}
		}
		if helperCount != 1 || helperNames != 1 {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		optional := false
		for _, snapshot := range origin.Fields {
			plan := snapshot.Plan
			sourceOK := plan.Collection == (literalFalseCollectionSource{}) && a.booleanFalseRoot(plan.Field.Schema)
			if plan.Collection.Leaf != nil {
				current := a.literalFalseCollectionFieldSource(plan.Field, origin.Schema, origin.Pointer, true, nil, nil)
				sourceOK = current != nil && *current == plan.Collection
			}
			if !sourceOK || get(get(origin.Schema, "properties"), plan.Field.Wire) != plan.Field.Schema || plan.Field.Pointer != naming.Pointer(origin.Pointer+"/properties", plan.Field.Wire) || plan.Index < 0 || plan.Index >= len(model.Properties) {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			p := &model.Properties[plan.Index]
			if p != snapshot.Property || len(snapshot.Types) == 0 {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			for _, snapshot := range snapshot.Types {
				value := *snapshot.Node
				value.SchemaSteps = 0
				if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
					return schemaReferenceUnsupported(plan.Field.Pointer)
				}
			}
			if falsePropertySignature(*p) != snapshot.Signature {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			if plan.Collection.OwnDefault {
				literal := snapshot.Literal
				if literal == nil || literal.Owner != origin.Owner || literal.Schema != plan.Field.Schema || literal.Value != get(plan.Field.Schema, "default") || literal.Encoded != plan.Encoded || plan.Encoded == "" || !sameBorrowedHierarchyNodes(literal.Nodes, literal.Nodes) {
					return schemaReferenceUnsupported(plan.Field.Pointer + "/default")
				}
			} else if snapshot.Literal != nil || plan.Encoded != "" {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			optional = optional || p.OmitDefault
		}
		if optional != origin.Absent {
			return schemaReferenceUnsupported(origin.Pointer)
		}
	}
	if len(a.literalFalseFieldOrigins) != len(models) {
		return schemaReferenceUnsupported("/components/schemas")
	}
	return nil
}

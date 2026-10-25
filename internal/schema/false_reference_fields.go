package schema

import (
	"encoding/json"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"slices"
	"strings"
)

// Ordinary direct named references are an explicit field role. Other callers
// retain referenceType's historical false-source refusal.
func (a *analyzer) falseReferenceFieldType(f field, owner *document.Value, pointer string, ordinary bool, common []Property, typed map[string]typedLeafField) (TypeRef, *component, *diagnostic.Diagnostic) {
	if !a.componentRoot(owner, pointer) || !ordinary || a.usage != SharedUsage || common != nil || len(typed) != 0 || has(owner, "allOf") || a.hierarchySources[owner] || !has(f.Schema, "$ref") || has(f.Schema, "readOnly") || has(f.Schema, "writeOnly") {
		return TypeRef{}, nil, nil
	}
	declared, err := a.resolveDeclared(text(get(f.Schema, "$ref")), f.Pointer)
	if err != nil {
		return TypeRef{}, nil, nil
	}
	root := declared
	if root.BindingTarget != nil {
		root = root.BindingTarget
	}
	if !root.FalseRoot {
		return TypeRef{}, nil, nil
	}
	keys := []string{"$ref", "nullable"}
	if has(f.Schema, "default") {
		keys = append(keys, "default")
	}
	if err := checkKeys(f.Schema, keys, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	if _, _, err := schemaType(f.Schema, f.Pointer); err != nil {
		return TypeRef{}, nil, err
	}
	target, err := a.resolveSchema(text(get(f.Schema, "$ref")), f.Pointer)
	if err != nil {
		return TypeRef{}, nil, err
	}
	ty, err := a.referenceTypeRole(target, nullableOverride(f.Schema), f.Depth, f.Pointer, true)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if has(f.Schema, "default") {
		// Logical nullability precedes optional-presence wrapping. A false
		// non-null source never gains a default role from its absence carrier.
		if ty.Kind != Nullable {
			return TypeRef{}, nil, nil
		}
		if get(f.Schema, "default").Kind() != document.Null {
			return TypeRef{}, nil, invalid("A nullable false field default must be null.", f.Pointer+"/default")
		}
	}
	return ty, root, nil
}

func falseAbsentSerializerName(name string) string {
	if name == "AbsentSerializer" {
		return "AbsentSerializer2"
	}
	return "AbsentSerializer"
}

func (a *analyzer) useFalseAbsentSerializer(root *component) string {
	a.falseAbsentUses[root] = true
	for d, origin := range a.falseValueOrigins {
		if origin.Owner == root {
			d.AbsentSerializer = true
		}
	}
	return a.options.PackageName + ".models." + root.Name + "." + falseAbsentSerializerName(root.Name)
}

// Authority belongs to the actual named constructor and original source nodes,
// independently of semantic equality used by later presentation transforms.
type falseFieldPlan struct {
	Index int
	Field field
	Root  *component
}
type falseFieldSnapshot struct {
	Plan      falseFieldPlan
	Property  *Property
	Signature string
	Types     []namedCollectionSourceType
}
type falseFieldOrigin struct {
	Schema        *document.Value
	Pointer, Name string
	Nodes         []borrowedAllOfNode
	Fields        []falseFieldSnapshot
}
type falseFieldCaptureBudget struct{ Registry *falseRootSourceRegistry }

func (b falseFieldCaptureBudget) charge(work, _ int, source string) *diagnostic.Diagnostic {
	return b.Registry.charge(work, source)
}
func falsePropertySignature(p Property) string {
	// Source work is checked by the resolver before this retained field proof.
	p.Type = falseFieldSemanticType(p.Type)
	value, _ := json.Marshal(p)
	return string(value)
}
func falseFieldSemanticType(t TypeRef) TypeRef {
	t.SchemaSteps = 0
	if t.Element != nil {
		child := falseFieldSemanticType(*t.Element)
		t.Element = &child
	}
	return t
}
func (a *analyzer) recordFalseFieldOrigin(model *Model, source *document.Value, pointer string, plans []falseFieldPlan) *diagnostic.Diagnostic {
	if err := a.captureFalseRootSource(plans[0].Root); err != nil {
		return err
	}
	nodes, _, err := a.borrowedAllOfSourceBudget(source, pointer, pointer, falseFieldCaptureBudget{a.falseRootSources})
	if err != nil {
		return err
	}
	origin := falseFieldOrigin{Schema: source, Pointer: pointer, Name: model.ClassName, Nodes: nodes}
	for _, plan := range plans {
		property := &model.Properties[plan.Index]
		origin.Fields = append(origin.Fields, falseFieldSnapshot{plan, property, falsePropertySignature(*property), dtoOriginalGetterTypes(&property.Type)})
	}
	a.falseFieldOrigins[model] = origin
	return nil
}
func (a *analyzer) validateFalseFieldBindings(ds []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	models := make([]*Model, 0, len(a.falseFieldOrigins))
	for model := range a.falseFieldOrigins {
		models = append(models, model)
	}
	slices.SortFunc(models, func(x, y *Model) int {
		return strings.Compare(a.falseFieldOrigins[x].Pointer, a.falseFieldOrigins[y].Pointer)
	})
	uses := map[*component]bool{}
	for _, model := range models {
		origin := a.falseFieldOrigins[model]
		if actual[model] != 1 || names[origin.Name] != 1 || model.ClassName != origin.Name || a.modelOrigins[model] != origin.Schema || a.modelPointers[model] != origin.Pointer || !a.componentRoot(origin.Schema, origin.Pointer) || !sameBorrowedHierarchyNodes(origin.Nodes, origin.Nodes) {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		if !a.falseObjectFactoryMatches(model, origin) {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		for _, snapshot := range origin.Fields {
			plan := snapshot.Plan
			if plan.Index >= len(model.Properties) {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			p := &model.Properties[plan.Index]
			if p != snapshot.Property || get(get(origin.Schema, "properties"), plan.Field.Wire) != plan.Field.Schema || len(snapshot.Types) == 0 {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			for _, ty := range snapshot.Types {
				value := *ty.Node
				value.SchemaSteps = 0
				if value != ty.Value || namedCollectionChecks(value) != ty.Checks {
					return schemaReferenceUnsupported(plan.Field.Pointer)
				}
			}
			if falsePropertySignature(*p) != snapshot.Signature {
				return schemaReferenceUnsupported(plan.Field.Pointer)
			}
			if p.OmitDefault {
				uses[plan.Root] = true
			}
		}
	}
	if len(uses) != len(a.falseAbsentUses) {
		return schemaReferenceUnsupported("/components/schemas")
	}
	for root := range uses {
		if !a.falseAbsentUses[root] {
			return schemaReferenceUnsupported(root.Pointer)
		}
	}
	return nil
}

package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Isolated source analysis exports original facts, never temporary declarations,
// names, default initializers or type-planning registrations.
type tupleInlineConstructorFact struct {
	Pointer  string
	Nullable bool
	Height   int
}

type tupleInlineAllOfUse struct {
	Source    *document.Value
	Pointer   string
	Snapshots []namedCollectionSourceType
}

type tupleInlineAllOfSelection struct {
	Array   *document.Value
	Source  *document.Value
	Pointer string
}

func (a *analyzer) tupleInlineAllOf(v *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if err := checkKeys(v, []string{"type", "nullable", "allOf", "properties", "required", "minProperties", "maxProperties"}, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if a.hierarchySources[v] || a.containsObjectCompositionUnion(v, 0, map[*document.Value]bool{}) {
		return tupleScalarSignature{}, unsupported(pointer)
	}
	if a.tupleAllOfSources == nil {
		a.tupleAllOfSources = &tupleAllOfSourceBudget{Seen: map[*document.Value]bool{}}
	}
	if !a.tupleAllOfSources.Seen[v] {
		bytes, _ := v.MarshalJSON()
		if err := a.tupleAllOfSources.charge(0, len(bytes), pointer); err != nil {
			return tupleScalarSignature{}, err
		}
		_, references, err := a.borrowedAllOfSourceBudget(v, pointer, pointer, a.tupleAllOfSources)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		if err := a.captureTupleInlineAllOfCatalogs(references, pointer); err != nil {
			return tupleScalarSignature{}, err
		}
		a.tupleAllOfSources.Seen[v] = true
	}
	if err := a.registerTupleInlineSource(v, pointer, depth, true); err != nil {
		return tupleScalarSignature{}, err
	}
	nullable := false
	height := 0
	if a.tupleSourceValidation {
		shape, err := a.modelShape(v, pointer, depth, true)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		nullable = shape.Nullable != nil && *shape.Nullable
		height = tupleInlineAllOfShapeHeight(shape, depth)
	} else {
		fact, ok := a.tupleObjects.Proof.Constructors[v]
		if !ok || fact.Pointer != pointer {
			return tupleScalarSignature{}, schemaReferenceUnsupported(pointer)
		}
		nullable = fact.Nullable
		height = fact.Height
	}
	a.tupleObjects.Heights[v] = height
	return tupleScalarSignature{Kind: "object", Nullable: nullable, ModelSource: v}, nil
}

func (a *analyzer) captureTupleInlineAllOfCatalogs(refs []borrowedAllOfShapeRef, pointer string) *diagnostic.Diagnostic {
	if a.tupleNativeSources == nil {
		a.tupleNativeSources = &tupleNativeSourceRegistry{}
	}
	r := a.tupleNativeSources
	if r.Catalogs == nil {
		r.Catalogs = map[*component]borrowedAllOfShapeRef{}
	}
	for _, ref := range refs {
		if old, ok := r.Catalogs[ref.Owner]; ok {
			if old.Schema != ref.Schema || old.Pointer != ref.Pointer || old.Reference != ref.Reference {
				return schemaReferenceUnsupported(pointer)
			}
		} else {
			if err := r.charge(1, 0, pointer); err != nil {
				return err
			}
			r.Catalogs[ref.Owner] = ref
		}
	}
	return nil
}

func tupleInlineAllOfShapeHeight(shape *shape, depth int) int {
	height := 0
	for _, f := range append(append([]field{}, shape.Fields...), shape.Discarded...) {
		height = max(height, f.Depth-depth)
	}
	return height
}

func (a *analyzer) tupleInlineAllOfSourceWork(element TypeRef, decision *arrayItemsDecision) TypeRef {
	if !a.aliasPlanning || a.tupleObjects == nil || len(a.tupleObjects.AllOf) == 0 || decision == nil {
		return element
	}
	for _, prefix := range decision.Prefix {
		if a.tupleObjects.AllOf[prefix.Schema] {
			element.SchemaSteps = max(element.SchemaSteps, a.tupleObjects.Heights[prefix.Schema])
		}
	}
	if a.tupleObjects.AllOf[decision.Selected] {
		element.SchemaSteps = max(element.SchemaSteps, a.tupleObjects.Heights[decision.Selected])
	}
	return element
}

func (a *analyzer) selectTupleInlineAllOf(array *document.Value, pointer string, decision *arrayItemsDecision) {
	r := a.tupleObjects
	if r == nil || decision.Projection != nil || !r.AllOf[decision.Selected] {
		return
	}
	for _, previous := range r.Selected {
		if previous.Array == array {
			return
		}
	}
	r.Selected = append(r.Selected, tupleInlineAllOfSelection{array, decision.Selected, pointer})
}

func (a *analyzer) recordTupleInlineAllOfUse(t *TypeRef, source *document.Value) {
	r := a.tupleObjects
	r.Uses[t] = tupleInlineAllOfUse{source, r.Seen[source], dtoOriginalGetterTypes(t)}
}

func tupleInlineAllOfUseMatches(t *TypeRef, use tupleInlineAllOfUse) bool {
	if len(use.Snapshots) == 0 || use.Snapshots[0].Node != t {
		return false
	}
	for _, snapshot := range use.Snapshots {
		value := *snapshot.Node
		value.SchemaSteps = 0
		if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
			return false
		}
	}
	return true
}

func (a *analyzer) tupleInlineAllOfFact(source *document.Value) (tupleInlineConstructorFact, bool) {
	registry := a.tupleObjects.Proof
	if a.tupleSourceValidation {
		registry = a.tupleNativeSources
	}
	if registry == nil {
		return tupleInlineConstructorFact{}, false
	}
	fact, ok := registry.Constructors[source]
	return fact, ok
}

func (a *analyzer) onlyTupleInlineAllOfPeers(ds []Declaration) bool {
	if len(ds) == 0 || a.tupleObjects == nil {
		return false
	}
	for _, d := range ds {
		model, ok := d.(*Model)
		if !ok || len(model.Properties) == 0 || !a.tupleObjects.AllOf[a.modelOrigins[model]] {
			return false
		}
	}
	return true
}

// Only positional sources route new operation peers; ordinary operation
// allOf/union roots and non-tuple containers retain their admission rules.
func tupleInlineAllOfContainerCandidate(v *document.Value) bool {
	for depth := 0; depth <= MaxSchemaDepth && v != nil; depth++ {
		if has(v, "$ref") {
			return false
		}
		if prefix := get(v, "prefixItems"); prefix.Kind() == document.Array {
			for _, member := range prefix.Items() {
				if composedModelCandidate(member) {
					return true
				}
			}
			if composedModelCandidate(get(v, "items")) {
				return true
			}
		}
		kind, _, err := schemaType(v, "")
		if err != nil {
			return false
		}
		if kind == "array" {
			v = get(v, "items")
		} else if dictionarySchema(v) {
			v = get(v, "additionalProperties")
		} else {
			return false
		}
	}
	return false
}

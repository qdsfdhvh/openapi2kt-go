package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type tupleCollectionFact struct {
	Pointer  string
	Nullable bool
	Kind     TypeKind
	Height   int
}

func (a *analyzer) onlyTupleCollectionPeers(ds []Declaration) bool {
	if len(ds) == 0 || a.tupleObjects == nil || len(a.tupleObjects.Collections) == 0 {
		return false
	}
	for _, d := range ds {
		var node *document.Value
		var pointer string
		switch d := d.(type) {
		case *Model:
			node, pointer = a.modelOrigins[d], a.modelPointers[d]
		case *StringEnum:
			origin := a.enumOrigins[d]
			node, pointer = origin.Schema, origin.Pointer
		case *FalseValue:
			if a.tupleBooleans != nil {
				source := a.tupleBooleans.Declarations[d]
				node, pointer = source.Leaf, source.LeafPointer
			}
		}
		if node == nil {
			return false
		}
		captured := false
		for _, source := range a.tupleObjects.Tasks {
			if source.Collection && strings.HasPrefix(pointer, source.Pointer+"/") {
				if saved, ok := a.tupleNativeSources.Nodes[node]; ok && saved.Node == node {
					captured = true
					break
				}
			}
		}
		if !captured {
			return false
		}
	}
	return true
}

func tupleCollectionContainerCandidate(v *document.Value) bool {
	for depth := 0; depth <= MaxSchemaDepth && v != nil; depth++ {
		if has(v, "$ref") {
			return false
		}
		if prefix := get(v, "prefixItems"); prefix != nil {
			for _, member := range prefix.Items() {
				if tupleCollectionCandidate(member) {
					return true
				}
			}
			if tupleCollectionCandidate(get(v, "items")) {
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

type tupleCollectionUse struct {
	Source    *document.Value
	Pointer   string
	Snapshots []namedCollectionSourceType
	Final     bool
}

func tupleCollectionCandidate(v *document.Value) bool {
	if v == nil || v.Kind() != document.Object || has(v, "$ref") {
		return false
	}
	kind, _, err := schemaType(v, "")
	open, _ := get(v, "additionalProperties").Bool()
	return err == nil && (kind == "array" || dictionarySchema(v) && (get(v, "additionalProperties").Kind() == document.Object || open))
}

func (a *analyzer) tupleCollectionSource(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, source); err != nil {
		return tupleScalarSignature{}, err
	}
	if original == terminal && has(original, "default") {
		return tupleScalarSignature{}, unsupported(naming.Pointer(source, "default"))
	}
	a.initTupleInlineDTOs()
	a.tupleObjects.Collections[original] = true
	if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	_, shapeRefs, err := a.borrowedAllOfSourceBudget(terminal, pointer, source, nil)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	if err := a.captureTupleInlineAllOfCatalogs(shapeRefs, source); err != nil {
		return tupleScalarSignature{}, err
	}
	if err := a.captureTupleCollectionGraph(original, source); err != nil {
		return tupleScalarSignature{}, err
	}
	if err := a.registerTupleInlineSource(original, source, depth, false); err != nil {
		return tupleScalarSignature{}, err
	}
	var nullable bool
	var height int
	if a.tupleSourceValidation {
		_, nullable, _ = schemaType(terminal, pointer)
		if original != terminal {
			target, err := a.resolveSchema(text(get(original, "$ref")), source)
			if err != nil {
				return tupleScalarSignature{}, err
			}
			ty, err := a.referenceType(target, nullableOverride(original), depth, source)
			if err != nil {
				return tupleScalarSignature{}, err
			}
			nullable = ty.Kind == Nullable
		}
	} else {
		fact, ok := a.tupleObjects.Proof.Collections[original]
		if !ok || fact.Pointer != source {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
		nullable, height = fact.Nullable, fact.Height
	}
	a.tupleObjects.Heights[original] = height
	return tupleScalarSignature{Kind: "collection", Nullable: nullable, ModelSource: terminal}, nil
}

// Charge each distinct positional use for its reachable original schema graph.
// Local sharing/nominal recursion terminates at original node identity; separate
// uses pay overlapping work. Only schema edges are followed: default/const/enum
// objects and metadata may contain JSON keys named $ref without being references.
func (a *analyzer) captureTupleCollectionGraph(original *document.Value, source string) *diagnostic.Diagnostic {
	if a.tupleObjects.CollectionCaptured[original] {
		return nil
	}
	seen := map[*document.Value]bool{}
	var walk func(*document.Value, string, bool) *diagnostic.Diagnostic
	walk = func(v *document.Value, pointer string, capture bool) *diagnostic.Diagnostic {
		if v == nil || seen[v] {
			return nil
		}
		seen[v] = true
		if err := a.tupleAllOfSources.charge(1+len(v.Members())+len(v.Items()), 0, source); err != nil {
			return err
		}
		if capture {
			bytes, _ := v.MarshalJSON()
			if err := a.tupleAllOfSources.charge(0, len(bytes), source); err != nil {
				return err
			}
			if err := a.captureTupleNativeSource(v, pointer, v, pointer); err != nil {
				return err
			}
		}
		if has(v, "$ref") {
			target, p, err := a.defaultSource(text(get(v, "$ref")), pointer)
			if err != nil {
				return err
			}
			if err := a.captureTupleNativeSource(v, pointer, target, p); err != nil {
				return err
			}
			if err := walk(target, p, true); err != nil {
				return err
			}
		}
		if properties := get(v, "properties"); properties != nil {
			if err := a.tupleAllOfSources.charge(1+len(properties.Members()), 0, source); err != nil {
				return err
			}
			for _, property := range properties.Members() {
				if err := walk(property.Value, naming.Pointer(naming.Pointer(pointer, "properties"), property.Name), false); err != nil {
					return err
				}
			}
		}
		for _, key := range []string{"items", "additionalProperties"} {
			if err := walk(get(v, key), naming.Pointer(pointer, key), false); err != nil {
				return err
			}
		}
		for _, key := range []string{"prefixItems", "allOf", "oneOf", "anyOf"} {
			if entries := get(v, key); entries != nil {
				for i, child := range entries.Items() {
					if err := walk(child, naming.Pointer(naming.Pointer(pointer, key), strconv.Itoa(i)), false); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	if err := walk(original, source, true); err != nil {
		return err
	}
	a.tupleObjects.CollectionCaptured[original] = true
	return nil
}

// Compute original source height before positional projection erases a branch.
// Reference depth follows finite collection sources; nominal DTOs remain
// recursion barriers, as in the existing collection resolver.
func (a *analyzer) tupleCollectionSourceHeight(v *document.Value, pointer string, depth int) (int, *diagnostic.Diagnostic) {
	work := 0
	active := map[*document.Value]bool{}
	var walk func(*document.Value, string, int, bool) (int, *diagnostic.Diagnostic)
	walk = func(v *document.Value, p string, d int, product bool) (int, *diagnostic.Diagnostic) {
		if err := depthCheck(d, p); err != nil {
			return 0, err
		}
		work++
		if work > 16384 {
			return 0, diagnostic.At("E_MODEL_LIMIT", "Tuple collection source work exceeds the supported limit.", p)
		}
		if active[v] {
			return 0, schemaReferenceUnsupported(p)
		}
		active[v] = true
		defer delete(active, v)
		product = product || a.tupleProductUnionCandidate(v)
		if has(v, "$ref") {
			target, q, err := a.defaultSource(text(get(v, "$ref")), p)
			if err != nil {
				return 0, err
			}
			if product || has(target, "$ref") || tupleCollectionCandidate(target) || a.tupleUnionCandidate(target) {
				h, err := walk(target, q, d+1, product)
				return h + 1, err
			}
			return 1, nil
		}
		height := 0
		var edges []unionBranch
		// Preserve every original composition edge, including empty or discarded
		// operands erased by the Cartesian shape planner. Value-member defaults
		// and literal data never become schema edges here.
		if product && has(v, "allOf") {
			for i, member := range get(v, "allOf").Items() {
				at := naming.Pointer(naming.Pointer(p, "allOf"), strconv.Itoa(i))
				h, err := walk(member, at, d+1, true)
				if err != nil {
					return 0, err
				}
				height = max(height, h+1)
			}
			if properties := get(v, "properties"); properties != nil {
				for _, member := range properties.Members() {
					at := naming.Pointer(naming.Pointer(p, "properties"), member.Name)
					h, err := walk(member.Value, at, d+1, false)
					if err != nil {
						return 0, err
					}
					height = max(height, h+1)
				}
			}
			if extra := get(v, "additionalProperties"); extra.Kind() == document.Object {
				h, err := walk(extra, naming.Pointer(p, "additionalProperties"), d+1, false)
				if err != nil {
					return 0, err
				}
				height = max(height, h+1)
			}
			return height, nil
		}
		if unionNominalCandidate(v) {
			view, err := a.readUnion(v, p)
			if err != nil {
				return 0, err
			}
			for _, branch := range view.Branches {
				resolved, at, branchDepth, err := a.resolveUnionBranch(branch, d+1)
				if err != nil {
					return 0, err
				}
				h, err := walk(resolved, at, branchDepth, product)
				if err != nil {
					return 0, err
				}
				height = max(height, branchDepth-d+h)
			}
			return height, nil
		}
		if kind, _, _ := schemaType(v, p); kind == "array" {
			if item := get(v, "items"); item != nil {
				edges = append(edges, unionBranch{Schema: item, Pointer: naming.Pointer(p, "items")})
			}
			if prefix := get(v, "prefixItems"); prefix != nil {
				for i, item := range prefix.Items() {
					edges = append(edges, unionBranch{Schema: item, Pointer: naming.Pointer(naming.Pointer(p, "prefixItems"), strconv.Itoa(i))})
				}
			}
		} else if dictionarySchema(v) {
			if extra := get(v, "additionalProperties"); extra.Kind() != document.Boolean {
				edges = append(edges, unionBranch{Schema: extra, Pointer: naming.Pointer(p, "additionalProperties")})
			}
		} else if directInternalModel(v, p) || composedModelCandidate(v) {
			shape, err := a.modelShape(v, p, d, true)
			if err != nil {
				return 0, err
			}
			for _, f := range append(append([]field{}, shape.Fields...), shape.Discarded...) {
				h, err := walk(f.Schema, f.Pointer, f.Depth, false)
				if err != nil {
					return 0, err
				}
				height = max(height, f.Depth-d+h)
			}
			return height, nil
		}
		for _, edge := range edges {
			h, err := walk(edge.Schema, edge.Pointer, d+1, false)
			if err != nil {
				return 0, err
			}
			height = max(height, h+1)
		}
		return height, nil
	}
	return walk(v, pointer, depth, false)
}

func (a *analyzer) recordTupleCollectionWork(element *TypeRef, decision *arrayItemsDecision) {
	if !a.aliasPlanning || a.tupleObjects == nil || len(a.tupleObjects.Collections)+len(a.tupleObjects.Unions) == 0 || decision == nil {
		return
	}
	height := 0
	for _, prefix := range decision.Prefix {
		if a.tupleObjects.Collections[prefix.Schema] || a.tupleObjects.Unions[prefix.Schema] != nil {
			height = max(height, a.tupleObjects.Heights[prefix.Schema])
		}
	}
	if a.tupleObjects.Collections[decision.Selected] || a.tupleObjects.Unions[decision.Selected] != nil {
		height = max(height, a.tupleObjects.Heights[decision.Selected])
	}
	if height != 0 {
		a.tupleObjects.CollectionWork[element] = max(a.tupleObjects.CollectionWork[element], height)
	}
}

// Ordinary typed structure already accounts for preserved source edges. Add
// only erased work after forward backings are complete, before alias heights
// are cached. Recursion is still diagnosed by the existing alias validator.
func (a *analyzer) finalizeTupleCollectionWork() {
	if a.tupleObjects == nil || len(a.tupleObjects.CollectionWork) == 0 {
		return
	}
	finished := map[*TypeRef]int{}
	active := map[*TypeRef]bool{}
	var height func(*TypeRef, int) int
	height = func(t *TypeRef, depth int) int {
		if t == nil {
			return 0
		}
		if depth > MaxSchemaDepth || active[t] {
			return MaxSchemaDepth + 1
		}
		if h, ok := finished[t]; ok {
			return h
		}
		active[t] = true
		h := t.SchemaSteps
		if t.Kind == Nullable {
			h += height(t.Element, depth)
		} else if t.Kind == List || t.Kind == Map || t.Kind == Named && t.Element != nil {
			h += 1 + height(t.Element, depth+1)
		}
		if required := a.tupleObjects.CollectionWork[t]; required > h {
			t.SchemaSteps += required - h
			h = required
		}
		delete(active, t)
		finished[t] = h
		return h
	}
	for t := range a.tupleObjects.CollectionWork {
		height(t, 0)
	}
}

func (a *analyzer) recordTupleCollectionUse(t *TypeRef, source *document.Value) {
	if _, ok := a.tupleObjects.CollectionUses[t]; !ok {
		a.tupleObjects.CollectionUseOrder = append(a.tupleObjects.CollectionUseOrder, t)
	}
	a.tupleObjects.CollectionUses[t] = tupleCollectionUse{Source: source, Pointer: a.tupleObjects.Seen[source], Snapshots: dtoOriginalGetterTypes(t)}
}

func (a *analyzer) selectTupleCollection(array *document.Value, pointer string, decision *arrayItemsDecision) {
	r := a.tupleObjects
	if r == nil || decision.Projection != nil || !r.Collections[decision.Selected] {
		return
	}
	for _, selection := range r.CollectionSelected {
		if selection.Array == array {
			return
		}
	}
	r.CollectionSelected = append(r.CollectionSelected, tupleInlineAllOfSelection{array, decision.Selected, pointer})
}

func (a *analyzer) tupleCollectionDefaultElement(v *document.Value, pointer string, actual TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if a.tupleSpecials != nil && a.tupleSpecials.Sources[v] != nil {
		return a.tupleSpecialDefaultElement(v, pointer, actual)
	}
	if a.tupleCovers != nil && a.tupleCovers.Sources[v] != nil {
		return a.tupleReferenceDefaultElement(v, pointer, actual, depth)
	}
	if a.tupleObjects != nil && a.tupleObjects.Unions[v] != nil {
		return a.tupleUnionDefaultElement(v, pointer, actual, depth)
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	validation, err := a.objectDefaultStep(pointer)
	if err != nil {
		return nil, err
	}
	if has(v, "$ref") {
		if err := checkKeys(v, a.tupleSourceKeys(v, pointer, []string{"$ref", "nullable"}), pointer); err != nil {
			return nil, err
		}
		target, p, err := a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, err
		}
		return validation.tupleCollectionDefaultElement(target, p, actual, depth+1)
	}
	if !tupleCollectionCandidate(v) {
		return validation.tupleReferenceDefaultElement(v, pointer, actual, depth)
	}
	ty := defaultBacking(actual)
	bare := ty.WithoutRootNull()
	if bare.Kind == JSONObject && dictionarySchema(v) {
		open, _ := get(v, "additionalProperties").Bool()
		if !open {
			return nil, schemaReferenceUnsupported(pointer)
		}
		if _, err := validation.dictionaryValue(v, pointer); err != nil {
			return nil, err
		}
		return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, rawJSON: true}, nil
	}
	if (bare.Kind != List && bare.Kind != Map) || bare.Element == nil {
		return nil, schemaReferenceUnsupported(pointer)
	}
	child, p := get(v, "items"), naming.Pointer(pointer, "items")
	var decision *arrayItemsDecision
	if bare.Kind == Map {
		child, p = get(v, "additionalProperties"), naming.Pointer(pointer, "additionalProperties")
	} else {
		decision, err = validation.arrayItemsOwnDefault(v, pointer, depth, a.tupleSourceOwnDefault(v, pointer))
		if err != nil {
			return nil, err
		}
		child, p = decision.Selected, decision.Pointer
	}
	var leaf *nestedDefaultNode
	if decision != nil && decision.Projection != nil {
		leaf = &nestedDefaultNode{schema: v, pointer: p, ty: *bare.Element, rawJSON: true}
	} else {
		leaf, err = validation.tupleCollectionDefaultElement(child, p, *bare.Element, depth+1)
		if err != nil {
			return nil, err
		}
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: leaf, dtoUniqueKeys: a.dtoUniqueKeys}, nil
}

func (a *analyzer) tupleCollectionFieldDefault(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	if nested == nil {
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	element, err := a.collectionElementType(decision.Selected, name+"Inner", decision.Pointer, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	child, err := a.tupleCollectionDefaultElement(decision.Selected, decision.Pointer, element, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	ty, err := a.checkedArrayItems(element, v, pointer, decision)
	if err != nil {
		return TypeRef{}, "", err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	if nullable {
		ty = ty.Null()
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: child, dtoUniqueKeys: a.dtoUniqueKeys}
	value, at := get(v, "default"), naming.Pointer(pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return TypeRef{}, "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return TypeRef{}, "", err
	}
	if len(encoded) > maxNestedDefaultBytes {
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical tuple collection default exceeds the supported 32 KiB limit.", at)
	}
	if value.Kind() == document.Null {
		return ty, "null", nil
	}
	if tupleCollectionNominalLeaf(ty) {
		return ty, "`openapi-kt default enum tree`(" + naming.Literal(encoded) + ")", nil
	}
	initializer, err := referencedDefaultInitializer(node, ty, value, encoded, at)
	return ty, initializer, err
}

func tupleCollectionNominalLeaf(t TypeRef) bool {
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if t.Kind == Named {
			return true
		}
		if t.Element == nil || t.Kind != Nullable && t.Kind != List && t.Kind != Map {
			return false
		}
		t = *t.Element
	}
	return false
}

func (a *analyzer) tupleCollectionFact(source *document.Value) (tupleCollectionFact, bool) {
	r := a.tupleObjects.Proof
	if a.tupleSourceValidation {
		r = a.tupleNativeSources
	}
	if r == nil {
		return tupleCollectionFact{}, false
	}
	fact, ok := r.Collections[source]
	return fact, ok
}

func tupleCollectionUseMatches(t *TypeRef, use tupleCollectionUse) bool {
	if !use.Final || len(use.Snapshots) == 0 || use.Snapshots[0].Node != t {
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

func (a *analyzer) finalizeTupleCollectionUses() *diagnostic.Diagnostic {
	if a.tupleObjects == nil {
		return nil
	}
	for _, t := range a.tupleObjects.UnionUseOrder {
		use := a.tupleObjects.UnionUses[t]
		if use.Final {
			continue
		}
		if len(use.Snapshots) == 0 || use.Snapshots[0].Node != t {
			return schemaReferenceUnsupported(use.Pointer)
		}
		root := *t
		root.SchemaSteps = 0
		expected := use.Snapshots[0].Value
		if binding := expected.deferredNominal; binding != nil && binding.Name != "" {
			source := a.tupleObjects.Unions[use.Source]
			if source == nil || binding.Schema != source.Terminal || binding.Pointer != source.TerminalPointer || a.deferredNominals[binding.Schema] != binding || expected.Name != "" {
				return schemaReferenceUnsupported(use.Pointer)
			}
			expected.Name, expected.deferredNominal = binding.Name, nil
		}
		if root != expected {
			return schemaReferenceUnsupported(use.Pointer)
		}
		use.Snapshots, use.Final = dtoOriginalGetterTypes(t), true
		a.tupleObjects.UnionUses[t] = use
	}
	for _, t := range a.tupleObjects.CollectionUseOrder {
		use := a.tupleObjects.CollectionUses[t]
		if use.Final {
			continue
		}
		if len(use.Snapshots) == 0 || use.Snapshots[0].Node != t {
			return schemaReferenceUnsupported(use.Pointer)
		}
		root := *t
		root.SchemaSteps = 0
		if root != use.Snapshots[0].Value {
			return schemaReferenceUnsupported(use.Pointer)
		}
		use.Snapshots = dtoOriginalGetterTypes(t)
		use.Final = true
		a.tupleObjects.CollectionUses[t] = use
	}
	return nil
}

func (a *analyzer) tupleCollectionDeferred(v *document.Value) bool {
	if !has(v, "prefixItems") {
		return false
	}
	var members []*document.Value
	if prefix := get(v, "prefixItems"); prefix != nil {
		members = append(members, prefix.Items()...)
	}
	members = append(members, get(v, "items"))
	for _, member := range members {
		seen := map[*document.Value]bool{}
		current := member
		for steps := 0; steps <= MaxSchemaDepth && has(current, "$ref"); steps++ {
			if seen[current] {
				break
			}
			seen[current] = true
			next, _, err := a.defaultSource(text(get(current, "$ref")), "")
			if err != nil {
				break
			}
			if tupleCollectionCandidate(next) || a.tupleUnionCandidate(next) || a.hierarchySources[next] {
				return true
			}
			current = next
		}
	}
	return false
}

func (a *analyzer) planTupleCollectionFieldDefault(plan referencedFieldDefault) (bool, *diagnostic.Diagnostic) {
	if !a.tupleCollectionDeferred(plan.Field.Schema) {
		return false, nil
	}
	validation := *a
	validation.warnings = nil
	v, pointer := plan.Field.Schema, plan.Field.Pointer
	decision, err := validation.arrayItemsOwnDefault(v, pointer, plan.Field.Depth, true)
	if err != nil {
		return true, err
	}
	var initializer string
	if decision.Projection != nil {
		_, initializer, err = validation.rawTupleFieldDefault(v, pointer, decision)
	} else {
		ty := plan.Type
		bare := ty.WithoutRootNull()
		if bare.Kind != List || bare.Element == nil {
			return true, schemaReferenceUnsupported(pointer)
		}
		child, failure := validation.tupleCollectionDefaultElement(decision.Selected, decision.Pointer, *bare.Element, plan.Field.Depth+1)
		if failure != nil {
			return true, failure
		}
		node := &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: child, dtoUniqueKeys: a.dtoUniqueKeys}
		value, at := get(v, "default"), naming.Pointer(pointer, "default")
		work, bytes := 0, 0
		if failure := nestedDefaultBudget(value, at, 0, &work, &bytes); failure != nil {
			return true, failure
		}
		encoded, failure := node.value(value, at)
		if failure != nil {
			return true, failure
		}
		if len(encoded) > maxNestedDefaultBytes {
			return true, diagnostic.At("E_MODEL_LIMIT", "Canonical tuple collection default exceeds the supported 32 KiB limit.", at)
		}
		if value.Kind() == document.Null {
			initializer = "null"
		} else if tupleCollectionNominalLeaf(ty) {
			initializer = "`openapi-kt default enum tree`(" + naming.Literal(encoded) + ")"
		} else if child.rawJSON && child.ty.WithoutRootNull().Kind == JSON {
			name := "JsonElement"
			if child.ty.Kind == Nullable {
				name = "Nullable JsonElement"
			}
			initializer = "`openapi-kt default array " + name + "`(" + naming.Literal(encoded) + ")"
		} else {
			initializer, err = referencedDefaultInitializer(node, ty, value, encoded, at)
		}
	}
	if err != nil {
		return true, err
	}
	if plan.Model != nil {
		property := &plan.Model.Properties[plan.Index]
		if property.DefaultNull && property.Type.Kind != PatchValue {
			property.Default = initializer
		}
	}
	return true, nil
}

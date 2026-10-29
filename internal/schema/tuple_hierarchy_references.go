package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// The named hierarchy planner already owns every physical declaration. Capture
// actual parent/child dependencies, rather than creating a tuple-owned graph.
func (a *analyzer) admitTupleHierarchyGraph(target *resolvedSchema, source string) (bool, *diagnostic.Diagnostic) {
	if !a.admitBorrowedHierarchy(target, source) {
		return false, nil
	}
	actual := map[string]*Hierarchy{}
	for d, proof := range a.tupleObjects.UnionDeclarations {
		if h, ok := d.(*Hierarchy); ok && proof.Source == a.hierarchyOrigins[h].Schema {
			actual[a.options.PackageName+".models."+h.Name()] = h
		}
	}
	h := actual[a.options.PackageName+".models."+target.Catalog.Name]
	if h == nil {
		return false, nil
	}
	seen := map[*Hierarchy]bool{}
	for h.Parent != "" {
		if seen[h] {
			return false, nil
		}
		seen[h] = true
		h = actual[h.Parent]
		if h == nil {
			return false, nil
		}
	}
	pending := []*Hierarchy{h}
	seen = map[*Hierarchy]bool{}
	for len(pending) > 0 {
		h = pending[0]
		pending = pending[1:]
		if seen[h] {
			return false, nil
		}
		seen[h] = true
		proof := a.tupleObjects.UnionDeclarations[h]
		c := a.componentsByOriginalSchema(proof.Source)
		if c == nil || c.Pointer != proof.Pointer || c.Name != h.Name() {
			return false, nil
		}
		leaf, err := a.resolveSchema("#"+c.Pointer, source)
		if err != nil || !a.admitBorrowedHierarchy(leaf, source) {
			return false, nil
		}
		if err := a.captureTupleNativeSource(c.Schema, c.Pointer, c.Schema, c.Pointer); err != nil {
			return false, err
		}
		if err := a.captureTupleCollectionGraph(c.Schema, c.Pointer); err != nil {
			return false, err
		}
		// Source order of declarations, rather than map iteration, determines
		// traversal and which original budget diagnostic wins.
		for _, entry := range a.componentSchemas.Members() {
			child := a.components[entry.Name]
			if child == nil {
				continue
			}
			if next := actual[a.options.PackageName+".models."+child.Name]; next != nil && next.Parent == a.options.PackageName+".models."+h.Name() {
				pending = append(pending, next)
			}
		}
	}
	return true, nil
}

// A retained hierarchy classifier is a nominal recursion barrier. Only erased
// use-site reference/alias hops enter the consuming container's cached height;
// the hierarchy planner separately checks every original dispatch graph.
func (a *analyzer) tupleHierarchyReferenceHeight(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (int, *diagnostic.Diagnostic) {
	v, p, steps := original, source, 0
	for has(v, "$ref") {
		if err := depthCheck(depth+steps, p); err != nil {
			return 0, err
		}
		next, at, err := a.defaultSource(text(get(v, "$ref")), p)
		if err != nil {
			return 0, err
		}
		v, p, steps = next, at, steps+1
		if steps > MaxSchemaDepth {
			return 0, schemaReferenceUnsupported(source)
		}
	}
	if v != terminal || p != pointer || !a.hierarchySources[v] {
		return 0, schemaReferenceUnsupported(source)
	}
	if err := depthCheck(depth+steps, p); err != nil {
		return 0, err
	}
	return steps, nil
}

func (a *analyzer) tupleHierarchyByName(name string) *Hierarchy {
	name = strings.TrimPrefix(name, a.options.PackageName+".models.")
	for d, proof := range a.tupleObjects.UnionDeclarations {
		if h, ok := d.(*Hierarchy); ok && h.Name() == name && a.hierarchyOrigins[h].Schema == proof.Source && tupleUnionDeclarationMatches(h, proof.Snapshot) {
			return h
		}
	}
	return nil
}

func (a *analyzer) validateTupleHierarchyGraphPlan(plan *Plan) *diagnostic.Diagnostic {
	active := false
	if a.tupleObjects != nil {
		for _, source := range a.tupleObjects.Unions {
			active = active || source.Fact.Hierarchy
		}
	}
	if !active {
		return nil
	}
	if err := a.validateBorrowedHierarchyLeaves(plan.Models); err != nil {
		return err
	}
	for _, entry := range a.componentSchemas.Members() {
		c := a.components[entry.Name]
		if c == nil {
			continue
		}
		use := a.borrowedHierarchyLeaves[c.Schema]
		if use == nil {
			continue
		}
		found := false
		for d, proof := range a.tupleObjects.UnionDeclarations {
			h, ok := d.(*Hierarchy)
			if !ok || proof.Source != c.Schema {
				continue
			}
			if found || !tupleUnionDeclarationMatches(h, proof.Snapshot) || a.hierarchyOrigins[h] != proof.Origin {
				return schemaReferenceUnsupported(use.Source)
			}
			for _, variant := range proof.Variants {
				if variant.Actual.ClassName != variant.Name || a.modelOrigins[variant.Actual] != variant.Schema || a.modelPointers[variant.Actual] != variant.Pointer {
					return schemaReferenceUnsupported(use.Source)
				}
			}
			found = true
		}
		if !found {
			return schemaReferenceUnsupported(use.Source)
		}
	}
	return nil
}

func (a *analyzer) tupleHierarchyDefaultEncoded(value *document.Value, at string, hierarchy *Hierarchy, depth int) (string, *diagnostic.Diagnostic) {
	if value.Kind() != document.Object {
		return "", invalid("A hierarchy default must select a known object leaf.", at)
	}
	current := hierarchy
	closed := false
	seen := map[*Hierarchy]bool{}
	for current.Leaf == nil {
		if err := depthCheck(depth, at); err != nil {
			return "", err
		}
		if seen[current] {
			return "", schemaReferenceUnsupported(at)
		}
		seen[current] = true
		selected := ""
		if current.Content() {
			for _, rule := range current.TagCombinations {
				match := true
				for _, field := range rule.Matches {
					wire, ok := unionTagContent(get(value, field.Wire))
					found := false
					if ok {
						for _, candidate := range field.Values {
							found = found || candidate == wire
						}
					}
					match = match && found
				}
				if match {
					selected = rule.Child
					break
				}
			}
			if selected == "" {
				if wire, ok := unionTagContent(get(value, current.TagField)); ok {
					for _, rule := range current.Tags {
						if rule.Value == wire {
							selected = rule.Child
							break
						}
					}
				}
			}
		} else {
			// Default sealed polymorphism dispatches on the actual leaf serial
			// name, which can differ from intermediate mapping names.
			closed = true
			wire, ok := get(value, "type").Text()
			if !ok {
				return "", invalid("A closed hierarchy default requires its class discriminator.", at)
			}
			for d := range a.tupleObjects.UnionDeclarations {
				leaf, ok := d.(*Hierarchy)
				if !ok || leaf.Leaf == nil {
					continue
				}
				tag := a.options.PackageName + ".models." + leaf.Name()
				if leaf.SerialName != nil {
					tag = *leaf.SerialName
				}
				if wire != tag {
					continue
				}
				ancestor := leaf
				for step := 0; step <= MaxSchemaDepth && ancestor != nil; step++ {
					if ancestor == current {
						if selected != "" {
							return "", schemaReferenceUnsupported(at)
						}
						selected = leaf.Name()
						break
					}
					ancestor = a.tupleHierarchyByName(ancestor.Parent)
				}
			}
		}
		if selected == "" {
			return "", invalid("A hierarchy default must select a known leaf; Unknown is not a default variant.", at)
		}
		next := a.tupleHierarchyByName(selected)
		if next == nil {
			return "", schemaReferenceUnsupported(at)
		}
		current = next
		depth++
	}
	proof, known := a.tupleObjects.UnionDeclarations[current]
	if !known || !tupleUnionDeclarationMatches(current, proof.Snapshot) || a.hierarchyOrigins[current] != proof.Origin || a.modelOrigins[current.Leaf] != proof.Source || a.modelPointers[current.Leaf] != proof.Pointer {
		return "", schemaReferenceUnsupported(at)
	}
	shape, err := a.modelShape(proof.Source, proof.Pointer, depth, true)
	if err != nil {
		return "", err
	}
	node, err := a.objectDefaultShapeNode(shape, proof.Pointer, true)
	if err != nil {
		return "", err
	}
	for _, field := range current.Fields {
		var vocabulary *StringEnum
		for d := range a.tupleObjects.UnionDeclarations {
			if h, ok := d.(*Hierarchy); ok {
				for _, enum := range h.Enums {
					if field.Type.Name == a.options.PackageName+".models."+enum.Name() {
						vocabulary = enum
					}
				}
			}
		}
		if vocabulary == nil {
			return "", schemaReferenceUnsupported(at)
		}
		property := -1
		for i := range node.objectFields {
			if node.objectFields[i].wire == field.WireName {
				property = i
				break
			}
		}
		if property < 0 {
			node.objectFields = append(node.objectFields, objectDefaultProperty{wire: field.WireName, node: &nestedDefaultNode{ty: TypeRef{Kind: Scalar, Name: "kotlin.String"}}})
			property = len(node.objectFields) - 1
		}
		node.objectFields[property].node.hierarchyTag = vocabulary
		if literal, ok := current.tagBodies[field.WireName]; ok {
			node.objectFields[property].node.hierarchyTagValue = &literal
		}
		for _, actual := range current.Leaf.Properties {
			if actual.WireName == field.WireName && actual.Default == "" && !actual.DefaultNull {
				node.objectFields[property].required = true
			}
		}
	}
	if closed {
		for _, field := range node.objectFields {
			if field.wire == "type" {
				return "", unsupported(at)
			}
		}
		node.objectFields = append(node.objectFields, objectDefaultProperty{wire: "type", required: true, node: &nestedDefaultNode{ty: TypeRef{Kind: Scalar, Name: "kotlin.String"}}})
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return "", err
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical hierarchy default exceeds the supported 32 KiB limit.", at)
	}
	return encoded, nil
}

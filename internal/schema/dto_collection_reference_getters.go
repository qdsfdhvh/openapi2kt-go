package schema

// Record original retained getters after final nominal/alias linking. This
// registry also covers a source first used by a later HTTP uniqueItems planner;
// a scratch key type cannot authorize its validation objects. It stays outside
// model identity and carries immutable check values as well as node addresses.
func (a *analyzer) recordDTOCollectionReferenceGetters(model *Model) {
	source := a.modelOrigins[model]
	if source == nil {
		return
	}
	fields, _, err := a.dtoUniqueOriginalFields(source, a.modelPointers[model])
	if err != nil {
		return
	}
	if composedModelCandidate(source) {
		a.recordDTOAllOfGetters(model, fields)
	}
	byWire := map[string]field{}
	for _, f := range fields {
		byWire[f.Wire] = f
	}
	for i := range model.Properties {
		p := &model.Properties[i]
		original := byWire[p.WireName].Schema
		if !dtoUniqueCollectionTreeHasReference(original, 0) || !dtoUniqueContainerType(p.Type) {
			continue
		}
		if a.dtoReferenceGetterTypes == nil {
			a.dtoReferenceGetterTypes = map[*Model]map[string][]namedCollectionSourceType{}
		}
		if a.dtoReferenceGetterTypes[model] == nil {
			a.dtoReferenceGetterTypes[model] = map[string][]namedCollectionSourceType{}
		}
		if snapshots := dtoOriginalGetterTypes(&p.Type); len(snapshots) != 0 {
			a.dtoReferenceGetterTypes[model][p.WireName] = snapshots
		}
	}
}

func (a *analyzer) dtoCollectionReferenceGetterMatches(model *Model, property Property) bool {
	snapshots := a.dtoReferenceGetterTypes[model][property.WireName]
	if len(snapshots) == 0 {
		return false
	}
	for _, snapshot := range snapshots {
		value := *snapshot.Node
		value.SchemaSteps = 0
		if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
			return false
		}
	}
	// The retained property must still have the same complete physical tree;
	// replacing its root by a copied value must not redirect the proof elsewhere.
	return dtoUniqueCollectionReferenceTypeSignature(property.Type) == dtoUniqueCollectionReferenceTypeSignature(*snapshots[0].Node)
}

// A transparent getter tree has one child per node. Capture both actual node
// identity and immutable validation values after original nominal links settle.
func dtoOriginalGetterTypes(root *TypeRef) []namedCollectionSourceType {
	snapshots := []namedCollectionSourceType{}
	seen := map[*TypeRef]bool{}
	for node, depth := root, 0; node != nil; {
		if depth > MaxSchemaDepth || seen[node] {
			return nil
		}
		seen[node] = true
		value := *node
		value.SchemaSteps = 0
		snapshots = append(snapshots, namedCollectionSourceType{node, value, namedCollectionChecks(value)})
		if node.Kind != Nullable {
			depth++
		}
		node = node.Element
	}
	return snapshots
}

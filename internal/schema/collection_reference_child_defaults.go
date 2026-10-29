package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type collectionReferenceChildDefault struct {
	Schema, Parent, Reference, Value, Target *document.Value
	TargetCatalog                            *component
	Pointer, TargetPointer, ValueBytes       string
	Encoded                                  string
	Uses                                     []*collectionReferenceChildDefaultUse
	FinalUseCount                            int
}

type collectionReferenceChildDefaultUse struct {
	Type        TypeRef
	Actual      *TypeRef
	Depth       int
	FinalDepth  int
	TypeProof   tupleCollectionUse
	ActualProof tupleCollectionUse
	Retained    bool
}

// Resolve only original physical paths; an equal copied node or a tuple
// projection must not acquire a collection-child annotation role.
func (a *analyzer) collectionReferenceChildParent(v *document.Value, pointer string) *document.Value {
	at := strings.LastIndex(pointer, "/")
	if a.root == nil || at < 0 {
		return nil
	}
	edge := pointer[at+1:]
	if edge != "items" && edge != "additionalProperties" {
		return nil
	}
	parent := a.root
	for _, raw := range strings.Split(strings.TrimPrefix(pointer[:at], "/"), "/") {
		key, err := naming.DecodePointerSegment(raw, pointer)
		if err != nil || parent == nil {
			return nil
		}
		switch parent.Kind() {
		case document.Object:
			parent = get(parent, key)
		case document.Array:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(parent.Items()) {
				return nil
			}
			parent = parent.Items()[index]
		default:
			return nil
		}
	}
	if parent == nil || get(parent, edge) != v {
		return nil
	}
	if edge == "additionalProperties" {
		if dictionarySchema(parent) {
			return parent
		}
		return nil
	}
	kind, _, err := schemaType(parent, pointer[:at])
	if err == nil && kind == "array" && !has(parent, "prefixItems") {
		return parent
	}
	return nil
}

func (a *analyzer) collectionReferenceChildDefault(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if !has(v, "$ref") || !has(v, "default") || a.collectionReferenceChildParent(v, pointer) == nil {
		return nil, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := checkKeys(v, []string{"$ref", "nullable", "default"}, pointer); err != nil {
		return nil, err
	}
	if _, _, err := schemaType(v, pointer); err != nil {
		return nil, err
	}
	reference, ok := get(v, "$ref").Text()
	if !ok {
		return nil, invalid("$ref must be a string.", pointer)
	}
	target, err := a.resolveSchema(reference, pointer)
	if err != nil {
		return nil, err
	}
	// This role only extends existing lexical and native enum references.
	// In particular, a default on a reference to an ordinary scalar, DTO, or
	// collection must still go through its existing admission rules.
	if !originalEnumWrapperDefaultSource(target.Schema) {
		return nil, nil
	}
	ty, err := a.referenceType(target, nullableOverride(v), depth, pointer)
	if err != nil {
		return nil, err
	}
	_, encoded, err := a.referencedDefaultValue(field{Schema: v, Pointer: pointer, Depth: depth}, ty)
	if err != nil {
		return nil, err
	}
	if a.collectionChildDefaults == nil {
		a.collectionChildDefaults = map[*document.Value]*collectionReferenceChildDefault{}
	}
	if a.collectionChildDefaultOrder == nil {
		a.collectionChildDefaultOrder = &[]*collectionReferenceChildDefault{}
	}
	source := a.collectionChildDefaults[v]
	if source == nil {
		if err := a.captureTupleNativeSource(v, pointer, target.Schema, target.Pointer); err != nil {
			return nil, err
		}
		value := get(v, "default")
		if err := a.captureTupleNativePath(value, naming.Pointer(pointer, "default"), pointer); err != nil {
			return nil, err
		}
		raw, _ := value.MarshalJSON()
		if err := a.tupleNativeSources.charge(0, len(raw), pointer); err != nil {
			return nil, err
		}
		source = &collectionReferenceChildDefault{
			Schema: v, Parent: a.collectionReferenceChildParent(v, pointer),
			Reference: get(v, "$ref"), Value: value, Pointer: pointer,
			Target: target.Schema, TargetPointer: target.Pointer,
			TargetCatalog: target.Catalog, ValueBytes: string(raw), Encoded: encoded,
		}
		a.collectionChildDefaults[v] = source
		*a.collectionChildDefaultOrder = append(*a.collectionChildDefaultOrder, source)
	} else if !a.validatedCollectionReferenceChildDefault(v, pointer) ||
		source.Target != target.Schema || source.TargetPointer != target.Pointer ||
		source.TargetCatalog != target.Catalog || source.Encoded != encoded {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if err := a.tupleNativeSources.charge(1, 0, pointer); err != nil {
		return nil, err
	}
	source.Uses = append(source.Uses, &collectionReferenceChildDefaultUse{Type: ty, Depth: depth})
	return &ty, nil
}

func (a *analyzer) validatedCollectionReferenceChildDefault(v *document.Value, pointer string) bool {
	record, ok := a.collectionChildDefaults[v]
	return ok && record != nil && record.Schema == v && record.Pointer == pointer &&
		record.Parent == a.collectionReferenceChildParent(v, pointer) &&
		record.Reference == get(v, "$ref") && record.Value == get(v, "default")
}

// Compare the linked type and its actual validation objects without requiring
// separately allocated nullable wrappers to have the same address.
func collectionChildDefaultTypesMatch(left, right TypeRef) bool {
	if dtoUniqueCollectionReferenceTypeSignature(left) != dtoUniqueCollectionReferenceTypeSignature(right) {
		return false
	}
	for depth := 0; ; depth++ {
		if depth > MaxSchemaDepth || left.Validation != right.Validation || left.Collection != right.Collection {
			return false
		}
		if left.Element == nil || right.Element == nil {
			return left.Element == nil && right.Element == nil
		}
		left, right = *left.Element, *right.Element
	}
}

// The element address belongs to the real List/Map planner. Keep it separate
// from the preflight type copy so a later replacement or changed checks cannot
// silently reuse the annotation's source permission.
func (a *analyzer) bindCollectionReferenceChildDefault(v *document.Value, pointer string, actual *TypeRef) *diagnostic.Diagnostic {
	source := a.collectionChildDefaults[v]
	if source == nil {
		return nil
	}
	if !a.validatedCollectionReferenceChildDefault(v, pointer) || actual == nil {
		return schemaReferenceUnsupported(pointer)
	}
	for i := len(source.Uses) - 1; i >= 0; i-- {
		use := source.Uses[i]
		if use.Actual != nil {
			continue
		}
		if !collectionChildDefaultTypesMatch(use.Type, *actual) {
			return schemaReferenceUnsupported(pointer)
		}
		use.Actual = actual
		return nil
	}
	return schemaReferenceUnsupported(pointer)
}

func (a *analyzer) collectionChildDefaultTarget(source *collectionReferenceChildDefault, use *collectionReferenceChildDefaultUse) *diagnostic.Diagnostic {
	if source == nil {
		return schemaReferenceUnsupported("")
	}
	if use == nil || !a.validatedCollectionReferenceChildDefault(source.Schema, source.Pointer) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	raw, _ := source.Value.MarshalJSON()
	if string(raw) != source.ValueBytes {
		return schemaReferenceUnsupported(source.Pointer)
	}
	target, err := a.resolveSchema(text(source.Reference), source.Pointer)
	if err != nil {
		return err
	}
	if target.Schema != source.Target || target.Pointer != source.TargetPointer || target.Catalog != source.TargetCatalog || !originalEnumWrapperDefaultSource(target.Schema) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	ty, err := a.referenceType(target, nullableOverride(source.Schema), use.Depth, source.Pointer)
	if err != nil {
		return err
	}
	if !collectionChildDefaultTypesMatch(use.Type, ty) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	_, encoded, err := a.referencedDefaultValue(field{Schema: source.Schema, Pointer: source.Pointer, Depth: use.Depth}, use.Type)
	if err != nil {
		return err
	}
	if encoded != source.Encoded {
		return schemaReferenceUnsupported(source.Pointer)
	}
	return nil
}

// Operation planning can add further uses after component linking, so freeze
// every original use only after both component and operation planning finish.
func (a *analyzer) finalizeCollectionReferenceChildDefaults(declarations []Declaration, client *Client) *diagnostic.Diagnostic {
	if a.collectionChildDefaultOrder == nil {
		return nil
	}
	a.collectionChildDefaultClient = client
	retained := collectionChildDefaultRetainedTypes(declarations, client)
	for _, source := range *a.collectionChildDefaultOrder {
		if source == nil {
			return schemaReferenceUnsupported("")
		}
		if len(source.Uses) == 0 || a.collectionChildDefaults[source.Schema] != source {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, use := range source.Uses {
			if use.Actual == nil || !collectionChildDefaultTypesMatch(use.Type, *use.Actual) {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if err := a.collectionChildDefaultTarget(source, use); err != nil {
				return err
			}
			use.FinalDepth = use.Depth
			use.TypeProof = tupleCollectionUse{Source: source.Schema, Pointer: source.Pointer, Snapshots: dtoOriginalGetterTypes(&use.Type), Final: true}
			use.ActualProof = tupleCollectionUse{Source: source.Schema, Pointer: source.Pointer, Snapshots: dtoOriginalGetterTypes(use.Actual), Final: true}
			use.Retained = retained[use.Actual]
		}
		source.FinalUseCount = len(source.Uses)
	}
	return nil
}

func (a *analyzer) validateCollectionReferenceChildDefaults(declarations []Declaration) *diagnostic.Diagnostic {
	if a.collectionChildDefaultOrder == nil {
		return nil
	}
	retained := collectionChildDefaultRetainedTypes(declarations, a.collectionChildDefaultClient)
	for _, source := range *a.collectionChildDefaultOrder {
		if source == nil {
			return schemaReferenceUnsupported("")
		}
		if a.collectionChildDefaults[source.Schema] != source || source.FinalUseCount == 0 || source.FinalUseCount != len(source.Uses) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, use := range source.Uses {
			if use == nil || use.Actual == nil || use.Retained && !retained[use.Actual] || use.Depth != use.FinalDepth || use.TypeProof.Source != source.Schema || use.TypeProof.Pointer != source.Pointer || !tupleCollectionUseMatches(&use.Type, use.TypeProof) ||
				use.ActualProof.Source != source.Schema || use.ActualProof.Pointer != source.Pointer || !tupleCollectionUseMatches(use.Actual, use.ActualProof) || !collectionChildDefaultTypesMatch(use.Type, *use.Actual) {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if err := a.collectionChildDefaultTarget(source, use); err != nil {
				return err
			}
		}
	}
	return nil
}

// Scratch validation may see the same original schema, but only addresses in
// returned declarations and operations are retained for output.
func collectionChildDefaultRetainedTypes(declarations []Declaration, client *Client) map[*TypeRef]bool {
	retained := map[*TypeRef]bool{}
	var typeRef func(*TypeRef)
	typeRef = func(ty *TypeRef) {
		if ty == nil || retained[ty] {
			return
		}
		retained[ty] = true
		typeRef(ty.Element)
	}
	var declaration func(Declaration)
	declaration = func(d Declaration) {
		switch d := d.(type) {
		case *Model:
			for i := range d.Properties {
				typeRef(&d.Properties[i].Type)
			}
			for i := range d.Constants {
				typeRef(&d.Constants[i].Type)
			}
			for _, nested := range d.Nested {
				declaration(nested)
			}
		case *ValueAlias:
			typeRef(&d.Type)
		case *ObjectUnion:
			for i := range d.Common {
				typeRef(&d.Common[i].Type)
			}
			for _, variant := range d.Variants {
				declaration(variant)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				declaration(d.Object)
			}
		case *Hierarchy:
			for i := range d.Fields {
				typeRef(&d.Fields[i].Type)
			}
			if d.Leaf != nil {
				declaration(d.Leaf)
			}
		case *SSEEvents:
			for i := range d.Variants {
				typeRef(d.Variants[i].Data)
			}
		}
	}
	for _, d := range declarations {
		declaration(d)
	}
	if client == nil {
		return retained
	}
	for _, d := range client.Models {
		declaration(d)
	}
	for gi := range client.Groups {
		for oi := range client.Groups[gi].Operations {
			op := &client.Groups[gi].Operations[oi]
			typeRef(&op.Response)
			for pi := range op.Parameters {
				typeRef(&op.Parameters[pi].Type)
			}
			if op.Body != nil {
				typeRef(&op.Body.Type)
			}
			for ei := range op.Errors {
				typeRef(op.Errors[ei].Type)
			}
			if op.SSE != nil {
				declaration(op.SSE)
			}
			for _, d := range op.Models {
				declaration(d)
			}
		}
	}
	return retained
}

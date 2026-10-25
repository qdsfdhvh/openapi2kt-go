package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// DTOUniqueKey is enforcing metadata, independent of source provenance. The
// original source and final retained constructor are checked on the analyzer.
type DTOUniqueKey struct {
	Fields []DTOUniqueField
	digest string
}
type DTOUniqueField struct {
	Name string
	Type TypeRef
}

// Hash the enforcing descriptor once. Canonical type fingerprints must not
// expand a shared field graph at every referring collection occurrence.
func (k *DTOUniqueKey) MarshalJSON() ([]byte, error) {
	digest := k.digest
	if digest == "" {
		digest = k.fieldDigest()
	}
	return json.Marshal(struct{ DTOKey string }{digest})
}
func (k *DTOUniqueKey) fieldDigest() string {
	encoded, _ := json.Marshal(k.Fields)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

type dtoUniqueSource struct {
	Composed              bool
	ShapeRefs             []borrowedAllOfShapeRef
	Required              map[string]bool
	Fields                []field
	Defaults              []*string
	DefaultWork           []int
	ContainerInitializers map[int]dtoUniqueContainerInitializer
	ReferenceFields       map[int]*dtoUniqueScalarReference
	Actual                *Model
	EnumActual            map[*document.Value]*StringEnum
	Schema, OwnerSchema   *document.Value
	Owner                 *component
	Pointer, Bytes        string
	Nodes                 []borrowedAllOfNode
	Key                   *DTOUniqueKey
}
type dtoUniqueUse struct {
	DefaultOnly bool
	Element     *TypeRef
	Source      *dtoUniqueSource
	Pointer     string
}
type dtoUniqueDefaultCache struct {
	Key   *DTOUniqueKey
	Value *document.Value
}
type dtoUniqueDefaultKeys struct {
	Sources map[*DTOUniqueKey]*dtoUniqueSource
	Work    int
	Seen    map[dtoUniqueDefaultCache]string
}

func (b *dtoUniqueDefaultKeys) key(key *DTOUniqueKey, v *document.Value, source string) (string, *diagnostic.Diagnostic) {
	if b == nil {
		return "", unsupported(source)
	}
	cache := dtoUniqueDefaultCache{key, v}
	if old, ok := b.Seen[cache]; ok {
		return old, nil
	}
	src := b.Sources[key]
	if src == nil {
		return "", unsupported(source)
	}
	cost := 1
	if v.Kind() != document.Null {
		cost += len(key.Fields)
		for i, field := range key.Fields {
			if !dtoUniqueContainerType(field.Type) {
				continue
			}
			work := src.DefaultWork[i]
			if value := get(v, src.Fields[i].Wire); value != nil {
				work = dtoUniqueLiteralWork(value)
			}
			cost += work - 1
		}
	}
	if cost > 16384-b.Work {
		return "", diagnostic.At("E_MODEL_LIMIT", "DTO uniqueness default keys exceed the cumulative work budget.", source)
	}
	b.Work += cost
	value := dtoUniqueDefaultKey(src, v)
	b.Seen[cache] = value
	return value, nil
}
func (a *analyzer) initDTOUnique() {
	if a.dtoReferenceDeclarationTypes == nil {
		a.dtoReferenceDeclarationTypes = map[*component]string{}
	}
	if a.dtoReferenceEnums == nil {
		a.dtoReferenceEnums = map[*document.Value]*StringEnum{}
	}
	if a.dtoReferenceEnumNames == nil {
		a.dtoReferenceEnumNames = map[*StringEnum]string{}
	}
	if a.dtoReferenceDeclarations == nil {
		a.dtoReferenceDeclarations = map[*component]Declaration{}
	}
	if a.dtoUniqueSources == nil {
		a.dtoUniqueSources = map[*document.Value]*dtoUniqueSource{}
	}
	if a.dtoUniqueCaptures == nil {
		a.dtoUniqueCaptures = &dtoUniqueCaptureBudget{}
	}
	if a.dtoUniqueUses == nil {
		a.dtoUniqueUses = &[]dtoUniqueUse{}
	}
	if a.dtoUniqueKeys == nil {
		a.dtoUniqueKeys = &dtoUniqueDefaultKeys{Seen: map[dtoUniqueDefaultCache]string{}, Sources: map[*DTOUniqueKey]*dtoUniqueSource{}}
	}
}

type dtoUniqueCaptureBudget struct {
	Work, Bytes        int
	ReferenceNodes     map[*document.Value]bool
	ReferenceSnapshots map[*document.Value]borrowedAllOfNode
	ReferenceTrees     map[*document.Value]*dtoUniqueReferenceTree
}

func (b *dtoUniqueCaptureBudget) charge(work, bytes int, source string) *diagnostic.Diagnostic {
	if work > 16384-b.Work || bytes > (8<<20)-b.Bytes {
		return diagnostic.At("E_MODEL_LIMIT", "DTO uniqueness source capture exceeds the cumulative work or byte budget.", source)
	}
	b.Work += work
	b.Bytes += bytes
	return nil
}

// The first DTO key role is a direct ordinary constructor with mapped scalar
// fields. A spelling, projection or provisional binding alone is insufficient.
func (a *analyzer) dtoUniqueSource(v *document.Value, pointer, source string) (*dtoUniqueSource, *diagnostic.Diagnostic) {
	a.initDTOUnique()
	if has(v, "$ref") {
		target, e := a.resolveSchema(text(get(v, "$ref")), naming.Pointer(pointer, "$ref"))
		if e != nil {
			return nil, e
		}
		v, pointer = target.Schema, target.Pointer
	}
	if !strings.HasPrefix(pointer, "/components/schemas/") || (!directInternalModel(v, pointer) && !composedModelCandidate(v)) || a.hierarchySources[v] || hasObjectCounts(v) || has(v, "default") {
		return nil, nil
	}
	target, e := a.lookupSchemaReference("#"+pointer, source)
	if e != nil {
		return nil, e
	}
	owner := target.SourceCatalog
	if target.Catalog != nil {
		owner = target.Catalog
	}
	if target.Schema != v || owner == nil || (!directInternalOwner(owner.Schema) && !composedModelCandidate(owner.Schema)) || a.hierarchySources[owner.Schema] || target.Catalog == nil && (!target.DirectLexical || target.ComposedLexical) && !composedModelCandidate(v) {
		return nil, nil
	}
	if old := a.dtoUniqueSources[v]; old != nil {
		if old.Owner != owner || old.Pointer != pointer {
			return nil, schemaReferenceUnsupported(source)
		}
		return old, nil
	}
	bytes, _ := v.MarshalJSON()
	nodes, shapeRefs, failure := a.borrowedAllOfSourceBudget(v, pointer, source, a.dtoUniqueCaptures)
	if failure != nil {
		return nil, failure
	}
	if failure := a.dtoUniqueCaptures.charge(0, len(bytes), source); failure != nil {
		return nil, failure
	}
	if a.dtoUniqueCaptures.ReferenceNodes == nil {
		a.dtoUniqueCaptures.ReferenceNodes = map[*document.Value]bool{}
	}
	for _, node := range nodes {
		a.dtoUniqueCaptures.ReferenceNodes[node.Node] = true
	}
	fields, required, e := a.dtoUniqueOriginalFields(v, pointer)
	if e != nil {
		// Excluded composition roles retain the historical consuming
		// uniqueItems diagnostic. Invalid original fields keep their locations.
		if composedModelCandidate(v) && e.Code == "E_SCHEMA_UNSUPPORTED" && e.Pointer != nil && *e.Pointer == pointer {
			return nil, nil
		}
		return nil, e
	}
	if len(fields) > MaxMembers {
		return nil, diagnostic.At("E_MODEL_LIMIT", "DTO uniqueness constructor exceeds the supported member limit.", pointer)
	}
	key := &DTOUniqueKey{}
	sourceFields := []field{}
	constructorDefaults := []*string{}
	defaultWork := []int{}
	containerInitializers := map[int]dtoUniqueContainerInitializer{}
	referenceFields := map[int]*dtoUniqueScalarReference{}
	for _, originalField := range fields {
		member := document.Member{Name: originalField.Wire, Value: originalField.Schema}
		p := originalField.Pointer
		if has(member.Value, "oneOf") || has(member.Value, "anyOf") || readOnlySchema(member.Value) || writeOnlySchema(member.Value) {
			return nil, nil
		}
		if has(member.Value, "const") {
			if has(member.Value, "x-native-type") || has(member.Value, "x-kotlin-value-class") {
				return nil, nil
			}
			name, failure := naming.PropertyName(member.Name, p)
			if failure != nil {
				return nil, failure
			}
			value, failure := constant(name, member.Name, member.Value, p, 0)
			if failure != nil {
				return nil, failure
			}
			if !dtoUniqueFieldScalarType(value.Type) {
				return nil, nil
			}
			key.Fields = append(key.Fields, DTOUniqueField{Name: name, Type: dtoUniqueFieldType(value.Type)})
			sourceFields = append(sourceFields, originalField)
			constructorDefaults = append(constructorDefaults, dtoUniqueConstInitializer(value.Type, get(member.Value, "const")))
			defaultWork = append(defaultWork, 1)
			continue
		}
		enum := has(member.Value, "enum")
		stringEnum := enum && a.uniqueStringEnumSource(member.Value, p, 0) || a.dtoUniqueStringEnumReferenceSource(member.Value, p)
		nativeEnum := (enum || has(member.Value, "$ref")) && a.uniqueNativeEnumSource(member.Value, p, 0)
		kind, _, kindFailure := schemaType(member.Value, p)
		nullValue := kindFailure == nil && kind == "null" && !has(member.Value, "x-native-type") && !has(member.Value, "x-kotlin-value-class")
		referenceContainer := a.dtoUniqueBuiltinCollectionReferenceSource(member.Value, p)
		container, containerFailure := a.dtoUniqueContainerCandidate(member.Value, p, 0, true)
		if containerFailure != nil {
			return nil, containerFailure
		}
		if !a.uniqueScalarSource(member.Value, p, 0) && !stringEnum && !nativeEnum && !nullValue && !container && !referenceContainer {
			return nil, nil
		}
		f := originalField
		var nested []Declaration
		var ty TypeRef
		var failure *diagnostic.Diagnostic
		if stringEnum && !has(member.Value, "$ref") {
			if has(member.Value, "default") {
				failure = a.enumDefaultValue(member.Value, p)
			}
			if failure != nil {
				return nil, failure
			}
			_, failure = buildStringEnumKeys("DTOUnique", member.Value, p, has(member.Value, "default"))
			_, nullable, e := schemaType(member.Value, p)
			if e != nil {
				return nil, e
			}
			ty = TypeRef{Kind: Named, Name: dtoUniqueEnumKind}
			if nullable {
				ty = ty.Null()
			}
		} else if container && dtoUniqueStringEnumTreeSource(member.Value) {
			ty, failure = dtoUniqueStringEnumTreeType(member.Value, p, 0)
		} else if nullValue && enum && has(member.Value, "default") {
			ty, failure = a.enumDefaultType(f, "DTOUnique", nil, nil)
		} else if nativeEnum && !has(member.Value, "$ref") {
			ty, _, failure = a.nativeEnumFieldType(f, has(member.Value, "default"))
		} else if has(member.Value, "default") {
			ty, _, failure = a.modelDefaultType(f, true, "DTOUnique", &nested, map[string]bool{})
		} else {
			ty, _, failure = a.modelPropertyType(member.Value, "DTOUnique", p, &nested, map[string]bool{}, f.Depth)
		}
		if failure != nil {
			return nil, failure
		}
		physicalType := ty
		if referenceContainer {
			ty, failure = a.dtoUniqueBuiltinCollectionReferenceMappedType(f, physicalType)
			if failure != nil {
				return nil, failure
			}
		}
		if (!dtoUniqueFieldScalarType(ty) && !stringEnum && !(container && dtoUniqueContainerTreeType(ty, 0)) && !(referenceContainer && dtoUniqueBuiltinCollectionReferenceType(ty, 0))) || len(nested) != 0 {
			return nil, nil
		}
		annotationType := ty
		if !required[member.Name] {
			ty = ty.Null()
		}
		name, failure := naming.PropertyName(member.Name, p)
		if failure != nil {
			return nil, failure
		}
		if has(member.Value, "$ref") || referenceContainer {
			referenceActual := ty
			if referenceContainer {
				referenceActual = physicalType
				if !required[member.Name] {
					referenceActual = referenceActual.Null()
				}
			}
			proof, failure := a.dtoUniqueScalarReference(f, annotationType, referenceActual, required[member.Name], source)
			if failure != nil {
				return nil, failure
			}
			referenceFields[len(key.Fields)] = proof
		}
		field := DTOUniqueField{Name: name, Type: dtoUniqueFieldType(ty)}
		var defaultKey *string
		if value := get(member.Value, "default"); value != nil && !required[member.Name] && value.Kind() != document.Null {
			s := dtoUniqueFieldDefaultKey(ty, value)
			defaultKey = &s
		}
		key.Fields = append(key.Fields, field)
		sourceFields = append(sourceFields, f)
		constructorDefaults = append(constructorDefaults, defaultKey)
		work := 1
		if defaultKey != nil && (container || referenceContainer) {
			work = dtoUniqueLiteralWork(get(member.Value, "default"))
		}
		defaultWork = append(defaultWork, work)
		if referenceContainer {
			containerInitializers[len(key.Fields)-1] = referenceFields[len(key.Fields)-1].Initializer
		}
		if container {
			initializer, failure := a.modelDefault(f, ty, required[member.Name])
			if failure != nil {
				return nil, failure
			}
			nullInitializer := ty.Kind == Nullable
			if has(member.Value, "default") && required[member.Name] {
				nullInitializer = false
			}
			containerInitializers[len(key.Fields)-1] = dtoUniqueContainerInitializer{Literal: initializer, DefaultNull: nullInitializer}
		}
	}
	key.digest = key.fieldDigest()
	capture := &dtoUniqueSource{Composed: composedModelCandidate(v), ShapeRefs: shapeRefs, Required: required, Fields: sourceFields, Defaults: constructorDefaults, DefaultWork: defaultWork, ContainerInitializers: containerInitializers, ReferenceFields: referenceFields, Schema: v, OwnerSchema: owner.Schema, Owner: owner, Pointer: pointer, Bytes: string(bytes), Nodes: nodes, Key: key}
	a.dtoUniqueSources[v] = capture
	a.dtoUniqueKeys.Sources[key] = capture
	return capture, nil
}

// JsonNull keeps its existing typed codec; this local equality role never
// grants JSON/raw/nominal projection authority to other collection planners.
func dtoUniqueFieldScalarType(t TypeRef) bool {
	if uniqueScalarType(t) {
		return true
	}
	if t.Kind == Nullable {
		return dtoUniqueFieldScalarType(*t.Element)
	}
	return t.Kind == Scalar && t.Name == "kotlinx.serialization.json.JsonNull" && t.Serializer == "" && t.Element == nil
}

func dtoUniqueConstInitializer(t TypeRef, v *document.Value) *string {
	if v.Kind() == document.Null {
		return nil
	}
	key := uniqueScalarDefaultKey(t, v)
	return &key
}

const dtoUniqueEnumKind = "enum serialName"

func (a *analyzer) dtoUniqueEnumField(model *Model, parent string, ty TypeRef, f field, src *dtoUniqueSource) bool {
	if !uniqueStringEnumType(ty) || !a.uniqueStringEnumSource(f.Schema, f.Pointer, 0) {
		return false
	}
	count := 0
	var actual *StringEnum
	for _, d := range model.Nested {
		enum, ok := d.(*StringEnum)
		if !ok {
			continue
		}
		origin, ok := a.enumOrigins[enum]
		if ok && origin.Schema == f.Schema && origin.Pointer == f.Pointer && (bare(ty).Name == enum.ClassName || bare(ty).Name == parent+"."+enum.ClassName) {
			count++
			actual = enum
		}
	}
	if count != 1 {
		return false
	}
	expected, failure := buildStringEnumKeys(actual.ClassName, f.Schema, f.Pointer, has(f.Schema, "default"))
	if failure != nil || len(actual.Members) != len(expected.Members) {
		return false
	}
	for i, member := range actual.Members {
		if member != expected.Members[i] {
			return false
		}
	}
	if src.EnumActual == nil {
		src.EnumActual = map[*document.Value]*StringEnum{}
	}
	if previous := src.EnumActual[f.Schema]; previous != nil && previous != actual {
		return false
	}
	src.EnumActual[f.Schema] = actual
	return true
}

func dtoUniqueFieldType(t TypeRef) TypeRef {
	if t.Kind == List || t.Kind == Map {
		child := dtoUniqueFieldType(*t.Element)
		return TypeRef{Kind: t.Kind, Element: &child}
	}

	if t.Kind == Nullable {
		child := dtoUniqueFieldType(*t.Element)
		return child.Null()
	}
	if t.Kind == Named {
		if t.Element != nil {
			return dtoUniqueFieldType(defaultBacking(t))
		}
		return TypeRef{Kind: Named, Name: dtoUniqueEnumKind}
	}
	return TypeRef{Kind: Scalar, Name: t.Name}
}

func (a *analyzer) validateDTOUniqueUses(plan *Plan) *diagnostic.Diagnostic {
	if a.dtoUniqueUses == nil || len(*a.dtoUniqueUses) == 0 {
		return nil
	}
	type occurrence struct {
		Model *Model
		Name  string
	}
	models := map[*document.Value][]occurrence{}
	a.dtoReferenceFinalEnums = map[*StringEnum]int{}
	a.dtoReferenceFinalEnumNames = map[*StringEnum]string{}
	a.dtoReferenceFinalEnumSources = map[*document.Value]int{}
	var visit func(Declaration, string)
	visit = func(d Declaration, parent string) {
		switch x := d.(type) {
		case *StringEnum:
			a.dtoReferenceFinalEnums[x]++
			a.dtoReferenceFinalEnumNames[x] = parent + "." + x.ClassName
			if origin, ok := a.enumOrigins[x]; ok {
				a.dtoReferenceFinalEnumSources[origin.Schema]++
			}
		case *Model:
			name := parent + "." + x.ClassName
			if origin := a.modelOrigins[x]; origin != nil {
				models[origin] = append(models[origin], occurrence{x, name})
			}
			for _, child := range x.Nested {
				visit(child, name)
			}
		case *ObjectUnion:
			for _, child := range x.Variants {
				visit(child, parent+"."+x.ClassName)
			}
		case *PrimitiveUnion:
			if x.Object != nil {
				visit(x.Object, parent)
			}
		case *Hierarchy:
			if x.Leaf != nil {
				visit(x.Leaf, parent)
			}
		}
	}
	for _, d := range plan.Models {
		visit(d, a.options.PackageName+".models")
	}
	if plan.Client != nil {
		for _, d := range plan.Client.Models {
			visit(d, a.options.PackageName+".api.models")
		}
		for _, g := range plan.Client.Groups {
			for _, op := range g.Operations {
				for _, d := range op.Models {
					visit(d, a.options.PackageName+".api.models")
				}
			}
		}
	}
	collectionReferences := false
	for _, use := range *a.dtoUniqueUses {
		for _, proof := range use.Source.ReferenceFields {
			collectionReferences = collectionReferences || proof.Collection
		}
		if collectionReferences {
			break
		}
	}
	if collectionReferences {
		if err := a.validateNamedCollectionBindings(plan.Models); err != nil {
			return err
		}
		if err := a.validateBorrowedCollectionBindings(plan.Models); err != nil {
			return err
		}
	}
	validated := map[*dtoUniqueSource]bool{}
	for _, use := range *a.dtoUniqueUses {
		src := use.Source
		if !validated[src] {
			if src.Key.digest != src.Key.fieldDigest() {
				return unsupported(use.Pointer)
			}
			bytes, _ := src.Schema.MarshalJSON()
			if src.Owner == nil || a.components[src.Owner.Wire] != src.Owner || src.Owner.Schema != src.OwnerSchema || string(bytes) != src.Bytes {
				return unsupported(use.Pointer)
			}
			found, e := a.lookupSchemaReference("#"+src.Pointer, use.Pointer)
			if e != nil || found.Schema != src.Schema || found.Catalog != src.Owner && found.SourceCatalog != src.Owner {
				return unsupported(use.Pointer)
			}
			for _, node := range src.Nodes {
				if len(node.Node.Members()) != len(node.Members) || len(node.Node.Items()) != len(node.Items) {
					return unsupported(use.Pointer)
				}
				for i, m := range node.Members {
					if node.Node.Members()[i] != m || get(node.Node, m.Name) != m.Value {
						return unsupported(use.Pointer)
					}
				}
				for i, item := range node.Items {
					if node.Node.Items()[i] != item {
						return unsupported(use.Pointer)
					}
				}
			}
			if src.Composed && !a.dtoUniqueOriginalShapeMatches(src) {
				return unsupported(use.Pointer)
			}
			all := models[src.Schema]
			if len(all) != 1 {
				return unsupported(use.Pointer)
			}
			actual := all[0]
			if src.Composed && !a.dtoAllOfOriginalGettersMatch(actual.Model) {
				return unsupported(use.Pointer)
			}
			if src.Actual != nil && src.Actual != actual.Model {
				return unsupported(use.Pointer)
			}
			if actual.Model.Directional || actual.Model.PropertyCount != nil || a.modelPointers[actual.Model] != src.Pointer || len(actual.Model.Properties)+len(actual.Model.Constants) != len(src.Key.Fields) || len(src.DefaultWork) != len(src.Key.Fields) {
				return unsupported(use.Pointer)
			}
			propertyIndex, constantIndex := 0, 0
			for i, originalField := range src.Fields {
				if has(originalField.Schema, "const") {
					if src.DefaultWork[i] != 1 {
						return unsupported(use.Pointer)
					}

					if constantIndex >= len(actual.Model.Constants) {
						return unsupported(use.Pointer)
					}
					actualConstant := actual.Model.Constants[constantIndex]
					constantIndex++
					name, failure := naming.PropertyName(originalField.Wire, originalField.Pointer)
					if failure != nil {
						return unsupported(use.Pointer)
					}
					expected, failure := constant(name, originalField.Wire, originalField.Schema, originalField.Pointer, 0)
					if failure != nil || !dtoUniqueFieldScalarType(expected.Type) {
						return unsupported(use.Pointer)
					}
					left, _ := json.Marshal(actualConstant)
					right, _ := json.Marshal(expected)
					keyField := DTOUniqueField{Name: name, Type: dtoUniqueFieldType(actualConstant.Type)}
					fieldBytes, _ := json.Marshal(keyField)
					expectedFieldBytes, _ := json.Marshal(src.Key.Fields[i])
					initializer := dtoUniqueConstInitializer(actualConstant.Type, get(originalField.Schema, "const"))
					if string(left) != string(right) || string(fieldBytes) != string(expectedFieldBytes) || (src.Defaults[i] == nil) != (initializer == nil) || initializer != nil && *src.Defaults[i] != *initializer {
						return unsupported(use.Pointer)
					}
					continue
				}
				if propertyIndex >= len(actual.Model.Properties) {
					return unsupported(use.Pointer)
				}
				p := actual.Model.Properties[propertyIndex]
				propertyIndex++
				f := src.Key.Fields[i]
				left, _ := json.Marshal(dtoUniqueFieldType(p.Type))
				right, _ := json.Marshal(f.Type)
				expectedName, failure := naming.PropertyName(src.Fields[i].Wire, src.Fields[i].Pointer)
				allowed := dtoUniqueFieldScalarType(p.Type) || a.dtoUniqueContainerSource(originalField.Schema, originalField.Pointer, 0, true) && dtoUniqueContainerTreeType(p.Type, 0) && a.dtoUniqueContainerTypeMatches(p.Type, originalField.Schema, originalField.Pointer, 0)
				if proof := src.ReferenceFields[i]; proof != nil {
					if proof.Collection && !a.dtoCollectionReferenceGetterMatches(actual.Model, p) {
						return unsupported(use.Pointer)
					}
					allowed = (allowed || proof.EnumSchema != nil && uniqueStringEnumType(p.Type) || proof.Collection && a.dtoUniqueBuiltinCollectionReferenceSource(originalField.Schema, originalField.Pointer) && dtoUniqueBuiltinCollectionReferenceType(p.Type, 0)) && a.dtoUniqueScalarReferenceMatches(proof, p, plan)
				} else if has(originalField.Schema, "$ref") {
					allowed = false
				}
				if container := dtoUniqueContainerType(p.Type); container && allowed && src.ReferenceFields[i] == nil {
					allowed = a.dtoUniqueContainerEnumsMatch(actual.Model, actual.Name, p.Type, originalField.Schema, originalField.Pointer, src, 0)
				}
				if bare(f.Type).Kind == Named && src.ReferenceFields[i] == nil {
					allowed = a.dtoUniqueEnumField(actual.Model, actual.Name, p.Type, src.Fields[i], src)
				}
				if failure != nil || !allowed || p.Name != f.Name || f.Name != expectedName || p.WireName != src.Fields[i].Wire || string(left) != string(right) {
					return unsupported(use.Pointer)
				}
				if initializer, ok := src.ContainerInitializers[i]; ok {
					if !dtoUniqueContainerType(p.Type) || p.Default != initializer.Literal || p.DefaultNull != initializer.DefaultNull {
						return unsupported(use.Pointer)
					}
				} else if dtoUniqueContainerType(p.Type) {
					return unsupported(use.Pointer)
				}
				original := originalField.Schema
				var expected *string
				if value := get(original, "default"); p.Default != "" && value != nil && value.Kind() != document.Null {
					key := dtoUniqueFieldDefaultKey(p.Type, value)
					expected = &key
				}
				expectedWork := 1
				if expected != nil && dtoUniqueContainerType(p.Type) {
					expectedWork = dtoUniqueLiteralWork(get(original, "default"))
				}
				if src.DefaultWork[i] != expectedWork {
					return unsupported(use.Pointer)
				}
				if (expected == nil) != (src.Defaults[i] == nil) || expected != nil && *expected != *src.Defaults[i] {
					return unsupported(use.Pointer)
				}
			}
			if propertyIndex != len(actual.Model.Properties) || constantIndex != len(actual.Model.Constants) {
				return unsupported(use.Pointer)
			}
			src.Actual = actual.Model
			validated[src] = true
		}
		all := models[src.Schema]
		if len(all) != 1 || all[0].Model != src.Actual {
			return unsupported(use.Pointer)
		}
		actual := all[0]
		element := use.Element.WithoutRootNull()
		local := src.Pointer == strings.TrimSuffix(use.Pointer, "/uniqueItems")+"/items" && element.Name == actual.Model.ClassName
		typed := element.Kind == Named && element.Element == nil && (element.Name == actual.Name || local)
		annotation := use.DefaultOnly && element.Kind == JSON
		if !typed && !annotation {
			return unsupported(use.Pointer)
		}
	}
	return nil
}

// Values have already passed their original object/leaf default checks. Keys
// include constructor initializers for omitted optional properties.
func dtoUniqueDefaultKey(src *dtoUniqueSource, v *document.Value) string {
	key := src.Key
	if v.Kind() == document.Null {
		return "null"
	}
	values := make([]any, 0, len(key.Fields))
	for i, field := range key.Fields {
		if value := get(v, src.Fields[i].Wire); value != nil && value.Kind() != document.Null {
			values = append(values, dtoUniqueFieldDefaultKey(field.Type, value))
		} else if value != nil {
			values = append(values, nil)
		} else if src.Defaults[i] != nil {
			values = append(values, *src.Defaults[i])
		} else {
			values = append(values, nil)
		}
	}
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Directional models are operation-owned views of already validated nominal
// DTOs. Source graphs and shared declarations are never rewritten or expanded.
type directionalDTO struct {
	model    *Model
	fields   map[string]field
	required map[string]bool
	eligible bool
}

type directionalModelSource struct {
	Schema   *document.Value
	Pointer  string
	Fields   []field
	Required map[string]bool
}

type directionalIndex struct {
	a            *analyzer
	declarations map[string]Declaration
	models       map[string]*directionalDTO
	blocked      [2]map[string]string
	affected     [2]map[string]bool
	cloneCount   int
}

func (a *analyzer) directionalModels(plan *Plan) *diagnostic.Diagnostic {
	if plan.Client == nil {
		return nil
	}
	index := &directionalIndex{a: a, declarations: map[string]Declaration{}, models: map[string]*directionalDTO{}, blocked: [2]map[string]string{{}, {}}}
	var collect func(Declaration, string) *diagnostic.Diagnostic
	collect = func(d Declaration, parent string) *diagnostic.Diagnostic {
		full := parent + "." + d.Name()
		index.declarations[full] = d
		switch d := d.(type) {
		case *Model:
			entry := &directionalDTO{model: d, fields: map[string]field{}}
			if original := a.modelOrigins[d]; original != nil {
				pointer := a.modelPointers[d]
				entry.eligible = !a.hierarchySources[original] && !has(original, "allOf") && !has(original, "oneOf") && !has(original, "anyOf") && !has(original, "discriminator")
				if properties := get(original, "properties"); properties != nil {
					for _, member := range properties.Members() {
						entry.fields[member.Name] = field{Wire: member.Name, Schema: member.Value, Pointer: naming.Pointer(pointer+"/properties", member.Name)}
					}
				}
				if source, ok := a.directionalModelSources[d]; ok && source.Schema == original && source.Pointer == pointer &&
					!a.hierarchySources[original] && !a.containsObjectCompositionUnion(original, 0, map[*document.Value]bool{}) {
					entry.eligible = true
					entry.fields = map[string]field{}
					for _, f := range source.Fields {
						entry.fields[f.Wire] = f
					}
					entry.required = source.Required
				}
			}
			if !entry.eligible {
				if original := a.modelOrigins[d]; original != nil {
					for direction, key := range []string{"readOnly", "writeOnly"} {
						at, err := a.directionalAnnotation(original, a.modelPointers[d], key)
						if err != nil {
							return err
						}
						if at != "" {
							index.blocked[direction][full] = at
						}
					}
				}
			}
			index.models[full] = entry
			for _, child := range d.Nested {
				if err := collect(child, full); err != nil {
					return err
				}
			}
		case *ObjectUnion:
			if origin, ok := a.unionOrigins[d]; ok {
				for direction, key := range []string{"readOnly", "writeOnly"} {
					at, err := a.directionalAnnotation(origin.Schema, origin.Pointer, key)
					if err != nil {
						return err
					}
					if at != "" {
						index.blocked[direction][full] = at
					}
				}
			}
			for _, child := range d.Variants {
				if err := collect(child, full); err != nil {
					return err
				}
			}
		case *PrimitiveUnion:
			if origin, ok := a.unionOrigins[d]; ok {
				for direction, key := range []string{"readOnly", "writeOnly"} {
					at, err := a.directionalAnnotation(origin.Schema, origin.Pointer, key)
					if err != nil {
						return err
					}
					if at != "" {
						index.blocked[direction][full] = at
					}
				}
			}
			if d.Object != nil {
				if err := collect(d.Object, parent); err != nil {
					return err
				}
			}
		case *Hierarchy:
			if origin, ok := a.hierarchyOrigins[d]; ok {
				for direction, key := range []string{"readOnly", "writeOnly"} {
					at, err := a.directionalAnnotation(origin.Schema, origin.Pointer, key)
					if err != nil {
						return err
					}
					if at != "" {
						index.blocked[direction][full] = at
					}
				}
			}
			if d.Leaf != nil {
				if err := collect(d.Leaf, parent); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, d := range plan.Models {
		if err := collect(d, a.options.PackageName+".models"); err != nil {
			return err
		}
	}
	for _, d := range plan.Client.Models {
		if err := collect(d, a.options.PackageName+".api.models"); err != nil {
			return err
		}
	}
	for _, request := range []bool{true, false} {
		p := index.projector(nil, request)
		index.affected[p.direction()] = p.affected
	}
	for gi := range plan.Client.Groups {
		for oi := range plan.Client.Groups[gi].Operations {
			op := &plan.Client.Groups[gi].Operations[oi]
			for _, request := range []bool{true, false} {
				projection := index.projector(op, request)
				if request {
					if op.Body == nil {
						continue
					}
					ty, err := projection.projectType(op.Body.Type, "", 0)
					if err != nil {
						return err
					}
					op.Body.Type = ty
				} else {
					ty, err := projection.projectType(op.Response, "", 0)
					if err != nil {
						return err
					}
					op.Response = ty
					if op.SSE != nil {
						for vi := range op.SSE.Variants {
							variant := &op.SSE.Variants[vi]
							if variant.Data == nil {
								continue
							}
							ty, err := projection.projectType(*variant.Data, "", 0)
							if err != nil {
								return err
							}
							variant.Data = &ty
						}
					}
					for ei := range op.Errors {
						if op.Errors[ei].Type == nil {
							continue
						}
						ty, err := projection.projectType(*op.Errors[ei].Type, "", 0)
						if err != nil {
							return err
						}
						op.Errors[ei].Type = &ty
					}
				}
				op.Models = append(op.Models, projection.models...)
			}
		}
	}
	plan.Client.Models = nil
	return a.operationDeclarations(plan.Client)
}

func (i *directionalIndex) qualify(name, owner string) string {
	if i.declarations[name] != nil {
		return name
	}
	for scope := owner; scope != ""; {
		if candidate := scope + "." + name; i.declarations[candidate] != nil {
			return candidate
		}
		at := strings.LastIndexByte(scope, '.')
		if at < 0 {
			break
		}
		scope = scope[:at]
	}
	return name
}

type directionalProjector struct {
	index    *directionalIndex
	op       *Operation
	request  bool
	affected map[string]bool
	clones   map[string]string
	models   []Declaration
}

func (i *directionalIndex) projector(op *Operation, request bool) *directionalProjector {
	p := &directionalProjector{index: i, op: op, request: request, affected: map[string]bool{}, clones: map[string]string{}}
	if cached := i.affected[p.direction()]; cached != nil {
		p.affected = cached
		return p
	}
	for name := range i.blocked[p.direction()] {
		p.affected[name] = true
	}
	for name, entry := range i.models {
		for _, property := range entry.model.Properties {
			f := entry.fields[property.WireName]
			if request && readOnlySchema(f.Schema) && property.Type.Kind != PatchValue || !request && writeOnlySchema(f.Schema) {
				p.affected[name] = true
			}
		}
		for _, c := range entry.model.Constants {
			f := entry.fields[c.WireName]
			if request && readOnlySchema(f.Schema) || !request && writeOnlySchema(f.Schema) {
				p.affected[name] = true
			}
		}
	}
	// Propagate over reverse typed edges once. Nominal cycles form a finite
	// worklist rather than repeated source expansion or quadratic rescans.
	reverse := map[string][]string{}
	for name, entry := range i.models {
		dependencies := map[string]bool{}
		for _, property := range entry.model.Properties {
			i.dependencies(property.Type, name, dependencies, map[string]bool{})
		}
		for dependency := range dependencies {
			reverse[dependency] = append(reverse[dependency], name)
		}
	}
	queue := make([]string, 0, len(p.affected))
	for name := range p.affected {
		queue = append(queue, name)
	}
	for n := 0; n < len(queue); n++ {
		for _, parent := range reverse[queue[n]] {
			if !p.affected[parent] {
				p.affected[parent] = true
				queue = append(queue, parent)
			}
		}
	}
	return p
}

func (i *directionalIndex) dependencies(t TypeRef, owner string, result, seen map[string]bool) {
	if t.Kind == Named {
		name := i.qualify(t.Name, owner)
		result[name] = true
		if !seen[name] {
			seen[name] = true
			if alias, ok := i.declarations[name].(*ValueAlias); ok {
				i.dependencies(alias.Type, name, result, seen)
			}
		}
	}
	if t.Element != nil {
		i.dependencies(*t.Element, owner, result, seen)
	}
}

func (p *directionalProjector) direction() int {
	if p.request {
		return 0
	}
	return 1
}

func (p *directionalProjector) needs(t TypeRef, owner string, seen map[string]bool) bool {
	if t.Kind == Named {
		name := p.index.qualify(t.Name, owner)
		if p.affected[name] {
			return true
		}
		if !seen[name] {
			seen[name] = true
			if alias, ok := p.index.declarations[name].(*ValueAlias); ok {
				if p.needs(alias.Type, name, seen) {
					return true
				}
			}
		}
	}
	return t.Element != nil && p.needs(*t.Element, owner, seen)
}

func (p *directionalProjector) projectType(t TypeRef, owner string, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if owner == "" && !p.needs(t, owner, map[string]bool{}) {
		return t, nil
	}
	// TypeRef nullable/alias wrappers are not schema edges; source depths were
	// already checked before this pass. Bound IR work without lowering that cap.
	if depth > 16384 {
		return TypeRef{}, diagnostic.At("E_SCHEMA_DEPTH", "Directional model nesting exceeds the supported limit.", p.op.Pointer)
	}
	if t.Kind == Named {
		name := p.index.qualify(t.Name, owner)
		if p.affected[name] {
			projected, err := p.projectModel(name)
			if err != nil {
				return TypeRef{}, err
			}
			t.Name = projected
		} else if alias, ok := p.index.declarations[name].(*ValueAlias); ok && p.needs(alias.Type, name, map[string]bool{}) {
			backing := alias.Type
			if t.Serializer != "" {
				backing.Serializer = t.Serializer
			}
			if t.Collection != nil {
				backing.Collection = t.Collection
			}
			if t.ArrayAliasView == NonNullArrayAlias {
				backing = backing.WithoutRootNull()
			}
			return p.projectType(backing, name, depth+1)
		} else {
			t.Name = name
		}
	}
	if t.Element != nil {
		element, err := p.projectType(*t.Element, owner, depth+1)
		if err != nil {
			return TypeRef{}, err
		}
		t.Element = &element
		if t.Kind == Nullable && element.Kind == Nullable {
			// A nullable collection alias can already wrap its backing type.
			// Replacing that alias must not produce an invalid Kotlin T?? view.
			if t.Serializer != "" {
				element.Serializer = t.Serializer
			}
			return element, nil
		}
	}
	return t, nil
}

func (p *directionalProjector) projectModel(name string) (string, *diagnostic.Diagnostic) {
	if existing := p.clones[name]; existing != "" {
		return existing, nil
	}
	if at := p.index.blocked[p.direction()][name]; at != "" {
		return "", unsupported(at)
	}
	entry := p.index.models[name]
	if entry == nil {
		return "", unsupported(p.op.Pointer)
	}
	if !entry.eligible {
		return "", unsupported(p.index.a.modelPointers[entry.model])
	}
	if entry.model.PropertyCount != nil {
		return "", rejectObjectCounts(p.index.a.modelOrigins[entry.model], p.index.a.modelPointers[entry.model])
	}
	p.index.cloneCount++
	if p.index.cloneCount > 16384 {
		return "", diagnostic.At("E_SCHEMA_LIMIT", "Too many directional DTO declarations.", p.op.Pointer)
	}
	role := "PublicResponse"
	if p.request {
		role = "Request"
	}
	if len(p.clones) != 0 {
		role += strconv.Itoa(len(p.clones) + 1)
	}
	short, err := p.index.a.operationModelName(p.op.ModelOwner, role, p.op.Pointer)
	if err != nil {
		return "", err
	}
	full := p.index.a.options.PackageName + ".api.models." + short
	p.clones[name] = full // Install the nominal recursion barrier before children.
	model := &Model{ClassName: short, Nullable: entry.model.Nullable, Directional: true}
	if p.index.a.modelDiagnosticPointers != nil {
		p.index.a.modelDiagnosticPointers[model] = p.index.a.modelPointers[entry.model]
	}
	p.models = append(p.models, model)
	for _, property := range entry.model.Properties {
		f := entry.fields[property.WireName]
		if !p.request && writeOnlySchema(f.Schema) {
			continue
		}
		if property.Default != "" && p.needs(property.Type, name, map[string]bool{}) {
			return "", unsupported(naming.Pointer(f.Pointer, "default"))
		}
		property.Type, err = p.projectType(property.Type, name, 0)
		if err != nil {
			return "", err
		}
		if p.request && property.Type.Kind != PatchValue {
			if readOnlySchema(f.Schema) {
				element := property.Type.WithoutRootNull()
				property.Type = TypeRef{Kind: PatchValue, Name: p.index.a.patchPackage() + ".Patch", Element: &element}
				property.Default, property.DefaultNull = "", false
			} else {
				nullable, e := p.index.a.requestNullable(f.Schema, f.Pointer)
				if e != nil {
					return "", e
				}
				if f.Schema != nil && nullable && !p.required(entry, property.WireName) {
					element := property.Type.WithoutRootNull()
					property.Type = TypeRef{Kind: PatchValue, Name: p.index.a.patchPackage() + ".Patch", Element: &element}
					property.Default, property.DefaultNull = "", false
				} else if p.required(entry, property.WireName) {
					property.Default, property.DefaultNull = "", false
				}
			}
		}
		model.Properties = append(model.Properties, property)
	}
	for _, c := range entry.model.Constants {
		f := entry.fields[c.WireName]
		if !p.request && writeOnlySchema(f.Schema) {
			continue
		}
		if p.request && readOnlySchema(f.Schema) {
			return "", unsupported(f.Pointer)
		}
		if p.needs(c.Type, name, map[string]bool{}) {
			return "", unsupported(f.Pointer)
		}
		c.Type, err = p.projectType(c.Type, name, 0)
		if err != nil {
			return "", err
		}
		model.Constants = append(model.Constants, c)
	}
	return full, nil
}

func (p *directionalProjector) required(entry *directionalDTO, wire string) bool {
	if entry.required != nil {
		return entry.required[wire]
	}
	original := p.index.a.modelOrigins[entry.model]
	required := get(original, "required")
	if required == nil {
		return false
	}
	for _, item := range required.Items() {
		if text(item) == wire {
			return true
		}
	}
	return false
}

func writeOnlySchema(v *document.Value) bool {
	value, _ := v.Get("writeOnly")
	flag, _ := value.Bool()
	return flag
}

package schema

import (
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type HierarchyTag struct{ Value, Child string }
type HierarchyTagRule struct {
	Matches []ObjectTagMatch
	Child   string
}
type Hierarchy struct {
	// Original typed mapping bodies retained for strict annotation validation.
	// Runtime emission continues to use the actual Leaf constants.
	tagBodies map[string]string
	// Type data without scalar selectors must not use a conflicting class discriminator.
	RawOnly                     bool
	Plain                       bool
	ClassName, Parent, TagField string
	Enums                       []*StringEnum
	Fields, Inherited           []Property
	Tags                        []HierarchyTag
	TagCombinations             []HierarchyTagRule
	OwnFields                   []string // Introduced here, not inherited overrides.
	Leaf                        *Model
	SerialName                  *string
	// New nullable hierarchy views require their physical singleton codecs to
	// reject null; historical non-null singleton behavior remains unchanged.
	NonNullSingletonCodec bool
	// Content-dispatched parents must encode through these actual child
	// serializers to retain a closed descendant's class discriminator.
	SerializeVia []string `json:",omitempty"`
}

func (n *Hierarchy) Name() string { return n.ClassName }
func (*Hierarchy) declaration()   {}
func (n *Hierarchy) Content() bool {
	return n.Leaf == nil && (n.RawOnly || n.TagField != "" || len(n.TagCombinations) != 0)
}

type typedLeafField struct {
	Type, Pointer string
	Body          *string
}

func typedFieldKeys(fields map[string]typedLeafField) []string {
	keys := make([]string, 0, len(fields))
	for wire := range fields {
		keys = append(keys, wire)
	}
	sort.Strings(keys)
	return keys
}

type hierarchyInfo struct {
	Wire, Type string
	Values     []string
	Fixed      *string
}
type hierarchyEdge struct{ Key, Value, Pointer string }
type hierarchyBranches struct {
	Keyword, Field string
	Edges          []hierarchyEdge
}
type hierarchyPlanner struct {
	typeData map[string]bool // Bottom-up facts from original leaf fields, not inferred tags.
	a        *analyzer
	graph    map[string]*hierarchyBranches
	shapes   map[string]*shape
	parents  map[string]string
	heights  map[string]int
	work     int
}

func hierarchyError(code, pointer string) *diagnostic.Diagnostic {
	message := map[string]string{"E_REF_CYCLE": "Discriminator reference graph is cyclic.", "E_SCHEMA_DEPTH": "Discriminator graph exceeds the supported depth.", "E_SCHEMA_LIMIT": "Discriminator graph exceeds the supported work budget.", "E_NAME_COLLISION": "Hierarchy-owned symbols collide with another declaration."}[code]
	return diagnostic.At(code, message, pointer)
}
func hierarchyProperty(info hierarchyInfo) (Property, *diagnostic.Diagnostic) {
	name, err := naming.PropertyName(info.Wire, "")
	if err != nil {
		return Property{}, err
	}
	return Property{Name: name, WireName: info.Wire, Type: TypeRef{Kind: Named, Name: info.Type}}, nil
}
func infoKeys(context map[string]hierarchyInfo) []string {
	keys := make([]string, 0, len(context))
	for key := range context {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func infoProperties(context map[string]hierarchyInfo) ([]Property, *diagnostic.Diagnostic) {
	properties := make([]Property, 0, len(context))
	for _, key := range infoKeys(context) {
		property, err := hierarchyProperty(context[key])
		if err != nil {
			return nil, err
		}
		properties = append(properties, property)
	}
	return properties, nil
}
func copyInfos(context map[string]hierarchyInfo) map[string]hierarchyInfo {
	out := make(map[string]hierarchyInfo, len(context))
	for key, value := range context {
		out[key] = value
	}
	return out
}

func (p *hierarchyPlanner) visit(key string, depth int, stack []string, pointer string) *diagnostic.Diagnostic {
	p.work++
	if p.work > 16384 {
		return hierarchyError("E_SCHEMA_LIMIT", pointer)
	}
	if depth > MaxSchemaDepth {
		return hierarchyError("E_SCHEMA_DEPTH", pointer)
	}
	if slices.Contains(stack, key) {
		return hierarchyError("E_REF_CYCLE", pointer)
	}
	if height, seen := p.heights[key]; seen {
		if depth+height-1 > MaxSchemaDepth {
			return hierarchyError("E_SCHEMA_DEPTH", pointer)
		}
		return nil
	}
	c := p.a.components[key]
	v := c.Schema
	stack = append(stack, key)
	keyword := ""
	if has(v, "oneOf") {
		keyword = "oneOf"
	} else if has(v, "anyOf") {
		keyword = "anyOf"
	}
	if keyword == "" {
		shape, err := p.a.objectShape(v, c.Pointer, depth)
		if err != nil {
			return err
		}
		if shape.Nullable != nil && *shape.Nullable && !c.NullableHierarchy {
			return unsupported(c.Pointer)
		}
		if f := shapeField(shape, "type"); f != nil {
			c, err := p.a.fieldConstant(f.Schema, "validation", f.Wire, f.Pointer, f.Depth, false, true)
			if err != nil {
				return err
			}
			if c != nil && structuredConstant(*c) {
				if p.typeData == nil {
					p.typeData = map[string]bool{}
				}
				p.typeData[key] = true
			}
		}
		p.shapes[key] = shape
		p.heights[key] = 1
		return nil
	}
	if err := checkKeys(v, []string{keyword, "discriminator", "nullable"}, c.Pointer); err != nil {
		return err
	}
	_, nullable, err := schemaType(v, c.Pointer)
	if err != nil {
		return err
	}
	if nullable && !c.NullableHierarchy {
		return unsupported(c.Pointer)
	}
	bp := naming.Pointer(c.Pointer, keyword)
	branches := get(v, keyword)
	if branches.Kind() != document.Array || len(branches.Items()) == 0 {
		return invalid("Composition must be a non-empty schema array.", bp)
	}
	if len(branches.Items()) > MaxMembers {
		return diagnostic.At("E_UNION_LIMIT", "Union exceeds the supported 240-branch limit.", bp)
	}
	type mapping struct {
		Value, Key, Pointer string
		Alias               bool
	}
	mappings := []mapping{}
	node := &hierarchyBranches{Keyword: keyword}
	if discriminator, ok := v.Get("discriminator"); ok {
		dp := naming.Pointer(c.Pointer, "discriminator")
		if err := object(discriminator, dp); err != nil {
			return err
		}
		if err := checkKeys(discriminator, []string{"propertyName", "mapping"}, dp); err != nil {
			return err
		}
		fp := naming.Pointer(dp, "propertyName")
		wire, ok := get(discriminator, "propertyName").Text()
		if !ok || wire == "" {
			return invalid("Discriminator propertyName must be a non-empty string.", fp)
		}
		property, err := naming.PropertyName(wire, fp)
		if err != nil {
			return err
		}
		if strings.EqualFold(strings.Trim(property, "`"), "rawElement") {
			return hierarchyError("E_NAME_COLLISION", fp)
		}
		node.Field = wire
		if values, ok := discriminator.Get("mapping"); ok {
			mp := naming.Pointer(dp, "mapping")
			if err := object(values, mp); err != nil {
				return err
			}
			for _, entry := range values.Members() {
				location := naming.Pointer(mp, entry.Name)
				reference, ok := entry.Value.Text()
				if !ok {
					return invalid("Mapping targets must be references.", location)
				}
				child := reference
				component := p.a.components[reference]
				if component == nil {
					var err *diagnostic.Diagnostic
					component, err = p.a.resolveDeclared(reference, location)
					if err != nil {
						return err
					}
					child = component.Wire
				}
				if component.NullableDTO || component.NullableEnum || component.NullableValueClass || component.NullableTemporal || component.NullableBigint || component.NullableBytes || component.NullableCallerNative || component.NullableRaw || component.NullablePrimitive || component.NullableNativeEnum {
					return unsupported(location)
				}
				alias := component.BindingTarget != nil
				if alias {
					if component.BindingTarget.NullableDTO || component.BindingTarget.NullableEnum || component.BindingTarget.NullableValueClass || component.BindingTarget.NullableTemporal || component.BindingTarget.NullableBigint || component.BindingTarget.NullableBytes || component.BindingTarget.NullableCallerNative || component.BindingTarget.NullableRaw || component.BindingTarget.NullablePrimitive || component.BindingTarget.NullableNativeEnum || component.BindingNullable != nil && *component.BindingNullable && !p.a.hierarchySources[component.BindingTarget.Schema] {
						return unsupported(location)
					}
					if depth+1+component.AliasHeight > MaxSchemaDepth {
						return hierarchyError("E_SCHEMA_DEPTH", location)
					}
					child = component.BindingTarget.Wire
				}
				mappings = append(mappings, mapping{entry.Name, child, location, alias})
			}
		}
	}
	height, nullBranches := 1, 0
	for index, branch := range branches.Items() {
		location := naming.Pointer(bp, strconv.Itoa(index))
		if err := object(branch, location); err != nil {
			return err
		}
		if text(get(branch, "type")) == "null" {
			if err := checkKeys(branch, []string{"type"}, location); err != nil {
				return err
			}
			p.work++
			if p.work > 16384 {
				return hierarchyError("E_SCHEMA_LIMIT", location)
			}
			if err := depthCheck(depth+1, location); err != nil {
				return err
			}
			height = max(height, 2)
			nullBranches++
			continue
		}
		// An inline branch carries no reference. Its synthetic component was
		// registered from the discriminator constant before this walk, so it is
		// handled here instead of through the reference key set.
		if !has(branch, "$ref") {
			leaf := p.a.hierarchyInlineBranches[branch]
			if leaf == nil {
				// An inline branch without a discriminator constant has no name
				// source, so it keeps the original rejection location.
				return unsupported(naming.Pointer(location, "type"))
			}
			if _, owned := p.parents[leaf.Wire]; owned {
				return unsupported(location)
			}
			if err := p.visit(leaf.Wire, depth+1, stack, location); err != nil {
				return err
			}
			p.parents[leaf.Wire] = key
			value := leaf.Value
			for _, mapping := range mappings {
				if mapping.Key == leaf.Wire || mapping.Key == leaf.Class {
					value = mapping.Value
					break
				}
			}
			node.Edges = append(node.Edges, hierarchyEdge{leaf.Wire, value, location})
			height = max(height, 1+p.heights[leaf.Wire])
			continue
		}
		if err := checkKeys(branch, []string{"$ref", "nullable"}, location); err != nil {
			return err
		}
		if _, _, err := schemaType(branch, location); err != nil {
			return err
		}
		rp := naming.Pointer(location, "$ref")
		reference, ok := get(branch, "$ref").Text()
		if !ok {
			return unsupported(rp)
		}
		component, steps, nullable, err := p.a.resolveHierarchyBranch(reference, rp, depth)
		if err != nil {
			return err
		}
		if override := nullableOverride(branch); override != nil {
			nullable = *override
		}
		if nullable {
			nullBranches++
		}
		child := component.Wire
		// Canonical graph keys do not erase original alias-edge work/depth.
		p.work += steps - 1
		if err := p.visit(child, depth+steps, stack, rp); err != nil {
			return err
		}
		if _, owned := p.parents[child]; owned {
			return unsupported(rp)
		}
		p.parents[child] = key
		value := lowerHierarchyWire(child)
		for _, mapping := range mappings {
			if mapping.Key == child {
				value = mapping.Value
				break
			}
		}
		node.Edges = append(node.Edges, hierarchyEdge{child, value, rp})
		height = max(height, steps+p.heights[child])
	}
	if len(node.Edges) == 0 {
		return unsupported(bp)
	}
	if nullBranches > 1 && keyword == "oneOf" {
		p.a.warn("W_ONEOF_NULL_OVERLAP", "Hierarchy nullable folding does not enforce exact oneOf match count for overlapping null branches.", bp)
	}
	for _, mapping := range mappings {
		if mapping.Alias && !slices.ContainsFunc(node.Edges, func(e hierarchyEdge) bool { return e.Key == mapping.Key }) {
			return unsupported(mapping.Pointer)
		}
	}
	for _, edge := range node.Edges {
		if p.typeData[edge.Key] {
			p.typeData[key] = true
			break
		}
	}
	p.graph[key] = node
	p.heights[key] = height
	return nil
}

func (p *hierarchyPlanner) actual(f *field) (*document.Value, *diagnostic.Diagnostic) {
	if err := object(f.Schema, f.Pointer); err != nil {
		return nil, err
	}
	if has(f.Schema, "$ref") {
		if err := checkKeys(f.Schema, []string{"$ref", "nullable"}, f.Pointer); err != nil {
			return nil, err
		}
		rp := naming.Pointer(f.Pointer, "$ref")
		reference, ok := get(f.Schema, "$ref").Text()
		if !ok {
			return nil, invalid("$ref must be a string.", rp)
		}
		component, err := p.a.resolve(reference, rp)
		if err != nil {
			return nil, err
		}
		return component.Schema, nil
	}
	return f.Schema, nil
}

// mappedField follows actual parent edges, not guessed leaf classifier names.
// The graph has already passed cycle/depth/ownership validation in visit.
func (p *hierarchyPlanner) mappedField(key, wire string) (hierarchyEdge, bool) {
	for parentKey, ok := p.parents[key]; ok; parentKey, ok = p.parents[key] {
		parent := p.graph[parentKey]
		if parent != nil && parent.Field == wire {
			for _, edge := range parent.Edges {
				if edge.Key == key {
					return edge, true
				}
			}
		}
		key = parentKey
	}
	return hierarchyEdge{}, false
}

func (p *hierarchyPlanner) pins(key, wire string) (bool, []string, *diagnostic.Diagnostic) {
	shape := p.shapes[key]
	if shape == nil {
		return true, []string{}, nil
	}
	f := shapeField(shape, wire)
	if f == nil {
		if edge, ok := p.mappedField(key, wire); ok {
			return true, []string{edge.Value}, nil
		}
		return true, []string{}, nil
	}
	v, err := p.actual(f)
	if err != nil {
		return false, nil, err
	}
	if has(v, "const") {
		c, err := constant("validation", wire, v, f.Pointer, f.Depth)
		if err != nil {
			return false, nil, err
		}
		if c.Type.Kind != Scalar || c.Type.Name != "kotlin.String" {
			return false, nil, unsupported(f.Pointer)
		}
		return true, []string{text(get(v, "const"))}, nil
	}
	if has(v, "enum") {
		e, err := buildStringEnum("Validation", v, f.Pointer)
		if err != nil {
			return false, nil, err
		}
		values := make([]string, 0, len(e.Members))
		for _, member := range e.Members {
			values = append(values, member.Wire)
		}
		return false, values, nil
	}
	parent := p.graph[p.parents[key]]
	if parent != nil && parent.Field == wire {
		kind, _, err := schemaType(v, f.Pointer)
		if err != nil {
			return false, nil, err
		}
		if kind != "string" {
			return false, nil, unsupported(f.Pointer)
		}
		values := []string{}
		for _, edge := range parent.Edges {
			if edge.Key == key {
				values = append(values, edge.Value)
			}
		}
		return true, values, nil
	}
	return true, []string{}, nil
}
func (p *hierarchyPlanner) collect(key, wire string) (bool, []string, *diagnostic.Diagnostic) {
	node := p.graph[key]
	if node == nil {
		return p.pins(key, wire)
	}
	all := true
	values := []string{}
	seen := map[string]bool{}
	for _, edge := range node.Edges {
		constant, pins, err := p.collect(edge.Key, wire)
		if err != nil {
			return false, nil, err
		}
		all = all && constant
		for _, value := range pins {
			if !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	}
	return all, values, nil
}
func (p *hierarchyPlanner) info(key, wire string) (hierarchyInfo, *diagnostic.Diagnostic) {
	all, values, err := p.collect(key, wire)
	if err != nil {
		return hierarchyInfo{}, err
	}
	c := p.a.components[key]
	field, err := naming.ClassName(wire, c.Pointer)
	if err != nil {
		return hierarchyInfo{}, err
	}
	info := hierarchyInfo{Wire: wire, Type: p.a.options.PackageName + ".models." + c.Name + field, Values: values}
	if all && len(values) == 1 {
		value := values[0]
		info.Fixed = &value
	}
	return info, nil
}
func (p *hierarchyPlanner) implicit(key string) ([]hierarchyInfo, *diagnostic.Diagnostic) {
	node := p.graph[key]
	first := p.shapes[node.Edges[0].Key]
	if first == nil {
		return nil, nil
	}
	infos := []hierarchyInfo{}
	for _, field := range first.Fields {
		present := true
		for _, edge := range node.Edges {
			shape := p.shapes[edge.Key]
			present = present && shape != nil && shapeField(shape, field.Wire) != nil
		}
		if !present {
			continue
		}
		// JSON body data cannot become an inferred string discriminator,
		// including when another leaf gives the same wire a scalar pin.
		structured := false
		for _, edge := range node.Edges {
			f := shapeField(p.shapes[edge.Key], field.Wire)
			c, err := p.a.fieldConstant(f.Schema, "validation", f.Wire, f.Pointer, f.Depth, false, true)
			if err != nil {
				return nil, err
			}
			structured = structured || c != nil && structuredConstant(*c)
		}
		if structured {
			continue
		}
		info, err := p.info(key, field.Wire)
		if err != nil {
			return nil, err
		}
		if info.Fixed == nil && len(info.Values) >= 2 {
			infos = append(infos, info)
		}
	}
	return infos, nil
}
func reserveHierarchy(name, pointer string, used map[string]bool) *diagnostic.Diagnostic {
	if used[strings.ToLower(name)] {
		return hierarchyError("E_NAME_COLLISION", pointer)
	}
	used[strings.ToLower(name)] = true
	if naming.ReservedPathSegment(name) {
		return diagnostic.At("E_NAME_INVALID", "Generated peer name is reserved.", pointer)
	}
	return nil
}

// Compound rules keep finite sets per field, without multiplying schema
// branches. All-const branches claim first, then branches with enum fields.
func (p *hierarchyPlanner) compoundTags(node *hierarchyBranches, own []hierarchyInfo, result *Hierarchy) *diagnostic.Diagnostic {
	type candidate struct {
		rule     HierarchyTagRule
		constant bool
		key      string
	}
	var candidates []candidate
	for _, edge := range node.Edges {
		candidate := candidate{rule: HierarchyTagRule{Child: p.a.components[edge.Key].Name}, constant: true}
		for _, info := range own {
			constant, values, err := p.pins(edge.Key, info.Wire)
			if err != nil {
				return err
			}
			if len(values) == 0 {
				return unsupported(edge.Pointer)
			}
			p.work += 1 + len(values)
			if p.work > 16384 {
				return hierarchyError("E_SCHEMA_LIMIT", edge.Pointer)
			}
			candidate.constant = candidate.constant && constant
			candidate.rule.Matches = append(candidate.rule.Matches, ObjectTagMatch{Wire: info.Wire, Values: values})
		}
		// Set equality controls duplicate claims; emission keeps source value order.
		canonical := make([]ObjectTagMatch, len(candidate.rule.Matches))
		for i, match := range candidate.rule.Matches {
			canonical[i] = ObjectTagMatch{Wire: match.Wire, Values: slices.Clone(match.Values)}
			sort.Strings(canonical[i].Values)
		}
		encoded, _ := json.Marshal(canonical)
		candidate.key = string(encoded)
		candidates = append(candidates, candidate)
	}
	seen := map[string]bool{}
	for _, pass := range []bool{true, false} {
		for _, candidate := range candidates {
			if candidate.constant == pass && !seen[candidate.key] {
				seen[candidate.key] = true
				result.TagCombinations = append(result.TagCombinations, candidate.rule)
			}
		}
	}
	return nil
}

func (p *hierarchyPlanner) build(key string, context map[string]hierarchyInfo, parentCustom bool, used map[string]bool, out map[string]*Hierarchy) *diagnostic.Diagnostic {
	c := p.a.components[key]
	result := &Hierarchy{ClassName: c.Name}
	if parent, ok := p.parents[key]; ok {
		result.Parent = p.a.options.PackageName + ".models." + p.a.components[parent].Name
	}
	infos := copyInfos(context)
	if node := p.graph[key]; node != nil {
		var ownInfos []hierarchyInfo
		if node.Field != "" {
			info, err := p.info(key, node.Field)
			if err != nil {
				return err
			}
			ownInfos = append(ownInfos, info)
		} else {
			var err *diagnostic.Diagnostic
			ownInfos, err = p.implicit(key)
			if err != nil {
				return err
			}
		}
		result.RawOnly = len(ownInfos) == 0 && p.typeData[key]
		for i := range ownInfos {
			own := &ownInfos[i]
			name, err := naming.PropertyName(own.Wire, c.Pointer)
			if err != nil {
				return err
			}
			if strings.EqualFold(strings.Trim(name, "`"), "rawElement") {
				return hierarchyError("E_NAME_COLLISION", c.Pointer)
			}
			if own.Fixed != nil && len(node.Edges) != 1 || len(own.Values) == 0 {
				return unsupported(c.Pointer)
			}
			if inherited, reused := context[own.Wire]; reused {
				own.Type = inherited.Type
			} else {
				enumName := own.Type[strings.LastIndex(own.Type, ".")+1:]
				if err := reserveHierarchy(enumName, c.Pointer, used); err != nil {
					return err
				}
				e, err := stringEnumValues(enumName, own.Values, naming.Pointer(c.Pointer, "discriminator"))
				if err != nil {
					return err
				}
				result.Enums = append(result.Enums, e)
				result.OwnFields = append(result.OwnFields, own.Wire)
			}
			infos[own.Wire] = *own
		}
		childInfos := copyInfos(infos)
		for _, wire := range infoKeys(context) {
			all, values, err := p.collect(key, wire)
			if err != nil {
				return err
			}
			inherited := context[wire]
			inherited.Fixed = nil
			if all && len(values) == 1 {
				value := values[0]
				inherited.Fixed = &value
			}
			childInfos[wire] = inherited
		}
		fields, err := infoProperties(context)
		if err != nil {
			return err
		}
		result.Fields = fields
		if len(ownInfos) != 0 || result.RawOnly {
			for _, own := range ownInfos {
				if _, inherited := context[own.Wire]; !inherited {
					property, err := hierarchyProperty(own)
					if err != nil {
						return err
					}
					result.Fields = append(result.Fields, property)
				}
			}
			for _, peer := range []string{c.Name + "Serializer", "Unknown" + c.Name, "Unknown" + c.Name + "Serializer"} {
				if err := reserveHierarchy(peer, c.Pointer, used); err != nil {
					return err
				}
			}
			if len(ownInfos) > 1 {
				if err := p.compoundTags(node, ownInfos, result); err != nil {
					return err
				}
			} else if len(ownInfos) == 1 {
				own := ownInfos[0]
				result.TagField = own.Wire
				seen := map[string]bool{}
				add := func(value, child string) {
					if !seen[value] {
						seen[value] = true
						result.Tags = append(result.Tags, HierarchyTag{value, p.a.components[child].Name})
					}
				}
				if node.Field != "" {
					for _, edge := range node.Edges {
						add(edge.Value, edge.Key)
					}
					for _, edge := range node.Edges {
						constant, pins, err := p.pins(edge.Key, own.Wire)
						if err != nil {
							return err
						}
						if !constant {
							for _, pin := range pins {
								add(pin, edge.Key)
							}
						}
					}
				} else {
					for _, pass := range []bool{true, false} {
						for _, edge := range node.Edges {
							constant, pins, err := p.pins(edge.Key, own.Wire)
							if err != nil {
								return err
							}
							if constant == pass {
								for _, pin := range pins {
									add(pin, edge.Key)
								}
							}
						}
					}
				}
			}
		}
		result.Inherited, err = infoProperties(infos)
		if err != nil {
			return err
		}
		p.a.warn("W_OBJECT_UNION_APPROXIMATION", "Hierarchy dispatch maps fields or sealed class discriminators, not exact schema-match validation.", naming.Pointer(c.Pointer, node.Keyword))
		for _, edge := range node.Edges {
			if err := p.build(edge.Key, childInfos, result.Content(), used, out); err != nil {
				return err
			}
		}
	} else {
		shape := p.shapes[key]
		mapped := map[string]typedLeafField{}
		for _, wire := range infoKeys(context) {
			info := context[wire]
			field := shapeField(shape, wire)
			if field == nil {
				edge, ok := p.mappedField(key, wire)
				if !ok {
					return unsupported(c.Pointer)
				}
				literal := edge.Value
				mapped[wire] = typedLeafField{Type: info.Type, Body: &literal, Pointer: edge.Pointer}
				continue
			}
			v, err := p.actual(field)
			if err != nil {
				return err
			}
			value := typedLeafField{Type: info.Type, Pointer: field.Pointer}
			if literal, ok := get(v, "const").Text(); ok {
				if info.Fixed != nil {
					literal = *info.Fixed
				}
				value.Body = &literal
			} else if has(v, "enum") { /* required typed constructor */
			} else {
				parent := p.graph[p.parents[key]]
				if parent == nil || parent.Field != wire {
					return unsupported(field.Pointer)
				}
				for _, edge := range parent.Edges {
					if edge.Key == key {
						literal := edge.Value
						value.Body = &literal
						break
					}
				}
			}
			mapped[wire] = value
		}
		model, err := p.a.buildTypedModel(c.Name, c.Schema, c.Pointer, mapped)
		if err != nil {
			return err
		}
		if !parentCustom && len(model.Properties) == 0 && len(model.Constants) > 0 && !slices.ContainsFunc(model.Constants, structuredConstant) {
			return unsupported(c.Pointer)
		}
		if !parentCustom {
			for _, edge := range p.graph[p.parents[key]].Edges {
				if edge.Key == key {
					value := edge.Value
					result.SerialName = &value
					break
				}
			}
		}
		result.Leaf = model
		result.tagBodies = map[string]string{}
		for wire, field := range mapped {
			if field.Body != nil {
				result.tagBodies[wire] = *field.Body
			}
		}
		for owner := key; owner != ""; owner = p.parents[owner] {
			if p.a.components[owner].MemberNullableHierarchy {
				result.NonNullSingletonCodec = true
				break
			}
		}
		result.Fields, err = infoProperties(context)
		if err != nil {
			return err
		}
		if result.Parent == "" {
			return unsupported(c.Pointer)
		}
	}
	out[key] = result
	aOrigin := nominalOrigin{Schema: c.Schema, Pointer: c.Pointer, Nullable: c.NullableHierarchy}
	if p.a.hierarchyOrigins != nil {
		p.a.hierarchyOrigins[result] = aOrigin
	}
	p.a.recordTupleUnionDeclaration(result, c.Schema, c.Pointer)
	return nil
}

// discoverHierarchySources records only original named graph nodes. Nullable
// alias admission is provisional until planHierarchy validates actual ownership.
func (a *analyzer) discoverHierarchySources() *diagnostic.Diagnostic {
	a.hierarchySources = map[*document.Value]bool{}
	seen := map[string]bool{}
	pending := []string{}
	for key, c := range a.components {
		if has(c.Schema, "discriminator") && (has(c.Schema, "oneOf") || has(c.Schema, "anyOf")) {
			pending = append(pending, key)
		}
	}
	sort.Strings(pending) // Bounded discovery must not depend on map iteration.
	for len(pending) != 0 {
		key := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[key] || len(seen) >= 16384 {
			continue
		}
		seen[key] = true
		c := a.components[key]
		if c == nil {
			continue
		}
		if c.Reference != "" {
			if target, err := a.resolveDeclared(c.Reference, c.Pointer+"/$ref"); err == nil {
				pending = append(pending, target.Wire)
			}
			continue
		}
		a.hierarchySources[c.Schema] = true
		// Named catalog candidates already own their classifier spelling. This
		// provisional entry is replaced by actual hierarchy declarations before
		// emission, so internal chains cannot authorize a borrowed nested owner.
		a.internalNamedModels[c.Schema] = a.options.PackageName + ".models." + c.Name
		branches := get(c.Schema, "oneOf")
		if branches.Kind() != document.Array {
			branches = get(c.Schema, "anyOf")
		}
		if branches.Kind() != document.Array {
			continue
		}
		for _, branch := range branches.Items() {
			if reference, ok := get(branch, "$ref").Text(); ok {
				if target, err := a.resolveDeclared(reference, c.Pointer); err == nil {
					pending = append(pending, target.Wire)
				}
				continue
			}
			// An inline branch has no reference; its synthetic component was
			// registered before this discovery pass.
			if leaf := a.hierarchyInlineBranches[branch]; leaf != nil {
				pending = append(pending, leaf.Wire)
			}
		}
	}
	return a.discoverHierarchyNullability()
}

func (a *analyzer) planHierarchy(used map[string]bool) (map[string]*Hierarchy, *diagnostic.Diagnostic) {
	p := &hierarchyPlanner{a: a, graph: map[string]*hierarchyBranches{}, shapes: map[string]*shape{}, parents: map[string]string{}, heights: map[string]int{}}
	candidates := []string{}
	for wire, c := range a.components {
		if has(c.Schema, "discriminator") && (has(c.Schema, "oneOf") || has(c.Schema, "anyOf")) {
			candidates = append(candidates, wire)
		}
	}
	sort.Strings(candidates)
	for _, key := range candidates {
		if err := p.visit(key, 1, nil, a.components[key].Pointer); err != nil {
			return nil, err
		}
	}
	out := map[string]*Hierarchy{}
	for _, key := range candidates {
		if _, owned := p.parents[key]; !owned {
			if err := p.build(key, map[string]hierarchyInfo{}, false, used, out); err != nil {
				return nil, err
			}
		}
	}
	var routes func(string)
	routes = func(key string) {
		h := out[key]
		if h == nil || h.Leaf != nil {
			return
		}
		for _, edge := range p.graph[key].Edges {
			routes(edge.Key)
			child := out[edge.Key]
			if h.Content() && child != nil && child.Leaf == nil && (!child.Content() || len(child.SerializeVia) != 0) {
				h.SerializeVia = append(h.SerializeVia, child.Name())
			}
		}
		p.a.recordTupleUnionDeclaration(h, p.a.components[key].Schema, p.a.components[key].Pointer)
	}
	for _, key := range candidates {
		if _, owned := p.parents[key]; !owned {
			routes(key)
		}
	}
	return out, nil
}

// syntheticHierarchyLeaf records an inline oneOf/anyOf branch that carries its
// own discriminator constant. Option B: the branch has no component name, so the
// constant supplies both the class name and the dispatch value. A branch without
// a single-valued string constant has no name source and keeps its rejection.
type syntheticHierarchyLeaf struct {
	Wire, Class string
	Schema      *document.Value
	Pointer     string
	Value       string
}

// hierarchyInlineLeaves registers synthetic components for eligible inline
// branches so the existing name-keyed planner can walk them unchanged. It runs
// before discoverHierarchySources, which is the first consumer of a.components.
func (a *analyzer) hierarchyInlineLeaves(used map[string]bool) *diagnostic.Diagnostic {
	a.hierarchyInlineBranches = map[*document.Value]*syntheticHierarchyLeaf{}
	a.hierarchyInlineOrder = nil
	for _, key := range sortedComponentNames(a.components) {
		c := a.components[key]
		if c == nil || (!has(c.Schema, "oneOf") && !has(c.Schema, "anyOf")) || !has(c.Schema, "discriminator") {
			continue
		}
		branchPointer := naming.Pointer(c.Pointer, "oneOf")
		branches := get(c.Schema, "oneOf")
		if !has(c.Schema, "oneOf") {
			branchPointer = naming.Pointer(c.Pointer, "anyOf")
			branches = get(c.Schema, "anyOf")
		}
		if branches == nil || branches.Kind() != document.Array {
			continue
		}
		wire, ok := get(c.Schema, "discriminator").Get("propertyName")
		if !ok {
			continue
		}
		field, isText := wire.Text()
		if !isText || field == "" {
			continue
		}
		for index, branch := range branches.Items() {
			location := naming.Pointer(branchPointer, strconv.Itoa(index))
			if has(branch, "$ref") || text(get(branch, "type")) == "null" {
				continue
			}
			leaf, err := a.hierarchyInlineLeaf(c, branch, location, field, index)
			if err != nil {
				return err
			}
			if leaf == nil {
				continue
			}
			if _, exists := a.components[leaf.Wire]; exists {
				return diagnostic.At("E_NAME_COLLISION", "Inline hierarchy branch resolves to an existing component name.", location)
			}
			a.components[leaf.Wire] = &component{
				Wire: leaf.Wire, Name: leaf.Class, Pointer: location, Schema: leaf.Schema,
			}
			a.hierarchyInlineBranches[leaf.Schema] = leaf
			a.hierarchyInlineOrder = append(a.hierarchyInlineOrder, leaf)
		}
	}
	return nil
}

// hierarchyInlineLeaf decides whether an inline branch may become a synthetic
// leaf and derives its class name from the discriminator constant.
func (a *analyzer) hierarchyInlineLeaf(parent *component, branch *document.Value, location, field string, index int) (*syntheticHierarchyLeaf, *diagnostic.Diagnostic) {
	if err := object(branch, location); err != nil {
		return nil, err
	}
	kind, nullable, err := schemaType(branch, location)
	if err != nil {
		return nil, err
	}
	// Only an ordinary object leaf is admitted here; primitives, arrays and
	// nullable branches keep their existing separate contracts.
	if kind != "object" || nullable || dictionarySchema(branch) ||
		has(branch, "allOf") || has(branch, "oneOf") || has(branch, "anyOf") || has(branch, "const") {
		return nil, nil
	}
	value, err := a.hierarchyInlineTagValue(branch, location, field)
	if err != nil {
		return nil, err
	}
	if value == "" {
		// No single constant means no name source; the branch keeps its rejection
		// at the same place as before this contract.
		return nil, nil
	}
	className, err := naming.ClassName(value, location)
	if err != nil {
		return nil, err
	}
	className += a.options.Suffix
	wire := parent.Wire + "#" + strconv.Itoa(index) + value
	return &syntheticHierarchyLeaf{Wire: wire, Class: className, Schema: branch, Pointer: location, Value: value}, nil
}

// hierarchyInlineTagValue returns the single discriminator value declared by the
// branch itself: a String const, or a one-member String enum. Anything else has
// no unambiguous name and is left to its existing rejection.
func (a *analyzer) hierarchyInlineTagValue(branch *document.Value, location, field string) (string, *diagnostic.Diagnostic) {
	properties, ok := branch.Get("properties")
	if !ok || properties.Kind() != document.Object {
		return "", nil
	}
	tag, ok := properties.Get(field)
	if !ok || tag == nil {
		return "", nil
	}
	tagPointer := naming.Pointer(naming.Pointer(location, "properties"), field)
	kind, _, err := schemaType(tag, tagPointer)
	if err != nil {
		return "", err
	}
	if kind != "" && kind != "string" {
		return "", nil
	}
	if has(tag, "const") {
		if has(tag, "enum") || has(tag, "format") {
			return "", nil
		}
		constant, ok := get(tag, "const").Text()
		if !ok {
			return "", nil
		}
		return constant, nil
	}
	if has(tag, "enum") {
		members := get(tag, "enum").Items()
		if len(members) != 1 {
			return "", nil
		}
		value, ok := members[0].Text()
		if !ok {
			return "", nil
		}
		return value, nil
	}
	return "", nil
}

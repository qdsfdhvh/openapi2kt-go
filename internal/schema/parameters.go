package schema

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func (a *analyzer) parameters(v *document.Value, present bool, pointer string) ([]Parameter, *diagnostic.Diagnostic) {
	if !present {
		return nil, nil
	}
	if v.Kind() != document.Array {
		return nil, invalid("parameters must be an array.", pointer)
	}
	if len(v.Items()) > MaxMembers {
		return nil, diagnostic.At("E_OPERATION_LIMIT", "Too many parameters.", pointer)
	}
	var params []Parameter
	seen := map[string]bool{}
	for i, value := range v.Items() {
		p := naming.Pointer(pointer, strconv.Itoa(i))
		occurrence := p
		value, p, err := a.operationReference(value, p)
		if err != nil {
			return nil, err
		}
		if err := operationKeys(value, []string{"name", "in", "required", "schema", "description", "deprecated", "example", "examples", "style", "explode"}, p); err != nil {
			return nil, err
		}
		wire, ok := get(value, "name").Text()
		if !ok || wire == "" {
			return nil, invalid("Parameter name must be nonempty text.", p+"/name")
		}
		location := text(get(value, "in"))
		if !slices.Contains([]string{"path", "query", "header"}, location) {
			return nil, operationUnsupported(p + "/in")
		}
		if seen[location+":"+wire] {
			return nil, invalid("Duplicate parameter name and location.", occurrence)
		}
		seen[location+":"+wire] = true
		required := false
		if has(value, "required") {
			var ok bool
			required, ok = get(value, "required").Bool()
			if !ok {
				return nil, invalid("required must be boolean.", p+"/required")
			}
		}
		if location == "path" && !required {
			return nil, invalid("Path parameters must be required.", p+"/required")
		}
		if has(value, "explode") && get(value, "explode").Kind() != document.Boolean {
			return nil, invalid("explode must be boolean.", p+"/explode")
		}
		schema := get(value, "schema")
		kind, _, err := schemaType(schema, p+"/schema")
		if err != nil {
			return nil, err
		}
		if location == "query" && text(get(value, "style")) == "deepObject" && kind == "object" && get(schema, "properties").Kind() == document.Object && len(get(schema, "properties").Members()) > 0 {
			validation := *a
			validation.warnings = nil
			// This impossible Kotlin name cannot reserve a real nested name.
			if _, _, err := validation.deepObject(schema, p+"/schema", "$Filter"); err != nil {
				return nil, err
			}
			ty := TypeRef{Kind: Named, Name: "$Filter"}
			if !required {
				ty = ty.Null()
			}
			params = append(params, Parameter{Wire: wire, Location: location, Type: ty, Required: required, DeepObject: &DeepObject{Schema: schema, Pointer: p + "/schema"}})
			continue
		}
		var nested []Declaration
		ty, err := a.parameterType(get(value, "schema"), p+"/schema", &nested)
		if err != nil {
			return nil, err
		}
		plain := ty
		if plain.Kind == Nullable {
			plain = *plain.Element
		}
		if location == "query" && text(get(value, "style")) == "deepObject" {
			// Inspect the already validated transparent backing for wire shape;
			// keep the original alias/type/check graph on the public parameter.
			backing := plain
			for backing.Kind == Named && backing.Element != nil {
				backing = bare(*backing.Element)
			}
			if backing.Kind == Map {
				plain = backing
			}
		}
		wireType := plain
		elementNullable := false
		if plain.Kind == List || plain.Kind == Map {
			elementNullable = plain.Element.Kind == Nullable
			wireType = bare(*plain.Element)
		}
		var inline *StringEnum
		leaf := get(value, "schema")
		if plain.Kind == List {
			leaf = get(leaf, "items")
		}
		if len(nested) == 1 && has(leaf, "enum") && wireType.Kind == Named {
			inline, _ = nested[0].(*StringEnum)
		}
		enum := inline != nil || a.parameterEnum(wireType)
		if len(nested) != 0 && inline == nil {
			return nil, operationUnsupported(p + "/schema")
		}
		deepMap := location == "query" && text(get(value, "style")) == "deepObject" && plain.Kind == Map
		if plain.Kind == Map && !deepMap && !(location == "path" && (text(get(value, "style")) == "label" || text(get(value, "style")) == "matrix")) {
			enum = false // Map values are forwarded directly, not serialName-unwrapped.
		}
		if wireType.Kind != Scalar && wireType.Kind != InlineValue && !enum || plain.Kind == Map && location != "query" && text(get(value, "style")) != "label" && text(get(value, "style")) != "matrix" {
			a.warn("W_HTTP_PARAMETER_PROJECTION", "A structured parameter uses Ktor toString, not recursive encoding or enum-value unwrapping.", p+"/schema")
		}
		if location == "path" && (text(get(value, "style")) == "label" || text(get(value, "style")) == "matrix") && wireType.Kind != Scalar && wireType.Kind != InlineValue && !enum {
			return nil, operationUnsupported(p + "/schema")
		}
		if deepMap && wireType.Kind != Scalar && wireType.Kind != InlineValue && !enum {
			return nil, operationUnsupported(p + "/schema")
		}
		format, err := parameterFormat(value, location, plain.Kind, p)
		if err != nil {
			return nil, err
		}
		if location == "path" && ty.Kind == Nullable {
			return nil, operationUnsupported(p + "/schema")
		}
		if !required {
			ty = ty.Null()
		}
		params = append(params, Parameter{Wire: wire, Location: location, Type: ty, Required: required, Enum: enum, Format: format, InlineEnum: inline, ElementNullable: elementNullable, ValueClass: wireType.Kind == InlineValue})
	}
	return params, nil
}

func (a *analyzer) parameterEnum(t TypeRef) bool {
	if t.Kind != Named {
		return false
	}
	for _, c := range a.components {
		if t.Name == a.options.PackageName+".models."+c.Name && c.Scalar == "" && has(c.Schema, "enum") {
			return true
		}
	}
	return false
}

func parameterFormat(v *document.Value, location string, kind TypeKind, pointer string) (CollectionFormat, *diagnostic.Diagnostic) {
	style := "simple"
	if location == "query" {
		style = "form"
	}
	if has(v, "style") {
		var ok bool
		style, ok = get(v, "style").Text()
		if !ok {
			return 0, invalid("style must be text.", pointer+"/style")
		}
	}
	exploded := style == "form"
	if has(v, "explode") {
		exploded, _ = get(v, "explode").Bool()
	}
	if location != "query" {
		if location == "path" && kind != JSONObject {
			switch style {
			case "label":
				if exploded {
					return LabelExplodedParameter, nil
				}
				return LabelParameter, nil
			case "matrix":
				if exploded {
					return MatrixExplodedParameter, nil
				}
				return MatrixParameter, nil
			}
		}
		if style != "simple" {
			return 0, operationUnsupported(pointer + "/style")
		}
		if kind == List {
			return CSVParameter, nil
		}
		return ScalarParameter, nil
	}
	if kind == Map || kind == JSONObject {
		if style == "deepObject" && kind == Map {
			if has(v, "explode") && !exploded {
				return 0, operationUnsupported(pointer + "/explode")
			}
			return MapDeepObjectParameter, nil
		}
		if style != "form" {
			return 0, operationUnsupported(pointer + "/style")
		}
		if !exploded {
			return MapCSVParameter, nil
		}
		return MapParameter, nil
	}
	if kind != List {
		if style != "form" {
			return 0, operationUnsupported(pointer + "/style")
		}
		return ScalarParameter, nil
	}
	if !slices.Contains([]string{"form", "spaceDelimited", "pipeDelimited", "deepObject"}, style) {
		return 0, operationUnsupported(pointer + "/style")
	}
	if style == "deepObject" {
		return IndexedParameter, nil
	}
	if exploded {
		return MultiParameter, nil
	}
	switch style {
	case "spaceDelimited":
		return SpaceParameter, nil
	case "pipeDelimited":
		return PipeParameter, nil
	default:
		return CSVParameter, nil
	}
}

func mergeParameters(base, override []Parameter) []Parameter {
	// Operation-local parameters precede inherited ones within each location;
	// this is part of the Kotlin positional call-site contract.
	out := slices.Clone(override)
	for _, p := range base {
		if !slices.ContainsFunc(override, func(b Parameter) bool { return b.Location == p.Location && b.Wire == p.Wire }) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return parameterPosition(out[i]) < parameterPosition(out[j])
	})
	return out
}

func parameterPosition(p Parameter) int {
	if p.DeepObject != nil {
		return 2
	}
	switch p.Location {
	case "path":
		return 0
	case "query":
		return 1
	default:
		return 3
	}
}

func nameParameters(params []Parameter, pointer string, body *RequestBody) *diagnostic.Diagnostic {
	// Allocate globally before emission. A naturally named idQuery must not be
	// stolen when a colliding id needs a suffix, irrespective of source order.
	indices := make([]int, len(params))
	counts := map[string]int{}
	slots := 0
	reserved := map[string]bool{"requestBuilder": true}
	if body != nil {
		slots = body.Type.Slots()
		reserved["input"] = true
	}
	for i := range params {
		indices[i] = i
		name, err := naming.APICamelName(params[i].Wire, pointer+"/parameters")
		if params[i].Location == "header" && strings.EqualFold(params[i].Wire, "Last-Event-ID") {
			name = "lastEventId"
			err = nil
		}
		if err != nil {
			return err
		}
		params[i].Name = name
		counts[name]++
		slots += params[i].Type.Slots()
	}
	if slots > MaxMembers {
		return diagnostic.At("E_OPERATION_LIMIT", "Operation exceeds the supported JVM parameter size.", pointer+"/parameters")
	}
	sort.SliceStable(indices, func(i, j int) bool {
		a, b := params[indices[i]], params[indices[j]]
		if a.Location != b.Location {
			return slices.Index([]string{"path", "query", "header"}, a.Location) < slices.Index([]string{"path", "query", "header"}, b.Location)
		}
		return a.Wire < b.Wire
	})
	taken := map[string]bool{}
	for name := range reserved {
		taken[name] = true
	}
	for _, p := range params {
		if counts[p.Name] == 1 && !reserved[p.Name] {
			taken[p.Name] = true
		}
	}
	for _, i := range indices {
		p := &params[i]
		if counts[p.Name] == 1 && !reserved[p.Name] {
			continue
		}
		base := p.Name
		if taken[base] {
			location := strings.ToUpper(p.Location[:1]) + p.Location[1:]
			base += location
			candidate := base
			for n := 2; taken[candidate]; n++ {
				candidate = base + strconv.Itoa(n)
			}
			p.Name = candidate
		}
		taken[p.Name] = true
	}
	return nil
}

func pathParameters(path string, params []Parameter, pointer string) *diagnostic.Diagnostic {
	used := map[string]bool{}
	for rest := path; rest != ""; {
		start := strings.IndexAny(rest, "{}")
		if start < 0 {
			break
		}
		if rest[start] != '{' {
			return invalid("Malformed path template.", pointer)
		}
		end := strings.IndexByte(rest[start+1:], '}')
		if end < 0 {
			return invalid("Malformed path template.", pointer)
		}
		wire := rest[start+1 : start+1+end]
		if wire == "" || strings.Contains(wire, "{") {
			return invalid("Malformed path template.", pointer)
		}
		if !slices.ContainsFunc(params, func(p Parameter) bool { return p.Location == "path" && p.Wire == wire }) {
			return invalid("Path template parameter is not declared.", pointer+"/parameters")
		}
		used[wire] = true
		rest = rest[start+end+2:]
	}
	for _, p := range params {
		if p.Location == "path" && !used[p.Wire] {
			return invalid("Declared path parameter is not in the template.", pointer+"/parameters")
		}
	}
	return nil
}

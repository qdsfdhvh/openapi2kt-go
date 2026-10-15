package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// SSEEvents is transport metadata, not a JSON envelope serializer. Payload
// declarations and the client decoder share the same planned TypeRef.
type SSEEvents struct {
	Plain     bool
	ClassName string
	Variants  []SSEVariant
}
type SSEVariant struct {
	Name, Event string
	Data        *TypeRef
	Required    bool
}

func (m *SSEEvents) Name() string { return m.ClassName }
func (*SSEEvents) declaration()   {}

func sseMedia(content *document.Value) string {
	for _, entry := range httpMembers(content) {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(entry.Name, ";", 2)[0]), "text/event-stream") {
			return entry.Name
		}
	}
	return ""
}

func (a *analyzer) sseSchema(v *document.Value, owner, pointer string) (*SSEEvents, []Declaration, *diagnostic.Diagnostic) {
	name, err := a.operationModelName(owner, "Event", pointer)
	if err != nil {
		return nil, nil, err
	}
	result := &SSEEvents{ClassName: name}
	var declarations []Declaration
	used := map[string]bool{strings.ToLower(name): true, "companion": true}
	seen := map[string]bool{}
	var visit func(*document.Value, string, int) *diagnostic.Diagnostic
	visit = func(v *document.Value, p string, depth int) *diagnostic.Diagnostic {
		if err := depthCheck(depth, p); err != nil {
			return err
		}
		if err := object(v, p); err != nil {
			return err
		}
		if has(v, "$ref") {
			if err := checkKeys(v, []string{"$ref"}, p); err != nil {
				return err
			}
			if get(v, "$ref").Kind() != document.String {
				return invalid("$ref must be a string.", p)
			}
			target, steps, err := a.resolveNominal(text(get(v, "$ref")), p, depth)
			if err != nil {
				return err
			}
			return visit(target.Schema, target.Pointer, depth+steps)
		}
		union, err := a.readUnion(v, p)
		if err != nil {
			return err
		}
		if union != nil {
			if union.HasNull || union.Nullable {
				return operationUnsupported(p)
			}
			for _, branch := range union.Branches {
				if err := visit(branch.Schema, branch.Pointer, depth+1); err != nil {
					return err
				}
			}
			return nil
		}
		kind, nullable, err := schemaType(v, p)
		if err != nil {
			return err
		}
		if kind != "object" || nullable {
			return operationUnsupported(p)
		}
		if err := checkKeys(v, []string{"type", "properties", "required"}, p); err != nil {
			return err
		}
		properties := get(v, "properties")
		if err := object(properties, p+"/properties"); err != nil {
			return err
		}
		for _, field := range properties.Members() {
			if field.Name != "event" && field.Name != "data" && field.Name != "id" {
				return operationUnsupported(naming.Pointer(p+"/properties", field.Name))
			}
		}
		required := map[string]bool{}
		if value := get(v, "required"); value != nil {
			if value.Kind() != document.Array {
				return invalid("required must be an array.", p+"/required")
			}
			for i, entry := range value.Items() {
				field, valid := entry.Text()
				if !valid || required[field] || !has(properties, field) {
					return invalid("required must name unique declared fields.", naming.Pointer(p+"/required", strconv.Itoa(i)))
				}
				required[field] = true
			}
		}
		event := "message"
		if field := get(properties, "event"); field != nil {
			ep := p + "/properties/event"
			constant, err := constant("event", "event", field, ep, depth+1)
			if err != nil {
				return err
			}
			if constant.Type.Name != "kotlin.String" || get(field, "const") == nil || get(field, "const").Kind() != document.String {
				return operationUnsupported(ep)
			}
			event = text(get(field, "const"))
			if event == "" || strings.ContainsAny(event, "\r\n\x00") {
				return invalid("SSE event must be nonempty text without line breaks.", ep+"/const")
			}
		}
		if field := get(properties, "id"); field != nil {
			ip := p + "/properties/id"
			kind, _, err := schemaType(field, ip)
			if err != nil {
				return err
			}
			scalar, err := scalarType(kind, field, ip)
			if err != nil {
				return err
			}
			if scalar != "kotlin.String" {
				return operationUnsupported(ip)
			}
		}
		if seen[event] {
			return invalid("SSE event names must be unique.", p)
		}
		seen[event] = true
		if len(result.Variants) == MaxMembers {
			return diagnostic.At("E_UNION_LIMIT", "SSE exceeds the supported 240-event limit.", p)
		}
		variantName, err := naming.ClassName(event, p)
		if err != nil {
			return err
		}
		base := variantName
		for n := 2; used[strings.ToLower(variantName)]; n++ {
			variantName = base + strconv.Itoa(n)
		}
		used[strings.ToLower(variantName)] = true
		variant := SSEVariant{Name: variantName, Event: event, Required: required["data"]}
		if data := get(properties, "data"); data != nil {
			dataName, err := a.operationModelName(owner, "Data"+variantName, p+"/properties/data")
			if err != nil {
				return err
			}
			start := len(declarations)
			context := *a
			context.usage = SharedUsage
			ty, err := context.propertyType(data, dataName, p+"/properties/data", &declarations, map[string]bool{}, depth+1)
			if err != nil {
				return err
			}
			a.warnings = context.warnings
			// Container payloads can own Value/Inner peers instead of the root
			// data name. Qualify every newly planned root declaration so client
			// decoders and event models use the exact same resolved type tree.
			for _, decl := range declarations[start:] {
				ty = qualifyOperationType(ty, decl.Name(), a.options.PackageName+".api.models."+decl.Name())
			}
			if !variant.Required {
				ty = ty.Null()
			}
			variant.Data = &ty
		}
		result.Variants = append(result.Variants, variant)
		return nil
	}
	if err := visit(v, pointer, 0); err != nil {
		return nil, nil, err
	}
	if len(result.Variants) == 0 {
		return nil, nil, invalid("SSE must declare at least one event.", pointer)
	}
	declarations = append(declarations, result)
	return result, declarations, nil
}

func sseParameters(params []Parameter, pointer string, body *RequestBody) ([]Parameter, *diagnostic.Diagnostic) {
	found := false
	for _, p := range params {
		if p.Location == "header" && strings.EqualFold(p.Wire, "Last-Event-ID") {
			if found {
				return nil, invalid("SSE Last-Event-ID header must be declared only once.", pointer+"/parameters")
			}
			ty := p.Type
			if ty.Kind == Nullable {
				ty = *ty.Element
			}
			if ty.Kind != Scalar || ty.Name != "kotlin.String" || p.Enum {
				return nil, operationUnsupported(pointer + "/parameters")
			}
			found = true
		}
	}
	if !found {
		params = append(params, Parameter{Name: "lastEventId", Wire: "Last-Event-ID", Location: "header", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}.Null()})
	}
	if err := nameParameters(params, pointer, body); err != nil {
		return nil, err
	}
	// Keep the transport cursor's public name even when another location
	// declares lastEventId. Respect naturally occupied suffixes when moving it.
	used := map[string]bool{}
	for _, p := range params {
		used[p.Name] = true
	}
	for i := range params {
		p := &params[i]
		if p.Name == "lastEventId" && !(p.Location == "header" && strings.EqualFold(p.Wire, "Last-Event-ID")) {
			base := "lastEventId" + strings.ToUpper(p.Location[:1]) + p.Location[1:]
			name := base
			for n := 2; used[name]; n++ {
				name = base + strconv.Itoa(n)
			}
			p.Name = name
			used[name] = true
		}
	}
	for i := range params {
		if params[i].Location == "header" && strings.EqualFold(params[i].Wire, "Last-Event-ID") {
			params[i].Name = "lastEventId"
		}
	}
	return params, nil
}

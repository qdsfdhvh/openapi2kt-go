package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// JSON keeps precedence. Exact form spelling precedes case/parameter variants;
// ties use the same sorted media inventory as the rest of HTTP planning.
func formRequestMedia(content *document.Value) string {
	best := ""
	for _, entry := range httpMembers(content) {
		if entry.Name == "application/x-www-form-urlencoded" {
			return entry.Name
		}
		base := strings.ToLower(strings.TrimSpace(strings.SplitN(entry.Name, ";", 2)[0]))
		if best == "" && base == "application/x-www-form-urlencoded" {
			best = entry.Name
		}
	}
	return best
}

// The ordinary request planner validates every source first. This selected-only
// profile additionally needs an actual retained DTO and a closed flat grammar;
// it never grants form encoding to a projected map, alias or nominal wrapper.
func formRequestFields(v *document.Value, pointer string, declarations []Declaration) ([]Property, *diagnostic.Diagnostic) {
	if v == nil || v.Kind() != document.Object {
		return nil, operationUnsupported(pointer)
	}
	if err := operationKeys(v, []string{"type", "properties", "required", "title", "description"}, pointer); err != nil {
		return nil, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	properties := get(v, "properties")
	if kind != "object" || nullable || properties == nil || len(properties.Members()) == 0 || len(declarations) != 1 {
		return nil, operationUnsupported(pointer)
	}
	model, ok := declarations[0].(*Model)
	if !ok || model.Nullable || len(model.Constants) != 0 || model.PropertyCount != nil || len(model.Properties) != len(properties.Members()) {
		return nil, operationUnsupported(pointer)
	}
	enums := map[string]*StringEnum{}
	for _, nested := range model.Nested {
		enum, ok := nested.(*StringEnum)
		if !ok {
			return nil, operationUnsupported(pointer)
		}
		enums[enum.ClassName] = enum
	}
	for _, entry := range properties.Members() {
		p := naming.Pointer(pointer+"/properties", entry.Name)
		if err := operationKeys(entry.Value, []string{"type", "format", "enum", "default", "title", "description", "minLength", "maxLength", "pattern", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"}, p); err != nil {
			return nil, err
		}
		if _, nullable, err := schemaType(entry.Value, p); err != nil {
			return nil, err
		} else if nullable {
			return nil, operationUnsupported(p)
		}
		if values := get(entry.Value, "enum"); values != nil && len(values.Items()) > MaxMembers {
			return nil, diagnostic.At("E_MODEL_LIMIT", "Form enum exceeds the supported 240-member limit.", naming.Pointer(p, "enum"))
		}
	}
	for _, field := range model.Properties {
		ty := field.Type
		if ty.Kind == Nullable && field.DefaultNull {
			ty = ty.WithoutRootNull() // Optional property null means omitted form key.
		} else if field.DefaultNull {
			return nil, operationUnsupported(naming.Pointer(pointer+"/properties", field.WireName))
		}
		// The ordinary planner validates each own annotation before presence
		// wrapping and required suppression. Encode the actual getter: an
		// optional constructor default sends a value; explicit null omits it.
		if ty.Serializer != "" {
			return nil, operationUnsupported(naming.Pointer(pointer+"/properties", field.WireName))
		}
		if ty.Kind == Named && enums[ty.Name] != nil {
			// Only an actual retained local String vocabulary owns serialName.
			// Alias/reference and formatted/nominal source roles stay separate.
			source := get(properties, field.WireName)
			if !has(source, "enum") || has(source, "format") {
				return nil, operationUnsupported(naming.Pointer(pointer+"/properties", field.WireName))
			}
			continue
		}
		if ty.Kind != Scalar {
			return nil, operationUnsupported(naming.Pointer(pointer+"/properties", field.WireName))
		}
		switch ty.Name {
		case "kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double",
			"kotlin.uuid.Uuid", "kotlinx.datetime.LocalDate", "kotlin.time.Instant":
		default:
			return nil, operationUnsupported(naming.Pointer(pointer+"/properties", field.WireName))
		}
	}
	return model.Properties, nil
}

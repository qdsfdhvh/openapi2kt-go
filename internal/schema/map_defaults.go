package schema

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Only ordinary DTO fields grant this default role. Source nodes remain borrowed;
// dictionary keys are data, including keys which spell schema keywords.
func (a *analyzer) primitiveMapDefault(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, "", err
	}
	if err := checkKeys(v, a.collectionKeys([]string{"type", "nullable", "properties", "required", "additionalProperties", "default"}), pointer); err != nil {
		return TypeRef{}, "", err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	if kind != "object" || !dictionarySchema(v) {
		return TypeRef{}, "", unsupported(naming.Pointer(pointer, "default"))
	}
	if required, ok := v.Get("required"); ok && (required.Kind() != document.Array || len(required.Items()) != 0) {
		return TypeRef{}, "", invalid("Dictionary schemas cannot require undeclared fixed properties.", naming.Pointer(pointer, "required"))
	}
	item := get(v, "additionalProperties")
	itemPointer := naming.Pointer(pointer, "additionalProperties")
	element, err := a.collectionDefaultElement(item, name+"Value", itemPointer, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	itemNull := element.Kind == Nullable
	leaf := bare(element)
	scalar := leaf.Name
	if leaf.Kind == Named {
		scalar = "kotlin.String" // Source literal validation; the type stays nominal.
	}
	ty, err := a.checkedCollection(TypeRef{Kind: Map, Element: &element}, v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	if err := a.bindInlineEnumCollectionChildDefault(item, itemPointer, ty.Element); err != nil {
		return TypeRef{}, "", err
	}
	if err := a.bindInlineBuiltinChildDefault(item, itemPointer, ty.Element); err != nil {
		return TypeRef{}, "", err
	}
	collection := ty.Collection
	if nullable {
		ty = ty.Null()
	}
	value := get(v, "default")
	at := naming.Pointer(pointer, "default")
	if value.Kind() == document.Null {
		if !nullable {
			return TypeRef{}, "", invalid("A null dictionary default requires explicit schema nullability.", at)
		}
		return ty, "null", nil
	}
	if value.Kind() != document.Object {
		return TypeRef{}, "", invalid("A dictionary field default must be a JSON object.", at)
	}
	if collection != nil {
		count := int64(len(value.Members()))
		if collection.Minimum != nil && count < *collection.Minimum || collection.Maximum != nil && count > *collection.Maximum {
			return TypeRef{}, "", invalid("Dictionary default must satisfy its retained collection counts.", at)
		}
	}
	// Validate in source order for stable diagnostic precedence; sort only the
	// resulting key/value pairs so semantic default identity ignores key order.
	type entry struct{ key, value string }
	entries := make([]entry, 0, len(value.Members()))
	for _, member := range value.Members() {
		location := naming.Pointer(at, member.Name)
		encoded := "null"
		if member.Value.Kind() == document.Null {
			if !itemNull {
				return TypeRef{}, "", invalid("A null default value requires a nullable value schema.", location)
			}
		} else {
			if _, err := scalarLiteral(scalar, member.Value, location); err != nil {
				return TypeRef{}, "", invalid("Default value must fit its mapped Kotlin value type.", location)
			}
			if scalar == "kotlin.time.Instant" && !collectionDefaultInstantPattern.MatchString(text(member.Value)) {
				return TypeRef{}, "", invalid("Timestamp default must fit the Kotlin Instant format and precision.", location)
			}
			if err := a.collectionDefaultMemberChecks(member.Value, item, leaf, itemPointer, location); err != nil {
				return TypeRef{}, "", err
			}
			switch scalar {
			case "kotlin.Float", "kotlin.Double":
				number, _ := floatLiteral(member.Value)
				bits := 64
				if scalar == "kotlin.Float" {
					number = float64(float32(number))
					bits = 32
				}
				encoded = strconv.FormatFloat(number, 'g', -1, bits)
			case "kotlin.Int", "kotlin.Long":
				number, _ := integerLiteral(member.Value)
				encoded = strconv.FormatInt(number, 10)
			default:
				bytes, marshalErr := member.Value.MarshalJSON()
				if marshalErr != nil {
					return TypeRef{}, "", invalid("Invalid dictionary default value.", location)
				}
				encoded = string(bytes)
			}
		}
		entries = append(entries, entry{member.Name, encoded})
	}
	slices.SortFunc(entries, func(x, y entry) int { return strings.Compare(x.key, y.key) })
	members := make([]string, 0, len(entries))
	for _, entry := range entries {
		key, _ := json.Marshal(entry.key)
		members = append(members, string(key)+":"+entry.value)
	}
	encoded := "{" + strings.Join(members, ",") + "}"
	// JVM method names cannot contain dots, even in backtick identifiers.
	if leaf.Kind == Named {
		return ty, "`openapi-kt default enum map`(" + naming.Literal(encoded) + ")", nil
	}
	name = scalar[strings.LastIndex(scalar, ".")+1:]
	helper := fmt.Sprintf("`openapi-kt default map %s`", name)
	if itemNull {
		helper = fmt.Sprintf("`openapi-kt default map %s nullable`", name)
	}
	return ty, helper + "(" + naming.Literal(encoded) + ")", nil
}

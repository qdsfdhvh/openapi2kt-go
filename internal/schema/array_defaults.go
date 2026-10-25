package schema

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Go's RFC3339 parser admits spellings that Kotlin Instant cannot decode:
// single-digit hours, comma fractions, excess precision and oversized offsets.
// Keep collection defaults within both parsers; scalarLiteral checks the date.
var collectionDefaultInstantPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]((0[0-9]|1[0-7]):[0-5][0-9]|18:00))$`)

func bareArrayDefaultSchema(v *document.Value) bool {
	kind, _, err := schemaType(v, "")
	return err == nil && kind == "array"
}

// Direct DTO scalar-array defaults borrow their original schema and value members.
// Container and element checks retain their original schema provenance; the
// default annotation never grants a lost-check path. Date/date-time leaves use
// the same literal validation and serializers as ordinary temporal fields.
func (a *analyzer) primitiveArrayDefault(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, "", err
	}
	if err := checkKeys(v, a.collectionKeys([]string{"type", "nullable", "items", "prefixItems", "default"}), pointer); err != nil {
		return TypeRef{}, "", err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	if kind != "array" {
		return TypeRef{}, "", unsupported(naming.Pointer(pointer, "default"))
	}
	item := get(v, "items")
	itemPointer := naming.Pointer(pointer, "items")
	positional := has(v, "prefixItems")
	if positional {
		decision, err := a.arrayItemsOwnDefault(v, pointer, depth, true)
		if err != nil {
			return TypeRef{}, "", err
		}
		if decision.Projection != nil {
			return TypeRef{}, "", unsupported(naming.Pointer(pointer, "prefixItems"))
		}
		item, itemPointer = decision.Selected, decision.Pointer
		if has(item, "$ref") {
			return TypeRef{}, "", unsupported(naming.Pointer(pointer, "prefixItems"))
		}
	}
	element, err := a.collectionDefaultElement(item, name+"Inner", itemPointer, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	itemNull := element.Kind == Nullable
	leaf := bare(element)
	scalar := leaf.Name
	if leaf.Kind == Named {
		scalar = "kotlin.String" // Validate source strings without erasing the enum API.
	}
	ty, err := a.checkedCollection(TypeRef{Kind: List, Element: &element}, v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	collection := ty.Collection
	if nullable {
		ty = ty.Null()
	}
	value := get(v, "default")
	at := naming.Pointer(pointer, "default")
	if positional {
		work, bytes := 0, 0
		if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
			return TypeRef{}, "", err
		}
	}
	if value.Kind() == document.Null {
		if !nullable {
			return TypeRef{}, "", invalid("A null array default requires explicit schema nullability.", at)
		}
		return ty, "null", nil
	}
	if value.Kind() != document.Array {
		return TypeRef{}, "", invalid("An array field default must be a JSON array.", at)
	}
	if collection != nil {
		count := int64(len(value.Items()))
		if collection.Minimum != nil && count < *collection.Minimum || collection.Maximum != nil && count > *collection.Maximum {
			return TypeRef{}, "", invalid("Array default must satisfy its retained collection counts.", at)
		}
	}
	var seen map[string]bool
	if collection != nil && collection.Unique {
		seen = make(map[string]bool)
	}
	members := make([]string, 0, len(value.Items()))
	for index, member := range value.Items() {
		location := fmt.Sprintf("%s/%d", at, index)
		if seen != nil {
			key := uniqueScalarDefaultKey(leaf, member)
			if seen[key] {
				return TypeRef{}, "", invalid("Array default members must satisfy uniqueItems.", location)
			}
			seen[key] = true
		}
		if member.Kind() == document.Null {
			if !itemNull {
				return TypeRef{}, "", invalid("A null default member requires a nullable element schema.", location)
			}
			members = append(members, "null")
			continue
		}
		if _, err := scalarLiteral(scalar, member, location); err != nil {
			return TypeRef{}, "", invalid("Default member must fit its mapped Kotlin element type.", location)
		}
		if scalar == "kotlin.time.Instant" && !collectionDefaultInstantPattern.MatchString(text(member)) {
			return TypeRef{}, "", invalid("Timestamp default must fit the Kotlin Instant format and precision.", location)
		}
		if err := a.collectionDefaultMemberChecks(member, item, leaf, itemPointer, location); err != nil {
			return TypeRef{}, "", err
		}
		switch scalar {
		case "kotlin.Float", "kotlin.Double":
			number, _ := floatLiteral(member)
			bits := 64
			if scalar == "kotlin.Float" {
				number = float64(float32(number))
				bits = 32
			}
			members = append(members, strconv.FormatFloat(number, 'g', -1, bits))
		default:
			encoded, marshalErr := member.MarshalJSON()
			if marshalErr != nil {
				return TypeRef{}, "", invalid("Invalid array default member.", location)
			}
			members = append(members, string(encoded))
		}
	}
	encoded := "[" + strings.Join(members, ",") + "]"
	if positional && len(encoded) > maxNestedDefaultBytes {
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical tuple field default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	// A JVM method name cannot contain dots, including in a backtick identifier.
	if leaf.Kind == Named {
		return ty, "`openapi-kt default enum array`(" + naming.Literal(encoded) + ")", nil
	}
	name = scalar[strings.LastIndex(scalar, ".")+1:]
	helper := fmt.Sprintf("`openapi-kt default array %s`", name)
	if itemNull {
		helper = fmt.Sprintf("`openapi-kt default array %s nullable`", name)
	}
	return ty, helper + "(" + naming.Literal(encoded) + ")", nil
}

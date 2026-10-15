package kotlin

import (
	"fmt"
	"slices"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func emitHierarchy(node *schema.Hierarchy, owner string, hints schema.EmissionHints, out *strings.Builder) {
	for _, enum := range node.Enums {
		emitStringEnum(enum, 0, owner, out)
		out.WriteByte('\n')
	}
	if node.Leaf != nil {
		emitModelContext(node.Leaf, 0, owner, node.Parent, node.Fields, node.SerialName, node.NonNullSingletonCodec, hints, out)
		return
	}
	name, base, unknown := node.ClassName, owner+"."+node.ClassName, owner+".Unknown"+node.ClassName
	if node.Content() {
		fmt.Fprintf(out, "@kotlinx.serialization.Serializable(with = %sSerializer::class)\n", name)
	} else {
		out.WriteString("@kotlinx.serialization.Serializable\n")
	}
	superclass := ""
	if node.Parent != "" {
		superclass = " : " + node.Parent + "()"
	}
	fmt.Fprintf(out, "public sealed class %s%s {\n", name, superclass)
	for _, field := range node.Fields {
		override := ""
		if node.Parent != "" && !slices.Contains(node.OwnFields, field.WireName) {
			override = "override "
		}
		fmt.Fprintf(out, "    public abstract %sval %s: %s\n", override, field.Name, RenderType(field.Type))
	}
	out.WriteString("}\n")
	if !node.Content() {
		return
	}
	emitUnknown(name, 0, owner, node.Inherited, true, node.Plain, out)
	fmt.Fprintf(out, "\ninternal object %sSerializer : kotlinx.serialization.json.JsonContentPolymorphicSerializer<%s>(%s::class) {\n    override fun selectDeserializer(element: kotlinx.serialization.json.JsonElement): kotlinx.serialization.DeserializationStrategy<%s> {\n", name, base, base, base)
	if len(node.Tags) == 0 && len(node.TagCombinations) == 0 {
		fmt.Fprintf(out, "        return %s.serializer()\n", unknown)
	} else {
		fmt.Fprintf(out, "        val obj = element.jsonObject\n")
		if len(node.Tags) != 0 {
			fmt.Fprintf(out, "        val tag = obj[%s]?.jsonPrimitive?.content\n", naming.Literal(node.TagField))
		}
		fmt.Fprintf(out, "        return when {\n")
		for _, rule := range node.TagCombinations {
			fmt.Fprintf(out, "            %s -> %s.%s.serializer()\n", objectTagConditions(rule.Matches), owner, rule.Child)
		}
		for _, tag := range node.Tags {
			fmt.Fprintf(out, "            tag == %s -> %s.%s.serializer()\n", naming.Literal(tag.Value), owner, tag.Child)
		}
		fmt.Fprintf(out, "            else -> %s.serializer()\n        }\n", unknown)
	}
	out.WriteString("    }\n}\n")
}

package kotlin

import (
	"fmt"
	"slices"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func hasPrimitiveUnion(d schema.Declaration) bool {
	switch d := d.(type) {
	case *schema.PrimitiveUnion:
		return true
	case *schema.Hierarchy:
		return d.Leaf != nil && hasPrimitiveUnion(d.Leaf)
	case *schema.ObjectUnion:
		for _, variant := range d.Variants {
			if hasPrimitiveUnion(variant) {
				return true
			}
		}
	case *schema.Model:
		for _, nested := range d.Nested {
			if hasPrimitiveUnion(nested) {
				return true
			}
		}
	}
	return false
}

func emitPrimitiveUnion(model *schema.PrimitiveUnion, indent int, owner string, hints schema.EmissionHints, out *strings.Builder) {
	pad, name := strings.Repeat("    ", indent), model.ClassName
	base := owner + "." + name
	fmt.Fprintf(out, "%s@kotlinx.serialization.Serializable(with = %sSerializer::class)\n%spublic sealed class %s\n", pad, name, pad, name)
	if model.Object != nil {
		out.WriteByte('\n')
		emitModelWithSuper(model.Object, indent, owner, base, hints, out)
	}
	if model.NullVariant {
		emitNullVariant(name, indent, owner, nil, out)
	}
	for _, kind := range model.Kinds {
		variant := kind.Prefix() + name
		qualified := owner + "." + variant
		fmt.Fprintf(out, "\n%s@kotlinx.serialization.Serializable(with = %sSerializer::class)\n%spublic %s %s(\n%s    public val value: %s,\n%s) : %s()", pad, variant, pad, classKeyword(model.Plain), variant, pad, kind.Scalar(), pad, base)
		if model.Plain {
			out.WriteString(" {\n")
			emitPlainMembers(variant, []schema.Property{{Name: "value"}}, nil, indent+1, out)
			fmt.Fprintf(out, "%s}\n", pad)
		} else {
			out.WriteByte('\n')
		}
		fmt.Fprintf(out, "\n%sinternal object %sSerializer : kotlinx.serialization.KSerializer<%s> {\n%s    override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor =\n%s        kotlinx.serialization.descriptors.PrimitiveSerialDescriptor(%s, kotlinx.serialization.descriptors.PrimitiveKind.%s)\n", pad, variant, qualified, pad, pad, naming.Literal(qualified), kind.Descriptor())
		fmt.Fprintf(out, "\n%s    override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s) {\n%s        encoder.encode%s(value.value)\n%s    }\n\n%s    override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s =\n%s        %s(decoder.decode%s())\n%s}\n", pad, qualified, pad, kind.Method(), pad, pad, qualified, pad, qualified, kind.Method(), pad)
	}
	fmt.Fprintf(out, "\n%sinternal object %sSerializer : kotlinx.serialization.json.JsonContentPolymorphicSerializer<%s>(%s::class) {\n%s    override fun selectDeserializer(element: kotlinx.serialization.json.JsonElement): kotlinx.serialization.DeserializationStrategy<%s> = when {\n", pad, name, base, base, pad, base)
	if model.NullVariant {
		fmt.Fprintf(out, "%s        element == kotlinx.serialization.json.JsonNull -> %s.Null%s.serializer()\n", pad, owner, name)
	}
	if model.Object != nil {
		fmt.Fprintf(out, "%s        element is kotlinx.serialization.json.JsonObject -> %s.%s.serializer()\n", pad, owner, model.Object.ClassName)
	}
	if slices.Contains(model.Kinds, schema.PrimitiveString) {
		fmt.Fprintf(out, "%s        element is kotlinx.serialization.json.JsonPrimitive && element.isString -> %s.String%s.serializer()\n", pad, owner, name)
	}
	for _, kind := range []schema.PrimitiveKind{schema.PrimitiveBoolean, schema.PrimitiveInteger, schema.PrimitiveNumber} {
		if slices.Contains(model.Kinds, kind) {
			fmt.Fprintf(out, "%s        element is kotlinx.serialization.json.JsonPrimitive && !element.isString && element.%s != null -> %s.%s%s.serializer()\n", pad, kind.Reader(), owner, kind.Prefix(), name)
		}
	}
	fmt.Fprintf(out, "%s        else -> throw kotlinx.serialization.SerializationException(\"Unexpected JSON kind for a primitive union.\")\n%s    }\n%s}\n", pad, pad, pad)
}

func unknownDefault(ty schema.TypeRef) string {
	switch ty.Kind {
	case schema.Nullable:
		return "null"
	case schema.List:
		return "kotlin.collections.emptyList()"
	case schema.Scalar:
		switch ty.Name {
		case "kotlin.String":
			return `""`
		case "kotlin.Boolean":
			return "false"
		case "kotlin.Int", "kotlin.Long":
			return "0"
		case "kotlin.Float":
			return "0.0f"
		case "kotlin.Double":
			return "0.0"
		}
	}
	return ""
}

func emitNullVariant(name string, indent int, owner string, common []schema.Property, out *strings.Builder) {
	pad, ty := strings.Repeat("    ", indent), owner+".Null"+name
	fmt.Fprintf(out, "\n%s@kotlinx.serialization.Serializable(with = Null%sSerializer::class)\n%spublic data object Null%s : %s.%s() {\n", pad, name, pad, name, owner, name)
	for _, p := range common {
		if fallback := unknownDefault(p.Type); fallback != "" {
			fmt.Fprintf(out, "%s    @kotlinx.serialization.Transient\n%s    public override val %s: %s = %s\n", pad, pad, p.Name, RenderType(p.Type), fallback)
		} else {
			fmt.Fprintf(out, "%s    public override val %s: %s\n%s        get() = throw kotlin.IllegalStateException(\"Property is not available on the null variant.\")\n", pad, p.Name, RenderType(p.Type), pad)
		}
	}
	fmt.Fprintf(out, "%s}\n\n%sinternal object Null%sSerializer : kotlinx.serialization.KSerializer<%s> {\n%s    override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor = kotlinx.serialization.json.JsonElement.serializer().descriptor\n", pad, pad, name, ty, pad)
	fmt.Fprintf(out, "\n%s    override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s) {\n%s        val json = encoder as? kotlinx.serialization.json.JsonEncoder ?: throw kotlinx.serialization.SerializationException(\"Root null variants require JSON.\")\n%s        json.encodeJsonElement(kotlinx.serialization.json.JsonNull)\n%s    }\n", pad, ty, pad, pad, pad)
	fmt.Fprintf(out, "\n%s    override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s {\n%s        val json = decoder as? kotlinx.serialization.json.JsonDecoder ?: throw kotlinx.serialization.SerializationException(\"Root null variants require JSON.\")\n%s        if (json.decodeJsonElement() != kotlinx.serialization.json.JsonNull) throw kotlinx.serialization.SerializationException(\"Expected JSON null for a root null variant.\")\n%s        return %s\n%s    }\n%s}\n", pad, ty, pad, pad, pad, ty, pad, pad)
}

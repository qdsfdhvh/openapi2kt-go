package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func hasObjectUnion(declaration schema.Declaration) bool {
	switch d := declaration.(type) {
	case *schema.ObjectUnion:
		return d.Content() || d.NullVariant || modelsHaveObjectUnion(d.Variants)
	case *schema.Hierarchy:
		return d.Content() || d.Leaf != nil && hasObjectUnion(d.Leaf)
	case *schema.Model:
		for _, nested := range d.Nested {
			if hasObjectUnion(nested) {
				return true
			}
		}
	case *schema.PrimitiveUnion:
		if d.Object != nil {
			return hasObjectUnion(d.Object)
		}
	}
	return false
}
func modelsHaveObjectUnion(models []*schema.Model) bool {
	for _, model := range models {
		if hasObjectUnion(model) {
			return true
		}
	}
	return false
}

func emitSingletonSerializer(model *schema.Model, indent int, owner string, nonNullSingleton bool, out *strings.Builder) {
	pad, ty := strings.Repeat("    ", indent), owner+"."+model.ClassName
	emit := func(format string, args ...any) {
		fmt.Fprintf(out, nullLiteralTemplate(format, modelNeedsNullLiteralAliases(model)), args...)
	}
	emit("\n%sinternal object Serializer : kotlinx.serialization.KSerializer<%s> {\n%s    override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor = kotlinx.serialization.json.JsonElement.serializer().descriptor\n\n%s    override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s) {\n%s        (encoder as kotlinx.serialization.json.JsonEncoder).encodeJsonElement(kotlinx.serialization.json.JsonObject(kotlin.collections.mapOf(\n", pad, ty, pad, pad, ty, pad)
	for _, c := range model.Constants {
		if c.Type.Kind == schema.Scalar && c.Type.Name == "kotlinx.serialization.json.JsonNull" {
			emit("%s            %s to value.%s,\n", pad, naming.Literal(c.WireName), c.Name)
			continue
		}
		suffix := ""
		if c.Type.Kind == schema.Named {
			suffix = ".serialName"
		}
		emit("%s            %s to kotlinx.serialization.json.JsonPrimitive(value.%s%s),\n", pad, naming.Literal(c.WireName), c.Name, suffix)
	}
	emit("%s        )))\n%s    }\n\n%s    override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s {\n", pad, pad, pad, ty)
	if model.Nullable || nonNullSingleton {
		// Nullable hierarchy views keep non-null physical singleton codecs.
		// Only the surrounding nullable adapter may consume JSON null.
		emit("%s        val element = (decoder as kotlinx.serialization.json.JsonDecoder).decodeJsonElement()\n%s        if (element == kotlinx.serialization.json.JsonNull) throw kotlinx.serialization.SerializationException(\"Expected a non-null hierarchy value.\")\n", pad, pad)
	} else {
		emit("%s        (decoder as kotlinx.serialization.json.JsonDecoder).decodeJsonElement()\n", pad)
	}
	emit("%s        return %s\n%s    }\n%s}\n", pad, ty, pad, pad)
}

func emitUnknown(name string, indent int, owner string, common []schema.Property, enumFallback, plain bool, out *strings.Builder) {
	pad, base, unknown := strings.Repeat("    ", indent), owner+"."+name, owner+".Unknown"+name
	fmt.Fprintf(out, "\n%s@kotlinx.serialization.Serializable(with = Unknown%sSerializer::class)\n%spublic %s Unknown%s(\n%s    public val rawElement: kotlinx.serialization.json.JsonElement,\n%s) : %s() {\n", pad, name, pad, classKeyword(plain), name, pad, pad, base)
	if plain {
		reserved := []string{}
		for _, p := range common {
			reserved = append(reserved, p.Name)
		}
		emitPlainMembers("Unknown"+name, []schema.Property{{Name: "rawElement"}}, reserved, indent+1, out)
	}
	for _, p := range common {
		if fallback := unknownDefault(p.Type); fallback != "" {
			fmt.Fprintf(out, "%s    @kotlinx.serialization.Transient\n%s    public override val %s: %s = %s\n", pad, pad, p.Name, RenderType(p.Type), fallback)
		} else if enumFallback && p.Type.Kind == schema.Named {
			fmt.Fprintf(out, "%s    @kotlinx.serialization.Transient\n%s    public override val %s: %s = %s.Unrecognized(\"\")\n", pad, pad, p.Name, p.Type.Name, p.Type.Name)
		} else {
			fmt.Fprintf(out, "%s    public override val %s: %s\n%s        get() = throw kotlin.IllegalStateException(\"Property is not available on the Unknown variant.\")\n", pad, p.Name, RenderType(p.Type), pad)
		}
	}
	fmt.Fprintf(out, "%s}\n\n%sinternal object Unknown%sSerializer : kotlinx.serialization.KSerializer<%s> {\n%s    override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor = kotlinx.serialization.json.JsonElement.serializer().descriptor\n\n%s    override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s =\n%s        %s((decoder as kotlinx.serialization.json.JsonDecoder).decodeJsonElement())\n\n%s    override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s) {\n%s        (encoder as kotlinx.serialization.json.JsonEncoder).encodeJsonElement(value.rawElement)\n%s    }\n%s}\n", pad, pad, name, unknown, pad, pad, unknown, pad, unknown, pad, unknown, pad, pad, pad)
}

// Both object products and inferred hierarchy levels use the same finite
// OR-within-field, AND-between-fields JSON dispatch grammar.
func objectTagConditions(matches []schema.ObjectTagMatch) string {
	conditions := make([]string, 0, len(matches))
	for _, match := range matches {
		var pins []string
		for _, value := range match.Values {
			pins = append(pins, "obj["+naming.Literal(match.Wire)+"]?.jsonPrimitive?.content == "+naming.Literal(value))
		}
		condition := strings.Join(pins, " || ")
		if len(pins) > 1 {
			condition = "(" + condition + ")"
		}
		conditions = append(conditions, condition)
	}
	return strings.Join(conditions, " && ")
}

func emitObjectUnion(model *schema.ObjectUnion, indent int, owner string, hints schema.EmissionHints, out *strings.Builder) {
	pad, name := strings.Repeat("    ", indent), model.ClassName
	base, unknown := owner+"."+name, owner+".Unknown"+name
	if model.Content() || model.NullVariant {
		fmt.Fprintf(out, "%s@kotlinx.serialization.Serializable(with = %sSerializer::class)\n", pad, name)
	} else {
		fmt.Fprintf(out, "%s@kotlinx.serialization.Serializable\n", pad)
	}
	fmt.Fprintf(out, "%spublic sealed class %s {\n", pad, name)
	for _, p := range model.Common {
		fmt.Fprintf(out, "%s    public abstract val %s: %s\n", pad, p.Name, RenderType(p.Type))
	}
	for index, variant := range model.Variants {
		out.WriteByte('\n')
		var serialName *string
		if !model.Content() {
			serialName = &model.SerialNames[index]
		}
		emitModelContext(variant, indent+1, base, base, model.Common, serialName, false, hints, out)
	}
	fmt.Fprintf(out, "%s}\n", pad)
	if model.NullVariant {
		emitNullVariant(name, indent, owner, model.Common, out)
	}
	if !model.Content() {
		if model.NullVariant {
			emitNullablePolymorphic(model, indent, owner, out)
		}
		return
	}
	emitUnknown(name, indent, owner, model.Common, false, model.Plain, out)
	fmt.Fprintf(out, "\n%sinternal object %sSerializer : kotlinx.serialization.json.JsonContentPolymorphicSerializer<%s>(%s::class) {\n%s    override fun selectDeserializer(element: kotlinx.serialization.json.JsonElement): kotlinx.serialization.DeserializationStrategy<%s> {\n", pad, name, base, base, pad, base)
	if model.NullVariant {
		fmt.Fprintf(out, "%s        if (element == kotlinx.serialization.json.JsonNull) return %s.Null%s.serializer()\n", pad, owner, name)
	}
	if len(model.Tags) == 0 && len(model.Presence) == 0 && len(model.TagCombinations) == 0 {
		fmt.Fprintf(out, "%s        return %s.serializer()\n", pad, unknown)
	} else {
		fmt.Fprintf(out, "%s        val obj = element.jsonObject\n", pad)
		if len(model.Tags) > 0 {
			fmt.Fprintf(out, "%s        val tag = obj[%s]?.jsonPrimitive?.content\n", pad, naming.Literal(model.TagField))
		}
		fmt.Fprintf(out, "%s        return when {\n", pad)
		for _, rule := range model.TagCombinations {
			fmt.Fprintf(out, "%s            %s -> %s.%s.serializer()\n", pad, objectTagConditions(rule.Matches), base, model.Variants[rule.Index].ClassName)
		}
		for _, rule := range model.Tags {
			fmt.Fprintf(out, "%s            tag == %s -> %s.%s.serializer()\n", pad, naming.Literal(rule.Value), base, model.Variants[rule.Index].ClassName)
		}
		for _, rule := range model.Presence {
			fmt.Fprintf(out, "%s            %s in obj -> %s.%s.serializer()\n", pad, naming.Literal(rule.Value), base, model.Variants[rule.Index].ClassName)
		}
		fmt.Fprintf(out, "%s            else -> %s.serializer()\n%s        }\n", pad, unknown, pad)
	}
	fmt.Fprintf(out, "%s    }\n%s}\n", pad, pad)
}

func emitNullablePolymorphic(model *schema.ObjectUnion, indent int, owner string, out *strings.Builder) {
	pad, name := strings.Repeat("    ", indent), model.ClassName
	ty := owner + "." + name
	fmt.Fprintf(out, "\n%s@kotlin.OptIn(kotlinx.serialization.InternalSerializationApi::class)\n%sinternal object %sSerializer : kotlinx.serialization.KSerializer<%s> {\n%s    private val nonNull: kotlinx.serialization.KSerializer<%s> by lazy {\n%s        kotlinx.serialization.SealedClassSerializer(\n%s            %s,\n%s            %s::class,\n%s            kotlin.arrayOf<kotlin.reflect.KClass<out %s>>(\n", pad, pad, name, ty, pad, ty, pad, pad, naming.Literal(ty), pad, ty, pad, ty)
	for _, variant := range model.Variants {
		fmt.Fprintf(out, "%s                %s.%s::class,\n", pad, ty, variant.ClassName)
	}
	fmt.Fprintf(out, "%s            ),\n%s            kotlin.arrayOf<kotlinx.serialization.KSerializer<out %s>>(\n", pad, pad, ty)
	for _, variant := range model.Variants {
		fmt.Fprintf(out, "%s                %s.%s.serializer(),\n", pad, ty, variant.ClassName)
	}
	fmt.Fprintf(out, "%s            ),\n%s        )\n%s    }\n%s    override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor get() = nonNull.descriptor\n\n%s    override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s) {\n%s        val json = encoder as? kotlinx.serialization.json.JsonEncoder ?: throw kotlinx.serialization.SerializationException(\"Nullable root polymorphism requires JSON.\")\n%s        val element = if (value === %s.Null%s) kotlinx.serialization.json.JsonNull else json.json.encodeToJsonElement(nonNull, value)\n%s        json.encodeJsonElement(element)\n%s    }\n\n%s    override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s {\n%s        val json = decoder as? kotlinx.serialization.json.JsonDecoder ?: throw kotlinx.serialization.SerializationException(\"Nullable root polymorphism requires JSON.\")\n%s        val element = json.decodeJsonElement()\n%s        return if (element == kotlinx.serialization.json.JsonNull) %s.Null%s else json.json.decodeFromJsonElement(nonNull, element)\n%s    }\n%s}\n", pad, pad, pad, pad, pad, ty, pad, pad, owner, name, pad, pad, pad, ty, pad, pad, pad, owner, name, pad, pad)
}

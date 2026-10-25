package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func emitFalseValue(d *schema.FalseValue, indent int, owner string, out *strings.Builder) {
	if indent > 0 {
		var nested strings.Builder
		emitFalseValue(d, 0, owner, &nested)
		for _, line := range strings.SplitAfter(nested.String(), "\n") {
			if line == "" {
				continue
			}
			if line != "\n" {
				out.WriteString(strings.Repeat("    ", indent))
			}
			out.WriteString(line)
		}
		return
	}
	serializer := "Serializer"
	if d.ClassName == serializer {
		serializer += "2"
	}
	qualified := owner + "." + d.ClassName
	fmt.Fprintf(out, "@kotlinx.serialization.Serializable(with = %s.%s::class)\n", qualified, serializer)
	fmt.Fprintf(out, "public sealed class %s private constructor() {\n", d.ClassName)
	fmt.Fprintf(out, "    public object %s : kotlinx.serialization.KSerializer<%s> {\n", serializer, qualified)
	fmt.Fprintf(out, "        override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor = kotlinx.serialization.descriptors.buildClassSerialDescriptor(%s)\n", naming.Literal(qualified))
	fmt.Fprintf(out, "        override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s): kotlin.Unit =\n", qualified)
	out.WriteString("            throw kotlinx.serialization.SerializationException(\"Schema false admits no JSON value.\")\n")
	fmt.Fprintf(out, "        override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s =\n", qualified)
	out.WriteString("            throw kotlinx.serialization.SerializationException(\"Schema false admits no JSON value.\")\n")
	out.WriteString("    }\n")
	if d.AbsentSerializer {
		name := "AbsentSerializer"
		if d.ClassName == name {
			name += "2"
		}
		out.WriteString("\n    @kotlin.OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)\n")
		fmt.Fprintf(out, "    public object %s : kotlinx.serialization.KSerializer<%s?> {\n", name, qualified)
		fmt.Fprintf(out, "        override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor = %s.%s.descriptor.nullable\n", qualified, serializer)
		fmt.Fprintf(out, "        override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: %s?): kotlin.Unit =\n", qualified)
		out.WriteString("            throw kotlinx.serialization.SerializationException(\"A false field must be absent.\")\n")
		fmt.Fprintf(out, "        override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): %s? =\n", qualified)
		out.WriteString("            throw kotlinx.serialization.SerializationException(\"A false field must be absent.\")\n")
		out.WriteString("    }\n")
	}
	out.WriteString("}\n")
}

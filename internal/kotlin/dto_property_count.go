package kotlin

import (
	"fmt"
	"openapi-kt/internal/schema"
	"strings"
)

// Private nested symbol avoids public API additions and is allocated against all
// existing members, without reserving names in unconstrained declarations.
func propertyCountSerializerName(model *schema.Model) string {
	used := map[string]bool{strings.ToLower(model.ClassName): true}
	for _, p := range model.Properties {
		used[strings.ToLower(strings.Trim(p.Name, "`"))] = true
	}
	for _, p := range model.Constants {
		used[strings.ToLower(strings.Trim(p.Name, "`"))] = true
	}
	for _, d := range model.Nested {
		used[strings.ToLower(d.Name())] = true
	}
	name := "OpenapiKtPropertyCountSerializer"
	for i := 2; used[strings.ToLower(name)]; i++ {
		name = fmt.Sprintf("OpenapiKtPropertyCountSerializer%d", i)
	}
	return name
}

func emitPropertyCountSerializer(model *schema.Model, indent int, owner string, out *strings.Builder) {
	pad := strings.Repeat("    ", indent)
	ty := owner + "." + model.ClassName
	lines := []string{
		"", "private object " + propertyCountSerializerName(model) + " : kotlinx.serialization.KSerializer<" + ty + "> {",
		"    private val delegate: kotlinx.serialization.KSerializer<" + ty + "> by kotlin.lazy {",
		"        " + ty + ".generatedSerializer()", "    }",
		"    override val descriptor: kotlinx.serialization.descriptors.SerialDescriptor",
		"        get() = delegate.descriptor", "",
		"    private fun check(element: kotlinx.serialization.json.JsonElement): kotlinx.serialization.json.JsonObject {",
		"        if (element !is kotlinx.serialization.json.JsonObject) {",
		"            throw kotlinx.serialization.SerializationException(\"Expected a JSON object for property counts.\")", "        }",
	}
	if model.PropertyCount.Minimum != nil {
		lines = append(lines, fmt.Sprintf("        if (element.size < %d) throw kotlinx.serialization.SerializationException(\"Object violates minProperties.\")", *model.PropertyCount.Minimum))
	}
	if model.PropertyCount.Maximum != nil {
		lines = append(lines, fmt.Sprintf("        if (element.size > %d) throw kotlinx.serialization.SerializationException(\"Object violates maxProperties.\")", *model.PropertyCount.Maximum))
	}
	lines = append(lines,
		"        return element", "    }", "",
		"    override fun serialize(encoder: kotlinx.serialization.encoding.Encoder, value: "+ty+") {",
		"        if (encoder !is kotlinx.serialization.json.JsonEncoder) {",
		"            throw kotlinx.serialization.SerializationException(\"Property counts require JSON.\")", "        }",
		"        val element = encoder.json.encodeToJsonElement(delegate, value)",
		"        encoder.encodeJsonElement(check(element))", "    }", "",
		"    override fun deserialize(decoder: kotlinx.serialization.encoding.Decoder): "+ty+" {",
		"        if (decoder !is kotlinx.serialization.json.JsonDecoder) {",
		"            throw kotlinx.serialization.SerializationException(\"Property counts require JSON.\")", "        }",
		"        val element = check(decoder.decodeJsonElement())",
		"        return decoder.json.decodeFromJsonElement(delegate, element)", "    }", "}",
	)
	for _, line := range lines {
		if line == "" {
			out.WriteByte('\n')
		} else {
			fmt.Fprintf(out, "%s%s\n", pad, line)
		}
	}
}

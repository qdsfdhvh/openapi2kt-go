package kotlin

import (
	"strings"

	"openapi-kt/internal/schema"
)

func typeHasJSONUnique(ty schema.TypeRef) bool {
	return ty.Collection != nil && ty.Collection.UniqueJSON || ty.Element != nil && typeHasJSONUnique(*ty.Element)
}
func declarationHasJSONUnique(d schema.Declaration) bool {
	switch d := d.(type) {
	case *schema.Model:
		for _, p := range d.Properties {
			if typeHasJSONUnique(p.Type) {
				return true
			}
		}
		for _, n := range d.Nested {
			if declarationHasJSONUnique(n) {
				return true
			}
		}
	case *schema.ObjectUnion:
		for _, v := range d.Variants {
			if declarationHasJSONUnique(v) {
				return true
			}
		}
	case *schema.PrimitiveUnion:
		return d.Object != nil && declarationHasJSONUnique(d.Object)
	case *schema.Hierarchy:
		return d.Leaf != nil && declarationHasJSONUnique(d.Leaf)
	case *schema.SSEEvents:
		for _, v := range d.Variants {
			if v.Data != nil && typeHasJSONUnique(*v.Data) {
				return true
			}
		}
	}
	return false
}
func groupHasJSONUnique(group schema.APIGroup) bool {
	for _, op := range group.Operations {
		for _, p := range op.Parameters {
			if typeHasJSONUnique(p.Type) {
				return true
			}
		}
		if op.Body != nil && typeHasJSONUnique(op.Body.Type) || typeHasJSONUnique(op.Response) {
			return true
		}
	}
	return false
}
func emitJSONUniqueHelper(out *strings.Builder) { out.WriteString(jsonUniqueSource) }

const jsonUniqueSource = `
private fun ` + "`openapi-kt JSON number`" + `(raw: kotlin.String): kotlin.String {
    require(raw.length <= 256) { "JSON number exceeds supported uniqueItems precision." }
    require(raw.matches("-?(0|[1-9][0-9]*)(\\.[0-9]+)?([eE][+-]?[0-9]+)?".toRegex())) { "Expected a finite JSON number for uniqueItems." }
    val exponentAt = raw.indexOfAny(charArrayOf('e', 'E'))
    val exponent = if (exponentAt < 0) 0 else raw.substring(exponentAt + 1).toIntOrNull()
        ?: throw kotlin.IllegalArgumentException("JSON number exceeds supported uniqueItems exponent.")
    require(exponent in -512..512) { "JSON number exceeds supported uniqueItems exponent." }
    val mantissa = if (exponentAt < 0) raw else raw.substring(0, exponentAt)
    val negative = mantissa.startsWith('-')
    val unsigned = if (negative) mantissa.substring(1) else mantissa
    val point = unsigned.indexOf('.')
    val fraction = if (point < 0) 0 else unsigned.length - point - 1
    val digits = unsigned.replace(".", "").trimStart('0')
    if (digits.isEmpty()) return "0"
    val significant = digits.trimEnd('0')
    val power = exponent - fraction + digits.length - significant.length
    return (if (negative) "-" else "") + significant + "e" + power
}
private fun ` + "`openapi-kt JSON unique append`" + `(out: kotlin.text.StringBuilder, value: kotlin.String, budget: kotlin.IntArray) {
    require(value.length <= 8388608 - budget[1]) { "JSON value exceeds supported uniqueItems size." }
    budget[1] += value.length
    out.append(value)
}
private fun ` + "`openapi-kt JSON equality key`" + `(value: kotlinx.serialization.json.JsonElement?, depth: kotlin.Int, budget: kotlin.IntArray, out: kotlin.text.StringBuilder) {
    require(depth <= 48 && budget[0] < 16384) { "JSON value exceeds supported uniqueItems traversal." }
    budget[0]++
    fun token(text: kotlin.String) { ` + "`openapi-kt JSON unique append`" + `(out, text, budget) }
    when (value) {
        null, kotlinx.serialization.json.JsonNull -> token("n;")
        is kotlinx.serialization.json.JsonPrimitive -> {
            when {
                value.isString -> { token("s" + value.content.length + ":"); token(value.content) }
                value.content == "true" || value.content == "false" -> token("b" + value.content + ";")
                else -> { val number = ` + "`openapi-kt JSON number`" + `(value.content); token("d" + number.length + ":"); token(number) }
            }
        }
        is kotlinx.serialization.json.JsonArray -> {
            token("a" + value.size + "[")
            for (element in value) ` + "`openapi-kt JSON equality key`" + `(element, depth + 1, budget, out)
            token("]")
        }
        is kotlinx.serialization.json.JsonObject -> {
            require(value.size <= 16384 - budget[0]) { "JSON value exceeds supported uniqueItems traversal." }
            token("o" + value.size + "{")
            for (key in value.keys.sorted()) {
                token("s" + key.length + ":"); token(key)
                ` + "`openapi-kt JSON equality key`" + `(value[key], depth + 1, budget, out)
            }
            token("}")
        }
    }
}
private fun ` + "`openapi-kt unique JSON items`" + `(input: kotlin.collections.List<kotlinx.serialization.json.JsonElement?>) {
    val seen = kotlin.collections.mutableSetOf<kotlin.String>()
    val budget = kotlin.intArrayOf(0, 0)
    for (value in input) {
        val key = kotlin.text.StringBuilder()
        ` + "`openapi-kt JSON equality key`" + `(value, 0, budget, key)
        require(seen.add(key.toString())) { "Field value contains duplicate entries for uniqueItems." }
    }
}
`

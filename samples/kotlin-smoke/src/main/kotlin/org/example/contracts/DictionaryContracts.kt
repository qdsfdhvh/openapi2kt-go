@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import org.example.dictionaries.models.BooleanChoicesValue
import org.example.dictionaries.models.Choices
import org.example.dictionaries.models.Closed
import org.example.dictionaries.models.Counts
import org.example.dictionaries.models.Fixed
import org.example.dictionaries.models.Holder
import org.example.dictionaries.models.Labels
import org.example.dictionaries.models.LabelsValue
import org.example.dictionaries.models.Lists
import org.example.dictionaries.models.ListsValueInner
import org.example.dictionaries.models.Nullable
import org.example.dictionaries.models.Objects
import org.example.dictionaries.models.ObjectsValue
import org.example.dictionaries.models.Open
import org.example.dictionaries.models.StringChoicesValue
import org.example.dictionaries.models.Tree
import org.example.dictionaries.models.TreeValue
import org.example.dictionaries_suffix.models.CountsDto
import org.example.dictionaries_suffix.models.ObjectsDto
import org.example.dictionaries_suffix.models.ObjectsDtoValue

fun main() {
    val raw = Json.parseToJsonElement("""{"null":null,"array":[1,false],"object":{"x":"y"}}""").jsonObject
    val wire = """{
      "counts":{"max":9223372036854775807,"min":-9223372036854775808},
      "nullableValues":{"present":7,"missing":null},
      "list":[{"on":true}],"inline":{"entry":{"id":"inline"}},
      "statuses":{"known":"ready","new":"future"},
      "nested":{"outer":{"inner":[1.5,2.0]}},
      "open":{"null":null,"array":[1,false],"object":{"x":"y"}},
      "objects":{"a":{"id":"root"}},"choice":{"s":"text","b":true}
    }"""
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val holder = json.decodeFromString<Holder>(wire)
        check(holder.counts["max"] == Long.MAX_VALUE && holder.counts["min"] == Long.MIN_VALUE)
        check(holder.nullableValues.containsKey("missing") && holder.nullableValues["missing"] == null)
        check(holder.inline.getValue("entry") == Holder.InlineValue("inline"))
        check(holder.statuses["known"] === Holder.StatusesValue.READY)
        check(holder.statuses["new"] == Holder.StatusesValue.Unrecognized("future"))
        check(holder.nested["outer"]?.get("inner") == listOf(1.5, 2.0))
        check(holder.open == raw && holder.objects["a"] == ObjectsValue("root"))
        check(holder.optional == null && holder.nullable == null && holder.wrapped == null && holder.folded == null)
        check(holder.choice["s"] == Holder.StringChoiceValue("text"))
        check(holder.choice["b"] == Holder.BooleanChoiceValue(true))
        val encoded = json.encodeToJsonElement(Holder.serializer(), holder).jsonObject
        check(encoded.getValue("open") == raw)
        check(encoded.getValue("nullableValues").jsonObject.getValue("missing") == JsonNull)
        check(encoded.containsKey("optional") == (defaults && nulls))
        check(json.decodeFromJsonElement(Holder.serializer(), encoded) == holder)
        val empty = holder.copy(optional = emptyMap(), wrapped = mapOf("one" to 1L), folded = mapOf("two" to "value"))
        check(json.decodeFromString<Holder>(json.encodeToString(empty)) == empty)
        check(json.encodeToJsonElement(Holder.serializer(), empty).jsonObject["optional"] == JsonObject(emptyMap()))
        check(json.decodeFromString<Open>(json.encodeToString<Open>(raw)) == raw)
        check(json.decodeFromString<Counts>("""{"x":1}""") == mapOf("x" to 1L))
        check(json.decodeFromString<Nullable>("null") == null)
        check(json.decodeFromString<Nullable>("""{"x":"y"}""") == mapOf("x" to "y"))
        val labels = json.decodeFromString<Labels>("""{"new":"future"}""")
        check(labels["new"] == LabelsValue.Unrecognized("future"))
        check(json.encodeToString<Labels>(labels) == """{"new":"future"}""")
        val choices = json.decodeFromString<Choices>("""{"s":"text","b":false}""")
        check(choices["s"] == StringChoicesValue("text") && choices["b"] == BooleanChoicesValue(false))
        check(json.decodeFromString<Choices>(json.encodeToString<Choices>(choices)) == choices)
        val lists = json.decodeFromString<Lists>("""{"x":[{"id":"nested"}]}""")
        check(lists["x"] == listOf(ListsValueInner("nested")))
        val tree = json.decodeFromString<Tree>("""{"root":{"children":{"leaf":{}}}}""")
        check(tree["root"]?.children?.get("leaf") == TreeValue())
        check(json.decodeFromString<Tree>(json.encodeToString<Tree>(tree)) == tree)
        check(json.decodeFromString<CountsDto>("""{"x":2}""")["x"] == 2L)
        check(json.decodeFromString<ObjectsDto>("""{"x":{"id":"suffix"}}""")["x"] == ObjectsDtoValue("suffix"))
        for (bad in listOf("[]", "true", "null", "\"string\"")) {
            check(runCatching { json.decodeFromString<Open>(bad) }.isFailure)
            check(runCatching { json.decodeFromString<Counts>(bad) }.isFailure)
        }
        check(runCatching { json.decodeFromString<Counts>("""{"x":null}""") }.isFailure)
        check(runCatching { json.decodeFromString<Objects>("""{"x":{}}""") }.isFailure)
        check(runCatching { json.decodeFromJsonElement(Holder.serializer(), JsonObject(encoded - "counts")) }.isFailure)
        // Closed/open extras are schema constraints, not secretly configured JSON behavior.
        check(json.decodeFromString<Closed>("""{"stillPresent":1}""").containsKey("stillPresent"))
        check(runCatching { json.decodeFromString<Fixed>("""{"id":"x","extra":1}""") }.isFailure)
        val tolerant = Json(json) { ignoreUnknownKeys = true }
        val fixed = tolerant.decodeFromString<Fixed>("""{"id":"x","extra":1}""")
        check(tolerant.encodeToString(fixed) == """{"id":"x"}""")
    }
    check(org.example.dictionary_limits.models.Wide.serializer().descriptor.elementsCount == 240)
    println("Generated dictionary alias/value/nullability/unknown-enum/recursion/extras contracts passed")
}

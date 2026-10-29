package org.example.platform

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.platform.models.Item

@OptIn(ExperimentalSerializationApi::class)
fun main() {
    val input = """{"id":42,"tags":["one",null],"details":{"count":2},"score":1.5,"rating":2.0,"enabled":true,"metadata":{"unknown":[1,true,null]}}"""
    for (defaults in listOf(false, true)) {
        for (nulls in listOf(false, true)) {
            val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
            modelIncrementContracts(json)
            roadmapIncrementContracts(json)
            dtoGetterContracts(json)
            enumFieldBoundsContracts(json)
            val value = json.decodeFromString<Item>(input)
            check(value.id == 42L)
            check(value.tags == listOf("one", null))
            check(value.parent == null)
            val encoded = json.encodeToString(value)
            val tree = json.parseToJsonElement(encoded).jsonObject
            check(tree.getValue("id").jsonPrimitive.content == "42")
            check(tree.getValue("metadata") == json.parseToJsonElement(input).jsonObject.getValue("metadata"))
            if (defaults && nulls) check(tree["parent"] == JsonNull)
            check(json.decodeFromString<Item>(encoded).tags == value.tags)
        }
    }
    println("Generated platform contracts passed (four Json configurations; catalog, allOf/byte/self defaults, roadmap tuple/independent factories/structural uniqueness/primitive union aliases, DTO scalar/enum/const getters and lexical enum field bounds)")
}

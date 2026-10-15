package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.root_unions.models.*
import org.example.root_unions_suffix.models.ImplicitDto
import org.example.root_unions_suffix.models.StateDto

fun main() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val warm = State.Warm(kind = "warm", id = "root")
        val wire = json.encodeToJsonElement(State.serializer(), warm).jsonObject
        check(wire.getValue("kind").jsonPrimitive.content == "warm" && "type" !in wire)
        check(json.decodeFromJsonElement(State.serializer(), wire) == warm)
        check(warm.copy(kind = "different").kind == "different")
        for (tag in listOf("cool", "colder")) {
            val cold = json.decodeFromString(State.serializer(), "{\"kind\":\"$tag\"}") as State.Cool
            check(cold.kind == tag && cold.id == null)
            check(json.decodeFromJsonElement(State.serializer(), json.encodeToJsonElement(State.serializer(), cold)) == cold)
        }
        val unknownRaw = json.parseToJsonElement("{\"kind\":\"future\",\"nested\":[{\"x/y~z\":null}],\"count\":1}")
        val unknown = json.decodeFromJsonElement(State.serializer(), unknownRaw) as UnknownState
        check(unknown.kind == "" && unknown.id == null && unknown.rawElement == unknownRaw)
        check(json.encodeToJsonElement(State.serializer(), unknown) == unknownRaw)
        val missingId = json.parseToJsonElement("{\"kind\":\"warm\"}")
        if (nulls) check(runCatching { json.decodeFromJsonElement(State.serializer(), missingId) }.exceptionOrNull() is SerializationException)
        else check((json.decodeFromJsonElement(State.serializer(), missingId) as State.Warm).id == null)
        val plain: Untyped = Untyped.Variant1(shared = "ordinary")
        val plainWire = json.encodeToJsonElement(Untyped.serializer(), plain).jsonObject
        check(plainWire.getValue("type").jsonPrimitive.content == "Variant1")
        check(json.decodeFromJsonElement(Untyped.serializer(), plainWire) == plain)
        for (bad in listOf("{\"type\":\"future\"}", "{}", "[]")) {
            check(runCatching { json.decodeFromString(Untyped.serializer(), bad) }.exceptionOrNull() is SerializationException)
        }
        val key = json.decodeFromString(Keys.serializer(), "{\"a\":9223372036854775807}") as Keys.Variant1
        check(key.a == Long.MAX_VALUE)
        check(json.encodeToJsonElement(Keys.serializer(), key) == buildJsonObject { put("a", Long.MAX_VALUE) })
        check(json.decodeFromString(Keys.serializer(), "{\"b\":false}") is Keys.Variant2)
        check(json.decodeFromString(Keys.serializer(), "{}") is UnknownKeys)
        val implicit = json.decodeFromString(Implicit.serializer(), "{\"type\":\"LocalAB\",\"a\":\"ref\"}") as Implicit.LocalAB
        check(implicit.type == "LocalAB" && implicit.a == "ref")
        check(json.decodeFromJsonElement(Implicit.serializer(), json.encodeToJsonElement(Implicit.serializer(), implicit)) == implicit)
        check(json.decodeFromString(Implicit.serializer(), "{\"type\":\"Other\",\"b\":4}") is Implicit.Other)
        check(json.decodeFromString(Implicit.serializer(), "{\"type\":\"Local/a~b\"}") is UnknownImplicit)
        val suffixed = json.decodeFromString(ImplicitDto.serializer(), "{\"type\":\"LocalABDto\",\"a\":\"suffix\"}") as ImplicitDto.LocalABDto
        check(suffixed.type == "LocalABDto")
        check(json.decodeFromString(ImplicitDto.serializer(), "{\"type\":\"LocalAB\"}") is org.example.root_unions_suffix.models.UnknownImplicitDto)
        check(json.decodeFromJsonElement(StateDto.serializer(), wire) is StateDto.Warm)
        val holder = Holder(state = warm, states = listOf(unknown, State.Cool(kind = "cool")), self = Holder(state = warm, states = emptyList()))
        check(json.decodeFromString(Holder.serializer(), json.encodeToString(Holder.serializer(), holder)) == holder)
        check(json.decodeFromString(Tagged.serializer(), "{\"type\":\"idle\",\"extra\":true}") === Tagged.Idle)
        check(json.encodeToJsonElement(Tagged.serializer(), Tagged.Idle) == buildJsonObject { put("type", "idle") })
        val busy = json.decodeFromString(Tagged.serializer(), "{\"type\":\"busy\",\"values\":[1,2]}") as Tagged.Busy
        check(busy.values == listOf(1, 2))
    }
    println("Standalone root union generated-code contracts passed")
}

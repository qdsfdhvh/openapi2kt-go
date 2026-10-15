package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.*
import org.example.requestkinds.ApiClient
import org.example.requestkinds.api.models.*

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun requestKindContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var calls = 0
        var expectedPath = ""
        var expectedBody: JsonElement = JsonNull
        fun expect(path: String, body: JsonElement) {
            expectedPath = path
            expectedBody = body
        }
        val engine = MockEngine { request ->
            calls++
            check(request.url.encodedPath == expectedPath)
            val body = codec.parseToJsonElement((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString())
            check(body == expectedBody) { "body=$body, expected=$expectedBody" }
            respond("", HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            val empty = JsonObject(emptyMap())
            // These are the root request caller's empty variants, not payload wrappers.
            for (variant in listOf(RawChoiceInputDto.Variant1, RawChoiceInputDto.Variant2,
                RawChoiceInputDto.Variant3, RawChoiceInputDto.Variant4, RawChoiceInputDto.Variant5)) {
                expect("/raw", empty)
                api.rawChoice(variant)
                check(codec.encodeToJsonElement(RawChoiceInputDto.serializer(), variant) == empty)
                check(codec.decodeFromJsonElement(RawChoiceInputDto.serializer(), empty) is UnknownRawChoiceInputDto)
            }
            val payloads = listOf(JsonPrimitive("text"), JsonPrimitive(false), JsonPrimitive(Long.MAX_VALUE),
                JsonPrimitive(1.5), JsonArray(listOf(JsonPrimitive("item"))), JsonNull,
                buildJsonObject { put("future", true) })
            for (payload in payloads) {
                val decoded = codec.decodeFromJsonElement(RawChoiceInputDto.serializer(), payload)
                check(decoded is UnknownRawChoiceInputDto && decoded.rawElement == payload)
                expect("/raw", payload)
                api.rawChoice(decoded)
                check(codec.encodeToJsonElement(RawChoiceInputDto.serializer(), decoded) == payload)
            }
            val tagged = buildJsonObject {
                put("kind", "known"); put("value", 7)
                if (defaults && nulls) put("memo", JsonNull)
            }
            expect("/tagged", tagged)
            api.taggedChoice(TaggedChoiceInputDto.Known(value = 7))
            check(codec.decodeFromJsonElement(TaggedChoiceInputDto.serializer(), tagged) is TaggedChoiceInputDto.Known)
            expect("/tagged", empty)
            api.taggedChoice(TaggedChoiceInputDto.Variant1)
            expect("/tagged", JsonPrimitive("raw"))
            api.taggedChoice(UnknownTaggedChoiceInputDto(JsonPrimitive("raw")))
            val future = buildJsonObject { put("kind", "future"); put("extra", 1) }
            check(codec.decodeFromJsonElement(TaggedChoiceInputDto.serializer(), future) is UnknownTaggedChoiceInputDto)
            expect("/tagged", future)
            api.taggedChoice(UnknownTaggedChoiceInputDto(future))
            for (payload in listOf(JsonPrimitive("raw"), JsonNull, JsonArray(emptyList()))) {
                // Tags/presence retain their existing object-only decoding boundary.
                check(runCatching { codec.decodeFromJsonElement(TaggedChoiceInputDto.serializer(), payload) }.exceptionOrNull() is IllegalArgumentException)
                check(runCatching { codec.decodeFromJsonElement(PresenceChoiceInputDto.serializer(), payload) }.exceptionOrNull() is IllegalArgumentException)
            }
            val label = buildJsonObject { put("label", "value") }
            expect("/presence", label)
            api.presenceChoice(PresenceChoiceInputDto.Variant2(label = "value"))
            check(codec.decodeFromJsonElement(PresenceChoiceInputDto.serializer(), label) is PresenceChoiceInputDto.Variant2)
            expect("/presence", empty)
            api.presenceChoice(PresenceChoiceInputDto.Variant1)
            val array = JsonArray(listOf(JsonPrimitive(true)))
            expect("/presence", array)
            api.presenceChoice(UnknownPresenceChoiceInputDto(array))
            expect("/optional", JsonNull)
            api.optionalRaw()
            expect("/optional", array)
            api.optionalRaw(UnknownOptionalRawInputDto(array))
            check(codec.decodeFromJsonElement(OptionalRawInputDto.serializer(), JsonNull) is UnknownOptionalRawInputDto)
            val code = buildJsonObject { put("code", 8) }
            expect("/shared", code)
            api.sharedChoice(SharedChoiceInputDto.Variant2(code = 8))
            check(codec.decodeFromJsonElement(SharedChoiceInputDto.serializer(), code) is SharedChoiceInputDto.Variant2)
            expect("/pinned", empty)
            api.pinnedChoice(PinnedChoiceInputDto.Variant3)
            val outsidePins = JsonPrimitive("not-a-pin")
            expect("/pinned", outsidePins)
            val pinnedUnknown = codec.decodeFromJsonElement(PinnedChoiceInputDto.serializer(), outsidePins)
            check(pinnedUnknown is UnknownPinnedChoiceInputDto && pinnedUnknown.rawElement == outsidePins)
            api.pinnedChoice(pinnedUnknown)
            check(calls == 24)
            println("Root request-kind contracts passed: defaults=$defaults/nulls=$nulls; empty unreachable variants, Unknown raw kinds, tag/presence object-only boundary, ref/optional; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}

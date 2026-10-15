package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.requestunions.UnionPresenceApiClient
import org.example.requestunions.api.models.*
import org.example.requestunions.models.Patch

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun requestUnionContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // A root union still permits the old no-argument variant constructor.
        val root: ChooseInputDto = ChooseInputDto.Left(child = ChooseInputDto.Left.Child())
        val rootJson = codec.encodeToJsonElement(ChooseInputDto.serializer(), root).jsonObject
        check(("token" in rootJson) == (defaults && nulls))
        check(("note" in rootJson) == (defaults && nulls))
        check(("memo" in rootJson.getValue("child").jsonObject) == (defaults && nulls))
        check(codec.decodeFromString(ChooseInputDto.serializer(), """{"kind":"left"}""") is ChooseInputDto.Left)

        val clear = EnvelopeInputDto(EnvelopeInputDto.Payload.Left(
            token = null, note = null, child = EnvelopeInputDto.Payload.Left.Child(memo = Patch.Null),
            rows = listOf(EnvelopeInputDto.Payload.Left.RowsInner(memo = Patch.Null)),
            attributes = mapOf("k" to EnvelopeInputDto.Payload.Left.AttributesValue(flag = Patch.Null))
        ))
        val clearJson = codec.encodeToJsonElement(EnvelopeInputDto.serializer(), clear).jsonObject
        val payload = clearJson.getValue("payload").jsonObject
        check(("token" in payload) == nulls)
        check(("note" in payload) == (defaults && nulls))
        check(payload.getValue("child").jsonObject["memo"] == JsonNull)
        check(payload.getValue("rows").jsonArray[0].jsonObject["memo"] == JsonNull)
        check(payload.getValue("attributes").jsonObject.getValue("k").jsonObject["flag"] == JsonNull)
        val decoded = codec.decodeFromJsonElement(EnvelopeInputDto.serializer(), clearJson)
        check((decoded.payload as EnvelopeInputDto.Payload.Left).child?.memo === Patch.Null)
        val missing = """{"payload":{"kind":"left"}}"""
        if (nulls) {
            check(runCatching { codec.decodeFromString(EnvelopeInputDto.serializer(), missing) }.exceptionOrNull() is SerializationException)
        } else {
            check((codec.decodeFromString(EnvelopeInputDto.serializer(), missing).payload as EnvelopeInputDto.Payload.Left).token == null)
        }
        for (memo in listOf(Patch.Absent, Patch.Value("v"))) {
            val value = EnvelopeInputDto(EnvelopeInputDto.Payload.Left(token = "present", child = EnvelopeInputDto.Payload.Left.Child(memo)))
            val encoded = codec.encodeToJsonElement(EnvelopeInputDto.serializer(), value)
            val fields = encoded.jsonObject.getValue("payload").jsonObject.getValue("child").jsonObject
            if (memo === Patch.Absent) check("memo" !in fields) else check(fields["memo"] == JsonPrimitive("v"))
            check((codec.decodeFromJsonElement(EnvelopeInputDto.serializer(), encoded).payload as EnvelopeInputDto.Payload.Left).child?.memo == memo)
        }
        val unknownJson = codec.parseToJsonElement("""{"kind":"future","extra":7}""")
        val unknown = codec.decodeFromJsonElement(ChooseInputDto.serializer(), unknownJson)
        check(unknown is UnknownChooseInputDto)
        check(codec.encodeToJsonElement(ChooseInputDto.serializer(), unknown) == unknownJson)
        val envelopeUnknownJson = buildJsonObject { put("payload", unknownJson) }
        val envelopeUnknown = codec.decodeFromJsonElement(EnvelopeInputDto.serializer(), envelopeUnknownJson)
        check(envelopeUnknown.payload is EnvelopeInputDto.UnknownPayload)
        check(codec.encodeToJsonElement(EnvelopeInputDto.serializer(), envelopeUnknown) == envelopeUnknownJson)
        val folded = MaybeInputDto(token = null, memo = Patch.Null)
        val foldedJson = codec.encodeToJsonElement(MaybeInputDto.serializer(), folded)
        check(foldedJson.jsonObject["memo"] == JsonNull)
        check(("token" in foldedJson.jsonObject) == nulls)
        check(codec.decodeFromJsonElement(MaybeInputDto.serializer(), foldedJson).memo === Patch.Null)

        val expected = listOf(
            "/choice" to rootJson, "/choice" to unknownJson,
            "/envelope" to clearJson, "/envelope" to envelopeUnknownJson,
            "/maybe" to foldedJson, "/maybe" to JsonNull,
        )
        var calls = 0
        val engine = MockEngine { request ->
            val (path, body) = expected[calls++]
            check(request.url.encodedPath == path)
            val sent = (request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString()
            check(codec.parseToJsonElement(sent) == body)
            respond("", HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = UnionPresenceApiClient(http).default
            api.choose(root)
            api.choose(unknown)
            api.envelope(clear)
            api.envelope(envelopeUnknown)
            api.maybe(folded)
            api.maybe()
            check(calls == expected.size)
            println("Request union contracts passed: defaults=$defaults/nulls=$nulls, root/variant/nested context, Unknown passthrough and nullable fold; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}

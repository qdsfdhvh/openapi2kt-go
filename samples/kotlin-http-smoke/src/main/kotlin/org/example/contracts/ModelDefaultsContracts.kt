@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.client.engine.mock.respond
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpMethod
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import org.example.defaults.api.DefaultApi
import org.example.defaults.api.models.ApplyDefaultsInputDto
import org.example.defaults.api.models.ReadDefaultsResponseDto
import org.example.defaults.api.models.WatchDefaultsEventDto
import org.example.defaults.models.BoundsDto
import org.example.defaults.models.ChoiceDto
import org.example.defaults.models.HolderDto
import org.example.defaults.models.Patch
import org.example.defaults.models.RecordDto
import org.example.defaults.models.RequiredNullableDto
import org.example.defaults.models.ShadowDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ModelDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun modelDefaultsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> modelDefaultsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic default body")
}

public suspend fun modelDefaultsContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val minimal = "{\"required\":\"given\"}"
        val value: RecordDto = codec.decodeFromString(minimal)
        check(value == RecordDto(required = "given"))
        check(value.label == "ready" && value.enabled == true && value.count == 7)
        check(value.minimum == Long.MIN_VALUE && value.ratio == 1.25f)
        check(value.date.toString() == "2026-10-12" && value.instant.toString() == "2026-10-12T00:00:00Z")
        check(value.nullable == "fallback" && value.nullValue == null)
        check(runCatching { codec.decodeFromString<RecordDto>("{}") }.isFailure)
        val encoded = codec.encodeToJsonElement(value) as JsonObject
        check(encoded["required"] == JsonPrimitive("given"))
        check(("label" in encoded) == defaults)
        check(("nullValue" in encoded) == (defaults && nulls))
        val explicit = codec.decodeFromString<RecordDto>("{\"required\":\"given\",\"label\":null,\"nullable\":null}")
        check(explicit.label == null && explicit.nullable == null)
        check(codec.decodeFromString<RecordDto>("{\"required\":\"given\",\"label\":\"caller\"}").label == "caller")
        val lists: HolderDto = codec.decodeFromString("{\"rows\":[$minimal],\"map\":{\"x\":$minimal}}")
        check(lists.rows!!.single().label == "ready" && lists.map!!["x"]!!.count == 7)
        check(codec.decodeFromJsonElement<ChoiceDto>(JsonObject(mapOf("kind" to JsonPrimitive("a")))).value == "first")
        check(codec.decodeFromJsonElement<ChoiceDto>(JsonObject(mapOf("kind" to JsonPrimitive("b")))).value == "second")
        val shadow = ShadowDto(kotlin = "shadow")
        check(shadow.minimum == Long.MIN_VALUE && shadow.date.toString() == "2026-10-12" && shadow.instant.toString() == "2026-10-12T00:00:00Z")
        check(RequiredNullableDto(null).value == null)
        if (nulls) check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isFailure)
        val input = codec.decodeFromString<ApplyDefaultsInputDto>("{\"required\":\"given\",\"requiredNullable\":null,\"nested\":{}}")
        check(input.patch === Patch.Absent && input.clear === Patch.Absent && input.optional == "fallback")
        check(input.nested!!.value == "nested" && input.nested!!.patch === Patch.Absent)
        val request = codec.encodeToJsonElement(input) as JsonObject
        check("patch" !in request && "clear" !in request)
        val clearing = ApplyDefaultsInputDto("given", null, patch = Patch.Null, clear = Patch.Value("caller"))
        val clearingJson = codec.encodeToJsonElement(clearing) as JsonObject
        check(clearingJson["patch"] == JsonNull && clearingJson["clear"] == JsonPrimitive("caller"))
        check(codec.decodeFromString<ReadDefaultsResponseDto>("{}").value == "response")
        val bounds: BoundsDto = codec.decodeFromString("{}")
        check(bounds.value == 2 && bounds.label == "😀")
        if (checked) {
            check(runCatching { BoundsDto(value = 3) }.isFailure)
            check(runCatching { codec.decodeFromString<BoundsDto>("{\"value\":3}") }.isFailure)
        }
        var requests = 0
        val engine = ModelDefaultsEngine(MockEngineConfig().apply { addHandler { outgoing ->
            requests++
            when (outgoing.url.encodedPath) {
                "/defaults" -> {
                    if (outgoing.method == HttpMethod.Post) {
                        check(codec.parseToJsonElement(modelDefaultsBody(outgoing.body)) == request)
                        respond(minimal, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                    } else respond("{}", headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/watch" -> respond("event: sample\ndata: $minimal\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("unexpected synthetic route")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.applyDefaults(input).label == "ready")
            check(api.readDefaults().value == "response")
            val events = api.watchDefaults().toList()
            check((events.single() as WatchDefaultsEventDto.Sample).data!!.label == "ready")
            check(requests == 3)
        } finally { client.close() }
        println("ModelDefaults Json defaults=$defaults nulls=$nulls literal-required-Patch-source-HTTP-SSE passed")
    }
}

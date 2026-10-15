@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
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
import org.example.enumdefaults.api.DefaultApi
import org.example.enumdefaults.api.models.ApplyEnumsInputDto
import org.example.enumdefaults.api.models.ReadEnumsResponseDto
import org.example.enumdefaults.api.models.WatchEnumsEventDto
import org.example.enumdefaults.models.ChoiceDto
import org.example.enumdefaults.models.HolderDto
import org.example.enumdefaults.models.MergedDto
import org.example.enumdefaults.models.Patch
import org.example.enumdefaults.models.RecordDto
import org.example.enumdefaults.models.ShadowDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class EnumDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun enumDefaultsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> enumDefaultsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic enum default body")
}

public suspend fun enumDefaultsContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val minimal = "{\"required\":\"ready\"}"
        val value: RecordDto = codec.decodeFromString(minimal)
        check(value == RecordDto(required = RecordDto.Required.READY))
        check(value.state === RecordDto.State.READY && value.kind === RecordDto.Kind.BUSY)
        check(value.count == 2 && value.ratio == 1.25 && value.flag == true)
        check(value.nullable === RecordDto.Nullable.BUSY && value.nullValue == null)
        check(runCatching { codec.decodeFromString<RecordDto>("{}") }.isFailure)
        val encoded = codec.encodeToJsonElement(value) as JsonObject
        check(encoded["required"] == JsonPrimitive("ready"))
        check(("state" in encoded) == defaults)
        check(("nullValue" in encoded) == (defaults && nulls))
        check(codec.decodeFromString<RecordDto>("{\"required\":\"ready\",\"state\":null}").state == null)
        val unknown = codec.decodeFromString<RecordDto>("{\"required\":\"future\",\"state\":\"future\"}")
        check(unknown.required.serialName == "future" && unknown.state!!.serialName == "future")
        val lists: HolderDto = codec.decodeFromString("{\"rows\":[$minimal],\"map\":{\"x\":$minimal}}")
        check(lists.rows!!.single().state === RecordDto.State.READY && lists.map!!["x"]!!.kind === RecordDto.Kind.BUSY)
        val a = codec.decodeFromJsonElement<ChoiceDto>(JsonObject(mapOf("kind" to JsonPrimitive("a")))) as ChoiceDto.A
        val b = codec.decodeFromJsonElement<ChoiceDto>(JsonObject(mapOf("kind" to JsonPrimitive("b")))) as ChoiceDto.B
        check(a.state === ChoiceDto.A.State.READY && b.state === ChoiceDto.B.State.BUSY)
        check(ShadowDto(kotlin = "shadow").state === ShadowDto.State.READY)
        check(codec.decodeFromString<MergedDto>("{}").state === MergedDto.State.BUSY)
        val input = codec.decodeFromString<ApplyEnumsInputDto>("{\"required\":\"ready\",\"requiredNullable\":null,\"nested\":{}}")
        check(input.patch === Patch.Absent && input.clear === Patch.Absent && input.optional === ApplyEnumsInputDto.Optional.READY)
        check(input.nested!!.state === ApplyEnumsInputDto.Nested.State.READY && input.nested!!.patch === Patch.Absent)
        val request = codec.encodeToJsonElement(input) as JsonObject
        check("patch" !in request && "clear" !in request)
        val clearing = ApplyEnumsInputDto(ApplyEnumsInputDto.Required.READY, null, patch = Patch.Null, clear = Patch.Value(ApplyEnumsInputDto.Clear.BUSY))
        val clearingJson = codec.encodeToJsonElement(clearing) as JsonObject
        check(clearingJson["patch"] == JsonNull && clearingJson["clear"] == JsonPrimitive("busy"))
        check(codec.decodeFromString<ReadEnumsResponseDto>("{}").state === ReadEnumsResponseDto.State.READY)
        var requests = 0
        val engine = EnumDefaultsEngine(MockEngineConfig().apply { addHandler { outgoing ->
            requests++
            when (outgoing.url.encodedPath) {
                "/defaults" -> {
                    if (outgoing.method == HttpMethod.Post) {
                        check(codec.parseToJsonElement(enumDefaultsBody(outgoing.body)) == request)
                        respond(minimal, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                    } else respond("{}", headers = headersOf(HttpHeaders.ContentType, "application/json"))
                }
                "/watch" -> respond("event: sample\ndata: $minimal\n\n", headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("Unexpected synthetic enum default route")
            }
        } })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.applyEnums(input).state === RecordDto.State.READY)
            check(api.readEnums().state === ReadEnumsResponseDto.State.READY)
            val events = api.watchEnums().toList()
            check((events.single() as WatchEnumsEventDto.Sample).data!!.state === RecordDto.State.READY)
            check(requests == 3)
        } finally { client.close() }
        println("EnumDefaults Json defaults=$defaults nulls=$nulls vocabulary-required-Patch-lexical-source-HTTP-SSE passed checked=$checked")
    }
}

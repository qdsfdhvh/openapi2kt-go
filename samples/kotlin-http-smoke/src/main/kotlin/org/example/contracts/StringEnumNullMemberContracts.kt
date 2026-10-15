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
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.stringenumnullmembers.api.DefaultApi
import org.example.stringenumnullmembers.api.models.ApplyPatchInputDto
import org.example.stringenumnullmembers.api.models.WatchStateEventDto
import org.example.stringenumnullmembers.models.AliasDto
import org.example.stringenumnullmembers.models.BridgeDto
import org.example.stringenumnullmembers.models.ChoiceDto
import org.example.stringenumnullmembers.models.FalseDto
import org.example.stringenumnullmembers.models.NestedDto
import org.example.stringenumnullmembers.models.OwnerDto
import org.example.stringenumnullmembers.models.Patch
import org.example.stringenumnullmembers.models.RecordDto
import org.example.stringenumnullmembers.models.StateDto
import org.example.stringenumnullmembers.models.UnknownChoiceDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StringEnumNullMemberEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun stringEnumNullMemberBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> stringEnumNullMemberBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic String enum body")
}
private suspend fun stringEnumNullMemberReject(block: suspend () -> Unit) {
    val result = runCatching { block() }
    check(result.isFailure && generateSequence(result.exceptionOrNull()) { it.cause }.any { it is SerializationException })
}

suspend fun stringEnumNullMemberContracts() {
    check(StateDto.entries.map { it.serialName } == listOf("ready", "done", "null"))
    check(AliasDto.entries == StateDto.entries && NestedDto.entries == OwnerDto.State.entries)
    check(serializer<AliasDto>().descriptor.isNullable && serializer<BridgeDto>().descriptor.isNullable)
    check(!serializer<FalseDto>().descriptor.isNullable && !StateDto.serializer().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<AliasDto>("null") == null)
        check(codec.decodeFromString<AliasDto>("\"null\"") == StateDto.NULL)
        check(codec.decodeFromString<NestedDto>("\"future-雪\"") == OwnerDto.State.Unrecognized("future-雪"))
        stringEnumNullMemberReject { codec.decodeFromString<FalseDto>("null") }
        stringEnumNullMemberReject { codec.decodeFromString<AliasDto>("123") }
        val value = RecordDto(value = null, narrow = StateDto.READY, rows = listOf(null, StateDto.NULL, StateDto.Unrecognized("future")), map = mapOf("nil" to null, "future" to OwnerDto.State.Unrecognized("future")))
        check(value.defaulted == RecordDto.Defaulted.READY && value.nullDefault == null)
        check(codec.decodeFromString<RecordDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<RecordDto>("{\"narrow\":\"ready\"}").defaulted == RecordDto.Defaulted.READY)
        check(codec.decodeFromString<RecordDto>("{\"narrow\":\"ready\",\"defaulted\":null}").defaulted == null)
        check(codec.decodeFromString<RecordDto>("{\"narrow\":\"ready\",\"defaulted\":\"future\"}").defaulted == RecordDto.Defaulted.Unrecognized("future"))
        check(codec.decodeFromString<ChoiceDto>("{\"kind\":\"a\"}") is ChoiceDto.A)
        val unknown: ChoiceDto = codec.decodeFromString("{\"kind\":null}")
        check(unknown is UnknownChoiceDto && codec.encodeToJsonElement(unknown) == codec.parseToJsonElement("{\"kind\":null}"))
        val absent = ApplyPatchInputDto(requiredNull = null, narrow = StateDto.READY)
        val clear = ApplyPatchInputDto(requiredNull = null, narrow = StateDto.READY, state = Patch.Null)
        val set = ApplyPatchInputDto(requiredNull = null, narrow = StateDto.READY, state = Patch.Value(ApplyPatchInputDto.State.Unrecognized("future")))
        check("state" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["state"] == JsonNull)
        check((codec.encodeToJsonElement(set) as JsonObject)["state"] == JsonPrimitive("future"))
        for (patch in listOf(absent, clear, set)) check(codec.decodeFromString<ApplyPatchInputDto>(codec.encodeToString(patch)) == patch)
        var response = "null"
        var stream = "event: sample\ndata: null\n\nevent: sample\ndata: \"future\"\n\nevent: narrow\ndata: \"ready\"\n\n"
        var expectedPatch = codec.encodeToJsonElement(absent)
        val engine = StringEnumNullMemberEngine(MockEngineConfig().apply { addHandler { request ->
            when (request.url.encodedPath) {
                "/direct" -> respond(stringEnumNullMemberBody(request.body), headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/rows", "/record" -> respond(stringEnumNullMemberBody(request.body), headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/patch" -> { check(codec.parseToJsonElement(stringEnumNullMemberBody(request.body)) == expectedPatch); respond("", headers = headersOf(HttpHeaders.ContentType, "application/json")) }
                "/projection" -> respond(response, headers = headersOf(HttpHeaders.ContentType, "application/json"))
                "/watch" -> respond(stream, headers = headersOf(HttpHeaders.ContentType, "text/event-stream"))
                else -> error("Unexpected synthetic String enum route")
            }
        } })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(http, "https://example.test/", codec)
            for (state in listOf(null, StateDto.READY, StateDto.NULL, StateDto.Unrecognized("future"))) check(api.applyState(state) == state)
            check(api.applyRows(value.rows!!) == value.rows)
            check(api.applyRecord(value) == value)
            for (patch in listOf(absent, clear, set)) { expectedPatch = codec.encodeToJsonElement(patch); api.applyPatch(patch) }
            check(api.projectState() == "null")
            response = "\"future\""; check(api.projectState() == "\"future\"")
            val events = api.watchState().toList()
            check((events[0] as WatchStateEventDto.Sample).data == null)
            check((events[1] as WatchStateEventDto.Sample).data == StateDto.Unrecognized("future"))
            check((events[2] as WatchStateEventDto.Narrow).data == StateDto.READY)
            stream = "event: narrow\ndata: null\n\n"; stringEnumNullMemberReject { api.watchState().toList() }
            stream = "event: sample\ndata: 123\n\n"; stringEnumNullMemberReject { api.watchState().toList() }
            check(api.applyState(StateDto.READY) == StateDto.READY)
        } finally { http.close() }
        println("StringEnumNullMember Json defaults=$defaults nulls=$nulls original-open-vocabulary-owner-defaults-Patch-nullbody-result-SSE passed")
    }
}

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
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.serializer
import org.example.refstringscalar.RefStringScalar
import org.example.refstringscalar.api.models.PatchInputDto
import org.example.refstringscalar.api.models.WatchEventDto
import org.example.refstringscalar.models.AliasDto
import org.example.refstringscalar.models.HolderDto
import org.example.refstringscalar.models.ModeDto
import org.example.refstringscalar.models.NullableStatusDto
import org.example.refstringscalar.models.Patch
import org.example.refstringscalar.models.SourceDto
import org.example.refstringscalar.models.StatusDto
import org.example.refstringscalar.models.WordsDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StringScalarDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun stringScalarDefaultReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected invalid String enum wire to fail" }
}

private fun stringScalarDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> stringScalarDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedStringEnumScalarDefaultsContracts(checked: Boolean) {
    val escaped = "line\n\$\"\\"
    val value = HolderDto(required = StatusDto.ON, requiredNullable = null, kotlin = "shadow", kotlinx = "shadow", serializer = "shadow")
    check(value.status === StatusDto.ON && value.mode === ModeDto.RAW)
    val borrowed: AliasDto? = value.borrowed
    check(borrowed === SourceDto.Code.RAW && value.narrowed === StatusDto.OFF)
    check(value.words == WordsDto.fromSerialName(escaped) && value.words !is WordsDto.Unrecognized)
    check(value.nil == null && value.nested == null && HolderDto.Nested().status === StatusDto.OFF)
    check(HolderDto(required = StatusDto.OFF, requiredNullable = null).status === StatusDto.ON)
    check(serializer<AliasDto>().descriptor.serialName == SourceDto.Code.serializer().descriptor.serialName)
    val descriptor = HolderDto.serializer().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("required")))
    check(!descriptor.isElementOptional(descriptor.getElementIndex("requiredNullable")))
    check(descriptor.isElementOptional(descriptor.getElementIndex("status")))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val decoded = codec.decodeFromString<HolderDto>("""{"required":"on","requiredNullable":null}""")
        check(decoded.status === StatusDto.ON && decoded.mode === ModeDto.RAW && decoded.borrowed === SourceDto.Code.RAW)
        check(decoded.narrowed === StatusDto.OFF && decoded.words?.serialName == escaped && decoded.nil == null)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), decoded) as JsonObject
        check(encoded.containsKey("status") == defaults && encoded.containsKey("mode") == defaults && encoded.containsKey("borrowed") == defaults && encoded.containsKey("words") == defaults)
        check(encoded.containsKey("requiredNullable") == nulls)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == decoded)
        val supplied = codec.decodeFromString<HolderDto>("""{"required":"off","requiredNullable":"on","status":"off","borrowed":"wire","nil":"off","words":"😀"}""")
        check(supplied.status === StatusDto.OFF && supplied.borrowed === SourceDto.Code.WIRE && supplied.nil === NullableStatusDto.OFF && supplied.words?.serialName == "😀")
        check(codec.decodeFromString<HolderDto>("""{"required":"on","requiredNullable":null,"status":null,"borrowed":null}""").let { it.status == null && it.borrowed == null })
        val future = codec.decodeFromString<HolderDto>("""{"required":"future","requiredNullable":null,"status":"unknown","mode":"new","borrowed":"opaque"}""")
        check(future.required == StatusDto.Unrecognized("future") && future.status == StatusDto.Unrecognized("unknown"))
        check(future.mode == ModeDto.Unrecognized("new") && future.borrowed == SourceDto.Code.Unrecognized("opaque"))
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), future)) == future)
        check(codec.decodeFromString<AliasDto>("\"future\"") == SourceDto.Code.Unrecognized("future"))
        stringScalarDefaultReject { codec.decodeFromString<HolderDto>("{}") }
        check(runCatching { codec.decodeFromString<HolderDto>("""{"required":"on"}""") }.isSuccess == !nulls)
        for (bad in listOf("true", "1", "[]", "{}")) stringScalarDefaultReject {
            codec.decodeFromString<HolderDto>("""{"required":"on","requiredNullable":null,"status":$bad}""")
        }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.value == Patch.Absent)
        check(!(codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).containsKey("value"))
        val clear = codec.decodeFromString<PatchInputDto>("""{"required":null,"value":null}""")
        check(clear.value == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), clear) as JsonObject)["value"] === JsonNull)
        val update = PatchInputDto(required = StatusDto.OFF, value = Patch.Value(StatusDto.Unrecognized("future")))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        stringScalarDefaultWire(codec, decoded)
        println("Referenced String scalar defaults defaults=$defaults nulls=$nulls checked=$checked passed")
    }
}

private suspend fun stringScalarDefaultWire(codec: Json, value: HolderDto) {
    var payload = """{"required":"on","requiredNullable":null}"""
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = StringScalarDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(stringScalarDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(stringScalarDefaultBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> respond("event: holder\r\ndata: $payload\r\n\r\n", HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            else -> error("Unexpected String scalar request")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RefStringScalar(http, "http://example.test/", codec).defaults
        var callbacks = 0
        check(api.send(value, requestBuilder = { callbacks++ }).status === StatusDto.ON)
        check(callbacks == 1 && calls == 1)
        for (patch in listOf(PatchInputDto(required = null), PatchInputDto(required = StatusDto.ON, value = Patch.Null), PatchInputDto(required = StatusDto.OFF, value = Patch.Value(StatusDto.Unrecognized("future"))))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val events = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { events.toList() }.single() as WatchEventDto.Holder).data.borrowed === SourceDto.Code.RAW) }
        payload = """{"required":"on","requiredNullable":null,"status":"future"}"""
        check(api.send(value).status == StatusDto.Unrecognized("future"))
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.status == StatusDto.Unrecognized("future"))
        payload = """{"required":"on","requiredNullable":null,"status":true}"""
        stringScalarDefaultReject { api.send(value) }
        stringScalarDefaultReject { withTimeout(10000) { api.watch().toList() } }
        payload = """{"required":"on","requiredNullable":null}"""
        check(api.send(value).mode === ModeDto.RAW)
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.status === StatusDto.ON)
    } finally { http.close() }
}

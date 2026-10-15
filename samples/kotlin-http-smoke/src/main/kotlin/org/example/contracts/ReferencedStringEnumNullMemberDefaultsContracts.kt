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
import org.example.refnullstrings.RefNullStrings
import org.example.refnullstrings.api.models.PatchInputDto
import org.example.refnullstrings.api.models.WatchEventDto
import org.example.refnullstrings.models.AliasDto
import org.example.refnullstrings.models.HolderDto
import org.example.refnullstrings.models.ModeDto
import org.example.refnullstrings.models.Patch
import org.example.refnullstrings.models.SourceDto
import org.example.refnullstrings.models.StatusDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullStringDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nullStringDefaultReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected invalid null-member enum wire to fail" }
}

private fun nullStringDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullStringDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedStringEnumNullMemberDefaultsContracts(checked: Boolean) {
    val nullString = StatusDto.fromSerialName("null")
    val value = HolderDto(required = StatusDto.ON, kotlin = "shadow", kotlinx = "shadow", serializer = "shadow")
    check(value.scalar === StatusDto.ON && value.textNull == nullString && value.textNull !is StatusDto.Unrecognized)
    val borrowed: AliasDto = value.borrowed
    check(borrowed === SourceDto.Code.RAW && value.nil == null)
    check(value.rows == listOf(null, StatusDto.ON, nullString))
    check(value.maps == mapOf("a/~" to null, "\$ref" to nullString))
    check(value.nested == listOf(listOf(null, StatusDto.OFF), emptyList<StatusDto?>()))
    check(value.narrow == listOf(StatusDto.ON) && HolderDto.Child().scalar === StatusDto.OFF)
    check(StatusDto.entries.size == 3 && StatusDto.entries.all { it.serialName != "" })
    check(!StatusDto.serializer().descriptor.isNullable && serializer<AliasDto>().descriptor.isNullable)
    check(!HolderDto.serializer().descriptor.isElementOptional(HolderDto.serializer().descriptor.getElementIndex("required")))
    val second = HolderDto(required = null)
    check(value.rows !== second.rows && value.maps !== second.maps && value.nested !== second.nested)
    check(value.nested?.first() !== second.nested?.first())
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val decoded = codec.decodeFromString<HolderDto>("""{"required":null}""")
        check(decoded.scalar === StatusDto.ON && decoded.textNull == nullString && decoded.nil == null)
        check(decoded.borrowed === SourceDto.Code.RAW && decoded.rows == value.rows && decoded.maps == value.maps && decoded.nested == value.nested)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), decoded) as JsonObject
        check(encoded.containsKey("scalar") == defaults && encoded.containsKey("rows") == defaults && encoded.containsKey("maps") == defaults)
        check(encoded.containsKey("required") == nulls)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == decoded)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isSuccess == !nulls)
        val supplied = codec.decodeFromString<HolderDto>("""{"required":"off","scalar":null,"textNull":null,"rows":["future",null],"maps":{"x":"null"},"borrowed":"future"}""")
        check(supplied.scalar == null && supplied.textNull == null && supplied.required === StatusDto.OFF)
        check(supplied.rows == listOf(StatusDto.Unrecognized("future"), null) && supplied.maps == mapOf("x" to nullString))
        check(supplied.borrowed == SourceDto.Code.Unrecognized("future"))
        val suppliedRoundTrip = codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), supplied))
        check(suppliedRoundTrip.rows == supplied.rows && suppliedRoundTrip.maps == supplied.maps && suppliedRoundTrip.borrowed == supplied.borrowed)
        if (nulls) check(suppliedRoundTrip == supplied)
        else check(suppliedRoundTrip.scalar === StatusDto.ON && suppliedRoundTrip.textNull == nullString) // Omitted explicit nulls reapply non-null defaults.
        check(codec.decodeFromString<HolderDto>("""{"required":null,"rows":null,"maps":null,"nested":null}""").let { it.rows == null && it.maps == null && it.nested == null })
        check(codec.decodeFromString<AliasDto>("null") == null)
        check(codec.decodeFromString<ModeDto?>("\"future\"") == ModeDto.Unrecognized("future"))
        for (bad in listOf("true", "1", "[]", "{}")) nullStringDefaultReject {
            codec.decodeFromString<HolderDto>("""{"required":null,"scalar":$bad}""")
        }
        nullStringDefaultReject { codec.decodeFromString<HolderDto>("""{"required":null,"rows":[true]}""") }
        nullStringDefaultReject { codec.decodeFromString<HolderDto>("""{"required":null,"narrow":[null]}""") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.value == Patch.Absent && absent.rows == Patch.Absent)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "value" || it == "rows" })
        val clear = codec.decodeFromString<PatchInputDto>("""{"required":null,"value":null,"rows":null}""")
        check(clear.value == Patch.Null && clear.rows == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), clear) as JsonObject)["value"] === JsonNull)
        val update = PatchInputDto(required = StatusDto.ON, value = Patch.Value(nullString), rows = Patch.Value(listOf(null, StatusDto.Unrecognized("future"))))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        nullStringDefaultWire(codec, decoded)
        println("Referenced null-member String enum defaults defaults=$defaults nulls=$nulls checked=$checked passed")
    }
}

private suspend fun nullStringDefaultWire(codec: Json, value: HolderDto) {
    var payload = """{"required":null}"""
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NullStringDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nullStringDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nullStringDefaultBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> respond("event: holder\r\ndata: $payload\r\n\r\n", HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            else -> error("Unexpected null String request")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RefNullStrings(http, "http://example.test/", codec).defaults
        var callbacks = 0
        check(api.send(value, requestBuilder = { callbacks++ }).rows == value.rows)
        check(callbacks == 1 && calls == 1)
        for (patch in listOf(PatchInputDto(required = null), PatchInputDto(required = StatusDto.ON, value = Patch.Null, rows = Patch.Null), PatchInputDto(required = StatusDto.OFF, value = Patch.Value(StatusDto.Unrecognized("future")), rows = Patch.Value(listOf(null))))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val events = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { events.toList() }.single() as WatchEventDto.Holder).data.rows == value.rows) }
        payload = """{"required":null,"scalar":"future","rows":[null,"future"]}"""
        check(api.send(value).scalar == StatusDto.Unrecognized("future"))
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.rows == listOf(null, StatusDto.Unrecognized("future")))
        for (bad in listOf("""{"required":null,"scalar":true}""", """{"required":null,"narrow":[null]}""")) {
            payload = bad
            nullStringDefaultReject { api.send(value) }
            nullStringDefaultReject { withTimeout(10000) { api.watch().toList() } }
        }
        payload = """{"required":null}"""
        check(api.send(value).textNull?.serialName == "null")
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.borrowed === SourceDto.Code.RAW)
    } finally { http.close() }
}

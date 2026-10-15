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
import org.example.refnativescalar.RefNativeScalar
import org.example.refnativescalar.api.models.PatchInputDto
import org.example.refnativescalar.api.models.WatchEventDto
import org.example.refnativescalar.models.CodeDto
import org.example.refnativescalar.models.HolderDto
import org.example.refnativescalar.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NativeScalarDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nativeScalarDefaultReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected scalar default consumer failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nativeScalarDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nativeScalarDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedNativeEnumScalarDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = 1, requiredNullable = null, kotlin = "shadow", companion = "shadow", serializer = "shadow")
    check(value.i == 2 && value.l == 2L && value.f == 1.5f && value.d == 1.25 && value.b == true)
    check(value.nil == null && value.minInt == Int.MIN_VALUE && value.minLong == Long.MIN_VALUE)
    check(value.nested == null && HolderDto.Nested().value == 1.5f)
    check(serializer<CodeDto>().descriptor.serialName == "kotlin.Int")
    val descriptor = HolderDto.serializer().descriptor
    check(!descriptor.isElementOptional(descriptor.getElementIndex("required")))
    check(!descriptor.isElementOptional(descriptor.getElementIndex("requiredNullable")))
    check(descriptor.isElementOptional(descriptor.getElementIndex("i")))
    if (checked) nativeScalarDefaultReject("enum") { HolderDto(required = 9, requiredNullable = null) }
    else check(HolderDto(required = 9, requiredNullable = null, b = false).b == false)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val sparse = """{"required":1,"requiredNullable":null}"""
        val decoded = codec.decodeFromString<HolderDto>(sparse)
        check(decoded.i == 2 && decoded.l == 2L && decoded.f == 1.5f && decoded.d == 1.25 && decoded.b == true)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), decoded) as JsonObject
        check(encoded.containsKey("i") == defaults && encoded.containsKey("f") == defaults && encoded.containsKey("minInt") == defaults)
        check(encoded.containsKey("requiredNullable") == nulls)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == decoded)
        val supplied = codec.decodeFromString<HolderDto>("""{"required":2,"requiredNullable":1,"i":1,"f":1.0,"d":2.5,"nil":2}""")
        check(supplied.required == 2 && supplied.requiredNullable == 1 && supplied.i == 1 && supplied.f == 1.0f && supplied.d == 2.5 && supplied.nil == 2)
        check(codec.decodeFromString<HolderDto>("""{"required":1,"requiredNullable":null,"i":null,"f":null}""").let { it.i == null && it.f == null })
        check(codec.decodeFromString<CodeDto>("9") == 9) // Independent scalar aliases do not enforce the DTO checks.
        nativeScalarDefaultReject { codec.decodeFromString<HolderDto>("{}") }
        check(runCatching { codec.decodeFromString<HolderDto>("""{"required":1}""") }.isSuccess == !nulls)
        nativeScalarDefaultReject { codec.decodeFromString<HolderDto>("""{"required":1,"requiredNullable":null,"i":true}""") }
        if (checked) {
            nativeScalarDefaultReject("enum") { codec.decodeFromString<HolderDto>("""{"required":1,"requiredNullable":null,"i":9}""") }
            nativeScalarDefaultReject("enum") { codec.decodeFromString<HolderDto>("""{"required":1,"requiredNullable":null,"b":false}""") }
        } else check(codec.decodeFromString<HolderDto>("""{"required":1,"requiredNullable":null,"i":9,"b":false}""").let { it.i == 9 && it.b == false })
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.value == Patch.Absent)
        check(!(codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).containsKey("value"))
        val clear = codec.decodeFromString<PatchInputDto>("""{"required":null,"value":null}""")
        check(clear.value == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), clear) as JsonObject)["value"] === JsonNull)
        val update = PatchInputDto(required = 1, value = Patch.Value(2))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        if (checked) nativeScalarDefaultReject("enum") { PatchInputDto(required = null, value = Patch.Value(9)) }
        nativeScalarDefaultWire(codec, decoded, checked)
        println("Referenced native scalar defaults defaults=$defaults nulls=$nulls checked=$checked passed")
    }
}

private suspend fun nativeScalarDefaultWire(codec: Json, value: HolderDto, checked: Boolean) {
    var payload = """{"required":1,"requiredNullable":null}"""
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NativeScalarDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nativeScalarDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nativeScalarDefaultBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> respond("event: holder\r\ndata: $payload\r\n\r\n", HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            else -> error("Unexpected native scalar request")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RefNativeScalar(http, "http://example.test/", codec).defaults
        var callbacks = 0
        check(api.send(value, requestBuilder = { callbacks++ }).i == 2)
        check(callbacks == 1 && calls == 1)
        for (patch in listOf(PatchInputDto(required = null), PatchInputDto(required = 1, value = Patch.Null), PatchInputDto(required = 2, value = Patch.Value(1)))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val events = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { events.toList() }.single() as WatchEventDto.Holder).data.f == 1.5f) }
        payload = """{"required":1,"requiredNullable":null,"i":true}"""
        nativeScalarDefaultReject { api.send(value) }
        nativeScalarDefaultReject { withTimeout(10000) { api.watch().toList() } }
        if (checked) {
            payload = """{"required":1,"requiredNullable":null,"i":9}"""
            nativeScalarDefaultReject("enum") { api.send(value) }
            nativeScalarDefaultReject("enum") { withTimeout(10000) { api.watch().toList() } }
        }
        payload = """{"required":1,"requiredNullable":null}"""
        check(api.send(value).minLong == Long.MIN_VALUE)
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.i == 2)
    } finally { http.close() }
}

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
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.stringenumcollections.StringEnumCollections
import org.example.stringenumcollections.api.models.PatchInputDto
import org.example.stringenumcollections.api.models.WatchEventDto
import org.example.stringenumcollections.models.HolderDto
import org.example.stringenumcollections.models.Patch
import org.example.stringenumcollections.models.RequiredMapsDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StringEnumCollectionEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun stringEnumCollectionReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun stringEnumCollectionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> stringEnumCollectionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun stringEnumCollectionDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(HolderDto.RequiredInner.TWO), kotlin = "shadow", kotlinx = "shadow", labelsInner = "shadow")
    check(value.codes == listOf(HolderDto.CodesInner.ONE, null, HolderDto.CodesInner.TWO))
    check(value.labels == mapOf("a/~" to HolderDto.LabelsValue.YES, "\$ref" to HolderDto.LabelsValue.NO))
    check(value.special?.single()?.serialName == "quote\"\$😀")
    check(value.empty == emptyList<HolderDto.EmptyInner>() && value.emptyMap == emptyMap<String, HolderDto.EmptyMapValue>() && value.nil == null)
    check(HolderDto.Nested().labels == mapOf("one" to HolderDto.Nested.LabelsValue.NESTED, "nil" to null))
    check(HolderDto.CodesInner.fromSerialName("future") == HolderDto.CodesInner.Unrecognized("future"))
    check(HolderDto.CodesInner.entries == listOf(HolderDto.CodesInner.ONE, HolderDto.CodesInner.TWO))
    val second = HolderDto(required = value.required)
    check(value.codes !== second.codes && value.labels !== second.labels && value.empty !== second.empty && value.emptyMap !== second.emptyMap)
    val future = HolderDto(required = listOf(HolderDto.RequiredInner.Unrecognized("future")), codes = listOf(HolderDto.CodesInner.Unrecognized("future")), labels = mapOf("future" to HolderDto.LabelsValue.Unrecognized("future")))
    if (checked) {
        stringEnumCollectionReject("minItems") { HolderDto(required = emptyList()) }
        stringEnumCollectionReject("maxItems") { HolderDto(required = value.required, codes = List(4) { HolderDto.CodesInner.ONE }) }
        stringEnumCollectionReject("maxProperties") { HolderDto(required = value.required, labels = mapOf("a" to HolderDto.LabelsValue.YES, "b" to HolderDto.LabelsValue.YES, "c" to HolderDto.LabelsValue.YES)) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val requiredMaps = RequiredMapsDto(labels = mapOf("a" to RequiredMapsDto.LabelsValue.YES), nullableLabels = null)
        val mapWire = codec.encodeToJsonElement(RequiredMapsDto.serializer(), requiredMaps) as JsonObject
        check(mapWire.containsKey("nullableLabels") == nulls)
        check(codec.decodeFromJsonElement(RequiredMapsDto.serializer(), mapWire) == requiredMaps)
        stringEnumCollectionReject { codec.decodeFromString<RequiredMapsDto>("{}") }
        stringEnumCollectionReject { codec.decodeFromString<RequiredMapsDto>("""{"labels":{"a":null},"nullableLabels":null}""") }
        val nested = HolderDto.Nested()
        check(codec.decodeFromJsonElement(HolderDto.Nested.serializer(), codec.encodeToJsonElement(HolderDto.Nested.serializer(), nested)) == nested)
        val missing = codec.decodeFromString<HolderDto>("""{"required":["two"]}""")
        check(missing.codes == value.codes && missing.labels == value.labels)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"codes":null,"labels":null}""").let { it.codes == null && it.labels == null })
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"codes":["two"],"labels":{"new":"yes"}}""").let { it.codes == listOf(HolderDto.CodesInner.TWO) && it.labels == mapOf("new" to HolderDto.LabelsValue.YES) })
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), future)) == future)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("codes") == defaults && encoded.containsKey("labels") == defaults)
        if (defaults) {
            check(encoded["codes"] == codec.parseToJsonElement("""["one",null,"two"]"""))
            check(encoded["labels"] == codec.parseToJsonElement("""{"a/~":"yes","${'$'}ref":"no"}"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.codes == Patch.Absent && absent.labels == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "codes" || it == "labels" })
        val update = PatchInputDto(required = listOf(PatchInputDto.RequiredInner.TWO), codes = Patch.Value(listOf(PatchInputDto.CodesInner.Unrecognized("future"))), labels = Patch.Value(mapOf("yes" to PatchInputDto.LabelsValue.YES)))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val explicitNull = codec.decodeFromString<PatchInputDto>("""{"required":["two"],"codes":null,"labels":null}""")
        check(explicitNull.codes == Patch.Null && explicitNull.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), explicitNull) as JsonObject)["codes"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        stringEnumCollectionReject { codec.decodeFromString<HolderDto>("""{"required":[null]}""") }
        stringEnumCollectionReject { codec.decodeFromString<HolderDto>("""{"required":["two"],"labels":{"bad":1}}""") }
        if (checked) {
            stringEnumCollectionReject("minItems") { codec.decodeFromString<HolderDto>("""{"required":[]}""") }
            stringEnumCollectionReject("minItems") { PatchInputDto(required = null, codes = Patch.Value(emptyList())) }
        }
        stringEnumCollectionWire(codec, missing, checked)
        println("String enum collection defaults Json defaults=$defaults nulls=$nulls checked=$checked lexical-open-enum-constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun stringEnumCollectionWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":["two"]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = StringEnumCollectionEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(stringEnumCollectionBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(stringEnumCollectionBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected String enum collection path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StringEnumCollections(http, "http://example.test/", codec).defaults
        check(api.send(value).codes == value.codes)
        for (patch in listOf(
            PatchInputDto(required = listOf(PatchInputDto.RequiredInner.TWO)),
            PatchInputDto(required = null, codes = Patch.Null, labels = Patch.Null),
            PatchInputDto(required = listOf(PatchInputDto.RequiredInner.TWO), codes = Patch.Value(listOf(PatchInputDto.CodesInner.TWO)), labels = Patch.Value(mapOf("yes" to PatchInputDto.LabelsValue.Unrecognized("future"))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.labels == value.labels) }
        payload = """{"required":["future"],"codes":["future"],"labels":{"new":"future"}}"""
        check(api.send(value).required.single() == HolderDto.RequiredInner.Unrecognized("future"))
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.labels?.get("new") == HolderDto.LabelsValue.Unrecognized("future"))
        val invalid = mutableListOf<Pair<String, String?>>("""{"required":[null]}""" to null, """{"required":["two"],"labels":{"bad":1}}""" to null)
        if (checked) invalid += listOf("""{"required":[]}""" to "minItems", """{"required":["two"],"codes":["one","one","one","one"]}""" to "maxItems")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            stringEnumCollectionReject(rule) { api.send(value) }
            stringEnumCollectionReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":["two"]}"""
            failing = false
            check(api.send(value).codes == value.codes)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.codes == value.codes)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

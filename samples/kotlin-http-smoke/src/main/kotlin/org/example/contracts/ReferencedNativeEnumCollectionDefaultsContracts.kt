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
import kotlinx.serialization.serializer
import org.example.refnativedefaults.models.NilDto
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
import org.example.refnativedefaults.RefNativeDefaults
import org.example.refnativedefaults.api.models.PatchInputDto
import org.example.refnativedefaults.api.models.WatchEventDto
import org.example.refnativedefaults.models.HolderDto
import org.example.refnativedefaults.models.Patch
import org.example.refnativedefaults.models.RequiredNullableDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ReferencedNativeEnumCollectionDefaultEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun referencedNativeEnumCollectionDefaultReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun referencedNativeEnumCollectionDefaultBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> referencedNativeEnumCollectionDefaultBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun referencedNativeEnumCollectionFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(listOf(1)), requiredInts = listOf(3), kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow")
    check(value.ints == listOf(1, 3, 2))
    check(value.longs == listOf(2L, 1L))
    check(value.floats == listOf(1.5f, 0.5f))
    check(value.doubleMaps == mapOf("z" to -0.0, "a/~" to 1.25, "\$ref" to 1.25))
    check(value.bools == listOf(true, false, true))
    check(value.intMaps == mapOf("\$ref" to 1, "a/~" to 3))
    check(value.borrowed == listOf(2))
    check(value.empty == emptyList<Long>() && value.nil == null)
    check(HolderDto.Nested().value == listOf(listOf(1), listOf(3, 2)))
    check(serializer<org.example.refnativedefaults.models.IntsDto>().descriptor.serialName == "kotlin.collections.ArrayList")
    check(serializer<org.example.refnativedefaults.models.DoubleMapsDto>().descriptor.serialName == "kotlin.collections.LinkedHashMap")
    check(serializer<org.example.refnativedefaults.models.LongsDto>().descriptor.getElementDescriptor(0).serialName == "kotlin.Long")
    val descriptor = HolderDto.serializer().descriptor
    val requiredIndex = descriptor.getElementIndex("required")
    check(!descriptor.isElementOptional(requiredIndex))
    check(descriptor.getElementDescriptor(requiredIndex).getElementDescriptor(0).getElementDescriptor(0).serialName == "kotlin.Int")
    // Fresh evaluation identity: equal trees are never shared between instances.
    val second = HolderDto(required = listOf(listOf(1)), requiredInts = listOf(3))
    check(value.ints !== second.ints && value.doubleMaps !== second.doubleMaps)
    check(value.empty !== second.empty && value.borrowed !== second.borrowed)
    check(value.ints == second.ints)
    // Supplied values replace defaults; scalar wire keeps its native type.
    check(RequiredNullableDto(value = null).value == null)
    if (checked) {
        referencedNativeEnumCollectionDefaultReject("maxItems") { HolderDto(required = value.required, requiredInts = value.requiredInts, ints = List(4) { 1 }) }
        referencedNativeEnumCollectionDefaultReject("enum") { HolderDto(required = value.required, requiredInts = value.requiredInts, ints = listOf(9)) }
        referencedNativeEnumCollectionDefaultReject("minProperties") { RequiredNullableDto(value = emptyMap()) }
    }
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[3]}""").ints == value.ints)
        check(codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[3],"ints":null,"nil":null}""").let { it.ints == null && it.nil == null })
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire) == requiredNull)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val missing = codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[3]}""")
        check(missing.longs == value.longs && missing.floats == value.floats && missing.doubleMaps == value.doubleMaps)
        referencedNativeEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[3],"bools":null,"intMaps":null}""").let { it.bools == null && it.intMaps == null })
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("ints") == defaults && encoded.containsKey("longs") == defaults && encoded.containsKey("floats") == defaults)
        check(encoded.containsKey("doubleMaps") == defaults && encoded.containsKey("bools") == defaults && encoded.containsKey("intMaps") == defaults)
        check(encoded.containsKey("empty") == defaults && encoded.containsKey("borrowed") == defaults)
        if (defaults) {
            check(encoded["ints"] == codec.parseToJsonElement("""[1,3,2]"""))
            check(encoded["doubleMaps"] == codec.parseToJsonElement("""{"${'$'}ref":1.25,"a/~":1.25,"z":-0.0}"""))
            check(encoded["floats"] == codec.parseToJsonElement("""[1.5,0.5]"""))
            check(encoded["bools"] == codec.parseToJsonElement("""[true,false,true]"""))
            check(encoded["borrowed"] == codec.parseToJsonElement("""[2]"""))
        }
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        referencedNativeEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[null],"requiredInts":[3]}""") }
        referencedNativeEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[null]}""") }
        referencedNativeEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[["x"]],"requiredInts":[3]}""") }
        referencedNativeEnumCollectionDefaultReject { codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[3],"intMaps":{"bad":false}}""") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.labels == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "rows" || it == "labels" })
        val update = PatchInputDto(required = listOf(listOf(2)), rows = Patch.Value(listOf(listOf(3))), labels = Patch.Value(mapOf("a" to listOf(false))))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"labels":null}""")
        check(cleared.rows == Patch.Null && cleared.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["rows"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        if (checked) {
            referencedNativeEnumCollectionDefaultReject("enum") { codec.decodeFromString<HolderDto>("""{"required":[[9]],"requiredInts":[3]}""") }
            referencedNativeEnumCollectionDefaultReject("enum") { codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[9]}""") }
            referencedNativeEnumCollectionDefaultReject("enum") { codec.decodeFromString<HolderDto>("""{"required":[[1]],"requiredInts":[3],"intMaps":{"bad":9}}""") }
            referencedNativeEnumCollectionDefaultReject("minItems") { PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList()))) }
        }
        referencedNativeEnumCollectionDefaultWire(codec, missing, checked)
        println("Referenced native enum collection defaults Json defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private suspend fun referencedNativeEnumCollectionDefaultWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":[[1]],"requiredInts":[3]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = ReferencedNativeEnumCollectionDefaultEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(referencedNativeEnumCollectionDefaultBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(referencedNativeEnumCollectionDefaultBody(request.body)) == expectedPatch)
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
            else -> error("Unexpected nested default path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = RefNativeDefaults(http, "http://example.test/", codec).defaults
        var callbacks = 0
        val beforeCall = calls
        check(api.send(value, requestBuilder = { callbacks++ }).ints == value.ints)
        check(callbacks == 1 && calls == beforeCall + 1)
        for (patch in listOf(
            PatchInputDto(required = listOf(listOf(1))),
            PatchInputDto(required = null, rows = Patch.Null, labels = Patch.Null),
            PatchInputDto(required = listOf(listOf(2)), rows = Patch.Value(listOf(listOf(3))), labels = Patch.Value(mapOf("a" to listOf(false))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.ints == value.ints) }
        val invalid = mutableListOf<Pair<String, String?>>(
            """{"required":[null],"requiredInts":[3]}""" to null,
            """{"required":[["x"]],"requiredInts":[3]}""" to null,
            """{"required":[[1]],"requiredInts":[3],"bools":[null]}""" to null,
            "{" to null)
        invalid += listOf(
            """{"required":[[1]],"requiredInts":[3],"intMaps":{"bad":false}}""" to null,
            """{"required":[[1]],"requiredInts":[3],"empty":["x"]}""" to null)
        if (checked) invalid += listOf(
            """{"required":[[]],"requiredInts":[3]}""" to "minItems",
            """{"required":[[1]],"requiredInts":[]}""" to "minItems",
            """{"required":[[1]],"requiredInts":[3],"ints":[9]}""" to "enum")
        for ((bad, rule) in invalid) {
            payload = bad
            failing = true
            val beforeFailure = calls
            val beforeCallbacks = callbacks
            referencedNativeEnumCollectionDefaultReject(rule) { api.send(value, requestBuilder = { callbacks++ }) }
            check(calls == beforeFailure + 1 && callbacks == beforeCallbacks + 1)
            referencedNativeEnumCollectionDefaultReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":[[1]],"requiredInts":[3]}"""
            failing = false
            check(api.send(value).doubleMaps == value.doubleMaps)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.doubleMaps == value.doubleMaps)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

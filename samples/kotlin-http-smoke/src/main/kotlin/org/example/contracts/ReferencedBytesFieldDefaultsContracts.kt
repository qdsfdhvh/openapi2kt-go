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
import kotlinx.serialization.serializer
import org.example.refbytes.ReferencedBytes
import org.example.refbytes.api.models.PatchInputDto
import org.example.refbytes.api.models.WatchEventDto
import org.example.refbytes.models.AliasDto
import org.example.refbytes.models.BridgeDto
import org.example.refbytes.models.BytesDto
import org.example.refbytes.models.HolderDto
import org.example.refbytes.models.Patch
import org.example.refbytes.models.RequiredNullableDto
import org.example.refbytes.models.RowsAliasDto

private suspend fun refBytesReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected bytes decode failure" }
}

suspend fun referencedBytesFieldDefaultsContracts(checked: Boolean) {
    val value = HolderDto(requiredBlob = byteArrayOf(9), kotlin = "shadow", kotlinx = "shadow")
    val fresh = HolderDto(requiredBlob = byteArrayOf(9))
    val alias: AliasDto = value.blob
    val bytes: BytesDto = alias
    val builtin: ByteArray? = bytes
    val bridge: BridgeDto = checkNotNull(value.bridge)
    val rows: RowsAliasDto = value.rows
    check(builtin.contentEquals(byteArrayOf(-128, 0, 127)) && bridge.contentEquals(byteArrayOf(3)))
    check(value.internal.contentEquals(byteArrayOf(4)) && value.binary.contentEquals(byteArrayOf(1)))
    check(rows!![0].contentEquals(byteArrayOf(-128, 127)) && rows[1] == null && rows[2]!!.isEmpty())
    check(value.internalRows!![0].contentEquals(byteArrayOf(4)))
    check(value.bags!!["\$ref"].contentEquals(byteArrayOf(2)) && value.bags!!["a/~"] == null)
    check(value.nested!![0]["bytes"].contentEquals(byteArrayOf(3)))
    check(HolderDto.Child().blob.contentEquals(byteArrayOf(5)))
    check(value.blob !== fresh.blob && value.blob !== value.repeated && value.empty !== fresh.empty)
    check(value.rows!![0] !== fresh.rows!![0] && value.bags!!["z"] !== fresh.bags!!["z"])
    fresh.blob!![0] = 7
    fresh.rows!![0]!![0] = 7
    check(value.blob!![0] == (-128).toByte() && value.rows!![0]!![0] == (-128).toByte())
    check(serializer<BytesDto>().descriptor.isNullable && serializer<AliasDto>().descriptor.isNullable)
    check(!serializer<ByteArray>().descriptor.isNullable && !serializer<BridgeDto>().descriptor.isNullable)
    check(HolderDto::class.java.getDeclaredMethod("getBlob").returnType == ByteArray::class.java)
    val descriptor = HolderDto.serializer().descriptor
    check(descriptor.serialName == "org.example.refbytes.models.HolderDto")
    check(!descriptor.isElementOptional(descriptor.getElementIndex("requiredBlob")))
    check(descriptor.isElementOptional(descriptor.getElementIndex("blob")))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    if (checked) check(runCatching { HolderDto(requiredBlob = byteArrayOf(9), capped = emptyList()) }.exceptionOrNull()?.message?.contains("minItems") == true)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"requiredBlob":[9]}""")
        check(missing.blob.contentEquals(byteArrayOf(-128, 0, 127)) && missing.bridge.contentEquals(byteArrayOf(3)))
        check(missing.empty!!.isEmpty() && missing.nil == null && missing.emptyRows!!.isEmpty() && missing.nilRows == null)
        refBytesReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"requiredBlob":[9],"blob":null,"rows":null}""").let { it.blob == null && it.rows == null })
        check(codec.decodeFromString<HolderDto>("""{"requiredBlob":[9],"blob":[7],"rows":[[8]]}""").let { it.blob.contentEquals(byteArrayOf(7)) && it.rows!![0].contentEquals(byteArrayOf(8)) })
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        // Fresh ByteArray defaults retain reference equality, so even an empty
        // byte array is emitted; an empty List default uses structural equality.
        check(encoded["blob"] == codec.parseToJsonElement("[-128,0,127]"))
        check(encoded["empty"] == codec.parseToJsonElement("[]"))
        check(encoded["rows"] == codec.parseToJsonElement("[[-128,127],null,[]]"))
        check(encoded.containsKey("emptyRows") == defaults)
        check(encoded.containsKey("nil") == (defaults && nulls))
        val roundTrip = codec.decodeFromJsonElement(HolderDto.serializer(), encoded)
        check(codec.encodeToJsonElement(HolderDto.serializer(), roundTrip) == encoded)
        val requiredNull = codec.encodeToJsonElement(RequiredNullableDto.serializer(), RequiredNullableDto(blob = null)) as JsonObject
        check(requiredNull.containsKey("blob") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), requiredNull).blob == null)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val absent = codec.decodeFromString<PatchInputDto>("""{"requiredBlob":null}""")
        check(absent.blob == Patch.Absent && absent.rows == Patch.Absent)
        val absentWire = codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject
        check(!absentWire.containsKey("blob") && !absentWire.containsKey("rows"))
        val cleared = codec.decodeFromString<PatchInputDto>("""{"requiredBlob":null,"blob":null,"rows":null}""")
        check(cleared.blob == Patch.Null && cleared.rows == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject)["blob"] === JsonNull)
        val update = PatchInputDto(requiredBlob = byteArrayOf(2), blob = Patch.Value(byteArrayOf(1)), rows = Patch.Value(listOf(byteArrayOf(3))))
        val updated = codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update))
        check((updated.blob as Patch.Value).value.contentEquals(byteArrayOf(1)))
        check((updated.rows as Patch.Value).value[0].contentEquals(byteArrayOf(3)))
        for (bad in listOf("{}", """{"requiredBlob":null}""", """{"requiredBlob":[9],"blob":"AQI="}""", """{"requiredBlob":[9],"rows":[[null]]}""")) {
            refBytesReject { codec.decodeFromString<HolderDto>(bad) }
        }
        refBytesWire(codec, missing)
        println("Referenced bytes defaults defaults=$defaults nulls=$nulls checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RefBytesEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun refBytesBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> refBytesBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

private suspend fun refBytesWire(codec: Json, value: HolderDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"requiredBlob":[9]}"""
    var failing = false
    var calls = 0
    var expectedPatch = JsonObject(emptyMap())
    var rawBody = ""
    val engine = RefBytesEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/records" -> {
                check(codec.parseToJsonElement(refBytesBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(request.method.value == "PATCH")
                check(codec.parseToJsonElement(refBytesBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/raw" -> {
                check(refBytesBody(request.body) == rawBody)
                respond("null", HttpStatusCode.OK, headersOf("Content-Type", "application/octet-stream"))
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: record\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected referenced bytes path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ReferencedBytes(http, "http://example.test/", codec).default
        check(api.save(value).blob.contentEquals(byteArrayOf(-128, 0, 127)))
        for (patch in listOf(PatchInputDto(requiredBlob = null), PatchInputDto(requiredBlob = null, blob = Patch.Null), PatchInputDto(requiredBlob = byteArrayOf(2), blob = Patch.Value(byteArrayOf(1))))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        rawBody = "null"
        check(api.raw(null).contentEquals("null".encodeToByteArray()))
        rawBody = "raw-bytes"
        check(api.raw(rawBody.encodeToByteArray()).contentEquals("null".encodeToByteArray()))
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Record).data.blob.contentEquals(byteArrayOf(-128, 0, 127))) }
        for (bad in listOf("{}", """{"requiredBlob":null}""", """{"requiredBlob":[9],"blob":"AQI="}""", """{"requiredBlob":[9],"rows":[[null]]}""", "{")) {
            payload = bad
            failing = true
            refBytesReject { api.save(value) }
            refBytesReject { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"requiredBlob":[9]}"""
            failing = false
            check(api.save(value).blob.contentEquals(byteArrayOf(-128, 0, 127)))
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Record).data.bridge.contentEquals(byteArrayOf(3)))
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

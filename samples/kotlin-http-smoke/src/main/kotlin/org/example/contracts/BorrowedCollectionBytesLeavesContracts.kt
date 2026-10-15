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
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.borrowedbytesleaves.BorrowedCollectionBytesLeavesApiClient
import org.example.borrowedbytesleaves.api.models.SubmitInputDto
import org.example.borrowedbytesleaves.api.models.WatchEventDto
import org.example.borrowedbytesleaves.models.CollectionChoiceDto
import org.example.borrowedbytesleaves.models.HolderDto
import org.example.borrowedbytesleaves.models.RowsAliasDto
import org.example.borrowedbytesleaves.models.RowsBridgeDto
import org.example.borrowedbytesleaves.models.MapAliasDto
import org.example.borrowedbytesleaves.models.MapBridgeDto
import org.example.borrowedbytesleaves.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BorrowedBytesLeafStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun borrowedBytesLeafRawBody(body: OutgoingContent): ByteArray = when (body) {
    is OutgoingContent.ContentWrapper -> borrowedBytesLeafRawBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes()
    else -> error("Unexpected synthetic borrowed collection body ${body::class}")
}
private fun borrowedBytesLeafBody(body: OutgoingContent): String = borrowedBytesLeafRawBody(body).decodeToString()
private suspend fun borrowedBytesLeafReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing borrowed collection rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong borrowed collection failure: $cause" }
    }
}

suspend fun borrowedCollectionBytesLeavesContracts(checked: Boolean = false) = withTimeout(30000) {
    check(serializer<RowsAliasDto>().descriptor.isNullable)
    check(!serializer<RowsBridgeDto>().descriptor.isNullable)
    check(serializer<MapAliasDto>().descriptor.isNullable && !serializer<MapBridgeDto>().descriptor.isNullable)
    val bytes=byteArrayOf(-128,-1,0,127)
    val values: List<ByteArray?> = listOf(bytes, byteArrayOf(1,2), null)
    val mapping: MapBridgeDto = mapOf("key" to bytes, "nil" to null)
    fun holder(rows: List<ByteArray?> = values, map: MapBridgeDto = mapping) = HolderDto(rows = rows, mapping = map, direct = values, flags=listOf(bytes),counts=listOf(bytes),ratios=listOf(bytes),measures=listOf(bytes),narrow=listOf(bytes))
    val model = holder()
    if (checked) {
        borrowedBytesLeafReject("minItems") { holder(emptyList()) }
        borrowedBytesLeafReject("maxItems") { holder(List(4) { bytes }) }
        borrowedBytesLeafReject("minProperties") { holder(map = emptyMap()) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<RowsAliasDto>("[null,[-128,-1,0,127]]")?.let { it[0]==null && it[1]?.contentEquals(bytes)==true }==true)
        check(codec.decodeFromString<MapBridgeDto>("{\"key\":[-128,-1,0,127],\"nil\":null}").let { it["key"]?.contentEquals(bytes)==true && it["nil"]==null })
        val alias: RowsAliasDto = values
        check(codec.encodeToJsonElement(codec.decodeFromString<RowsAliasDto>(codec.encodeToString(alias))) == codec.encodeToJsonElement<RowsAliasDto>(values))
        check(codec.decodeFromString<RowsAliasDto>("null") == null)
        check(codec.encodeToJsonElement(codec.decodeFromString<MapBridgeDto>(codec.encodeToString(mapping))) == codec.encodeToJsonElement(mapping))
        check(codec.decodeFromString<MapAliasDto>("null") == null)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        borrowedBytesLeafReject("null") { codec.decodeFromString<RowsBridgeDto>("null") }
        borrowedBytesLeafReject("JSON") { codec.decodeFromString<RowsAliasDto>("{}") }
        val unsafe: RowsBridgeDto = codec.decodeFromString(codec.encodeToString(values))
        check(unsafe.first()?.contentEquals(bytes)==true && unsafe.last()==null)
        check(codec.encodeToString(values)=="[[-128,-1,0,127],[1,2],null]")
        borrowedBytesLeafReject("128") { codec.decodeFromString<RowsBridgeDto>("[[128]]") }
        val listChoice: CollectionChoiceDto = CollectionChoiceDto.Rows(values)
        val mapChoice: CollectionChoiceDto = CollectionChoiceDto.Map(mapping)
        for (choice in listOf(listChoice, mapChoice)) check(codec.encodeToJsonElement(codec.decodeFromString<CollectionChoiceDto>(codec.encodeToString(choice))) == codec.encodeToJsonElement(choice))
        if (checked) borrowedBytesLeafReject("minItems") { CollectionChoiceDto.Rows(emptyList()) }
        val absent = SubmitInputDto(requiredNull = null, required = values)
        val clear = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Null, map = Patch.Null)
        val assigned = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(values), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        check("optional" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["optional"] == JsonNull)
        if (nulls) borrowedBytesLeafReject("requiredNull") { codec.decodeFromJsonElement(SubmitInputDto.serializer(), JsonObject((codec.encodeToJsonElement(absent) as JsonObject).filterKeys { it != "requiredNull" })) }
        if (checked) {
            borrowedBytesLeafReject("minItems") { SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(emptyList())) }
            borrowedBytesLeafReject("minProperties") { SubmitInputDto(requiredNull = null, required = values, map = Patch.Value(emptyMap())) }
        }
        var expectedRaw = "null".encodeToByteArray()
        var expected = "null"
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BorrowedBytesLeafStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val bad = mode; val keepOpen = endless
                    producers.launch { try {
                        val frame = when (bad) {
                            1 -> "event: map\r\ndata: null\r\n\r\n"
                            2 -> "event: rows\r\ndata: []\r\n\r\n"
                            3 -> "event: rows\r\ndata: [[128]]\r\n\r\n"
                            4 -> "event: rows\r\ndata: {\r\n\r\n"
                            5 -> "event: map\r\ndata: {\"key\":{}}\r\n\r\n"
                            else -> "event: rows\r\ndata: ${codec.encodeToString(values)}\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(mapping)}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (bad != 0 || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/raw" -> { check(borrowedBytesLeafRawBody(request.body).contentEquals(expectedRaw)); respond(expectedRaw, HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                "/patch" -> { check(codec.parseToJsonElement(borrowedBytesLeafBody(request.body)) == codec.parseToJsonElement(expected)); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(borrowedBytesLeafBody(request.body)) == codec.parseToJsonElement(expected))
                    val response = when (mode) { 1 -> "null"; 2 -> if (request.url.encodedPath == "/map") "{}" else "[]"; 3 -> "[[128]]"; else -> expected }
                    respond(if (mode == 4) "{\"key\":{}}" else response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = BorrowedCollectionBytesLeavesApiClient(http, "https://example.test/", codec).default
            if (checked) {
                val before = calls; var callback = false
                borrowedBytesLeafReject("minItems") { api.echo(emptyList()) { callback = true } }
                borrowedBytesLeafReject("minProperties") { api.echoMap(emptyMap()) { callback = true } }
                check(calls == before && !callback)
            }
            expected = codec.encodeToString(values); check(codec.encodeToJsonElement(api.echo(values)) == codec.encodeToJsonElement(values))
            if (checked) { mode = 2; borrowedBytesLeafReject("minItems") { api.echo(values) } }; mode = 3; borrowedBytesLeafReject("128") { api.echo(values) }
            mode = 0; expected = codec.encodeToString(mapping); check(codec.encodeToJsonElement(api.echoMap(mapping)) == codec.encodeToJsonElement(mapping))
            mode = 1; borrowedBytesLeafReject("null") { api.echoMap(mapping) }
            if (checked) { mode = 2; borrowedBytesLeafReject("minProperties") { api.echoMap(mapping) } }
            mode = 4; borrowedBytesLeafReject("JSON") { api.echoMap(mapping) }
            mode = 0
            check(api.echoRaw(null)?.contentEquals(expectedRaw)==true)
            expectedRaw = bytes; check(api.echoRaw(bytes)?.contentEquals(bytes)==true)
            for (input in listOf(absent, clear, assigned)) { expected = codec.encodeToString(input); api.submit(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check(codec.encodeToJsonElement((events[0] as WatchEventDto.Rows).data) == codec.encodeToJsonElement(values) && codec.encodeToJsonElement((events[1] as WatchEventDto.Map).data) == codec.encodeToJsonElement(mapping))
            }
            for ((bad, message) in listOf(1 to "null", 3 to "128", 4 to "JSON", 5 to "JSON") + if (checked) listOf(2 to "minItems") else emptyList()) {
                mode = bad; withTimeout(3000) { borrowedBytesLeafReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true; withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { cold.toList() }.size == 2)
            expected = codec.encodeToString(mapping); check(codec.encodeToJsonElement(api.echoMap(mapping)) == codec.encodeToJsonElement(mapping))
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("BorrowedCollectionBytesLeaves Json defaults=$defaults nulls=$nulls actual-union-dispatch-carriers-source-nullable-Patch-union-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}

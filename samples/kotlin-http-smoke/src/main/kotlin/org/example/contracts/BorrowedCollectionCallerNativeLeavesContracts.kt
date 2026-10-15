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
import kotlinx.serialization.json.JsonPrimitive
import org.example.scalarvalues.Money
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.borrowedcallernativeleaves.BorrowedCollectionCallerNativeLeavesApiClient
import org.example.borrowedcallernativeleaves.api.models.SubmitInputDto
import org.example.borrowedcallernativeleaves.api.models.WatchEventDto
import org.example.borrowedcallernativeleaves.models.CollectionChoiceDto
import org.example.borrowedcallernativeleaves.models.HolderDto
import org.example.borrowedcallernativeleaves.models.RowsAliasDto
import org.example.borrowedcallernativeleaves.models.RowsBridgeDto
import org.example.borrowedcallernativeleaves.models.MapAliasDto
import org.example.borrowedcallernativeleaves.models.MapBridgeDto
import org.example.borrowedcallernativeleaves.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BorrowedCallerNativeLeafStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun borrowedCallerNativeLeafBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> borrowedCallerNativeLeafBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic borrowed collection body ${body::class}")
}
private suspend fun borrowedCallerNativeLeafReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing borrowed collection rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong borrowed collection failure: $cause" }
    }
}

suspend fun borrowedCollectionCallerNativeLeavesContracts(checked: Boolean = false) = withTimeout(30000) {
    check(serializer<RowsAliasDto>().descriptor.isNullable)
    check(!serializer<RowsBridgeDto>().descriptor.isNullable)
    check(serializer<MapAliasDto>().descriptor.isNullable && !serializer<MapBridgeDto>().descriptor.isNullable)
    val instant=kotlin.time.Instant.parse("2026-01-02T03:04:05Z")
    val ready=Money("12.50")
    val future=Money("caller-owned-future")
    val values: List<Money?> = listOf(ready, future, null)
    val mapping: MapBridgeDto = mapOf("key" to ready, "nil" to null)
    fun holder(rows: List<Money?> = values, map: MapBridgeDto = mapping) = HolderDto(rows = rows, mapping = map, direct = values, flags=listOf(Long.MAX_VALUE),counts=listOf(byteArrayOf(-128,-1,0,127)),ratios=listOf(JsonNull,JsonPrimitive("caller-json")),measures=listOf(instant),narrow=listOf(ready),nativeLong=Long.MAX_VALUE,objects=listOf(JsonObject(mapOf("nil" to JsonNull))),texts=listOf("text"))
    val model = holder()
    if (checked) {
        borrowedCallerNativeLeafReject("minItems") { holder(emptyList()) }
        borrowedCallerNativeLeafReject("maxItems") { holder(List(4) { ready }) }
        borrowedCallerNativeLeafReject("minProperties") { holder(map = emptyMap()) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<RowsAliasDto>("[null,\"12.50\"]") == listOf(null,ready))
        check(codec.decodeFromString<MapBridgeDto>("{\"key\":\"12.50\",\"nil\":null}") == mapping)
        val alias: RowsAliasDto = values
        check(codec.decodeFromString<RowsAliasDto>(codec.encodeToString(alias)) == values)
        check(codec.decodeFromString<RowsAliasDto>("null") == null)
        check(codec.decodeFromString<MapBridgeDto>(codec.encodeToString(mapping)) == mapping)
        check(codec.decodeFromString<MapAliasDto>("null") == null)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        borrowedCallerNativeLeafReject("null") { codec.decodeFromString<RowsBridgeDto>("null") }
        borrowedCallerNativeLeafReject("JSON") { codec.decodeFromString<RowsAliasDto>("{}") }
        val unsafe: RowsBridgeDto = codec.decodeFromString(codec.encodeToString(values))
        check(unsafe.first()==ready && unsafe.last()==null)
        borrowedCallerNativeLeafReject("JSON") { codec.decodeFromString<RowsBridgeDto>("[{}]") }
        val tree=codec.encodeToJsonElement(model) as JsonObject
        check(tree["nativeLong"]==JsonPrimitive(Long.MAX_VALUE))
        check(tree["flags"]==codec.parseToJsonElement("[9223372036854775807]"))
        check(tree["counts"]==codec.parseToJsonElement("[[-128,-1,0,127]]"))
        check(tree["ratios"]==codec.parseToJsonElement("[null,\"caller-json\"]"))
        check(tree["measures"]==codec.parseToJsonElement("[\"2026-01-02T03:04:05Z\"]"))
        check(tree["objects"]==codec.parseToJsonElement("[{\"nil\":null}]"))
        check(tree["texts"]==codec.parseToJsonElement("[\"text\"]"))
        check(codec.encodeToString(values)=="[\"12.50\",\"caller-owned-future\",null]")
        val listChoice: CollectionChoiceDto = CollectionChoiceDto.Rows(values)
        val mapChoice: CollectionChoiceDto = CollectionChoiceDto.Map(mapping)
        for (choice in listOf(listChoice, mapChoice)) check(codec.encodeToJsonElement(codec.decodeFromString<CollectionChoiceDto>(codec.encodeToString(choice))) == codec.encodeToJsonElement(choice))
        if (checked) borrowedCallerNativeLeafReject("minItems") { CollectionChoiceDto.Rows(emptyList()) }
        val absent = SubmitInputDto(requiredNull = null, required = values)
        val clear = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Null, map = Patch.Null)
        val assigned = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(values), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        check("optional" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["optional"] == JsonNull)
        if (nulls) borrowedCallerNativeLeafReject("requiredNull") { codec.decodeFromJsonElement(SubmitInputDto.serializer(), JsonObject((codec.encodeToJsonElement(absent) as JsonObject).filterKeys { it != "requiredNull" })) }
        if (checked) {
            borrowedCallerNativeLeafReject("minItems") { SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(emptyList())) }
            borrowedCallerNativeLeafReject("minProperties") { SubmitInputDto(requiredNull = null, required = values, map = Patch.Value(emptyMap())) }
        }
        var expected = "null"
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BorrowedCallerNativeLeafStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val bad = mode; val keepOpen = endless
                    producers.launch { try {
                        val frame = when (bad) {
                            1 -> "event: map\r\ndata: null\r\n\r\n"
                            2 -> "event: rows\r\ndata: []\r\n\r\n"
                            3 -> "event: rows\r\ndata: [{}]\r\n\r\n"
                            4 -> "event: rows\r\ndata: {\r\n\r\n"
                            5 -> "event: map\r\ndata: {\"key\":[]}\r\n\r\n"
                            else -> "event: rows\r\ndata: ${codec.encodeToString(values)}\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(mapping)}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (bad != 0 || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/patch" -> { check(codec.parseToJsonElement(borrowedCallerNativeLeafBody(request.body)) == codec.parseToJsonElement(expected)); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(borrowedCallerNativeLeafBody(request.body)) == codec.parseToJsonElement(expected))
                    val response = when (mode) { 1 -> "null"; 2 -> if (request.url.encodedPath == "/map") "{}" else "[]"; 3 -> "[{}]"; else -> expected }
                    respond(if (mode == 4) "{\"key\":[]}" else response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = BorrowedCollectionCallerNativeLeavesApiClient(http, "https://example.test/", codec).default
            if (checked) {
                val before = calls; var callback = false
                borrowedCallerNativeLeafReject("minItems") { api.echo(emptyList()) { callback = true } }
                borrowedCallerNativeLeafReject("minProperties") { api.echoMap(emptyMap()) { callback = true } }
                check(calls == before && !callback)
            }
            expected = codec.encodeToString(values); check(api.echo(values) == values)
            if (checked) { mode = 2; borrowedCallerNativeLeafReject("minItems") { api.echo(values) } }; mode = 3; borrowedCallerNativeLeafReject("JSON") { api.echo(values) }
            mode = 0; expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
            mode = 1; borrowedCallerNativeLeafReject("null") { api.echoMap(mapping) }
            if (checked) { mode = 2; borrowedCallerNativeLeafReject("minProperties") { api.echoMap(mapping) } }
            mode = 4; borrowedCallerNativeLeafReject("JSON") { api.echoMap(mapping) }
            mode = 0
            for (input in listOf(absent, clear, assigned)) { expected = codec.encodeToString(input); api.submit(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check((events[0] as WatchEventDto.Rows).data == values && (events[1] as WatchEventDto.Map).data == mapping)
            }
            for ((bad, message) in listOf(1 to "null", 4 to "JSON", 5 to "JSON") + if (checked) listOf(2 to "minItems") else emptyList()) {
                mode = bad; withTimeout(3000) { borrowedCallerNativeLeafReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 3
            withTimeout(3000) { borrowedCallerNativeLeafReject("JSON") { cold.toList() } }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            mode = 0; endless = true; withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { cold.toList() }.size == 2)
            expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("BorrowedCollectionCallerNativeLeaves Json defaults=$defaults nulls=$nulls actual-union-dispatch-carriers-source-nullable-Patch-union-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}

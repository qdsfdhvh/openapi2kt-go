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
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.borrowedrecursivecoverleaves.BorrowedCollectionRecursiveCoverLeavesApiClient
import org.example.borrowedrecursivecoverleaves.api.models.SubmitInputDto
import org.example.borrowedrecursivecoverleaves.api.models.WatchEventDto
import org.example.borrowedrecursivecoverleaves.models.CollectionChoiceDto
import org.example.borrowedrecursivecoverleaves.models.HolderDto
import org.example.borrowedrecursivecoverleaves.models.RowsAliasDto
import org.example.borrowedrecursivecoverleaves.models.RowsBridgeDto
import org.example.borrowedrecursivecoverleaves.models.MapAliasDto
import org.example.borrowedrecursivecoverleaves.models.MapBridgeDto
import org.example.borrowedrecursivecoverleaves.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BorrowedRecursiveCoverLeafStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun borrowedRecursiveCoverLeafBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> borrowedRecursiveCoverLeafBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic borrowed collection body ${body::class}")
}
private suspend fun borrowedRecursiveCoverLeafReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing borrowed collection rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong borrowed collection failure: $cause" }
    }
}

suspend fun borrowedCollectionRecursiveCoverLeavesContracts(checked: Boolean = false) = withTimeout(30000) {
    check(serializer<RowsAliasDto>().descriptor.isNullable)
    check(!serializer<RowsBridgeDto>().descriptor.isNullable)
    check(serializer<MapAliasDto>().descriptor.isNullable && !serializer<MapBridgeDto>().descriptor.isNullable)
    val values: List<JsonElement?> = listOf(JsonPrimitive("raw"), JsonArray(listOf(JsonPrimitive(1),JsonNull)), null)
    val mapping: MapBridgeDto = JsonObject(mapOf("key" to JsonPrimitive(1), "nil" to JsonNull))
    fun holder(rows: List<JsonElement?> = values, map: MapBridgeDto = mapping) = HolderDto(rows = rows, mapping = map, direct = values, flags=listOf(JsonNull,JsonPrimitive(true)),counts=listOf(JsonPrimitive(1)),ratios=listOf(JsonPrimitive(1.5)),measures=listOf(JsonNull),narrow=listOf(JsonNull))
    val model = holder()
    if (checked) {
        borrowedRecursiveCoverLeafReject("minItems") { holder(emptyList()) }
        borrowedRecursiveCoverLeafReject("maxItems") { holder(List(4) { JsonPrimitive("ok") }) }
        borrowedRecursiveCoverLeafReject("minProperties") { holder(map = JsonObject(emptyMap())) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<RowsAliasDto>("[null,\"ok\"]") == listOf(null,JsonPrimitive("ok")))
        check(codec.decodeFromString<MapBridgeDto>("{\"max\":9223372036854775807,\"min\":-9223372036854775808,\"nil\":null}") == JsonObject(mapOf("max" to JsonPrimitive(Long.MAX_VALUE),"min" to JsonPrimitive(Long.MIN_VALUE),"nil" to JsonNull)))
        val alias: RowsAliasDto = values
        check(codec.decodeFromString<RowsAliasDto>(codec.encodeToString(alias)) == values)
        check(codec.decodeFromString<RowsAliasDto>("null") == null)
        check(codec.decodeFromString<MapBridgeDto>(codec.encodeToString(mapping)) == mapping)
        check(codec.decodeFromString<MapAliasDto>("null") == null)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        borrowedRecursiveCoverLeafReject("null") { codec.decodeFromString<RowsBridgeDto>("null") }
        borrowedRecursiveCoverLeafReject("JSON") { codec.decodeFromString<RowsAliasDto>("{}") }
        val unsafe: RowsBridgeDto = codec.decodeFromString(codec.encodeToString(values))
        check(unsafe == values)
        check(codec.decodeFromString<RowsBridgeDto>("[null,true,1,\"s\",[],{}]")==listOf(null,JsonPrimitive(true),JsonPrimitive(1),JsonPrimitive("s"),JsonArray(emptyList()),JsonObject(emptyMap())))
        check(model.flags?.first()==JsonNull && model.narrow?.first()==JsonNull)
        borrowedRecursiveCoverLeafReject("JSON") { codec.decodeFromString<RowsBridgeDto>("{") }
        val listChoice: CollectionChoiceDto = CollectionChoiceDto.Rows(values)
        val mapChoice: CollectionChoiceDto = CollectionChoiceDto.Map(mapping)
        for (choice in listOf(listChoice, mapChoice)) check(codec.encodeToJsonElement(codec.decodeFromString<CollectionChoiceDto>(codec.encodeToString(choice))) == codec.encodeToJsonElement(choice))
        if (checked) borrowedRecursiveCoverLeafReject("minItems") { CollectionChoiceDto.Rows(emptyList()) }
        val absent = SubmitInputDto(requiredNull = null, required = values)
        val clear = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Null, map = Patch.Null)
        val assigned = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(values), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        check("optional" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["optional"] == JsonNull)
        if (nulls) borrowedRecursiveCoverLeafReject("requiredNull") { codec.decodeFromJsonElement(SubmitInputDto.serializer(), JsonObject((codec.encodeToJsonElement(absent) as JsonObject).filterKeys { it != "requiredNull" })) }
        if (checked) {
            borrowedRecursiveCoverLeafReject("minItems") { SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(emptyList())) }
            borrowedRecursiveCoverLeafReject("minProperties") { SubmitInputDto(requiredNull = null, required = values, map = Patch.Value(JsonObject(emptyMap()))) }
        }
        var expected = "null"
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BorrowedRecursiveCoverLeafStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val bad = mode; val keepOpen = endless
                    producers.launch { try {
                        val frame = when (bad) {
                            1 -> "event: map\r\ndata: null\r\n\r\n"
                            2 -> "event: rows\r\ndata: []\r\n\r\n"
                            3 -> "event: rows\r\ndata: {}\r\n\r\n"
                            4 -> "event: rows\r\ndata: {\r\n\r\n"
                            5 -> "event: map\r\ndata: []\r\n\r\n"
                            else -> "event: rows\r\ndata: ${codec.encodeToString(values)}\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(mapping)}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (bad != 0 || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/patch" -> { check(codec.parseToJsonElement(borrowedRecursiveCoverLeafBody(request.body)) == codec.parseToJsonElement(expected)); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(borrowedRecursiveCoverLeafBody(request.body)) == codec.parseToJsonElement(expected))
                    val response = when (mode) { 1 -> "null"; 2 -> if (request.url.encodedPath == "/map") "{}" else "[]"; 3 -> "{}"; else -> expected }
                    respond(if (mode == 4) "[]" else response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = BorrowedCollectionRecursiveCoverLeavesApiClient(http, "https://example.test/", codec).default
            if (checked) {
                val before = calls; var callback = false
                borrowedRecursiveCoverLeafReject("minItems") { api.echo(emptyList()) { callback = true } }
                borrowedRecursiveCoverLeafReject("minProperties") { api.echoMap(JsonObject(emptyMap())) { callback = true } }
                check(calls == before && !callback)
            }
            expected = codec.encodeToString(values); check(api.echo(values) == values)
            if (checked) { mode = 2; borrowedRecursiveCoverLeafReject("minItems") { api.echo(values) } }; mode = 3; borrowedRecursiveCoverLeafReject("JSON") { api.echo(values) }
            mode = 0; expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
            mode = 1; borrowedRecursiveCoverLeafReject("null") { api.echoMap(mapping) }
            if (checked) { mode = 2; borrowedRecursiveCoverLeafReject("minProperties") { api.echoMap(mapping) } }
            mode = 4; borrowedRecursiveCoverLeafReject("JSON") { api.echoMap(mapping) }
            mode = 0
            for (input in listOf(absent, clear, assigned)) { expected = codec.encodeToString(input); api.submit(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check((events[0] as WatchEventDto.Rows).data == values && (events[1] as WatchEventDto.Map).data == mapping)
            }
            for ((bad, message) in listOf(1 to "null", 3 to "JSON", 4 to "JSON", 5 to "JSON") + if (checked) listOf(2 to "minItems") else emptyList()) {
                mode = bad; withTimeout(3000) { borrowedRecursiveCoverLeafReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true; withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { cold.toList() }.size == 2)
            expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("BorrowedCollectionRecursiveCoverLeaves Json defaults=$defaults nulls=$nulls actual-union-dispatch-carriers-source-nullable-Patch-union-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}

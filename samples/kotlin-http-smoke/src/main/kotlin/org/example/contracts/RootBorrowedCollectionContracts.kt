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
import org.example.borrowedcollectionroots.RootBorrowedCollectionApiClient
import org.example.borrowedcollectionroots.api.models.SubmitInputDto
import org.example.borrowedcollectionroots.api.models.WatchEventDto
import org.example.borrowedcollectionroots.models.AgainBridgeDto
import org.example.borrowedcollectionroots.models.CollectionChoiceDto
import org.example.borrowedcollectionroots.models.HolderDto
import org.example.borrowedcollectionroots.models.ListBridgeDto
import org.example.borrowedcollectionroots.models.ListFalseBridgeDto
import org.example.borrowedcollectionroots.models.MapAliasDto
import org.example.borrowedcollectionroots.models.MapBridgeDto
import org.example.borrowedcollectionroots.models.MixedBridgeDto
import org.example.borrowedcollectionroots.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BorrowedCollectionStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun borrowedCollectionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> borrowedCollectionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic borrowed collection body ${body::class}")
}
private suspend fun borrowedCollectionReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing borrowed collection rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong borrowed collection failure: $cause" }
    }
}

suspend fun rootBorrowedCollectionContracts(checked: Boolean = false) = withTimeout(30000) {
    check(serializer<ListBridgeDto>().descriptor.isNullable && !serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<ListFalseBridgeDto>().descriptor.isNullable)
    check(serializer<MapAliasDto>().descriptor.isNullable && !serializer<MapBridgeDto>().descriptor.isNullable)
    check(!serializer<MixedBridgeDto>().descriptor.isNullable)
    val values = listOf(null, "ok")
    val mapping: MapBridgeDto = mapOf("key" to values)
    val mixed: MixedBridgeDto = listOf(null, mapping)
    fun holder(list: List<String?> = values, map: Map<String, List<String?>> = mapping, mix: MixedBridgeDto = mixed) = HolderDto(narrow = list, map = map, mixed = mix, list = values)
    val model = holder()
    if (checked) {
        borrowedCollectionReject("minItems") { holder(emptyList()) }
        borrowedCollectionReject("maxItems") { holder(listOf("aa", "bb", "cc", "dd")) }
        borrowedCollectionReject("uniqueItems") { holder(listOf("ok", "ok")) }
        borrowedCollectionReject("minLength") { holder(listOf("a")) }
        borrowedCollectionReject("minProperties") { holder(map = emptyMap()) }
        borrowedCollectionReject("minLength") { holder(map = mapOf("key" to listOf("a"))) }
        borrowedCollectionReject("minProperties") { holder(mix = listOf(emptyMap())) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val alias: ListBridgeDto = values
        val again: AgainBridgeDto = values
        check(codec.decodeFromString<ListBridgeDto>(codec.encodeToString(alias)) == values)
        check(codec.decodeFromString<AgainBridgeDto>(codec.encodeToString(again)) == values)
        check(codec.decodeFromString<ListBridgeDto>("null") == null)
        check(codec.decodeFromString<MapBridgeDto>(codec.encodeToString(mapping)) == mapping)
        check(codec.decodeFromString<MapAliasDto>("null") == null)
        check(codec.decodeFromString<MixedBridgeDto>(codec.encodeToString(mixed)) == mixed)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        borrowedCollectionReject("null") { codec.decodeFromString<ListFalseBridgeDto>("null") }
        borrowedCollectionReject("JSON") { codec.decodeFromString<ListBridgeDto>("{}") }
        val unsafe: ListFalseBridgeDto = codec.decodeFromString("[\"a\"]")
        if (checked) borrowedCollectionReject("minLength") { holder(unsafe) }
        val listChoice: CollectionChoiceDto = CollectionChoiceDto.List(values)
        val mapChoice: CollectionChoiceDto = CollectionChoiceDto.Map(mapping)
        for (choice in listOf(listChoice, mapChoice)) check(codec.encodeToJsonElement(codec.decodeFromString<CollectionChoiceDto>(codec.encodeToString(choice))) == codec.encodeToJsonElement(choice))
        if (checked) borrowedCollectionReject("minItems") { CollectionChoiceDto.List(emptyList()) }
        val absent = SubmitInputDto(requiredNull = null, requiredNarrow = values)
        val clear = SubmitInputDto(requiredNull = null, requiredNarrow = values, optional = Patch.Null, map = Patch.Null)
        val assigned = SubmitInputDto(requiredNull = null, requiredNarrow = values, optional = Patch.Value(values), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        check("optional" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["optional"] == JsonNull)
        if (nulls) borrowedCollectionReject("requiredNull") { codec.decodeFromString<SubmitInputDto>("{\"requiredNarrow\":[\"ok\"]}") }
        if (checked) {
            borrowedCollectionReject("minItems") { SubmitInputDto(requiredNull = null, requiredNarrow = values, optional = Patch.Value(emptyList())) }
            borrowedCollectionReject("minLength") { SubmitInputDto(requiredNull = null, requiredNarrow = values, map = Patch.Value(mapOf("key" to listOf("a")))) }
        }
        var expected = "null"
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BorrowedCollectionStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val bad = mode; val keepOpen = endless
                    producers.launch { try {
                        val frame = when (bad) {
                            1 -> "event: map\r\ndata: null\r\n\r\n"
                            2 -> "event: values\r\ndata: []\r\n\r\n"
                            3 -> "event: map\r\ndata: {\"key\":[\"a\"]}\r\n\r\n"
                            4 -> "event: values\r\ndata: {\r\n\r\n"
                            5 -> "event: map\r\ndata: {\"key\":[null,null]}\r\n\r\n"
                            else -> "event: values\r\ndata: null\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(mapping)}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (bad != 0 || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/patch" -> { check(codec.parseToJsonElement(borrowedCollectionBody(request.body)) == codec.parseToJsonElement(expected)); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(borrowedCollectionBody(request.body)) == codec.parseToJsonElement(expected))
                    val response = when (mode) { 1 -> "null"; 2 -> if (request.url.encodedPath == "/map") "{}" else "[]"; 3 -> expected.replace("ok", "a"); else -> expected }
                    respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootBorrowedCollectionApiClient(http, "https://example.test/", codec).default
            if (checked) {
                val before = calls; var callback = false
                borrowedCollectionReject("minItems") { api.echo(emptyList()) { callback = true } }
                borrowedCollectionReject("minLength") { api.echoMap(mapOf("key" to listOf("a"))) { callback = true } }
                check(calls == before && !callback)
            }
            check(api.echo(null) == null)
            expected = codec.encodeToString(values); check(api.echo(values) == values)
            if (checked) { mode = 2; borrowedCollectionReject("minItems") { api.echo(values) }; mode = 3; borrowedCollectionReject("minLength") { api.echo(values) } }
            mode = 0; expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
            mode = 1; borrowedCollectionReject("null") { api.echoMap(mapping) }
            if (checked) { mode = 2; borrowedCollectionReject("minProperties") { api.echoMap(mapping) }; mode = 3; borrowedCollectionReject("minLength") { api.echoMap(mapping) } }
            mode = 0; expected = codec.encodeToString(mixed); check(api.echoMixed(mixed) == mixed)
            for (input in listOf(absent, clear, assigned)) { expected = codec.encodeToString(input); api.submit(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check((events[0] as WatchEventDto.Values).data == null && (events[1] as WatchEventDto.Map).data == mapping)
            }
            for ((bad, message) in listOf(1 to "null", 4 to "JSON") + if (checked) listOf(2 to "minItems", 3 to "minLength", 5 to "uniqueItems") else emptyList()) {
                mode = bad; withTimeout(3000) { borrowedCollectionReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true; withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { cold.toList() }.size == 2)
            expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("RootBorrowedCollection Json defaults=$defaults nulls=$nulls original-graph-checks-nullable-Patch-union-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}

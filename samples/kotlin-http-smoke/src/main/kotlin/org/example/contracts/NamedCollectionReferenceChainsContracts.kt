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
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.namedcollectionchains.NamedCollectionReferenceChainsApiClient
import org.example.namedcollectionchains.api.models.SubmitInputDto
import org.example.namedcollectionchains.api.models.WatchEventDto
import org.example.namedcollectionchains.models.AgainBridgeDto
import org.example.namedcollectionchains.models.CollectionChoiceDto
import org.example.namedcollectionchains.models.HolderDto
import org.example.namedcollectionchains.models.ListBridgeDto
import org.example.namedcollectionchains.models.ListFalseBridgeDto
import org.example.namedcollectionchains.models.MapBridgeDto
import org.example.namedcollectionchains.models.MapFalseBridgeDto
import org.example.namedcollectionchains.models.OpenBridgeDto
import org.example.namedcollectionchains.models.Patch
import org.example.namedcollectionchains.models.RowsBridgeDto
import org.example.namedcollectionchains.models.ZRowsDtoInner

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class CollectionChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun collectionChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> collectionChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic collection-chain body ${body::class}")
}
private suspend fun collectionChainReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing collection-chain rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong collection-chain failure: $cause" }
    }
}

suspend fun namedCollectionReferenceChainsContracts(checked: Boolean = false) = withTimeout(30000) {
    check(serializer<ListBridgeDto>().descriptor.isNullable && serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<ListFalseBridgeDto>().descriptor.isNullable)
    check(serializer<MapBridgeDto>().descriptor.isNullable && !serializer<MapFalseBridgeDto>().descriptor.isNullable)
    check(serializer<OpenBridgeDto>().descriptor.isNullable && !serializer<RowsBridgeDto>().descriptor.isNullable)
    val values = listOf(null, "ok")
    val mapping: MapFalseBridgeDto = mapOf("key" to "ok", "nil" to null)
    val rows: RowsBridgeDto = listOf(ZRowsDtoInner("parent", listOf(ZRowsDtoInner("child"))))
    val open: OpenBridgeDto = JsonObject(mapOf("nil" to JsonNull, "text" to JsonPrimitive("kept")))
    fun holder(list: List<String?> = values, map: Map<String, String?> = mapping) = HolderDto(narrow = list,
        wrappedNarrow = list, mapNarrow = map, mapUseNarrow = map, rows = rows, nested = listOf(null, values), values = values, map = mapping, open = open)
    val model = holder()
    if (checked) {
        collectionChainReject("minItems") { holder(emptyList()) }
        collectionChainReject("uniqueItems") { holder(listOf("ok", "ok")) }
        collectionChainReject("minLength") { holder(listOf("a")) }
        collectionChainReject("minProperties") { holder(map = emptyMap()) }
        collectionChainReject("minLength") { ZRowsDtoInner("") }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val alias: ListBridgeDto = values
        val again: AgainBridgeDto = alias
        check(codec.decodeFromString<ListBridgeDto>(codec.encodeToString(alias)) == values)
        check(codec.decodeFromString<AgainBridgeDto>(codec.encodeToString(again)) == values)
        check(codec.decodeFromString<ListBridgeDto>("null") == null)
        check(codec.decodeFromString(ListSerializer(String.serializer().nullable).nullable, codec.encodeToString(alias)) == values)
        check(codec.decodeFromString<MapFalseBridgeDto>(codec.encodeToString(mapping)) == mapping)
        check(codec.decodeFromString<MapBridgeDto>("null") == null)
        check(codec.decodeFromString<OpenBridgeDto>(codec.encodeToString(open)) == open)
        check(codec.decodeFromString<RowsBridgeDto>(codec.encodeToString(rows)) == rows)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        collectionChainReject("null") { codec.decodeFromString<ListFalseBridgeDto>("null") }
        collectionChainReject("JSON") { codec.decodeFromString<ListBridgeDto>("{}") }
        val unsafe: ListBridgeDto = codec.decodeFromString("[\"a\"]")
        if (checked) collectionChainReject("minLength") { holder(unsafe!!) }
        val rowChoice: CollectionChoiceDto = CollectionChoiceDto.Rows(rows)
        val valueChoice: CollectionChoiceDto = CollectionChoiceDto.Values(values)
        for (choice in listOf(rowChoice, valueChoice)) check(codec.encodeToJsonElement(codec.decodeFromString<CollectionChoiceDto>(codec.encodeToString(choice))) == codec.encodeToJsonElement(choice))
        if (checked) collectionChainReject("minItems") { CollectionChoiceDto.Values(emptyList()) }
        val absent = SubmitInputDto(requiredNull = null, requiredNarrow = mapping)
        val clear = SubmitInputDto(requiredNull = null, requiredNarrow = mapping, optional = Patch.Null, map = Patch.Null)
        val assigned = SubmitInputDto(requiredNull = null, requiredNarrow = mapping, optional = Patch.Value(values), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        check("optional" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["optional"] == JsonNull)
        if (nulls) collectionChainReject("requiredNull") { codec.decodeFromString<SubmitInputDto>("{\"requiredNarrow\":{\"key\":\"ok\"}}") }
        if (checked) collectionChainReject("minItems") { SubmitInputDto(requiredNull = null, requiredNarrow = mapping, optional = Patch.Value(emptyList())) }
        var expected = "null"
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = CollectionChainStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val bad = mode; val keepOpen = endless
                    producers.launch { try {
                        val frame = when (bad) {
                            1 -> "event: map\r\ndata: null\r\n\r\n"
                            2 -> "event: values\r\ndata: []\r\n\r\n"
                            3 -> "event: map\r\ndata: {\"key\":\"a\"}\r\n\r\n"
                            4 -> "event: values\r\ndata: {\r\n\r\n"
                            else -> "event: values\r\ndata: null\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(mapping)}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (bad != 0 || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/patch" -> { check(codec.parseToJsonElement(collectionChainBody(request.body)) == codec.parseToJsonElement(expected)); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(collectionChainBody(request.body)) == codec.parseToJsonElement(expected))
                    val response = when (mode) { 1 -> "null"; 2 -> if (request.url.encodedPath == "/map") "{}" else "[]"; 3 -> expected.replace("ok", "a"); else -> expected }
                    respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = NamedCollectionReferenceChainsApiClient(http, "https://example.test/", codec).default
            if (checked) {
                val before = calls; var callback = false
                collectionChainReject("minItems") { api.echo(emptyList()) { callback = true } }
                check(calls == before && !callback)
            }
            check(api.echo(null) == null)
            expected = codec.encodeToString(values); check(api.echo(values) == values && api.echo2(values) == values)
            expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
            mode = 1; collectionChainReject("null") { api.echoMap(mapping) }
            if (checked) { mode = 2; collectionChainReject("minProperties") { api.echoMap(mapping) }; mode = 3; collectionChainReject("minLength") { api.echoMap(mapping) } }
            mode = 0; expected = codec.encodeToString(rows); check(api.rows(rows).first().id == "parent")
            for (input in listOf(absent, clear, assigned)) { expected = codec.encodeToString(input); api.submit(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check((events[0] as WatchEventDto.Values).data == null && (events[1] as WatchEventDto.Map).data == mapping)
            }
            for ((bad, message) in listOf(1 to "null", 4 to "JSON") + if (checked) listOf(2 to "minItems", 3 to "minLength") else emptyList()) {
                mode = bad; withTimeout(3000) { collectionChainReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true; withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { cold.toList() }.size == 2)
            expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("NamedCollectionReferenceChains Json defaults=$defaults nulls=$nulls linked-backing-peers-nullable-checks-Patch-union-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}

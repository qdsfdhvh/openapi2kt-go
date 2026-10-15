@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.rootcollectionprobes.models.ACountWideDto as RootCollectionCountWideDto
import org.example.rootcollectionprobes.models.ADeepDto as RootCollectionDeepDto
import org.example.rootcollectionprobes.models.ANearDto as RootCollectionNearDto
import org.example.rootcollectionprobes.models.AWideDto as RootCollectionWideDto
import org.example.rootcollections.RootCollectionAliasApiClient
import org.example.rootcollections.api.models.SubmitInputDto
import org.example.rootcollections.api.models.WatchEventDto
import org.example.rootcollections.models.AValuesDto
import org.example.rootcollections.models.AMapNarrowDto
import org.example.rootcollections.models.HolderDto
import org.example.rootcollections.models.Patch
import org.example.rootcollections.models.ZRowsDtoInner

private fun rootCollectionReject(block: () -> Unit) {
    try { block(); error("Missing root collection check") }
    catch (cause: IllegalArgumentException) { check(cause.message?.startsWith("Field value") == true) }
}

private suspend fun rootCollectionWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing root collection wire error") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong root collection failure $cause" }
    }
}

private fun rootCollectionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> rootCollectionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected root collection request body")
}

suspend fun rootCollectionAliasContracts() {
    val values = listOf(null, "ok")
    val map: AMapNarrowDto = mapOf("key" to "ok", "null" to null)
    val rows = listOf(ZRowsDtoInner("parent", children = listOf(ZRowsDtoInner("child"))))
    val holder = HolderDto(narrow = values, wrappedNarrow = values, mapNarrow = map, mapUseNarrow = map, rows = rows, nested = listOf(null, values))
    rootCollectionReject { holder.copy(narrow = emptyList()) }
    rootCollectionReject { holder.copy(wrapped = listOf("a")) }
    rootCollectionReject { holder.copy(nested = listOf(listOf("ok", "ok"))) }
    rootCollectionReject { holder.copy(mapNarrow = emptyMap()) }
    rootCollectionReject { holder.copy(mapUseNarrow = mapOf("bad" to "a")) }
    rootCollectionReject { holder.copy(forced = listOf("a")) }
    rootCollectionReject { ZRowsDtoInner("") }
    rootCollectionReject { holder.copy(open = JsonObject(emptyMap())) }
    val descriptor = HolderDto.serializer().descriptor
    for (name in listOf("narrow", "wrappedNarrow", "mapNarrow", "mapUseNarrow")) {
        check(!descriptor.getElementDescriptor(descriptor.getElementIndex(name)).isNullable)
        check(!descriptor.isElementOptional(descriptor.getElementIndex(name)))
    }
    check(descriptor.getElementDescriptor(descriptor.getElementIndex("forced")).isNullable)
    val requestDescriptor = SubmitInputDto.serializer().descriptor
    check(!requestDescriptor.isElementOptional(requestDescriptor.getElementIndex("requiredNull")))
    check(requestDescriptor.getElementDescriptor(requestDescriptor.getElementIndex("requiredNull")).isNullable)
    check(!requestDescriptor.getElementDescriptor(requestDescriptor.getElementIndex("requiredNarrow")).isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val streamCodec = Json(codec) { ignoreUnknownKeys = false }
        check(codec.decodeFromString<AValuesDto>("null") == null)
        check(codec.decodeFromString<AValuesDto>(codec.encodeToString<AValuesDto>(values)) == values)
        val unchecked: AValuesDto = codec.decodeFromString("[\"a\"]")
        rootCollectionReject { holder.copy(narrow = unchecked!!) }
        check(codec.decodeFromString<AMapNarrowDto>(codec.encodeToString<AMapNarrowDto>(map)) == map)
        val text = codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(text) == holder)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.parseToJsonElement(text)) == holder)
        rootCollectionReject { codec.decodeFromString<HolderDto>(text.replace("\"ok\"", "\"a\"")) }
        rootCollectionWireReject("null") { codec.decodeFromString<HolderDto>(text.replace("\"mapNarrow\":{\"key\":\"ok\",\"null\":null}", "\"mapNarrow\":null")) }
        check(codec.decodeFromString<RootCollectionWideDto>("{\"p0\":[null,\"ok\"],\"p1\":{\"key\":\"ok\"}}").p1 == mapOf("key" to "ok"))
        rootCollectionReject { codec.decodeFromString<RootCollectionWideDto>("{\"p1\":{\"key\":\"a\"}}") }
        rootCollectionReject { codec.decodeFromString<RootCollectionCountWideDto>("{\"p0\":{}}") }
        val near = RootCollectionNearDto(listOf("valid"))
        check(codec.decodeFromString<RootCollectionNearDto>(codec.encodeToString(near)) == near)
        val deepJson = "[".repeat(15) + "[null,\"ok\"]" + "]".repeat(15)
        val checkedDeep = codec.decodeFromString<RootCollectionDeepDto>("{\"v\":$deepJson}")
        check(codec.decodeFromString<RootCollectionDeepDto>(codec.encodeToString(checkedDeep)) == checkedDeep)
        rootCollectionReject { codec.decodeFromString<RootCollectionDeepDto>("{\"v\":${deepJson.replace("ok", "a")}}") }
        var mode = 0
        var calls = 0
        var expected = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { httpRequest ->
                calls++
                when (httpRequest.url.encodedPath) {
                    "/alias", "/direct", "/map" -> {
                        check(rootCollectionBody(httpRequest.body) == expected)
                        val response = when (mode) { 1 -> "null"; 2 -> if (httpRequest.url.encodedPath == "/map") "{}" else "[]"; 3 -> expected.replace("ok", "a"); else -> expected }
                        respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> { check(rootCollectionBody(httpRequest.body) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val frame = when (current) {
                                    1 -> "event: map\r\ndata: null\r\n\r\n"
                                    2 -> "event: values\r\ndata: []\r\n\r\n"
                                    3 -> "event: map\r\ndata: {\"key\":\"a\"}\r\n\r\n"
                                    4 -> "event: map\r\ndata: malformed\r\n\r\n"
                                    else -> "event: values\r\nid: binding-id\r\ndata: null\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(map)}\r\n\r\n"
                                }
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected root collection route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootCollectionAliasApiClient(http, "http://example.test/", streamCodec).default
            val before = calls
            var callback = false
            rootCollectionWireReject("minItems") { api.echo(emptyList()) { callback = true } }
            check(!callback && calls == before)
            expected = "null"; check(api.echo(null) == null)
            expected = codec.encodeToString(values); check(api.echo(values) == values); check(api.echo2(values) == values)
            expected = codec.encodeToString(map); check(api.echoMap(map) == map)
            for ((bad, message) in listOf(1 to "null", 2 to "minProperties", 3 to "minLength")) {
                mode = bad; rootCollectionWireReject(message) { api.echoMap(map) }
            }
            mode = 0
            for (state in listOf<Patch<List<String?>>>(Patch.Absent, Patch.Null, Patch.Value(values))) {
                val mapState: Patch<Map<String, String?>> = when (state) { Patch.Absent -> Patch.Absent; Patch.Null -> Patch.Null; is Patch.Value -> Patch.Value(map) }
                val input = SubmitInputDto(optional = state, map = mapState, requiredNull = null, requiredNarrow = map)
                val encoded = codec.encodeToJsonElement(input) as JsonObject
                check(when (state) { Patch.Absent -> "optional" !in encoded && "map" !in encoded; Patch.Null -> encoded["optional"] == JsonNull && encoded["map"] == JsonNull; is Patch.Value -> encoded["optional"] == codec.encodeToJsonElement(values) && encoded["map"] == codec.encodeToJsonElement(map) })
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(), encoded) == input)
                expected = codec.encodeToString(input); api.submit(input)
                rootCollectionReject { input.copy(map = Patch.Value(emptyMap())) }
                rootCollectionReject { input.copy(optional = Patch.Value(listOf("a"))) }
            }
            val beforeStream = calls
            val cold = api.watch()
            check(calls == beforeStream)
            repeat(2) {
                val events = withTimeout(10_000) { cold.toList() }
                check((events[0] as WatchEventDto.Values).data == null)
                check((events[1] as WatchEventDto.Map).data == map)
                check(events.all { it.id == "binding-id" })
            }
            check(calls == beforeStream + 2)
            for ((bad, message) in listOf(1 to "null", 2 to "minItems", 3 to "minLength", 4 to "JSON")) {
                mode = bad; rootCollectionWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Root binding error left live stream open" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 2)
            expected = codec.encodeToString(map); check(api.echoMap(map) == map)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Root collection alias contracts passed: defaults=$defaults/nulls=$nulls; backing views/canonical peers/240+16+48/Patch/HTTP/live SSE close-reuse-cold")
    }
}

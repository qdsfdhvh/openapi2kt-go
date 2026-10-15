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
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.structuredvalues.StructuredValues
import org.example.structuredvalues.api.models.WatchEventDto
import org.example.structuredvalues.models.ArrayAliasDto
import org.example.structuredvalues.models.ArrayDto
import org.example.structuredvalues.models.ChainDto
import org.example.structuredvalues.models.HolderDto
import org.example.structuredvalues.models.ObjectDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StructuredValuesEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers,
            response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers,
            response.version, body, response.callContext)
    }
}
private fun structuredValuesReject(block: () -> Unit) {
    try { block(); error("Expected typed structured value rejection") }
    catch (_: SerializationException) { }
}
private suspend fun structuredValuesWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected typed structured wire rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong structured wire failure: $cause" }
    }
}
private fun structuredValuesBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> structuredValuesBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected structured value body $body")
}

suspend fun structuredValuesContracts() {
    val obj: ObjectDto = JsonObject(mapOf("different" to JsonPrimitive(2)))
    val array: ArrayDto = JsonArray(listOf(JsonPrimitive("alternative"), JsonNull))
    val holder = HolderDto(kotlinx = "shadow", objects = listOf(obj, null), arrays = mapOf("one" to array), inlineObjects = listOf(obj), inlineArrays = mapOf("one" to array))
    check(holder.payload == Json.parseToJsonElement("{\"\$ref\":\"data\",\"n\":1}"))
    check(holder.rows == Json.parseToJsonElement("[null,true]"))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val aliasObject: ChainDto = codec.decodeFromString("{\"other\":true}")
        val aliasArray: ArrayAliasDto = codec.decodeFromString("[1,2]")
        check(aliasObject == codec.parseToJsonElement("{\"other\":true}") && aliasArray == codec.parseToJsonElement("[1,2]"))
        for (wrong in listOf("null", "false", "0", "\"object\"", "[]")) structuredValuesReject { codec.decodeFromString<ObjectDto>(wrong) }
        for (wrong in listOf("null", "true", "1", "\"array\"", "{}")) structuredValuesReject { codec.decodeFromString<ArrayAliasDto>(wrong) }
        val missing = codec.decodeFromString<HolderDto>("{}")
        check(missing.payload == holder.payload && missing.rows == holder.rows)
        val wire = codec.encodeToJsonElement(HolderDto.serializer(), holder) as JsonObject
        check(wire["payload"] == holder.payload && wire["rows"] == holder.rows)
        val back = codec.decodeFromJsonElement(HolderDto.serializer(), wire)
        check(back.objects == listOf(obj, null) && back.arrays?.get("one") == array)
        check(back.inlineObjects?.single() == obj && back.inlineArrays?.get("one") == array)
        structuredValuesReject { codec.decodeFromString<HolderDto>("{\"inlineObjects\":[[]]}") }
        structuredValuesReject { codec.decodeFromString<HolderDto>("{\"inlineArrays\":{\"one\":{}}}") }
        structuredValuesReject { codec.decodeFromString<HolderDto>("{\"payload\":null}") }
        val changed = codec.decodeFromString<HolderDto>("{\"payload\":{\"other\":true},\"rows\":[1,2]}")
        check(changed.payload == aliasObject && changed.rows == aliasArray)
        structuredValuesWire(codec, obj, array, holder)
        println("StructuredConstValues Json defaults=$defaults nulls=$nulls root-alias-body-leaf-kind-HTTP-liveSSE passed")
    }
}

private suspend fun structuredValuesWire(codec: Json, obj: JsonObject, array: JsonArray, holder: HolderDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0
    val engine = StructuredValuesEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/object" -> {
                    check(structuredValuesBody(request.body) == codec.encodeToString(obj))
                    respond(if (bad) "[]" else codec.encodeToString(obj), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/array" -> {
                    check(structuredValuesBody(request.body) == codec.encodeToString(array))
                    respond(if (bad) "{}" else codec.encodeToString(array), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/holder" -> {
                    check(structuredValuesBody(request.body) == codec.encodeToString(holder))
                    respond(codec.encodeToString(holder), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/objects" -> respond(if (bad) "[[]]" else codec.encodeToString(listOf(obj)), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/arrays" -> respond(if (bad) "{\"one\":{}}" else codec.encodeToString(mapOf("one" to array)), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/params" -> {
                    check(request.url.parameters["object"] == obj.toString())
                    check(request.url.parameters["array"] == array.toString())
                    respond("", HttpStatusCode.NoContent)
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val objectPayload = if (invalid) "[]" else codec.encodeToString(obj)
                            val frame = "event: object\r\nid: one\r\ndata: $objectPayload\r\n\r\n" +
                                if (invalid) "" else "event: array\r\nid: two\r\ndata: ${codec.encodeToString(array)}\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected structured value request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StructuredValues(http, "http://example.test/", codec).values
        check(api.sendObject(obj) == obj && api.sendArray(array) == array)
        check(api.sendHolder(holder).payload == holder.payload)
        check(api.getObjects() == listOf(obj) && api.getArrays()["one"] == array)
        api.getParams(obj, array)
        val before = calls
        val flow = api.watch()
        check(calls == before)
        repeat(2) {
            val events = withTimeout(10000) { flow.toList() }
            check((events[0] as WatchEventDto.Object).data == obj && events[0].id == "one")
            check((events[1] as WatchEventDto.Array).data == array && events[1].id == "two")
        }
        bad = true
        structuredValuesWireReject { api.sendObject(obj) }
        structuredValuesWireReject { api.sendArray(array) }
        structuredValuesWireReject { api.getObjects() }
        structuredValuesWireReject { api.getArrays() }
        val beforeBad = channels.size
        structuredValuesWireReject { withTimeout(10000) { api.watch().toList() } }
        check(channels.size == beforeBad + 1 && channels.last().isClosedForRead)
        bad = false
        check(api.sendObject(obj) == obj)
    } finally {
        http.close()
        producers.cancel()
        channels.forEach { it.close() }
    }
}

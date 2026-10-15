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
import org.example.nullablestructured.NullableStructured
import org.example.nullablestructured.api.models.WatchEventDto
import org.example.nullablestructured.models.ObjectDto
import org.example.nullablestructured.models.AliasDto
import org.example.nullablestructured.models.NarrowDto
import org.example.nullablestructured.models.ReopenDto
import org.example.nullablestructured.models.ArrayDto
import org.example.nullablestructured.models.ArrayNarrowDto
import org.example.nullablestructured.models.HolderDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullableStructuredEngine(config: MockEngineConfig) : MockEngine(config) {
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
private fun nullableStructuredReject(block: () -> Unit) {
    try { block(); error("Expected narrowed/kind rejection") }
    catch (_: SerializationException) { }
}
private suspend fun nullableStructuredWireReject(block: suspend () -> Unit) {
    try { block(); error("Expected nullable structured wire rejection") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong wire failure: $cause" }
    }
}
private fun nullableStructuredBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nullableStructuredBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable body $body")
}

suspend fun nullableStructuredContracts() {
    val obj: NarrowDto = JsonObject(mapOf("other" to JsonPrimitive(true)))
    val array: ArrayNarrowDto = JsonArray(listOf(JsonPrimitive(2)))
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        check(codec.decodeFromString<ObjectDto>("null") == null)
        check(codec.decodeFromString<AliasDto>("null") == null)
        check(codec.decodeFromString<ReopenDto>("null") == null)
        check(codec.decodeFromString<ArrayDto>("null") == null)
        check(codec.encodeToString<ObjectDto>(null) == "null")
        check(codec.encodeToString<ArrayDto>(null) == "null")
        check(codec.decodeFromString<NarrowDto>(codec.encodeToString(obj)) == obj)
        check(codec.decodeFromString<ArrayNarrowDto>(codec.encodeToString(array)) == array)
        nullableStructuredReject { codec.decodeFromString<NarrowDto>("null") }
        nullableStructuredReject { codec.decodeFromString<ArrayNarrowDto>("null") }
        nullableStructuredReject { codec.decodeFromString<ObjectDto>("[]") }
        nullableStructuredReject { codec.decodeFromString<ArrayDto>("{}") }
        val holder=HolderDto(label="value",items=listOf(obj,null),narrowItems=listOf(obj),literalItems=mapOf("one" to null,"two" to obj))
        val back=codec.decodeFromString<HolderDto>(codec.encodeToString(holder))
        check(back.items == listOf(obj,null) && back.narrowItems == listOf(obj))
        check(back.literalItems?.get("one") == null && back.literalItems?.get("two") == obj)
        check(holder.fixed == codec.parseToJsonElement("{\"n\":1}") && holder.fixedArray == codec.parseToJsonElement("[null]"))
        check(codec.decodeFromString<HolderDto>("{}").fixed == holder.fixed)
        val wire=codec.encodeToJsonElement(HolderDto.serializer(),holder) as JsonObject
        check(wire["fixed"] == holder.fixed && wire["fixedArray"] == holder.fixedArray)
        nullableStructuredReject { codec.decodeFromString<HolderDto>("{\"fixed\":null}") }
        nullableStructuredReject { codec.decodeFromString<HolderDto>("{\"narrowItems\":[null]}") }
        nullableStructuredWire(codec,obj,array,holder)
        println("NullableStructuredConst Json defaults=$defaults nulls=$nulls root-null-nearestfalse-bodydefault-leaf-HTTP-liveSSE passed")
    }
}

private suspend fun nullableStructuredWire(codec: Json,obj: JsonObject,array: JsonArray,holder: HolderDto) {
    val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
    val channels=CopyOnWriteArrayList<ByteChannel>()
    var expected=""
    var bad=false
    var nullEvents=false
    var calls=0
    val engine=NullableStructuredEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when(request.url.encodedPath) {
                "/object","/array","/narrow","/array-narrow","/holder" -> {
                    check(nullableStructuredBody(request.body) == expected)
                    respond(if(bad) "null" else expected,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                }
                "/events" -> {
                    val channel=ByteChannel(autoFlush=true);channels+=channel
                    val invalid=bad;val emitNull=nullEvents
                    producers.launch {
                        try {
                            val first=if(invalid) "[]" else if(emitNull) "null" else codec.encodeToString(obj)
                            val frame="event: object\r\ndata: $first\r\n\r\n" + if(invalid) "" else
                                "event: array\r\ndata: ${if(emitNull) "null" else codec.encodeToString(array)}\r\n\r\nevent: narrow\r\ndata: ${codec.encodeToString(obj)}\r\n\r\n"
                            for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if(invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                }
                else -> error("Unexpected nullable request")
            }
        }
    })
    val http=HttpClient(engine) { install(SSE);install(ContentNegotiation) { json(codec) } }
    try {
        val api=NullableStructured(http,"http://example.test/",codec).nullable
        for(value in listOf(obj,null)) {
            expected=codec.encodeToString<JsonObject?>(value)
            check(api.sendObject(value) == value)
        }
        for(value in listOf(array,null)) {
            expected=codec.encodeToString<JsonArray?>(value)
            check(api.sendArray(value) == value)
        }
        expected=codec.encodeToString(obj);check(api.sendNarrow(obj) == obj)
        expected=codec.encodeToString(array);check(api.sendArrayNarrow(array) == array)
        expected=codec.encodeToString(holder);check(api.sendHolder(holder).fixed == holder.fixed)
        val before=calls;val flow=api.watch();check(calls == before)
        for(emitNull in listOf(false,true)) {
            nullEvents=emitNull
            val events=withTimeout(10000) { flow.toList() }
            check(events.size == 3)
            check((events[0] as WatchEventDto.Object).data == if(emitNull) null else obj)
            check((events[1] as WatchEventDto.Array).data == if(emitNull) null else array)
            check((events[2] as WatchEventDto.Narrow).data == obj)
        }
        bad=true
        expected=codec.encodeToString(obj);nullableStructuredWireReject { api.sendNarrow(obj) }
        expected=codec.encodeToString(array);nullableStructuredWireReject { api.sendArrayNarrow(array) }
        val count=channels.size;nullableStructuredWireReject { withTimeout(10000) { api.watch().toList() } }
        check(channels.size == count+1 && channels.last().isClosedForRead)
        bad=false;expected=codec.encodeToString(obj);check(api.sendNarrow(obj) == obj)
    } finally { http.close();producers.cancel();channels.forEach { it.close() } }
}

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
import kotlinx.coroutines.yield
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.intrinsicnullableraw.IntrinsicNullableRawApiClient
import org.example.intrinsicnullableraw.api.models.SubmitInputDto
import org.example.intrinsicnullableraw.api.models.WatchEventDto
import org.example.intrinsicnullableraw.api.models.WatchNarrowEventDto
import org.example.intrinsicnullableraw.models.AInheritedDto
import org.example.intrinsicnullableraw.models.AFalseDto
import org.example.intrinsicnullableraw.models.CEmptyDto
import org.example.intrinsicnullableraw.models.HolderDto
import org.example.intrinsicnullableraw.models.Patch
import org.example.intrinsicnullableraw.models.ZRawDto
import org.example.intrinsicnullableraw.models.ZEmptyDto
import org.example.intrinsicnullablerawprobes.models.ANearDto as RawNearDto
import org.example.intrinsicnullablerawprobes.models.ANarrowDto as RawNarrowDto
import org.example.intrinsicnullablerawprobes.models.AWideDto as RawWideDto

private fun nullableRawBody(body: OutgoingContent): String=when(body) {
    is OutgoingContent.ContentWrapper -> nullableRawBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable raw body ${body::class}")
}
private suspend fun nullableRawReject(block: suspend () -> Unit) {
    try { block();error("Malformed raw JSON accepted") }
    catch(cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) }
}
suspend fun intrinsicNullableRawContracts() {
    check(serializer<ZRawDto>().descriptor.isNullable&&serializer<ZEmptyDto>().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable&&serializer<CEmptyDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable&&!JsonElement.serializer().descriptor.isNullable)
    val future=JsonObject(mapOf("unknown" to JsonPrimitive("kept")))
    val kinds=listOf(JsonNull,JsonPrimitive("text"),JsonPrimitive(42),JsonPrimitive(true),JsonArray(listOf(JsonNull,future)),future)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        for(value in listOf(null)+kinds) {
            val text=codec.encodeToString(serializer<AInheritedDto>(),value)
            val expected=if(value===JsonNull) null else value
            check(codec.decodeFromString<ZRawDto>(codec.encodeToString<ZRawDto>(value))==expected)
            check(codec.decodeFromString<ZEmptyDto>(codec.encodeToString<ZEmptyDto>(value))==expected)
            check(codec.decodeFromString(serializer<AInheritedDto>(),text)==expected)
            check(codec.decodeFromString(JsonElement.serializer().nullable,text)==expected)
        }
        check(codec.decodeFromString<RawNearDto>("{\"v\":null}").v==null)
        check(codec.decodeFromString<RawNarrowDto>("{\"v\":null}").v===JsonNull)
        val wide=codec.decodeFromJsonElement(RawWideDto.serializer(),JsonObject((0 until 240).associate { "p$it" to if(it==239) future else JsonNull }))
        check(wide.p0==null&&wide.p239==future&&codec.decodeFromString<RawWideDto>(codec.encodeToString(wide.copy(p0=JsonPrimitive(1))))==wide.copy(p0=JsonPrimitive(1)))
        val holder=HolderDto(inherited=null,forcedTrue=future,forcedFalse=JsonNull,wrapper=null,wrapperFalse=JsonNull,items=listOf(null,future),values=mapOf("nil" to null,"future" to future),requiredNull=null)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))==holder)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder.copy(inherited=JsonNull))).inherited==null)
        val absent=SubmitInputDto(items=emptyList(),values=emptyMap(),requiredNull=null)
        check("inherited" !in (codec.encodeToJsonElement(absent) as JsonObject))
        if(nulls) nullableRawReject { codec.decodeFromString<SubmitInputDto>("{\"items\":[],\"values\":{}}") }
        val clear=absent.copy(inherited=Patch.Null,wrapper=Patch.Null)
        val value=absent.copy(inherited=Patch.Value(future),wrapper=Patch.Value(JsonNull))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)).wrapper===Patch.Null)
        for(p in listOf(absent,clear,value)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(p)))==codec.encodeToJsonElement(p))
        var expected: JsonElement=JsonNull
        var syntax=false
        var calls=0
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=StreamingTestEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when(request.url.encodedPath) {
                "/watch","/watch-narrow" -> {
                    val narrow=request.url.encodedPath=="/watch-narrow"
                    val channel=ByteChannel(autoFlush=true);channels+=channel;val broken=syntax
                    producers.launch { try {
                        val event=if(narrow) "narrow" else "raw"
                        val frame=if(broken) "event: $event\r\ndata: {\r\n\r\n" else kinds.joinToString("") { "event: $event\r\ndata: ${codec.encodeToString(JsonElement.serializer(),it)}\r\n\r\n" }
                        for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if(broken) awaitCancellation()
                    } finally { channel.close() } }
                    respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                }
                "/submit" -> { check(codec.parseToJsonElement(nullableRawBody(request.body))==expected);respond("",HttpStatusCode.NoContent) }
                else -> { check(codec.parseToJsonElement(nullableRawBody(request.body))==expected);respond(codec.encodeToString(JsonElement.serializer(),expected),HttpStatusCode.OK,headersOf("Content-Type","application/json")) }
            }
        } })
        val http=HttpClient(engine) { install(SSE);install(ContentNegotiation) { json(codec) } }
        try {
            val api=IntrinsicNullableRawApiClient(http,"http://example.test/",codec).default
            check(api.echo(null)==null)
            for(value in kinds) {expected=value;check(api.echo(value)==if(value===JsonNull) null else value);check(api.echo2(value)==value)}
            for(p in listOf(absent,clear,value)) {expected=codec.encodeToJsonElement(p);api.submit(p)}
            val before=calls;val flow=api.watch();check(calls==before)
            val events=withTimeout(3000) { flow.toList() };check(events.size==kinds.size)
            events.forEachIndexed { i,event -> check((event as WatchEventDto.Raw).data==if(kinds[i]===JsonNull) null else kinds[i]) }
            val narrow=withTimeout(3000) { api.watchNarrow().toList() };check((narrow.first() as WatchNarrowEventDto.Narrow).data===JsonNull)
            syntax=true
            for(stream in listOf(false,true)) {withTimeout(3000) { nullableRawReject { if(stream) api.watchNarrow().toList() else api.watch().toList() } };withTimeout(3000) { while(!channels.last().isClosedForWrite) yield() } }
            syntax=false;expected=future;check(api.echo2(future)==future)
        } finally {http.close();producers.cancel()}
        println("IntrinsicNullableRaw Json defaults=$defaults nulls=$nulls rawnullable-allkinds-Patch-carriercollapse-HTTP-SSEsyntax-close-reuse-cold checks passed")
    }
}

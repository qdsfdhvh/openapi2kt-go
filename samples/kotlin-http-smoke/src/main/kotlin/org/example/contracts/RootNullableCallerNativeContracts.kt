@file:OptIn(kotlin.time.ExperimentalTime::class, kotlinx.serialization.ExperimentalSerializationApi::class)
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
import kotlin.time.Instant
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
import org.example.rootnullablecallernative.RootNullableCallerNativeApiClient
import org.example.rootnullablecallernative.api.models.SubmitInputDto
import org.example.rootnullablecallernative.api.models.WatchEventDto
import org.example.rootnullablecallernative.api.models.WatchNarrowEventDto
import org.example.rootnullablecallernative.models.AInheritedDto
import org.example.rootnullablecallernative.models.AFalseDto
import org.example.rootnullablecallernative.models.ATrueDto
import org.example.rootnullablecallernative.models.HolderDto
import org.example.rootnullablecallernative.models.Patch
import org.example.scalarvalues.Money
import org.example.rootnullablecallerprobes.models.ANearDto as CallerNearDto
import org.example.rootnullablecallerprobes.models.ANarrowDto as CallerNarrowDto
import org.example.rootnullablecallerprobes.models.AWideDto as CallerWideDto
import org.example.rootnullablecallerprobes.models.LongWideDto as CallerLongWideDto

private fun callerBody(body: OutgoingContent): ByteArray = when(body) {
    is OutgoingContent.ContentWrapper -> callerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes()
    else -> error("Unexpected caller-owned body ${body::class}")
}
private suspend fun callerReject(block: suspend () -> Unit) {
    try { block(); error("Invalid caller-owned wire accepted") }
    catch(cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong caller wire error $cause" } }
}
suspend fun rootNullableCallerNativeContracts() {
    val ready = AInheritedDto("12.50")
    val future = ATrueDto("caller-owned-future")
    check(ready == AFalseDto("12.50"))
    check(!AInheritedDto.serializer().descriptor.isNullable && serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults=defaults;explicitNulls=nulls }
        for(value in listOf(null,ready,future)) {
            val text=codec.encodeToString(serializer<AInheritedDto>(),value)
            check(codec.parseToJsonElement(text)==(value?.let { JsonPrimitive(it.decimal) } ?: JsonNull))
            check(codec.decodeFromString(serializer<AInheritedDto>(),text)==value)
            check(codec.decodeFromString(Money.serializer().nullable,text)==value)
        }
        callerReject { codec.decodeFromString(serializer<AInheritedDto>(),"{}") }
        check(codec.decodeFromString<CallerNearDto>("{\"v\":null}").v==null)
        check(codec.decodeFromString<CallerNarrowDto>("{\"v\":\"12.50\"}").v==ready)
        callerReject { codec.decodeFromString<CallerNarrowDto>("{\"v\":null}") }
        val wide=codec.decodeFromJsonElement(CallerWideDto.serializer(),JsonObject((0 until 240).associate { "p$it" to if(it==239) JsonPrimitive("12.50") else JsonNull }))
        check(wide.p0==null&&wide.p239==ready)
        check(codec.decodeFromString<CallerWideDto>(codec.encodeToString(wide.copy(p239=Money("future"))))==wide.copy(p239=Money("future")))
        val longWide=codec.decodeFromJsonElement(CallerLongWideDto.serializer(),JsonObject((0 until 120).associate { "p$it" to JsonPrimitive(Long.MAX_VALUE) }))
        check(longWide.p119==Long.MAX_VALUE&&codec.decodeFromString<CallerLongWideDto>(codec.encodeToString(longWide.copy(p0=Long.MIN_VALUE)))==longWide.copy(p0=Long.MIN_VALUE))
        val holder=HolderDto(inherited=null,forcedTrue=future,forcedFalse=ready,wrapper=null,wrapperFalse=ready,items=listOf(null,ready),values=mapOf("nil" to null,"ready" to ready),requiredNull=null,nativeLong=Long.MAX_VALUE,nativeBytes=byteArrayOf(-128,-1,0,127),nativeElement=JsonPrimitive("future-element"),nativeObject=JsonObject(mapOf("future" to JsonNull)),nativeString="text",nativeInstant=Instant.parse("2024-02-29T12:34:56Z"))
        val tree=codec.encodeToJsonElement(holder) as JsonObject
        check(tree["nativeLong"]==JsonPrimitive(Long.MAX_VALUE))
        check(tree["nativeBytes"]==JsonArray(listOf(-128,-1,0,127).map { JsonPrimitive(it) }))
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)))==tree)
        // Nullable JsonElement collapses the JsonNull carrier to Kotlin null;
        // explicitNulls=false may then omit the field. This is provider behavior.
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder.copy(nativeElement=JsonNull))).nativeElement==null)
        val absent=SubmitInputDto(items=emptyList(),values=emptyMap(),requiredNull=null)
        check("inherited" !in (codec.encodeToJsonElement(absent) as JsonObject))
        if(nulls) callerReject { codec.decodeFromString<SubmitInputDto>("{\"items\":[],\"values\":{}}") }
        val clear=absent.copy(inherited=Patch.Null,wrapper=Patch.Null,nativeLong=Patch.Null,nativeBytes=Patch.Null,nativeElement=Patch.Null)
        val value=absent.copy(inherited=Patch.Value(future),wrapper=Patch.Value(ready),nativeLong=Patch.Value(Long.MAX_VALUE),nativeBytes=Patch.Value(byteArrayOf(-1)),nativeElement=Patch.Value(JsonNull))
        for(p in listOf(absent,clear,value)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(p)))==codec.encodeToJsonElement(p))
        check((codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value))).nativeElement===Patch.Null)
        var expected="null".encodeToByteArray()
        var response: ByteArray?=null
        var failure=""
        var calls=0
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=StreamingTestEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when(request.url.encodedPath) {
                "/watch","/watch-narrow" -> {
                    val narrow=request.url.encodedPath=="/watch-narrow"
                    val channel=ByteChannel(autoFlush=true);channels+=channel;val broken=failure
                    producers.launch { try {
                        val name=if(narrow) "narrow" else "money"
                        val data=when(broken) {"kind"->"{}";"syntax"->"{";"null"->"null";else->"\"12.50\""}
                        val frame=if(broken.isEmpty()&&!narrow) "event: money\r\ndata: null\r\n\r\nevent: money\r\ndata: $data\r\n\r\n" else "event: $name\r\ndata: $data\r\n\r\n"
                        for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if(broken.isNotEmpty()) awaitCancellation()
                    } finally { channel.close() } }
                    respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                }
                "/optional","/submit" -> { check(callerBody(request.body).contentEquals(expected));respond("",HttpStatusCode.NoContent) }
                else -> { check(callerBody(request.body).contentEquals(expected));respond(response?:expected,HttpStatusCode.OK,headersOf("Content-Type","application/json")) }
            }
        } })
        val http=HttpClient(engine) { install(SSE);install(ContentNegotiation) { json(codec) } }
        try {
            val api=RootNullableCallerNativeApiClient(http,"http://example.test/",codec).default
            check(api.echo(null)==null);api.optional()
            expected="\"12.50\"".encodeToByteArray();check(api.echo(ready)==ready&&api.echo2(ready)==ready);api.optional(ready)
            expected="null".encodeToByteArray();check(api.echoLong(null)==null)
            expected=Long.MAX_VALUE.toString().encodeToByteArray();check(api.echoLong(Long.MAX_VALUE)==Long.MAX_VALUE)
            expected="null".encodeToByteArray();check(api.echoBytes(null)?.contentEquals(expected)==true&&api.echoText(null)=="null")
            expected=byteArrayOf(-128,-1,0,127);check(api.echoBytes(expected)?.contentEquals(expected)==true)
            expected="raw-text".encodeToByteArray();check(api.echoText("raw-text")=="raw-text")
            expected="null".encodeToByteArray();check(api.echoElement(null)==null&&api.echoObject(null)==null&&api.echoInstant(null)==null)
            val raw: JsonElement=JsonArray(listOf(JsonNull,JsonPrimitive(1)))
            expected=codec.encodeToString(JsonElement.serializer(),raw).encodeToByteArray();check(api.echoElement(raw)==raw)
            val obj=JsonObject(mapOf("future" to JsonNull));expected=codec.encodeToString(JsonObject.serializer(),obj).encodeToByteArray();check(api.echoObject(obj)==obj)
            val stamp=Instant.parse("2024-02-29T12:34:56Z");expected="\"2024-02-29T12:34:56Z\"".encodeToByteArray();check(api.echoInstant(stamp)==stamp)
            for(p in listOf(absent,clear,value)) { expected=codec.encodeToString(p).encodeToByteArray();api.submit(p) }
            expected="\"12.50\"".encodeToByteArray();response="{}".encodeToByteArray();callerReject { api.echo(ready) };response=null
            val before=calls;val flow=api.watch();check(calls==before)
            val events=withTimeout(3000) { flow.toList() };check(events.size==2&&(events[0] as WatchEventDto.Money).data==null&&(events[1] as WatchEventDto.Money).data==ready)
            check((withTimeout(3000) { api.watchNarrow().toList() }).single().let { it as WatchNarrowEventDto.Narrow }.data==ready)
            for(bad in listOf("kind","syntax","null")) { failure=bad;withTimeout(3000) { callerReject { api.watchNarrow().toList() } };withTimeout(3000) { while(!channels.last().isClosedForWrite) yield() } }
            failure="";check(api.echo2(ready)==ready)
        } finally { http.close();producers.cancel() }
        println("RootNullableCallerNative Json defaults=$defaults nulls=$nulls providerowned-codecs-Patch-nativeprofiles-rawStringBytes-SSEclose-reuse-cold checks passed")
    }
}

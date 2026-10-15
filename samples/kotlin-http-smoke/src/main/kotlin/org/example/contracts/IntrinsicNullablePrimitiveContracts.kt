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
import kotlinx.serialization.builtins.serializer
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
import org.example.intrinsicprimitives.IntrinsicPrimitiveApiClient
import org.example.intrinsicprimitives.api.models.SubmitInputDto
import org.example.intrinsicprimitives.api.models.WatchEventDto
import org.example.intrinsicprimitives.models.AFalseDto
import org.example.intrinsicprimitives.models.AInheritedDto
import org.example.intrinsicprimitives.models.ATrueDto
import org.example.intrinsicprimitives.models.CFalseInheritedDto
import org.example.intrinsicprimitives.models.HolderDto
import org.example.intrinsicprimitives.models.Patch
import org.example.intrinsicprimitives.models.ZBooleanDto
import org.example.intrinsicprimitives.models.ZDoubleDto
import org.example.intrinsicprimitives.models.ZFloatDto
import org.example.intrinsicprimitives.models.ZIntDto
import org.example.intrinsicprimitives.models.ZLongDto
import org.example.intrinsicprimitives.models.ZStringDto
import org.example.intrinsicprimitiveprobes.ApiClient as BoundClient
import org.example.intrinsicprimitiveprobes.api.models.PatchInputDto
import org.example.intrinsicprimitiveprobes.models.BoundHolderDto
import org.example.intrinsicprimitiveprobes.models.CheckedDto
import org.example.intrinsicprimitiveprobes.models.NearDto
import org.example.intrinsicprimitiveprobes.models.NarrowDto
import org.example.intrinsicprimitiveprobes.models.WideDto
import org.example.intrinsicprimitiveprobes.models.ZBoundLongDto
import org.example.intrinsicprimitiveprobes.models.ZBoundStringDto
import org.example.intrinsicprimitiveprobes.models.Patch as BoundPatch

private fun primitiveBody(body: OutgoingContent): String = when(body) {
    is OutgoingContent.ContentWrapper -> primitiveBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected primitive body ${body::class}")
}
private suspend fun primitiveReject(block: suspend () -> Unit) {
    try { block();error("Invalid primitive accepted") }
    catch(cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException || it is IllegalArgumentException }) { "Wrong primitive failure $cause" } }
}
suspend fun intrinsicNullablePrimitiveContracts() {
    check(serializer<ZStringDto>().descriptor.isNullable && serializer<ZBooleanDto>().descriptor.isNullable)
    check(serializer<ZIntDto>().descriptor.isNullable && serializer<ZLongDto>().descriptor.isNullable)
    check(serializer<ZFloatDto>().descriptor.isNullable && serializer<ZDoubleDto>().descriptor.isNullable)
    check(serializer<AInheritedDto>().descriptor.isNullable && serializer<ATrueDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable && !serializer<CFalseInheritedDto>().descriptor.isNullable)
    check(!Long.serializer().descriptor.isNullable && Long.serializer().nullable.descriptor.isNullable)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        check(codec.decodeFromString<ZStringDto>(codec.encodeToString<ZStringDto>("future"))=="future")
        check(codec.decodeFromString<ZStringDto>("null")==null)
        check(codec.decodeFromString<ZBooleanDto>(codec.encodeToString<ZBooleanDto>(true))==true)
        check(codec.decodeFromString<ZBooleanDto>("null")==null)
        for(v in listOf(Int.MIN_VALUE,Int.MAX_VALUE)) check(codec.decodeFromString<ZIntDto>(codec.encodeToString<ZIntDto>(v))==v)
        check(codec.decodeFromString<ZIntDto>("null")==null)
        for(v in listOf(Long.MIN_VALUE,Long.MAX_VALUE)) check(codec.decodeFromString<ZLongDto>(codec.encodeToString<ZLongDto>(v))==v)
        check(codec.decodeFromString<ZLongDto>("null")==null)
        for(v in listOf(-0.0f,1.25f)) check(codec.decodeFromString<ZFloatDto>(codec.encodeToString<ZFloatDto>(v))!!.toBits()==v.toBits())
        check(codec.decodeFromString<ZFloatDto>("null")==null)
        for(v in listOf(-0.0,1.25)) check(codec.decodeFromString<ZDoubleDto>(codec.encodeToString<ZDoubleDto>(v))!!.toBits()==v.toBits())
        check(codec.decodeFromString<ZDoubleDto>("null")==null)
        check(codec.decodeFromString<AInheritedDto>("null")==null)
        primitiveReject { codec.decodeFromString<AFalseDto>("null") }
        primitiveReject { codec.decodeFromString<ZIntDto>("2147483648") }
        primitiveReject { codec.decodeFromString<ZLongDto>("9223372036854775808") }
        primitiveReject { codec.decodeFromString<ZBooleanDto>("{}") }
        val holder=HolderDto(string="future",boolean=true,int=Int.MAX_VALUE,long=Long.MIN_VALUE,float=1.25f,double=1.25,narrow="ok",narrowInherited="ok",list=listOf(null,Long.MAX_VALUE),map=mapOf("nil" to null,"long" to Long.MIN_VALUE))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder.copy(inherited="x")))==holder.copy(inherited="x"))
        check(codec.decodeFromString<NearDto>("{\"v\":null}").v==null)
        val wide=codec.decodeFromJsonElement(WideDto.serializer(),JsonObject((0 until 240).associate { "p$it" to if(it==239) JsonPrimitive(Long.MAX_VALUE) else JsonNull }))
        check(wide.p0==null && wide.p239==Long.MAX_VALUE && codec.decodeFromString<WideDto>(codec.encodeToString(wide.copy(p0=1L)))==wide.copy(p0=1L))
        val narrow=codec.decodeFromJsonElement(NarrowDto.serializer(),JsonObject((0 until 120).associate { "p$it" to JsonPrimitive(Long.MIN_VALUE) }))
        check(narrow.p119==Long.MIN_VALUE && codec.decodeFromString<NarrowDto>(codec.encodeToString(narrow.copy(p0=Long.MAX_VALUE)))==narrow.copy(p0=Long.MAX_VALUE))
        val bound=BoundHolderDto(s=null,l=null,narrow=2,items=listOf(null,"ok"),values=mapOf("null" to null,"valid" to 3L))
        check(codec.decodeFromString<BoundHolderDto>(codec.encodeToString(bound))==bound)
        primitiveReject { bound.copy(s="x") };primitiveReject { bound.copy(l=4L) };primitiveReject { bound.copy(items=listOf("x")) };primitiveReject { bound.copy(values=mapOf("bad" to 4L)) }
        primitiveReject { codec.decodeFromString<BoundHolderDto>("{\"s\":\"x\",\"l\":null,\"narrow\":2,\"items\":[],\"values\":{}}") }
        // Transparent aliases are NOT standalone validation barriers.
        check(codec.decodeFromString<ZBoundStringDto>("\"x\"")=="x" && codec.decodeFromString<ZBoundLongDto>("4")==4L)
        var deep: JsonElement=JsonPrimitive("ok")
        repeat(16) { deep=JsonArray(listOf(deep)) }
        val checked=codec.decodeFromJsonElement(CheckedDto.serializer(),JsonObject(mapOf("v" to deep)))
        check(codec.decodeFromString<CheckedDto>(codec.encodeToString(checked))==checked)
        val absent=SubmitInputDto(requiredNull=null,requiredFalse=Long.MAX_VALUE)
        check("long" !in (codec.encodeToJsonElement(absent) as JsonObject))
        if(nulls) primitiveReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":1}") }
        val clear=absent.copy(string=Patch.Null,boolean=Patch.Null,int=Patch.Null,long=Patch.Null,float=Patch.Null,double=Patch.Null)
        val value=absent.copy(string=Patch.Value("text"),boolean=Patch.Value(false),int=Patch.Value(Int.MIN_VALUE),long=Patch.Value(Long.MIN_VALUE),float=Patch.Value(1.25f),double=Patch.Value(1.25))
        for(p in listOf(absent,clear,value)) check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(p))==p)
        val boundAbsent=PatchInputDto(required=null)
        primitiveReject { boundAbsent.copy(s=BoundPatch.Value("x")) };primitiveReject { boundAbsent.copy(l=BoundPatch.Value(4L)) }
        var expected="null";var expectedJson: JsonElement=JsonNull;var bad="";var calls=0
        val channels=CopyOnWriteArrayList<ByteChannel>();val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=StreamingTestEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when(request.url.encodedPath) {
                "/events","/bound-events" -> {
                    val channel=ByteChannel(autoFlush=true);channels+=channel;val failure=bad;val bounded=request.url.encodedPath=="/bound-events"
                    producers.launch { try {
                        val frame=when(failure) {
                            "narrow" -> "event: narrowLong\r\ndata: null\r\n\r\n"
                            "range" -> "event: long\r\ndata: 9223372036854775808\r\n\r\n"
                            "bounds" -> "event: bounded\r\ndata: 4\r\n\r\n"
                            "syntax" -> "event: long\r\ndata: {\r\n\r\n"
                            else -> if(bounded) "event: bounded\r\ndata: null\r\n\r\nevent: bounded\r\ndata: 2\r\n\r\n" else "event: string\r\ndata: null\r\n\r\nevent: string\r\ndata: \"future\"\r\n\r\nevent: boolean\r\ndata: true\r\n\r\nevent: int\r\ndata: 2147483647\r\n\r\nevent: long\r\ndata: -9223372036854775808\r\n\r\nevent: float\r\ndata: 1.25\r\n\r\nevent: double\r\ndata: 1.25\r\n\r\nevent: narrowString\r\ndata: \"ok\"\r\n\r\nevent: narrowLong\r\ndata: 2\r\n\r\n"
                        }
                        for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if(failure.isNotEmpty()) awaitCancellation()
                    } finally { channel.close() } }
                    respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                }
                "/query" -> { check(request.url.parameters["q"]==if(expected=="null") null else expected);respond("",HttpStatusCode.NoContent) }
                "/submit","/patch" -> { check(codec.parseToJsonElement(primitiveBody(request.body))==expectedJson);respond("",HttpStatusCode.NoContent) }
                else -> {
                    check(primitiveBody(request.body)==expected) { "Wrong primitive body $expected: ${primitiveBody(request.body)}" }
                    val response=when(bad) { "range" -> "9223372036854775808";"bounds" -> "4";"syntax" -> "{";else -> expected }
                    respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                }
            }
        } })
        val http=HttpClient(engine) { install(SSE);install(ContentNegotiation) { json(codec) } }
        try {
            val api=IntrinsicPrimitiveApiClient(http,"http://example.test/",codec).default
            expected="null";check(api.echoString(null)=="null" && api.echoString2(null)=="null")
            expected="text";check(api.echoString("text")=="text" && api.echoString2("text")=="text")
            expected="null";check(api.echoBoolean(null)==null && api.echoInt(null)==null && api.echoLong(null)==null && api.echoFloat(null)==null && api.echoDouble(null)==null)
            expected="true";check(api.echoBoolean(true)==true)
            expected=Int.MIN_VALUE.toString();check(api.echoInt(Int.MIN_VALUE)==Int.MIN_VALUE)
            expected=Long.MAX_VALUE.toString();check(api.echoLong(Long.MAX_VALUE)==Long.MAX_VALUE)
            expected="1.25";check(api.echoFloat(1.25f)==1.25f && api.echoDouble(1.25)==1.25)
            expected="null";api.query(null);expected="2";api.query(2L)
            for(p in listOf(absent,clear,value)) { expectedJson=codec.encodeToJsonElement(p);api.submit(p) }
            val bounded=BoundClient(http,"http://example.test/",codec).default
            expected="ok";check(bounded.boundString("ok")=="ok")
            expected="2";check(bounded.boundLong(2L)==2L)
            val before=calls;primitiveReject { bounded.boundLong(4L) };primitiveReject { bounded.boundString("x") };check(calls==before)
            for(p in listOf(boundAbsent,boundAbsent.copy(s=BoundPatch.Null,l=BoundPatch.Null),boundAbsent.copy(s=BoundPatch.Value("ok"),l=BoundPatch.Value(2L)))) { expectedJson=codec.encodeToJsonElement(p);bounded.patchPatch(p) }
            expected="2";bad="bounds";primitiveReject { bounded.boundLong(2L) };bad="range";primitiveReject { api.echoLong(2L) };bad="syntax";primitiveReject { api.echoLong(2L) };bad=""
            val cold=calls;val flow=api.watch();check(calls==cold)
            val events=withTimeout(3000) { flow.toList() }
            check(events.size==9 && (events[0] as WatchEventDto.String).data==null && (events[1] as WatchEventDto.String).data=="future")
            check((events[2] as WatchEventDto.Boolean).data==true && (events[3] as WatchEventDto.Int).data==Int.MAX_VALUE)
            check((events[4] as WatchEventDto.Long).data==Long.MIN_VALUE && (events[5] as WatchEventDto.Float).data==1.25f && (events[6] as WatchEventDto.Double).data==1.25)
            check((events[7] as WatchEventDto.NarrowString).data=="ok" && (events[8] as WatchEventDto.NarrowLong).data==2L)
            check(withTimeout(3000) { bounded.watchBounded().toList() }.size==2)
            for(failure in listOf("narrow","range","syntax","bounds")) { bad=failure;withTimeout(3000) { primitiveReject { if(failure=="bounds") bounded.watchBounded().toList() else api.watch().toList() } };withTimeout(3000) { while(!channels.last().isClosedForWrite) yield() } }
            bad="";expected="2";check(api.echoLong(2L)==2L)
        } finally { http.close();producers.cancel() }
        println("IntrinsicNullablePrimitive Json defaults=$defaults nulls=$nulls sixcodecs-nativewire-bounds-Patch-HTTP-SSEclose-reuse-cold checks passed")
    }
}

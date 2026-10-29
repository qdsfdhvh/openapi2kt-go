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
import kotlin.time.Instant
import kotlinx.datetime.LocalDate
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
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonNull
import org.example.arraydefaults.ArrayDefaults
import org.example.arraydefaults.models.HolderDto
import org.example.arraydefaults.models.RequiredTemporalDto
import org.example.arraydefaults.models.Patch
import org.example.arraydefaults.api.models.PatchInputDto
import org.example.arraydefaults.api.models.WatchEventDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ArrayDefaultsEngine(config:MockEngineConfig):MockEngine(config) {
    override val supportedCapabilities=super.supportedCapabilities+SSECapability
    override suspend fun execute(data:HttpRequestData):HttpResponseData {
        val response=super.execute(data)
        val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey)?:return response
        val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext)?:return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun arrayDefaultsReject(block:()->Unit) {
    try { block();error("Expected array default typed failure") } catch(_:SerializationException) { }
}
private suspend fun arrayDefaultsWireReject(block:suspend ()->Unit) {
    try { block();error("Expected array default wire failure") } catch(cause:Throwable) {
        check(generateSequence(cause){it.cause}.any {it is SerializationException}) {"Wrong array default failure $cause"}
    }
}
private fun arrayDefaultsBody(body:OutgoingContent):String=when(body) {
    is OutgoingContent.ContentWrapper -> arrayDefaultsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected array body $body")
}

private suspend fun temporalArraysReject(block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: Throwable) { cause }
    check(failure != null && generateSequence(failure) { it.cause }.any {
        it is SerializationException || it is IllegalArgumentException
    }) { "Expected temporal codec failure, got $failure" }
}

suspend fun primitiveArrayDefaultsContracts() {
    val value=HolderDto(required=listOf("supplied"),kotlin="shadow",kotlinx="shadow")
    check(value.names==listOf("quote \" and \$value","line\nnext"))
    check(value.longs==listOf(Long.MIN_VALUE,Long.MAX_VALUE))
    check(value.floats==listOf(1.25f,null) && value.flags==null && value.empty==emptyList<Int>())
    check(value.childOnly == null)
    val day = LocalDate.parse("2024-02-29")
    val instant = Instant.parse("2024-02-29T10:04:56.123456789Z")
    check(value.dates == listOf(day) && value.times == listOf(instant, null))
    check(value.emptyDates == emptyList<LocalDate>() && value.nullDates == null)
    check(value.timeEdges == listOf(
        Instant.parse("2024-02-29T12:34:56.1Z"),
        Instant.parse("2024-02-28T18:34:56Z"),
        Instant.parse("2024-03-01T06:34:56Z"),
        Instant.parse("2024-02-29T12:34:56Z"),
        Instant.parse("0000-01-01T00:00:00Z")
    ))
    check(value.dates !== HolderDto(required = emptyList()).dates)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        val missing=codec.decodeFromString<HolderDto>("{\"required\":[]}")
        check(missing.names==value.names && missing.longs==value.longs && missing.floats==value.floats && missing.empty==value.empty)
        check(missing.childOnly == null)
        check(codec.decodeFromString<HolderDto>("""{"required":[],"childOnly":[4]}""").childOnly == listOf(4))
        check(missing.dates == value.dates && missing.times == value.times && missing.emptyDates == value.emptyDates && missing.timeEdges == value.timeEdges)
        check(codec.decodeFromString<HolderDto>("""{"required":[],"dates":null,"times":null}""").dates == null)
        val supplied = codec.decodeFromString<HolderDto>("""{"required":[],"dates":["2024-03-01"],"times":[null]}""")
        check(supplied.dates == listOf(LocalDate.parse("2024-03-01")) && supplied.times == listOf<Instant?>(null))
        for (bad in listOf(
            """{"required":[],"dates":["2023-02-29"]}""",
            """{"required":[],"dates":[null]}""",
            """{"required":[],"times":["invalid"]}"""
        )) temporalArraysReject { codec.decodeFromString<HolderDto>(bad) }
        temporalArraysReject { codec.decodeFromString<RequiredTemporalDto>("{}") }
        temporalArraysReject { codec.decodeFromString<RequiredTemporalDto>("""{"dates":null,"times":null}""") }
        if (nulls) temporalArraysReject { codec.decodeFromString<RequiredTemporalDto>("""{"dates":[]}""") }
        else check(codec.decodeFromString<RequiredTemporalDto>("""{"dates":[]}""").times == null)
        val required = RequiredTemporalDto(dates = listOf(day), times = null)
        check(codec.decodeFromString<RequiredTemporalDto>(codec.encodeToString(required)) == required)
        arrayDefaultsReject {codec.decodeFromString<HolderDto>("{}")}
        arrayDefaultsReject {codec.decodeFromString<HolderDto>("{\"required\":null}")}
        arrayDefaultsReject {codec.decodeFromString<HolderDto>("{\"required\":[],\"names\":[null]}")}
        arrayDefaultsReject {codec.decodeFromString<HolderDto>("{\"required\":[],\"longs\":[true]}")}
        check(codec.decodeFromString<HolderDto>("{\"required\":[],\"names\":null}").names==null)
        check(codec.decodeFromString<HolderDto>("{\"required\":[],\"names\":[\"different\"]}").names==listOf("different"))
        val wire=codec.encodeToJsonElement(HolderDto.serializer(),value) as JsonObject
        check(wire.containsKey("names")==defaults && wire.containsKey("longs")==defaults)
        check(wire.containsKey("dates") == defaults && wire.containsKey("times") == defaults)
        if (defaults) {
            check(wire["dates"] == codec.parseToJsonElement("""["2024-02-29"]"""))
            check(wire["times"] == codec.parseToJsonElement("""["2024-02-29T10:04:56.123456789Z",null]"""))
        }
        val temporalRoundtrip = codec.decodeFromJsonElement(HolderDto.serializer(), wire)
        check(temporalRoundtrip.dates == value.dates && temporalRoundtrip.times == value.times && temporalRoundtrip.timeEdges == value.timeEdges)
        check(codec.decodeFromJsonElement(HolderDto.serializer(),wire).longs==value.longs)
        val patches=listOf(PatchInputDto(required=null),PatchInputDto(required=null,values=Patch.Null,dates=Patch.Null),PatchInputDto(required=listOf("value"),values=Patch.Value(listOf(3)),dates=Patch.Value(listOf(day,null))))
        for(patch in patches) {
            val json=codec.encodeToJsonElement(PatchInputDto.serializer(),patch) as JsonObject
            check(when(patch.values) {Patch.Absent -> !json.containsKey("values");Patch.Null -> json["values"]===JsonNull;is Patch.Value -> json["values"]==codec.parseToJsonElement("[3]")})
            check(codec.decodeFromJsonElement(PatchInputDto.serializer(),json).values==patch.values)
            check(when (patch.dates) {
                Patch.Absent -> !json.containsKey("dates")
                Patch.Null -> json["dates"] === JsonNull
                is Patch.Value -> json["dates"] == codec.parseToJsonElement("""["2024-02-29",null]""")
            })
            check(codec.decodeFromJsonElement(PatchInputDto.serializer(), json).dates == patch.dates)
        }
        if(nulls) arrayDefaultsReject {codec.decodeFromString<PatchInputDto>("{}")}
        else check(codec.decodeFromString<PatchInputDto>("{}").required==null)
        primitiveArrayDefaultsWire(codec,value,patches)
        println("PrimitiveArrayDefaults Json defaults=$defaults nulls=$nulls typed-lists-required-null-Patch-shadow-HTTP-liveSSE passed")
    }
}

private suspend fun primitiveArrayDefaultsWire(codec:Json,value:HolderDto,patches:List<PatchInputDto>) {
    val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
    val channels=CopyOnWriteArrayList<ByteChannel>();var bad=false;var patchBody="";var calls=0
    var temporalBad: String? = null
    val engine=ArrayDefaultsEngine(MockEngineConfig().apply {addHandler {request ->
        calls++
        when(request.url.encodedPath) {
            "/holder" -> {check(arrayDefaultsBody(request.body)==codec.encodeToString(value));respond(temporalBad ?: if(bad) "{\"required\":[],\"names\":[null]}" else "{\"required\":[\"wire\"]}",HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
            "/patch" -> {check(arrayDefaultsBody(request.body)==patchBody);respond("",HttpStatusCode.NoContent)}
            "/events" -> {
                val channel=ByteChannel(autoFlush=true);channels+=channel;val invalid=bad || temporalBad != null
                val temporalPayload = temporalBad
                producers.launch {try {
                    val payload=temporalPayload ?: if(invalid) "{\"required\":[],\"longs\":[true]}" else "{\"required\":[]}"
                    val frame="event: holder\r\ndata: $payload\r\n\r\n"
                    for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if(invalid) awaitCancellation()
                } finally {channel.close()} }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
            else -> error("Unexpected array request")
        }
    }})
    val http=HttpClient(engine) {install(SSE);install(ContentNegotiation){json(codec)}}
    try {
        val api=ArrayDefaults(http,"http://example.test/",codec).defaults
        check(api.send(value).names==value.names)
        for(patch in patches){patchBody=codec.encodeToString(patch);api.patchPatch(patch)}
        val before=calls;val flow=api.watch();check(calls==before)
        repeat(2){val event=withTimeout(10000){flow.toList()}.single() as WatchEventDto.Holder;check(event.data.longs==value.longs && event.data.floats==value.floats && event.data.dates==value.dates && event.data.times==value.times)}
        bad=true;arrayDefaultsWireReject {api.send(value)};val count=channels.size
        arrayDefaultsWireReject {withTimeout(10000){api.watch().toList()}}
        check(channels.size==count+1 && channels.last().isClosedForRead)
        bad=false;check(api.send(value).longs==value.longs)
        for (payload in listOf(
            """{"required":[],"dates":["2023-02-29"]}""",
            """{"required":[],"times":["invalid"]}"""
        )) {
            temporalBad = payload
            temporalArraysReject { api.send(value) }
            temporalArraysReject { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            temporalBad = null
            val recovered = api.send(value)
            check(recovered.dates == value.dates && recovered.times == value.times)
        }
    } finally {http.close();producers.cancel();channels.forEach{it.close()}}
}

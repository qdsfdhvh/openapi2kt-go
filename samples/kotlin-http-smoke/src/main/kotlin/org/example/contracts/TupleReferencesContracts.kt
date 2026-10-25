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
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.datetime.LocalDate
import org.example.tuplereferences.PrimitiveTupleApiClient
import org.example.tuplereferences.api.models.PatchInputDto
import org.example.tuplereferences.api.models.WatchEventDto
import org.example.tuplereferences.models.HolderDto
import org.example.tuplereferences.models.StringTupleDto
import org.example.tuplereferences.models.LongTupleDto
import org.example.tuplereferences.models.MixedTupleDto
import org.example.tuplereferences.models.PriorityTupleDto
import org.example.tuplereferences.models.DateTupleDto
import org.example.tuplereferences.models.InstantTupleDto
import org.example.tuplereferences.models.MixedTemporalTupleDto
import org.example.tuplereferences.models.PriorityTemporalTupleDto
import org.example.tuplereferences.models.MaybeDatesDto
import org.example.tuplereferences.models.Patch
import java.util.concurrent.CopyOnWriteArrayList
import kotlin.time.Instant

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RefTupleStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun tupleReferenceBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> tupleReferenceBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic tuple request body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun tupleReferencesContracts() = withTimeout(15000) {
    val strings: StringTupleDto = listOf("one","two","tail")
    val longs: LongTupleDto = listOf(Long.MIN_VALUE,Long.MAX_VALUE)
    val mixed: MixedTupleDto = listOf(JsonPrimitive("one"),JsonPrimitive(true),JsonNull)
    val priority: PriorityTupleDto = listOf("items-wins")
    val dates: DateTupleDto = listOf(LocalDate.parse("2024-02-29"),LocalDate.parse("2026-10-21"))
    val moments: InstantTupleDto = listOf(Instant.parse("2026-01-02T03:04:05.123Z"),null)
    val mixedTime: MixedTemporalTupleDto = listOf(JsonPrimitive("not-a-date"),JsonPrimitive(true),JsonNull)
    val datePriority: PriorityTemporalTupleDto = dates
    val value=HolderDto(strings,longs,mixed,priority,listOf("v",null),mapOf("a" to listOf(true,false)),dates,moments,mixedTime,datePriority,mapOf("dates" to dates))
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        check(codec.decodeFromString<StringTupleDto>(codec.encodeToString(strings))==strings)
        check(codec.decodeFromString<LongTupleDto>(codec.encodeToString(longs))==longs)
        check(codec.decodeFromString<MixedTupleDto>(codec.encodeToString(mixed))==mixed)
        check(codec.decodeFromString<PriorityTupleDto>("[\"not-an-integer\"]")==listOf("not-an-integer"))
        check(codec.decodeFromString<DateTupleDto>(codec.encodeToString(dates))==dates)
        check(codec.decodeFromString<InstantTupleDto>(codec.encodeToString(moments))==moments)
        check(codec.decodeFromString<MixedTemporalTupleDto>(codec.encodeToString(mixedTime))==mixedTime) // no positional format validation
        check(codec.decodeFromString<PriorityTemporalTupleDto>(codec.encodeToString(datePriority))==dates)
        check(codec.decodeFromString<MaybeDatesDto>("null")==null)
        check(codec.decodeFromString<DateTupleDto>("[]").isEmpty()) // no positional length enforcement
        check(codec.decodeFromString<InstantTupleDto>("[\"2026-01-02T05:04:05.123+02:00\",null]")==moments)
        check(runCatching { codec.decodeFromString<DateTupleDto>("[\"2026-02-30\"]") }.isFailure)
        check(runCatching { codec.decodeFromString<InstantTupleDto>("[\"not-an-instant\"]") }.isFailure)
        check(runCatching { codec.decodeFromString<DateTupleDto>("[\"2024-02-29\",\"2026-10-21\",true]") }.isFailure) // typed tail narrows unrestricted values
        check(codec.decodeFromString<StringTupleDto>("[]").isEmpty()) // no positional length enforcement
        check(codec.decodeFromString<MixedTupleDto>("[{},[],null,7]").size==4) // no positional types
        check(runCatching { codec.decodeFromString<StringTupleDto>("[\"a\",\"b\",true]") }.isFailure) // typed tail narrowing
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value))==value)
        val raw=codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(raw["longs"]==codec.parseToJsonElement("[${Long.MIN_VALUE},${Long.MAX_VALUE}]"))
        check(raw["dates"]==codec.parseToJsonElement("[\"2024-02-29\",\"2026-10-21\"]"))
        check(raw["moments"]==codec.parseToJsonElement("[\"2026-01-02T03:04:05.123Z\",null]"))
        val badDates=JsonObject(raw+("dates" to codec.parseToJsonElement("[\"2026-02-30\"]"))).toString()
        val badInstants=JsonObject(raw+("moments" to codec.parseToJsonElement("[\"not-an-instant\"]"))).toString()
        check(codec.decodeFromString<HolderDto>("{\"strings\":[],\"longs\":[],\"mixed\":[],\"priority\":[],\"nullable\":null,\"mapping\":{}}").nullable==null)
        check(runCatching { codec.decodeFromString<HolderDto>("{\"strings\":null,\"longs\":[],\"mixed\":[],\"priority\":[],\"mapping\":{}}") }.isFailure)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var patchMode=0
        var bad=false
        var badTemporal: String?=null
        val engine=RefTupleStreamingEngine(MockEngineConfig().apply {
            addHandler { req ->
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(tupleReferenceBody(req.body))==value)
                        respond(badTemporal ?: codec.encodeToString(value),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/simple" -> {
                        check(codec.decodeFromString<List<Double>>(tupleReferenceBody(req.body))==listOf(1.5,2.0))
                        respond("[1.5,2.0]",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/dates" -> {
                        check(codec.decodeFromString<DateTupleDto>(tupleReferenceBody(req.body))==dates)
                        respond(codec.encodeToString(dates),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body=codec.parseToJsonElement(tupleReferenceBody(req.body)).jsonObject
                        when(patchMode) {
                            0 -> check("values" !in body && "dates" !in body)
                            1 -> check(body["values"]==JsonNull && body["dates"]==JsonNull)
                            2 -> check(body["values"]==codec.parseToJsonElement("[1,2]") && body["dates"]==raw["dates"])
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val data=badTemporal ?: if(bad) "{\"strings\":null}" else codec.encodeToString(value)
                        val frame="event: sample\r\ndata: $data\r\n\r\n"
                        writers.add(scope.launch {for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic tuple URL")
                }
            }
        })
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) };install(SSE) }
        try {
            val api=PrimitiveTupleApiClient(http,"http://example.test/",codec).default
            check(api.echo(value)==value)
            check(api.dates(dates)==dates)
            for (invalid in listOf(badDates,badInstants)) {
                badTemporal=invalid
                check(runCatching {api.echo(value)}.isFailure)
            }
            badTemporal=null
            check(api.echo(value)==value)
            check(api.simple(listOf(1.5,2.0))==listOf(1.5,2.0))
            api.patchPatch(PatchInputDto(requiredNullable=null))
            patchMode=1;api.patchPatch(PatchInputDto(values=Patch.Null,requiredNullable=null,dates=Patch.Null))
            patchMode=2;api.patchPatch(PatchInputDto(values=Patch.Value(listOf(1,2)),requiredNullable=listOf("ready"),dates=Patch.Value(dates)))
            val flow=api.watch();check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Sample).data==value)
            check(channels.last().isClosedForRead)
            for (invalid in listOf(badDates,badInstants)) {
                badTemporal=invalid
                check(runCatching {flow.take(1).toList()}.isFailure)
                check(channels.last().isClosedForRead)
            }
            badTemporal=null
            bad=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1 && api.echo(value)==value)
            println("TupleReferences Json defaults=$defaults nulls=$nulls builtin-temporal-aliases-ctor-serde-approximation-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally {http.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

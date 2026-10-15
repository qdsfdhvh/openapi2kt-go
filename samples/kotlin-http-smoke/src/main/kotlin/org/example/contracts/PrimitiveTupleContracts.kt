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
import kotlinx.serialization.json.jsonObject
import org.example.tuples.PrimitiveTupleApiClient
import org.example.tuples.api.models.PatchInputDto
import org.example.tuples.api.models.WatchEventDto
import org.example.tuples.models.HolderDto
import org.example.tuples.models.StringTupleDto
import org.example.tuples.models.LongTupleDto
import org.example.tuples.models.MixedTupleDto
import org.example.tuples.models.PriorityTupleDto
import org.example.tuples.models.Patch
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class TupleStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun tupleBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> tupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic tuple request body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun primitiveTupleContracts() = withTimeout(15000) {
    val strings: StringTupleDto = listOf("one","two","tail")
    val longs: LongTupleDto = listOf(Long.MIN_VALUE,Long.MAX_VALUE)
    val mixed: MixedTupleDto = listOf(JsonPrimitive("one"),JsonPrimitive(true),JsonNull)
    val priority: PriorityTupleDto = listOf("items-wins")
    val value=HolderDto(strings,longs,mixed,priority,listOf("v",null),mapOf("a" to listOf(true,false)))
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        check(codec.decodeFromString<StringTupleDto>(codec.encodeToString(strings))==strings)
        check(codec.decodeFromString<LongTupleDto>(codec.encodeToString(longs))==longs)
        check(codec.decodeFromString<MixedTupleDto>(codec.encodeToString(mixed))==mixed)
        check(codec.decodeFromString<PriorityTupleDto>("[\"not-an-integer\"]")==listOf("not-an-integer"))
        check(codec.decodeFromString<StringTupleDto>("[]").isEmpty()) // no positional length enforcement
        check(codec.decodeFromString<MixedTupleDto>("[{},[],null,7]").size==4) // no positional types
        check(runCatching { codec.decodeFromString<StringTupleDto>("[\"a\",\"b\",true]") }.isFailure) // typed tail narrowing
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value))==value)
        val raw=codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(raw["longs"]==codec.parseToJsonElement("[${Long.MIN_VALUE},${Long.MAX_VALUE}]"))
        check(codec.decodeFromString<HolderDto>("{\"strings\":[],\"longs\":[],\"mixed\":[],\"priority\":[],\"nullable\":null,\"mapping\":{}}").nullable==null)
        check(runCatching { codec.decodeFromString<HolderDto>("{\"strings\":null,\"longs\":[],\"mixed\":[],\"priority\":[],\"mapping\":{}}") }.isFailure)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var patchMode=0
        var bad=false
        val engine=TupleStreamingEngine(MockEngineConfig().apply {
            addHandler { req ->
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(tupleBody(req.body))==value)
                        respond(codec.encodeToString(value),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/simple" -> {
                        check(codec.decodeFromString<List<Double>>(tupleBody(req.body))==listOf(1.5,2.0))
                        respond("[1.5,2.0]",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body=codec.parseToJsonElement(tupleBody(req.body)).jsonObject
                        when(patchMode) {
                            0 -> check("values" !in body)
                            1 -> check(body["values"]==JsonNull)
                            2 -> check(body["values"]==codec.parseToJsonElement("[1,2]"))
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val data=if(bad) "{\"strings\":null}" else codec.encodeToString(value)
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
            check(api.simple(listOf(1.5,2.0))==listOf(1.5,2.0))
            api.patchPatch(PatchInputDto(requiredNullable=null))
            patchMode=1;api.patchPatch(PatchInputDto(values=Patch.Null,requiredNullable=null))
            patchMode=2;api.patchPatch(PatchInputDto(values=Patch.Value(listOf(1,2)),requiredNullable=listOf("ready")))
            val flow=api.watch();check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Sample).data==value)
            check(channels.last().isClosedForRead)
            bad=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1 && api.echo(value)==value)
            println("PrimitiveTuple Json defaults=$defaults nulls=$nulls aliases-ctor-serde-approximation-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally {http.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

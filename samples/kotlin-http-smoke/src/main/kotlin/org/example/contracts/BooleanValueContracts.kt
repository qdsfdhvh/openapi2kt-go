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
import org.example.booleanvalues.BooleanValuesApiClient
import org.example.booleanvalues.api.models.ApplyInputDto
import org.example.booleanvalues.api.models.WatchEventDto
import org.example.booleanvalues.models.HolderDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BooleanStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response=super.execute(data)
        val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun booleanBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> booleanBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic Boolean schema body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun booleanValueContracts() = withTimeout(20000) {
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        val kinds=listOf("null","true","false","7","1.5","\"text\"","[]","{}")
        val nodes=kinds.map {codec.parseToJsonElement(it)}
        val values=nodes.map { HolderDto(it,rows=nodes,mapping=mapOf("a" to nodes)) }
        for(value in values) {
            check(codec.decodeFromString<HolderDto>(codec.encodeToString(value))==value)
            check(codec.parseToJsonElement(codec.encodeToString(value)).jsonObject["value"]==value.value)
        }
        val explicitOptional=HolderDto(JsonNull,optional=JsonNull,rows=emptyList(),mapping=emptyMap())
        val encodedOptional=codec.encodeToString(explicitOptional)
        check(codec.parseToJsonElement(encodedOptional).jsonObject["optional"]==JsonNull)
        // Nullable optional field codec collapses JsonNull to Kotlin null, as for {}.
        check(codec.decodeFromString<HolderDto>(encodedOptional).optional==null)
        check(codec.decodeFromString<HolderDto>("{\"value\":null,\"rows\":[],\"mapping\":{}}").value==JsonNull)
        check(runCatching {codec.decodeFromString<HolderDto>("{\"rows\":[],\"mapping\":{}}")}.isFailure)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var expected=values[0]
        var inlineMode=0
        var bad=false
        val engine=BooleanStreamingEngine(MockEngineConfig().apply {
            addHandler { req ->
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(booleanBody(req.body))==expected)
                        respond(codec.encodeToString(expected),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/rows" -> {
                        check(codec.parseToJsonElement(booleanBody(req.body))==codec.parseToJsonElement(codec.encodeToString(nodes)))
                        respond(codec.encodeToString(nodes),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/inline" -> {
                        val body=codec.parseToJsonElement(booleanBody(req.body)).jsonObject
                        check(body["value"]==JsonNull)
                        when(inlineMode) {
                            0 -> check(("optional" !in body) || body["optional"]==JsonNull)
                            1 -> check(body["optional"]==JsonNull)
                            2 -> check(body["optional"]==JsonPrimitive("ready"))
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val payload=if(bad) "{\"rows\":[],\"mapping\":{}}" else codec.encodeToString(expected)
                        val frame="event: sample\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic Boolean schema URL")
                }
            }
        })
        val http=HttpClient(engine) {install(ContentNegotiation){json(codec)};install(SSE)}
        try {
            val api=BooleanValuesApiClient(http,"http://example.test/",codec).default
            for(value in values) {expected=value;check(api.echo(value)==value)}
            check(api.rows(nodes)==nodes)
            api.apply(ApplyInputDto(value=JsonNull))
            inlineMode=1;api.apply(ApplyInputDto(value=JsonNull,optional=JsonNull))
            inlineMode=2;api.apply(ApplyInputDto(value=JsonNull,optional=JsonPrimitive("ready")))
            val flow=api.watch();check(channels.isEmpty())
            for(value in values) {
                expected=value;check((flow.take(1).toList().single() as WatchEventDto.Sample).data==value)
                check(channels.last().isClosedForRead)
            }
            bad=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1 && api.echo(expected)==expected)
            println("BooleanValue Json defaults=$defaults nulls=$nulls allkind-JsonNull-required-optional-HTTP-liveSSEclose-reuse-cold passed")
        } finally {http.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

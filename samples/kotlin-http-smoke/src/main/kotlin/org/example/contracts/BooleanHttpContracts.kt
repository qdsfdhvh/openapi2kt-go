package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.ResponseException
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.client.request.setBody
import io.ktor.http.HttpMethod
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
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import org.example.booleanhttp.BooleanHttpApiClient
import org.example.booleanhttp.api.errorBody
import org.example.booleanhttp.api.models.WatchEventDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BooleanHttpStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities=super.supportedCapabilities+SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response=super.execute(data)
        val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun booleanHttpBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> booleanHttpBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic Boolean HTTP body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun booleanHttpContracts()=withTimeout(20000) {
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        val nodes=listOf("null","true","false","7","1.5","\"quoted\"","[]","{}").map {codec.parseToJsonElement(it)}
        var expected: JsonElement=JsonNull
        var bad=false
        var failError=false
        var calls=0
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=BooleanHttpStreamingEngine(MockEngineConfig().apply {
            addHandler {request ->
                calls++
                when(request.url.encodedPath) {
                    "/echo","/borrowed","/optional" -> {
                        check(request.method==HttpMethod.Post)
                        check(codec.parseToJsonElement(booleanHttpBody(request.body))==expected)
                        check(request.body.contentType?.toString()?.startsWith("application/json")==true)
                        respond(if(bad) "[}" else codec.encodeToString(expected),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/priority" -> respond(codec.encodeToString(expected),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    "/errors" -> respond(codec.encodeToString(expected),if(failError) HttpStatusCode.BadRequest else HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    "/watch" -> {
                        check(request.method==HttpMethod.Post && codec.parseToJsonElement(booleanHttpBody(request.body))==expected)
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val payload=if(bad) "[}" else codec.encodeToString(expected)
                        val frame="event: sample\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic Boolean HTTP URL")
                }
            }
        })
        val http=HttpClient(engine) {expectSuccess=true;install(ContentNegotiation){json(codec)};install(SSE)}
        try {
            val api=BooleanHttpApiClient(http,"http://example.test/",codec).default
            for(node in nodes) {expected=node;check(api.echo(node)==node);check(api.borrowed(node)==node);check(api.optional(node)==node);check(api.priority()==node);check(api.errors()==node)}
            expected=JsonNull;check(api.optional()==JsonNull);check(api.optional(null)==JsonNull)
            expected=JsonPrimitive("override");check(api.echo(JsonNull){setBody(expected)}==expected)
            bad=true;check(runCatching {api.echo(expected)}.isFailure);bad=false;check(api.echo(expected)==expected)
            failError=true
            for(node in nodes) {expected=node;val failure=runCatching {api.errors()}.exceptionOrNull();check(failure is ResponseException);check(failure.errorBody<JsonElement>()==node);check(failure.errorBody<JsonElement>()==node)}
            failError=false;check(api.echo(expected)==expected)
            val count=calls;val flow=api.watch(input=expected);check(calls==count && channels.isEmpty())
            for(node in nodes) {expected=node;val current=api.watch(input=node);check((current.take(1).toList().single() as WatchEventDto.Sample).data==node);check(channels.last().isClosedForRead)}
            bad=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1);check(channels.last().isClosedForRead);check(api.echo(expected)==expected)
            println("BooleanHttp Json defaults=$defaults nulls=$nulls directtrue-allkind-JsonNull-optionalnull-priority-borrowed-errors-callback-HTTP-POSTSSEclose-reuse-cold passed")
        } finally {http.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

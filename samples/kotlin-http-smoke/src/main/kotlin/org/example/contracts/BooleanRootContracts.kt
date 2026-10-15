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
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.jsonObject
import org.example.booleanroots.BooleanRootsApiClient
import org.example.booleanroots.api.models.PatchInputDto
import org.example.booleanroots.api.models.WatchEventDto
import org.example.booleanroots.models.AChainDto
import org.example.booleanroots.models.HolderDto
import org.example.booleanroots.models.MiddleDto
import org.example.booleanroots.models.Patch
import org.example.booleanroots.models.ZAnyDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RootStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response=super.execute(data)
        val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun rootBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> rootBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic root body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun booleanRootContracts() = withTimeout(20000) {
    check(serializer<AChainDto>().descriptor==serializer<ZAnyDto>().descriptor)
    check(serializer<MiddleDto>().descriptor==serializer<ZAnyDto>().descriptor)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        val nodes=listOf("null","true","false","7","1.5","\"text\"","[]","{}").map {codec.parseToJsonElement(it)}
        for(node in nodes) {
            val root: ZAnyDto=node
            val alias: AChainDto=root
            check(codec.decodeFromString<ZAnyDto>(codec.encodeToString(root))==node)
            check(codec.decodeFromString<AChainDto>(codec.encodeToString(alias))==node)
        }
        val values=nodes.map { HolderDto(value=it,chain=it,rows=nodes,mapping=mapOf("a" to nodes),nullable=null) }
        for(value in values) check(codec.decodeFromString<HolderDto>(codec.encodeToString(value))==value)
        check(runCatching {codec.decodeFromString<HolderDto>("{}")}.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(values[0])).value==JsonNull)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var expected=values[0]
        var mode=0
        var bad=false
        val engine=RootStreamingEngine(MockEngineConfig().apply {
            addHandler { req ->
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(rootBody(req.body))==expected)
                        respond(codec.encodeToString(expected),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/any" -> {
                        val raw=rootBody(req.body);check(codec.parseToJsonElement(raw)==expected.value)
                        respond(raw,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body=codec.parseToJsonElement(rootBody(req.body)).jsonObject
                        // Preserve the existing nullable field wire policy:
                        // explicitNulls=false omits Kotlin null, even required.
                        check(body["requiredNullable"]==if(nulls) JsonNull else null)
                        when(mode) {
                            0 -> check("value" !in body)
                            1 -> check(body["value"]==JsonNull)
                            2 -> check(body["value"]==expected.value)
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/watch" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val payload=if(bad) "{}" else codec.encodeToString(expected)
                        val frame="event: sample\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic root URL")
                }
            }
        })
        val http=HttpClient(engine) {install(ContentNegotiation){json(codec)};install(SSE)}
        try {
            val api=BooleanRootsApiClient(http,"http://example.test/",codec).default
            for(value in values) {expected=value;check(api.echo(value)==value);check(api.any(value.value)==value.value)}
            api.patchPatch(PatchInputDto(requiredNullable=null))
            mode=1;api.patchPatch(PatchInputDto(requiredNullable=null,value=Patch.Null))
            mode=2;api.patchPatch(PatchInputDto(requiredNullable=null,value=Patch.Value(expected.value)))
            val flow=api.watch();check(channels.isEmpty())
            for(value in values) {
                expected=value;check((flow.take(1).toList().single() as WatchEventDto.Sample).data==value)
                check(channels.last().isClosedForRead)
            }
            bad=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1 && api.any(expected.value)==expected.value)
            println("BooleanRoot Json defaults=$defaults nulls=$nulls canonical-alias-allkind-JsonNull-ctor-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally {http.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

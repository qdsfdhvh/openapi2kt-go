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
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.serializer
import org.example.merge.MergeApiClient
import org.example.merge.api.models.ApplyInputDto
import org.example.merge.api.models.OneResponseDto
import org.example.merge.api.models.TwoResponseDto
import org.example.merge.api.models.WatchEventDto
import org.example.merge.models.AlphaDto
import org.example.merge.models.ZetaDto
import org.example.merge.models.ChildADto
import org.example.merge.models.ChildZDto
import org.example.merge.models.ParentADto
import org.example.merge.models.ParentZDto
import org.example.merge.models.Patch
import org.example.mergeprobes.ApiClient as ProbeApiClient
import org.example.mergeprobes.models.LooseADto
import org.example.mergeprobes.models.LooseZDto
import org.example.mergeprobes.models.StatusADto
import org.example.mergeprobes.models.StatusZDto
import org.example.mergeprobes.models.CountADto
import org.example.mergeprobes.models.CountZDto
import org.example.mergeprobes.models.ConstADto
import org.example.mergeprobes.models.ConstZDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class MergeStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
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
private fun mergeBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> mergeBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun mergeModelContracts(plain: Boolean = false) = withTimeout(15000) {
    val old: ZetaDto = AlphaDto("ready")
    val response: OneResponseDto = old
    val peer: TwoResponseDto = response
    val child: ChildZDto = ChildADto(1)
    val parent: ParentZDto = ParentADto(child)
    check(old == peer && old.hashCode() == peer.hashCode())
    if (plain) check(old.toString() == "AlphaDto")
    check(serializer<ZetaDto>().descriptor.serialName == serializer<AlphaDto>().descriptor.serialName)
    check(serializer<ChildZDto>().descriptor.serialName == serializer<ChildADto>().descriptor.serialName)
    check(runCatching { ChildZDto(0) }.isFailure)
    val nested: LooseZDto = LooseADto(LooseADto.Status.fromSerialName("fast"), LooseADto.Meta(true))
    val status: StatusZDto = StatusADto.fromSerialName("future")
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        check(codec.decodeFromString<ZetaDto>(codec.encodeToString(old))==old)
        check(codec.decodeFromString<ParentZDto>(codec.encodeToString(parent))==parent)
        check(codec.decodeFromString<LooseZDto>(codec.encodeToString(nested))==nested)
        check(codec.decodeFromString<StatusZDto>(codec.encodeToString(status))==status)
        check(codec.parseToJsonElement(codec.encodeToString(ConstADto("v"))).jsonObject["kind"].toString()=="\"a\"")
        check(codec.parseToJsonElement(codec.encodeToString(ConstZDto("v"))).jsonObject["kind"].toString()=="\"z\"")
        check(codec.encodeToString(CountADto(1))=="{\"n\":1}")
        check(runCatching { codec.encodeToString(CountZDto(1)) }.isFailure)
        check(runCatching { codec.decodeFromString<ChildZDto>("{\"n\":0}") }.isFailure)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var bad=false
        val probePaths=mutableListOf<String>()
        val engine=MergeStreamingEngine(MockEngineConfig().apply {
            addHandler { req ->
                when(req.url.encodedPath) {
                    "/watch" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val frame="event: update\r\ndata: {\"child\":{\"n\":"+(if(bad)"0" else "1")+"}}\r\n\r\n"
                        writers.add(scope.launch { for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation() })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    "/aa", "/bb", "/cc" -> {
                        probePaths.add(req.url.encodedPath)
                        check(codec.decodeFromString<LooseZDto>(mergeBody(req.body))==nested)
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/parent" -> respond("{\"child\":{\"n\":1}}",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    "/patch" -> {
                        val body=codec.parseToJsonElement(mergeBody(req.body)).jsonObject
                        check(body["value"]==JsonNull)
                        respond("{\"value\":\"ready\"}",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    else -> respond("{\"value\":\"ready\"}",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                }
            }
        })
        val http=HttpClient(engine) { install(ContentNegotiation){json(codec)};install(SSE) }
        try {
            val api=MergeApiClient(http,"http://example.test/",codec).default
            val probe=ProbeApiClient(http,"http://example.test/").default
            probe.submit(nested)
            probe.submit3(nested)
            probe.submit2(nested)
            check(probePaths==listOf("/aa","/bb","/cc"))
            check(api.one()==old && api.two()==old && api.parent()==parent)
            check(api.apply(ApplyInputDto(Patch.Null))==old)
            val flow=api.watch();check(channels.isEmpty())
            check((flow.take(1).toList().single() as WatchEventDto.Update).data==parent)
            check(channels.last().isClosedForRead)
            bad=true;check(runCatching { flow.take(1).toList() }.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1 && api.one()==old)
            println("MergeModels Json defaults=$defaults nulls=$nulls plain=$plain resolvedaliases-codecs-bounds-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally { http.close();scope.cancel();writers.forEach { it.cancel() };channels.forEach { it.cancel(null) } }
    }
}

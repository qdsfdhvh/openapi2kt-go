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
import org.example.booleanrootnullable.BooleanRootNullableApiClient
import org.example.booleanrootnullable.api.models.PatchInputDto
import org.example.booleanrootnullable.api.models.WatchEventDto
import org.example.booleanrootnullable.models.ANullableDto
import org.example.booleanrootnullable.models.BInheritedDto
import org.example.booleanrootnullable.models.CFalseDto
import org.example.booleanrootnullable.models.DNullableAgainDto
import org.example.booleanrootnullable.models.HolderDto
import org.example.booleanrootnullable.models.Patch
import org.example.booleanrootnullable.models.ZAnyDto
import java.util.concurrent.CopyOnWriteArrayList

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NullableRootStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response=super.execute(data)
        val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}
private fun nullableRootBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> nullableRootBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic nullable-root body")
}
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun booleanRootNullableContracts() = withTimeout(20000) {
    check(serializer<ANullableDto>().descriptor==serializer<BInheritedDto>().descriptor)
    check(serializer<DNullableAgainDto>().descriptor==serializer<BInheritedDto>().descriptor)
    check(serializer<CFalseDto>().descriptor==serializer<ZAnyDto>().descriptor)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults;explicitNulls=nulls }
        val nodes=listOf("null","true","false","7","1.5","\"text\"","[]","{}").map {codec.parseToJsonElement(it)}
        val nullableNodes=nodes.map {if(it==JsonNull) null else it}
        for(node in nodes) {
            val canonical: ZAnyDto=node
            val alias: ANullableDto=canonical
            val narrowed: CFalseDto=canonical
            check(codec.decodeFromString<ANullableDto>(codec.encodeToString(alias))==if(node==JsonNull) null else node)
            check(codec.decodeFromString<BInheritedDto>(codec.encodeToString(alias))==if(node==JsonNull) null else node)
            check(codec.decodeFromString<DNullableAgainDto>(codec.encodeToString(alias))==if(node==JsonNull) null else node)
            check(codec.decodeFromString<CFalseDto>(codec.encodeToString(narrowed))==node)
        }
        val values=nodes.map { HolderDto(canonical=it,inherited=if(it==JsonNull) null else it,narrowed=it,rows=nullableNodes,mapping=mapOf("a" to null,"b" to it.takeUnless {v -> v==JsonNull})) }
        for(value in values) check(codec.decodeFromString<HolderDto>(codec.encodeToString(value))==value)
        check(runCatching {codec.decodeFromString<HolderDto>("{}")}.isFailure)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var expected=values[0]
        var mode=0
        var bad=false
        val engine=NullableRootStreamingEngine(MockEngineConfig().apply {
            addHandler { req ->
                when(req.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(nullableRootBody(req.body))==expected)
                        respond(codec.encodeToString(expected),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/nullable", "/narrow" -> {
                        val raw=nullableRootBody(req.body);check(codec.parseToJsonElement(raw)==expected.canonical)
                        respond(raw,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {
                        val body=codec.parseToJsonElement(nullableRootBody(req.body)).jsonObject
                        check(body["requiredNullable"]==if(nulls) JsonNull else null)
                        check(body["narrowed"]==JsonNull)
                        when(mode) {
                            0 -> check("value" !in body)
                            1 -> check(body["value"]==JsonNull)
                            2 -> check(body["value"]==expected.canonical)
                            3 -> check(body["value"]==JsonNull)
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
                    else -> error("Unexpected synthetic nullable-root URL")
                }
            }
        })
        val http=HttpClient(engine) {install(ContentNegotiation){json(codec)};install(SSE)}
        try {
            val api=BooleanRootNullableApiClient(http,"http://example.test/",codec).default
            for(value in values) {
                expected=value;check(api.echo(value)==value)
                check(api.nullable(value.inherited)==value.inherited)
                check(api.narrow(value.narrowed)==value.narrowed)
            }
            api.patchPatch(PatchInputDto(requiredNullable=null,narrowed=JsonNull))
            mode=1;api.patchPatch(PatchInputDto(requiredNullable=null,narrowed=JsonNull,value=Patch.Null))
            mode=2;api.patchPatch(PatchInputDto(requiredNullable=null,narrowed=JsonNull,value=Patch.Value(expected.canonical)))
            mode=3;api.patchPatch(PatchInputDto(requiredNullable=null,narrowed=JsonNull,value=Patch.Value(JsonNull)))
            val encodedPatch=codec.encodeToString(PatchInputDto(requiredNullable=null,narrowed=JsonNull,value=Patch.Value(JsonNull)))
            check(codec.decodeFromString<PatchInputDto>(encodedPatch).value==Patch.Null)
            val flow=api.watch();check(channels.isEmpty())
            for(value in values) {expected=value;check((flow.take(1).toList().single() as WatchEventDto.Sample).data==value);check(channels.last().isClosedForRead)}
            bad=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead)
            bad=false;check(flow.take(1).toList().size==1 && api.narrow(expected.narrowed)==expected.narrowed)
            println("BooleanRootNullable Json defaults=$defaults nulls=$nulls alias-nearestfalse-renull-allkind-JsonNull-collapse-ctor-Patch-HTTP-liveSSEclose-reuse-cold passed")
        } finally {http.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

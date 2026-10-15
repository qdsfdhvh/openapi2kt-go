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
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.roothierarchyaliases.RootHierarchyAliasApiClient
import org.example.roothierarchyaliases.api.models.SubmitInputDto
import org.example.roothierarchyaliases.api.models.WatchEventDto
import org.example.roothierarchyaliases.models.ASignalDto
import org.example.roothierarchyaliases.models.AChannelDto
import org.example.roothierarchyaliases.models.AFluxDto
import org.example.roothierarchyaliases.models.AQuietDto
import org.example.roothierarchyaliases.models.AStatusABDto
import org.example.roothierarchyaliases.models.HolderDto
import org.example.roothierarchyaliases.models.Patch
import org.example.roothierarchyaliases.models.ZSignalDto
import org.example.roothierarchyaliases.models.ZChannelDto
import org.example.roothierarchyaliases.models.ZFluxDto
import org.example.roothierarchyaliases.models.ZQuietDto
import org.example.roothierarchyaliases.models.ZSignalDtoFamily
import org.example.roothierarchyaliases.models.ZChannelDtoPhase
import org.example.roothierarchyaliases.models.UnknownZSignalDto
import org.example.roothierarchyaliases.models.UnknownZChannelDto
import org.example.roothierarchyprobes.models.ANearDto as HierarchyNearDto
import org.example.roothierarchyprobes.models.AWideDto as HierarchyWideDto
import org.example.roothierarchyprobes.models.LeafDto as ProbeLeafDto
import org.example.roothierarchyprobes.models.ZRootDtoTag as ProbeTagDto

private fun hierarchyAliasReject(block: () -> Unit) {
    try {block();error("Missing hierarchy alias bound")}
    catch (cause: IllegalArgumentException){check(cause.message?.contains("minLength")==true)}
}
private suspend fun hierarchyAliasWireReject(message: String, block: suspend () -> Unit) {
    try {block();error("Missing hierarchy alias wire error")}
    catch (cause: Throwable){check(generateSequence(cause){it.cause}.any {
        (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message,ignoreCase=true)==true
    }) {"Wrong hierarchy alias failure $cause"}}
}
private fun hierarchyAliasBody(body: OutgoingContent): String = when(body){
    is OutgoingContent.ContentWrapper -> hierarchyAliasBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected hierarchy alias body")
}

suspend fun rootHierarchyAliasContracts() {
    val leaf=AFluxDto(ZChannelDtoPhase.DAWN,"valid",listOf(Long.MIN_VALUE,Long.MAX_VALUE))
    val parent: AChannelDto=leaf
    val signal: ASignalDto=parent
    val canonical: ZSignalDto=signal
    check(canonical===leaf && AQuietDto===ZQuietDto)
    check(ASignalDto.serializer().descriptor==ZSignalDto.serializer().descriptor)
    check(AChannelDto.serializer().descriptor==ZChannelDto.serializer().descriptor)
    check(AFluxDto.serializer().descriptor==ZFluxDto.serializer().descriptor)
    check(leaf.family===ZSignalDtoFamily.STREAM)
    hierarchyAliasReject {AFluxDto(ZChannelDtoPhase.DAWN,"",emptyList())}
    hierarchyAliasReject {leaf.copy(note="")}
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)){
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls;ignoreUnknownKeys=true}
        val streamCodec=Json(codec){ignoreUnknownKeys=false}
        rootHierarchyAliasBackendContracts(codec)
        val outerRaw=codec.parseToJsonElement("{\"family\":\"future\",\"payload\":[null,1]}")
        val innerRaw=codec.parseToJsonElement("{\"family\":\"stream\",\"phase\":\"future\",\"payload\":9}")
        val unknown: ASignalDto=UnknownZSignalDto(outerRaw)
        val inner: ASignalDto=UnknownZChannelDto(innerRaw)
        check((inner as UnknownZChannelDto).family==ZSignalDtoFamily.Unrecognized(""))
        check(inner.phase==ZChannelDtoPhase.Unrecognized(""))
        val status: ASignalDto=AStatusABDto(false)
        for(value in listOf(signal,status,AQuietDto,unknown,inner)){
            val encoded=codec.encodeToString<ASignalDto>(value)
            check(codec.decodeFromString<ASignalDto>(encoded)==value)
        }
        check(codec.encodeToJsonElement<ASignalDto>(unknown)==outerRaw)
        check(codec.encodeToJsonElement<ASignalDto>(inner)==innerRaw)
        check(codec.decodeFromString<AChannelDto>(codec.encodeToString<AChannelDto>(leaf))==leaf)
        check(codec.decodeFromString<AFluxDto>(codec.encodeToString<AFluxDto>(leaf))==leaf)
        check(codec.decodeFromString<ASignalDto>("{\"family\":\"stream\",\"phase\":\"ignored-alias\"}") is UnknownZChannelDto)
        val noon=leaf.copy(phase=ZChannelDtoPhase.NOON)
        check(codec.decodeFromString<ASignalDto>(codec.encodeToString<ASignalDto>(noon))==noon)
        hierarchyAliasReject {codec.decodeFromString<ASignalDto>("{\"family\":\"stream\",\"phase\":\"dawn\",\"note\":\"\",\"samples\":[]}")}
        val holder=HolderDto(signal,listOf(status,AQuietDto,unknown,inner),leaf,mapOf("leaf" to leaf))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))==holder)
        val wide=codec.decodeFromString<HierarchyWideDto>("{\"p0\":{\"tag\":\"leaf\",\"id\":\"wide\"}}")
        check((wide.p0 as ProbeLeafDto).id=="wide")
        val near=HierarchyNearDto(ProbeLeafDto("near"));check(codec.decodeFromString<HierarchyNearDto>(codec.encodeToString(near))==near)
        check((near.v as ProbeLeafDto).tag==ProbeTagDto.LEAF)
        var calls=0;var mode=0;var expected=""
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=StreamingTestEngine(MockEngineConfig().apply {
            addHandler {request ->
                calls++
                when(request.url.encodedPath){
                    "/alias","/direct","/leaf"->{check(hierarchyAliasBody(request.body)==expected);val response=if(mode==1) "{\"family\":\"stream\",\"phase\":\"dawn\",\"note\":\"\",\"samples\":[]}" else expected;respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
                    "/patch"->{check(hierarchyAliasBody(request.body)==expected);respond("",HttpStatusCode.NoContent)}
                    "/events"->{val channel=ByteChannel(autoFlush=true);channels+=channel;val current=mode
                        producers.launch {try {
                            val frame=when(current){
                                1->"event: signal\r\ndata: {\"family\":\"stream\",\"phase\":\"dawn\",\"note\":\"\",\"samples\":[]}\r\n\r\n"
                                2->"event: signal\r\ndata: {\"family\":[]}\r\n\r\n"
                                3->"event: leaf\r\ndata: {\"phase\":\"dawn\",\"note\":\"valid\",\"samples\":[],\"extra\":true}\r\n\r\n"
                                else->"event: signal\r\nid: hierarchy-id\r\ndata: ${codec.encodeToString<ASignalDto>(inner)}\r\n\r\nevent: leaf\r\ndata: ${codec.encodeToString<AFluxDto>(leaf)}\r\n\r\n"
                            }
                            for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte))
                            if(current!=0)awaitCancellation()
                        } finally {channel.close()}}
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else->error("Unexpected hierarchy alias route")
                }
            }
        })
        val http=HttpClient(engine){install(SSE);install(ContentNegotiation){json(codec)}}
        try {
            val api=RootHierarchyAliasApiClient(http,"http://example.test/",streamCodec).default
            expected=codec.encodeToString<ASignalDto>(signal);check(api.echo(signal)==signal);check(api.echo2(signal)==signal)
            for(raw in listOf(unknown,inner)){expected=codec.encodeToString<ASignalDto>(raw);check(api.echo(raw)==raw)}
            expected=codec.encodeToString<ASignalDto>(signal);mode=1;hierarchyAliasWireReject("minLength"){api.echo(signal)};mode=0
            expected=codec.encodeToString<AFluxDto>(leaf);check(api.echoLeaf(leaf)==leaf)
            for(state in listOf<Patch<ZSignalDto>>(Patch.Absent,Patch.Null,Patch.Value(signal))){
                val leafState: Patch<ZFluxDto> = when(state){Patch.Absent->Patch.Absent;Patch.Null->Patch.Null;is Patch.Value->Patch.Value(leaf)}
                val input=SubmitInputDto(signal=state,leaf=leafState)
                val body=codec.encodeToJsonElement(input) as JsonObject
                check(when(state){Patch.Absent->"signal" !in body && "leaf" !in body;Patch.Null->body["signal"]==JsonNull && body["leaf"]==JsonNull;is Patch.Value->body["signal"]==codec.encodeToJsonElement<ASignalDto>(signal) && body["leaf"]==codec.encodeToJsonElement<AFluxDto>(leaf)})
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(),body)==input)
                expected=codec.encodeToString(input);api.submit(input)
            }
            val before=calls;val cold=api.watch();check(calls==before)
            repeat(2){val events=withTimeout(10_000){cold.toList()};check((events[0] as WatchEventDto.Signal).data==inner);check((events[1] as WatchEventDto.Leaf).data==leaf);check(events.all{it.id=="hierarchy-id"})}
            check(calls==before+2)
            for((bad,message) in listOf(1 to "minLength",2 to "primitive",3 to "extra")){mode=bad;hierarchyAliasWireReject(message){withTimeout(10_000){api.watch().toList()}};check(channels.last().isClosedForRead)}
            mode=0;check(withTimeout(10_000){api.watch().toList().size}==2)
            expected=codec.encodeToString<ASignalDto>(signal);check(api.echo(signal)==signal)
        } finally {http.close();engine.close();producers.cancel()}
        println("Root hierarchy alias contracts passed: defaults=$defaults/nulls=$nulls; canonical ancestor/leaf/typedtag/knownUnknown/240+48/Patch/HTTP/live SSE close-reuse-cold")
    }
}

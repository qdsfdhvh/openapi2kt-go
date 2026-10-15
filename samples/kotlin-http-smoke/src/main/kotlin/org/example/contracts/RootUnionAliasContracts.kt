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
import org.example.rootunionaliases.RootUnionAliasApiClient
import org.example.rootunionaliases.api.models.SubmitInputDto
import org.example.rootunionaliases.api.models.WatchEventDto
import org.example.rootunionaliases.models.AChoiceDto
import org.example.rootunionaliases.models.AKindDto
import org.example.rootunionaliases.models.AClosedDto
import org.example.rootunionaliases.models.HolderDto
import org.example.rootunionaliases.models.Patch
import org.example.rootunionaliases.models.ZChoiceDto
import org.example.rootunionaliases.models.ZClosedDto
import org.example.rootunionaliases.models.ZKindDto
import org.example.rootunionaliases.models.NullZKindDto
import org.example.rootunionaliases.models.NullZClosedDto
import org.example.rootunionaliases.models.StringZKindDto
import org.example.rootunionaliases.models.BooleanZKindDto
import org.example.rootunionaliases.models.UnknownZChoiceDto
import org.example.rootunionprobes.models.ANearDto as UnionNearDto
import org.example.rootunionprobes.models.AWideDto as UnionWideDto
import org.example.rootunionprobes.models.NullZKindDto as ProbeNullDto

private fun rootUnionAliasReject(block: () -> Unit) {
    try {block();error("Missing union alias bound")}
    catch (cause: IllegalArgumentException){check(cause.message?.contains("minLength")==true)}
}
private suspend fun rootUnionAliasWireReject(message: String, block: suspend () -> Unit) {
    try {block();error("Missing union alias wire error")}
    catch (cause: Throwable){check(generateSequence(cause){it.cause}.any {
        (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message,ignoreCase=true)==true
    }) {"Wrong union alias failure $cause"}}
}
private fun rootUnionAliasBody(body: OutgoingContent): String = when(body){
    is OutgoingContent.ContentWrapper -> rootUnionAliasBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected union alias body")
}

suspend fun rootUnionAliasContracts() {
    val note: AChoiceDto=ZChoiceDto.ZNoteDto("valid")
    val flag: AChoiceDto=ZChoiceDto.ZFlagDto(true)
    check(AChoiceDto.serializer().descriptor==ZChoiceDto.serializer().descriptor)
    rootUnionAliasReject {ZChoiceDto.ZNoteDto("")}
    rootUnionAliasReject {(note as ZChoiceDto.ZNoteDto).copy(text="")}
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)){
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls;ignoreUnknownKeys=true}
        val streamCodec=Json(codec){ignoreUnknownKeys=false}
        val future=codec.parseToJsonElement("{\"kind\":\"future\",\"payload\":[null,1]}")
        val unknown: AChoiceDto=UnknownZChoiceDto(future)
        for(value in listOf(note,flag,unknown)){
            val encoded=codec.encodeToString<AChoiceDto>(value)
            check(codec.decodeFromString<AChoiceDto>(encoded)==value)
            check(codec.decodeFromJsonElement(AChoiceDto.serializer(),codec.parseToJsonElement(encoded))==value)
        }
        check(codec.encodeToJsonElement<AChoiceDto>(unknown)==future)
        rootUnionAliasReject {codec.decodeFromString<AChoiceDto>("{\"kind\":\"note\",\"text\":\"\"}")}
        val kinds=listOf<AKindDto>(StringZKindDto("text"),BooleanZKindDto(true),NullZKindDto)
        for(value in kinds){val text=codec.encodeToString<AKindDto>(value);check(codec.decodeFromString<AKindDto>(text)==value)}
        check(codec.encodeToString<AKindDto>(NullZKindDto)=="null")
        val closed: AClosedDto=ZClosedDto.Variant1(3)
        for(config in listOf(codec,Json(codec){classDiscriminator="category"},Json(codec){useArrayPolymorphism=true})){
            check(config.decodeFromString<AClosedDto>(config.encodeToString<AClosedDto>(closed))==closed)
            check(config.decodeFromString<AClosedDto>("null")==NullZClosedDto)
            try { config.decodeFromString<AClosedDto>("{}"); error("Closed union lost discriminator requirement") }
            catch (_: SerializationException) { }
        }
        val holder=HolderDto(note,NullZKindDto,closed,listOf(note,unknown),mapOf("null" to NullZKindDto))
        val wire=codec.encodeToString(holder)
        check(codec.decodeFromString<HolderDto>(wire)==holder)
        check((codec.parseToJsonElement(wire) as JsonObject).containsKey("kind"))
        val wide=codec.decodeFromString<UnionWideDto>("{\"p0\":null}");check(wide.p0==null)
        val near=UnionNearDto(ProbeNullDto);check(codec.decodeFromString<UnionNearDto>(codec.encodeToString(near))==near)
        var calls=0;var mode=0;var expected=""
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=StreamingTestEngine(MockEngineConfig().apply {
            addHandler {request ->
                calls++
                when(request.url.encodedPath){
                    "/alias","/direct","/kind"->{check(rootUnionAliasBody(request.body)==expected);val response=if(mode==1) "{\"kind\":\"note\",\"text\":\"\"}" else expected;respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
                    "/patch"->{check(rootUnionAliasBody(request.body)==expected);respond("",HttpStatusCode.NoContent)}
                    "/events"->{val channel=ByteChannel(autoFlush=true);channels+=channel;val current=mode
                        producers.launch {try {
                            val frame=when(current){
                                1->"event: choice\r\ndata: {\"kind\":\"note\",\"text\":\"\"}\r\n\r\n"
                                2->"event: kind\r\ndata: {}\r\n\r\n"
                                3->"event: choice\r\ndata: {\"kind\":\"note\",\"text\":\"valid\",\"extra\":true}\r\n\r\n"
                                else->"event: choice\r\nid: union-id\r\ndata: ${codec.encodeToString<AChoiceDto>(unknown)}\r\n\r\nevent: kind\r\ndata: null\r\n\r\n"
                            }
                            for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte))
                            if(current!=0)awaitCancellation()
                        } finally {channel.close()}}
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else->error("Unexpected union alias route")
                }
            }
        })
        val http=HttpClient(engine){install(SSE);install(ContentNegotiation){json(codec)}}
        try {
            val api=RootUnionAliasApiClient(http,"http://example.test/",streamCodec).default
            expected=codec.encodeToString<AChoiceDto>(note);check(api.echo(note)==note);check(api.echo2(note)==note)
            expected=codec.encodeToString<AChoiceDto>(unknown);check(api.echo(unknown)==unknown)
            expected=codec.encodeToString<AChoiceDto>(note)
            mode=1;rootUnionAliasWireReject("minLength"){api.echo(note)};mode=0
            for(kind in kinds){expected=codec.encodeToString<AKindDto>(kind);check(api.echoKind(kind)==kind)}
            for(state in listOf<Patch<ZChoiceDto>>(Patch.Absent,Patch.Null,Patch.Value(note))){
                val kind: Patch<ZKindDto> = when(state){Patch.Absent->Patch.Absent;Patch.Null->Patch.Null;is Patch.Value->Patch.Value(NullZKindDto)}
                val input=SubmitInputDto(choice=state,kind=kind)
                val body=codec.encodeToJsonElement(input) as JsonObject
                check(when(state){Patch.Absent->"choice" !in body && "kind" !in body;Patch.Null->body["choice"]==JsonNull && body["kind"]==JsonNull;is Patch.Value->body["choice"]==codec.encodeToJsonElement<AChoiceDto>(note) && body["kind"]==JsonNull})
                // JSON null decodes to Patch.Null, not Patch.Value(native Null carrier).
                val restored=codec.decodeFromJsonElement(SubmitInputDto.serializer(),body)
                check(restored.choice==state && restored.kind==if(state is Patch.Value)Patch.Null else kind)
                expected=codec.encodeToString(input);api.submit(input)
            }
            val before=calls;val cold=api.watch();check(calls==before)
            repeat(2){val events=withTimeout(10_000){cold.toList()};check((events[0] as WatchEventDto.Choice).data==unknown);check((events[1] as WatchEventDto.Kind).data==NullZKindDto);check(events.all{it.id=="union-id"})}
            check(calls==before+2)
            for((bad,message) in listOf(1 to "minLength",2 to "JSON",3 to "extra")){mode=bad;rootUnionAliasWireReject(message){withTimeout(10_000){api.watch().toList()}};check(channels.last().isClosedForRead)}
            mode=0;check(withTimeout(10_000){api.watch().toList().size}==2)
            expected=codec.encodeToString<AChoiceDto>(note);check(api.echo(note)==note)
        } finally {http.close();engine.close();producers.cancel()}
        println("Root union alias contracts passed: defaults=$defaults/nulls=$nulls; canonical content/kind/closed/null/240+48/Patch/HTTP/live SSE close-reuse-cold")
    }
}

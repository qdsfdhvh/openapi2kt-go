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
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.plainunion.models.ZChoiceDto
import org.example.plainunion.models.UnknownZChoiceDto
import org.example.plainunion.models.StringZKindDto
import org.example.plainunion.models.BooleanZKindDto
import org.example.plainunion.models.NullZKindDto
import org.example.plainunion.models.ZClosedDto
import org.example.plainunion.models.NullZClosedDto
import org.example.plainunion.api.models.WatchEventDto as UnionEvent
import org.example.plainhierarchy.models.ZChannelDtoPhase
import org.example.plainhierarchy.models.ZFluxDto
import org.example.plainhierarchy.models.ZSignalDto
import org.example.plainhierarchy.models.UnknownZSignalDto
import org.example.plainhierarchy.models.ZQuietDto
import org.example.plainruntime.models.RecordDto
import org.example.plainruntime.models.Patch
import org.example.plainruntime.api.models.SubmitInputDto
import org.example.plainruntime.api.models.WatchEventDto as RuntimeEvent
import org.example.plainruntime.SharedRuntimeExampleApiClient

private fun verifyValue(a: Any, b: Any) {
 check(a==b && a.hashCode()==b.hashCode())
 check(a.toString()==a.javaClass.simpleName)
 check(a.javaClass.methods.none { it.name=="copy" || it.name.startsWith("component") })
}
private fun plainBody(body: OutgoingContent):String=when(body){is OutgoingContent.ContentWrapper->plainBody(body.delegate());is OutgoingContent.ByteArrayContent->body.bytes().decodeToString();else->error("Unexpected body")}
suspend fun plainStyleContextContracts()=withTimeout(15000){
 for(defaults in listOf(false,true))for(nulls in listOf(false,true)){
 val codec=Json{encodeDefaults=defaults;explicitNulls=nulls}
 val note=ZChoiceDto.ZNoteDto("same");verifyValue(note,ZChoiceDto.ZNoteDto("same"));check(note.kind=="note")
 check(codec.decodeFromJsonElement(ZChoiceDto.serializer(),codec.encodeToJsonElement<ZChoiceDto>(note))==note)
 val raw=codec.parseToJsonElement("{\"kind\":\"future\",\"unknown\":[1,true]}");val unknown=UnknownZChoiceDto(raw);verifyValue(unknown,UnknownZChoiceDto(raw));check(codec.encodeToJsonElement<ZChoiceDto>(unknown)==raw)
 verifyValue(StringZKindDto("sample"),StringZKindDto("sample"));verifyValue(BooleanZKindDto(true),BooleanZKindDto(true));check(codec.encodeToJsonElement(StringZKindDto("sample"))==JsonPrimitive("sample"));check(codec.encodeToJsonElement(BooleanZKindDto(true))==JsonPrimitive(true));check(codec.encodeToJsonElement(NullZKindDto)==JsonNull)
 val closed=ZClosedDto.Variant1(7);verifyValue(closed,ZClosedDto.Variant1(7));check(codec.decodeFromJsonElement(ZClosedDto.serializer(),codec.encodeToJsonElement<ZClosedDto>(closed))==closed);check(codec.encodeToJsonElement(NullZClosedDto)==JsonNull)
 val phase=ZChannelDtoPhase.fromSerialName("dawn");check(phase===ZChannelDtoPhase.fromSerialName("dawn"));val future=ZChannelDtoPhase.Unrecognized("future");verifyValue(future,ZChannelDtoPhase.Unrecognized("future"));check(codec.decodeFromJsonElement(ZChannelDtoPhase.serializer(),codec.encodeToJsonElement<ZChannelDtoPhase>(future))==future)
 val leaf=ZFluxDto(phase,"leaf",listOf(1L));verifyValue(leaf,ZFluxDto(phase,"leaf",listOf(1L)));check(codec.decodeFromJsonElement(ZFluxDto.serializer(),codec.encodeToJsonElement(leaf))==leaf)
 val hierarchyRaw=codec.parseToJsonElement("{\"family\":\"future\",\"extra\":1}");verifyValue(UnknownZSignalDto(hierarchyRaw),UnknownZSignalDto(hierarchyRaw));check(codec.encodeToJsonElement<ZSignalDto>(UnknownZSignalDto(hierarchyRaw))==hierarchyRaw)
 check(ZQuietDto===ZQuietDto)
 verifyValue(UnionEvent.Choice(note,"id"),UnionEvent.Choice(note,"id"));verifyValue(UnionEvent.Kind(StringZKindDto("sample")),UnionEvent.Kind(StringZKindDto("sample")))
 val record=RecordDto(23L,listOf(23L));verifyValue(record,RecordDto(23L,listOf(23L)));val recordJSON="{\"amount\":\"23\",\"values\":[23]}";check(codec.encodeToJsonElement(record)==codec.parseToJsonElement(recordJSON));verifyValue(SubmitInputDto("ok",Patch.Value(17L)),SubmitInputDto("ok",Patch.Value(17L)));verifyValue(RuntimeEvent.Message(record,"id"),RuntimeEvent.Message(record,"id"))
 try{SubmitInputDto("x");error("invalid plain ctor accepted")}catch(_:IllegalArgumentException){}
 val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default);val channels=CopyOnWriteArrayList<ByteChannel>();var streams=0;var submits=0;var bad=false
 val engine=StreamingTestEngine(MockEngineConfig().apply{addHandler{request->when(request.url.encodedPath){
 "/submit"->{val tree=codec.parseToJsonElement(plainBody(request.body)) as JsonObject;check(tree["id"]==JsonPrimitive("ok"));when(submits++%3){0->check("amount" !in tree);1->check(tree["amount"]==JsonNull);2->check(tree["amount"]==JsonPrimitive("17"))};respond(recordJSON,HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
 "/amount"->{check(plainBody(request.body)=="17");respond("23",HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
 "/watch"->{streams++;val channel=ByteChannel(autoFlush=true);channels+=channel;val value=if(bad)"{\"amount\":false,\"values\":[23]}" else recordJSON;scope.launch{try{for(byte in "id: style\r\ndata: $value\r\n\r\n".encodeToByteArray())channel.writeFully(byteArrayOf(byte));awaitCancellation()}finally{channel.close()}};respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))}
 else->error("Unexpected route")}}})
 val http=HttpClient(engine){install(ContentNegotiation){json(codec)};install(SSE)}
 try{val client=SharedRuntimeExampleApiClient(http,"https://api.example.test/",codec);for(patch in listOf<Patch<Long>>(Patch.Absent,Patch.Null,Patch.Value(17L)))check(client.default.submit(SubmitInputDto("ok",patch))==record);check(client.default.amount(17L)==23L);val cold=client.default.watch();check(streams==0);repeat(2){check((cold.take(1).toList().single() as RuntimeEvent.Message).data==record)};bad=true;try{cold.take(1).toList();error("invalid plain SSE accepted")}catch(cause:Throwable){check(generateSequence(cause){it.cause}.any{it is SerializationException})};check(channels.all{it.isClosedForRead||it.isClosedForWrite});check(client.default.amount(17L)==23L);check(streams==3&&submits==3)}finally{http.close();scope.cancel()}
 println("PlainStyleContexts Json defaults=$defaults nulls=$nulls enum-unrecognized-singletons-primitive-object-closed-union-hierarchy-Unknown-Patch-HTTP-fragmentedSSE-close-reuse-cold checks passed")
 }
}

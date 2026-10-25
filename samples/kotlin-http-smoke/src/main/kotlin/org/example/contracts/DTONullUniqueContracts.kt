@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, io.ktor.utils.io.InternalAPI::class)
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
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.encodeToJsonElement
import java.util.concurrent.CopyOnWriteArrayList
import org.example.dtonullunique.api.DefaultApi
import org.example.dtonullunique.api.models.PatchInputDto
import org.example.dtonullunique.api.models.WatchEventDto
import org.example.dtonullunique.models.Patch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.dtonullunique.models.ItemDto
import org.example.dtonullunique.models.HolderDto
import org.example.dtonullunique.models.InlineHolderDto
import org.example.dtonullunique.models.NullableHolderDto

private class DTONullUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
 override val supportedCapabilities=super.supportedCapabilities+SSECapability
 override suspend fun execute(data:HttpRequestData):HttpResponseData {
  val response=super.execute(data)
  val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey)?:return response
  val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext)?:return response
  return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
 }
}
private fun dtoNullUniqueBody(body:OutgoingContent):String=when(body){
 is OutgoingContent.ContentWrapper->dtoNullUniqueBody(body.delegate())
 is OutgoingContent.ByteArrayContent->body.bytes().decodeToString()
 else->error("Unexpected synthetic body")
}

suspend fun dtoNullUniqueContracts(checked: Boolean = true) {
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  val first=ItemDto(requiredNull=JsonNull,id="a");val second=ItemDto(requiredNull=JsonNull,id="b")
  val valid=HolderDto(values=listOf(first,second))
  check(codec.decodeFromString<HolderDto>(codec.encodeToString(valid))==valid)
  check(first.requiredNull===JsonNull && first.fixedNull===JsonNull && first.typedFixedNull===JsonNull)
  check(first.optionalNull==null && first.nullEnum==null)
  val suppliedNull=ItemDto(requiredNull=JsonNull,id="a",optionalNull=JsonNull)
  check(HolderDto(values=listOf(first,suppliedNull)).values?.size==2)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"id":"a","requiredNull":"null"}""")}.isFailure)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"id":"a","requiredNull":null,"fixedNull":"null"}""")}.isFailure)
  check(first.kind=="fixed" && !first.pinnedFlag && first.pinnedNumber==0.0)
  val changed=codec.decodeFromString<ItemDto>("""{"requiredNull":null,"id":"a","kind":"changed"}""")
  check(changed.kind=="changed")
  check(codec.encodeToJsonElement(changed).toString().contains("changed"))
  check(HolderDto(values=listOf(first,changed)).values?.size==2)
  val sameBody=codec.decodeFromString<ItemDto>("""{"requiredNull":null,"id":"a","kind":"fixed","pinnedFlag":false,"pinnedNumber":-0.0}""")
  check(runCatching{HolderDto(values=listOf(first,sameBody))}.isFailure==checked)
  val changedFlag=codec.decodeFromString<ItemDto>("""{"requiredNull":null,"id":"a","pinnedFlag":true}""")
  check(changedFlag.pinnedFlag)
  check(HolderDto(values=listOf(first,changedFlag)).values?.size==2)
  val changedFloat=codec.decodeFromString<ItemDto>("""{"requiredNull":null,"id":"a","pinnedNumber":1.0}""")
  check(changedFloat.pinnedNumber==1.0)
  check(HolderDto(values=listOf(first,changedFloat)).values?.size==2)
  check(first.state?.serialName=="ready")
  check(codec.decodeFromString<ItemDto>("""{"requiredNull":null,"id":"default"}""").state?.serialName=="ready")
  val known=ItemDto.State.READY
  val unknown=ItemDto.State.Unrecognized("ready")
  check(known.serialName==unknown.serialName)
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="e",state=known),ItemDto(requiredNull=JsonNull,id="e",state=unknown)))}.isFailure==checked)
  check(HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="e",state=known),ItemDto(requiredNull=JsonNull,id="e",state=ItemDto.State.Unrecognized("future")))).values?.size==2)
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="n",state=null),ItemDto(requiredNull=JsonNull,id="n",state=null)))}.isFailure==checked)
  check(HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="n",state=null),ItemDto(requiredNull=JsonNull,id="n",state=known))).values?.size==2)
  if(checked) {
   check(runCatching{ItemDto(requiredNull=JsonNull,id="bad",version=2)}.isFailure)
   check(runCatching{ItemDto(requiredNull=JsonNull,id="bad",weight=2f)}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="z",weight=0f),ItemDto(requiredNull=JsonNull,id="z",weight=-0f)))}.isFailure)
  }
  val duplicate=runCatching{HolderDto(values=listOf(first,ItemDto(requiredNull=JsonNull,id="a",version=0)))}
  val repeatedNull=runCatching{NullableHolderDto(values=listOf(null,null))}
  check(duplicate.isFailure==checked);check(repeatedNull.isFailure==checked)
  val inline=codec.decodeFromString<InlineHolderDto>("""{"values":[{"requiredNull":null,"id":"a"},{"requiredNull":null,"id":"b"}]}""")
  check(codec.decodeFromString<InlineHolderDto>(codec.encodeToString(inline))==inline)
  check(runCatching{codec.decodeFromString<InlineHolderDto>("""{"values":[{"requiredNull":null,"id":"a"},{"requiredNull":null,"id":"a"}]}""")}.isFailure==checked)

  check(NullableHolderDto(values=listOf(null,first)).values?.size==2)
  val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
  val channels=CopyOnWriteArrayList<ByteChannel>()
  var bad=false;var calls=0;var callbacks=0
  var expectedPatch:JsonObject=JsonObject(emptyMap())
  val engine=DTONullUniqueEngine(MockEngineConfig().apply{addHandler{request->
   calls++;check(calls==callbacks)
   if(request.url.encodedPath=="/patch"){
    check(codec.parseToJsonElement(dtoNullUniqueBody(request.body))==expectedPatch)
    respond("",HttpStatusCode.NoContent)
   }else{
    check(codec.parseToJsonElement(dtoNullUniqueBody(request.body))==codec.encodeToJsonElement(valid))
    val payload=if(bad) """{"values":[{"requiredNull":null,"id":"a"},{"requiredNull":null,"id":"a","version":0}]}""" else codec.encodeToString(valid)
    if(request.url.encodedPath=="/watch"){
     val channel=ByteChannel(autoFlush=true);channels+=channel
     val frame="event: record\r\ndata: $payload\r\n\r\n"
     scope.launch{try{for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte))}finally{channel.close()}}
     respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
    }else respond(payload,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
   }
  }})
  val http=HttpClient(engine){install(SSE);install(ContentNegotiation){json(codec)}}
  try{
   val api=DefaultApi(http,"http://example.test/",codec)
   check(api.send(valid){callbacks++}==valid)
   val before=calls;val flow=api.watch(input=valid){callbacks++};check(calls==before)
   check((withTimeout(10000){flow.toList()}.single() as WatchEventDto.Record).data==valid)
   for((patch,expected) in listOf(PatchInputDto() to JsonObject(emptyMap()),PatchInputDto(value=Patch.Null) to JsonObject(mapOf("value" to JsonNull)),PatchInputDto(value=Patch.Value(valid)) to JsonObject(mapOf("value" to codec.encodeToJsonElement(valid))))){
    expectedPatch=expected;api.patchPatch(patch){callbacks++}
   }
   if(checked){bad=true;check(runCatching{api.send(valid){callbacks++}}.isFailure);check(runCatching{withTimeout(10000){api.watch(input=valid){callbacks++}.toList()}}.isFailure);bad=false}
   check(channels.all{it.isClosedForRead})
   check(api.send(valid){callbacks++}==valid)
  }finally{http.close();scope.cancel();channels.forEach{it.close()}}

  if(checked) {
   check(runCatching{codec.decodeFromString<HolderDto>("""{"values":[{"requiredNull":null,"id":"a"},{"requiredNull":null,"id":"a","version":0}]}""")}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="z",amount=0.0),ItemDto(requiredNull=JsonNull,id="z",amount=-0.0)))}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="z",weight=Float.NaN)))}.isFailure)
   val left=kotlin.uuid.Uuid.parse("550E8400-E29B-41D4-A716-446655440000")
   val right=kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredNull=JsonNull,id="u",uuid=left),ItemDto(requiredNull=JsonNull,id="u",uuid=right)))}.isFailure)
  }
 }
}

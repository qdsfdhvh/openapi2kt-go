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
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import java.util.concurrent.CopyOnWriteArrayList
import org.example.nullvaluedefaults.api.DefaultApi
import org.example.nullvaluedefaults.api.models.PatchInputDto
import org.example.nullvaluedefaults.api.models.WatchEventDto
import org.example.nullvaluedefaults.models.HolderDto
import org.example.nullvaluedefaults.models.ItemDto
import org.example.nullvaluedefaults.models.NoInheritHolderDto
import org.example.nullvaluedefaults.models.Patch
import org.example.nullvaluedefaults.models.RequiredHolderDto
import org.example.nullvaluedefaults.models.SeedDto
import org.example.nullvaluedefaults.models.`SeedAliasDto default`
import org.example.nullvaluedefaults.models.`NullRowsDto default`
import org.example.nullvaluedefaults.models.`NullRowsAliasDto default`
import org.example.nullvaluedefaults.models.`NullEnumRowsDto default`
import org.example.nullvaluedefaults.models.`NullTreeDto default`

private class NullValueDefaultsEngine(config: MockEngineConfig) : MockEngine(config) {
 override val supportedCapabilities=super.supportedCapabilities+SSECapability
 override suspend fun execute(data:HttpRequestData):HttpResponseData {
  val response=super.execute(data)
  val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey)?:return response
  val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext)?:return response
  return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
 }
}
private fun nullValueDefaultsBody(body:OutgoingContent):String=when(body){
 is OutgoingContent.ContentWrapper->nullValueDefaultsBody(body.delegate())
 is OutgoingContent.ByteArrayContent->body.bytes().decodeToString()
 else->error("Unexpected synthetic body")
}
private fun nullValueDefaultsValue(value:HolderDto) {
 check(value.inline?.id=="inline" && value.inline?.fixed===JsonNull)
 check(value.named?.id=="named" && value.named?.fixed===JsonNull)
 check(value.combined?.id=="combined" && value.combined?.fixed===JsonNull)
 check(value.rows?.single()?.id=="row" && value.rows?.single()?.fixed===JsonNull)
 check(value.tree?.get("one")?.single()?.id=="tree" && value.tree?.get("one")?.single()?.fixed===JsonNull)
 check(value.only?.fixed===JsonNull)
 check(value.inline?.requiredNull===JsonNull && value.named?.requiredNull===JsonNull)
 check(value.combined?.requiredNull===JsonNull && value.rows?.single()?.requiredNull===JsonNull)
 check(value.named?.optionalNull==null)
 check(value.nulls==listOf(JsonNull,JsonNull))
 check(value.enumNulls==listOf(null,null))
 check(value.nullTree?.get("one")==listOf(JsonNull,JsonNull))
}

suspend fun nullValueDefaultsContracts() {
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  val valid=HolderDto()
  nullValueDefaultsValue(valid)
  nullValueDefaultsValue(codec.decodeFromString<HolderDto>("{}"))
  nullValueDefaultsValue(codec.decodeFromString<HolderDto>(codec.encodeToString(HolderDto.serializer(),valid)))
  check(valid.named !== HolderDto().named)
  check(valid.only !== HolderDto().only)
  check(valid.rows !== HolderDto().rows)
  check(codec.encodeToJsonElement(ItemDto.serializer(),ItemDto(requiredNull=JsonNull,id="body")).jsonObject["fixed"]===JsonNull)
  check(SeedDto.default().id=="seed" && SeedDto.default().fixed===JsonNull)
  check(`SeedAliasDto default`().id=="alias" && `SeedAliasDto default`().fixed===JsonNull)
  check(NoInheritHolderDto().value==null && NoInheritHolderDto().rows==null)
  check(`NullRowsDto default`()==listOf(JsonNull))
  check(`NullRowsAliasDto default`()==listOf(JsonNull,JsonNull))
  check(`NullEnumRowsDto default`()==listOf(null))
  check(`NullTreeDto default`()==mapOf("one" to listOf(JsonNull)))
  check(`NullRowsDto default`() !== `NullRowsDto default`())
  check(`NullTreeDto default`()["one"] !== `NullTreeDto default`()["one"])
  check(RequiredHolderDto(value=ItemDto(requiredNull=JsonNull,id="provided")).value.fixed===JsonNull)
  check(runCatching{codec.decodeFromString<RequiredHolderDto>("{}")}.isFailure)
  check(runCatching{codec.decodeFromString<HolderDto>("""{"named":{"requiredNull":null,"id":"bad","fixed":"null"}}""")}.isFailure)
  check(runCatching{codec.decodeFromString<HolderDto>("""{"nulls":["null"]}""")}.isFailure)
  check(runCatching{codec.decodeFromString<HolderDto>("""{"nullTree":{"one":[false]}}""")}.isFailure)
  val replaced=codec.decodeFromString<HolderDto>("""{"named":{"requiredNull":null,"id":"replaced","fixed":null}}""")
  check(replaced.named?.id=="replaced" && replaced.named?.fixed===JsonNull)
  val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
  val channels=CopyOnWriteArrayList<ByteChannel>()
  var bad=false;var calls=0;var callbacks=0
  var expectedPatch:JsonObject=JsonObject(emptyMap())
  val engine=NullValueDefaultsEngine(MockEngineConfig().apply{addHandler{request->
   calls++;check(calls==callbacks)
   if(request.url.encodedPath=="/patch"){
    check(codec.parseToJsonElement(nullValueDefaultsBody(request.body))==expectedPatch)
    respond("",HttpStatusCode.NoContent)
   }else{
    check(codec.parseToJsonElement(nullValueDefaultsBody(request.body))==codec.encodeToJsonElement(HolderDto.serializer(),valid))
    val payload=if(bad) """{"named":{"requiredNull":null,"id":"bad","fixed":"null"}}""" else "{}"
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
   nullValueDefaultsValue(api.send(valid){callbacks++})
   val before=calls;val flow=api.watch(input=valid){callbacks++};check(calls==before)
   nullValueDefaultsValue((withTimeout(10000){flow.toList()}.single() as WatchEventDto.Record).data)
   for((patch,expected) in listOf(PatchInputDto() to JsonObject(emptyMap()),PatchInputDto(value=Patch.Null) to JsonObject(mapOf("value" to JsonNull)),PatchInputDto(value=Patch.Value(valid)) to JsonObject(mapOf("value" to codec.encodeToJsonElement(HolderDto.serializer(),valid))))){
    expectedPatch=expected;api.patchPatch(patch){callbacks++}
   }
   bad=true;check(runCatching{api.send(valid){callbacks++}}.isFailure)
   check(runCatching{withTimeout(10000){api.watch(input=valid){callbacks++}.toList()}}.isFailure);bad=false
   check(channels.all{it.isClosedForRead})
   nullValueDefaultsValue(api.send(valid){callbacks++})
  }finally{http.close();scope.cancel();channels.forEach{it.close()}}
 }
}

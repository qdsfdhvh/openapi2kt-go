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
import org.example.dtonativecollectionreferenceunique.api.DefaultApi
import org.example.dtonativecollectionreferenceunique.api.models.PatchInputDto
import org.example.dtonativecollectionreferenceunique.api.models.WatchEventDto
import org.example.dtonativecollectionreferenceunique.models.Patch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.dtonativecollectionreferenceunique.models.ItemDto
import org.example.dtonativecollectionreferenceunique.models.HolderDto
import org.example.dtonativecollectionreferenceunique.models.InlineHolderDto
import org.example.dtonativecollectionreferenceunique.models.NullableHolderDto

import org.example.dtonativecollectionreferenceunique.models.`RefTextDto default`

private class DTONativeCollectionReferenceUniqueEngine(config: MockEngineConfig) : MockEngine(config) {
 override val supportedCapabilities=super.supportedCapabilities+SSECapability
 override suspend fun execute(data:HttpRequestData):HttpResponseData {
  val response=super.execute(data)
  val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey)?:return response
  val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext)?:return response
  return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
 }
}
private fun dtoNativeCollectionReferenceUniqueBody(body:OutgoingContent):String=when(body){
 is OutgoingContent.ContentWrapper->dtoNativeCollectionReferenceUniqueBody(body.delegate())
 is OutgoingContent.ByteArrayContent->body.bytes().decodeToString()
 else->error("Unexpected synthetic body")
}

suspend fun dtoNativeCollectionReferenceUniqueContracts(checked: Boolean = true) {
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  val first=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a");val second=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="b")
  val valid=HolderDto(values=listOf(first,second))
  check(codec.decodeFromString<HolderDto>(codec.encodeToString(valid))==valid)
  check(first.numbers==listOf(0f,1f))
  check(first.refInt==0 && first.refFloat==0f && first.refFlag==false)
  check(first.refBorrow==1L && first.refBridge==0L && first.refChain==0)
  check(first.refText==null && `RefTextDto default`()=="source")
  check(first.refUuid==kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000"))
  check(first.refInstant==kotlin.time.Instant.parse("2024-01-01T00:00:00Z"))
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",refFloat=-0f)))}.isFailure==checked)
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",refUuid=kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000"))))}.isFailure==checked)
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",refInstant=kotlin.time.Instant.parse("2024-01-01T01:00:00+01:00"))))}.isFailure==checked)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",refInt=null))).values?.size==2)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"requiredNull":null,"id":"missing"}""")}.isFailure)
  if(checked)check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="badRef",refInt=-1)}.isFailure)

  check(first.states?.single()?.serialName=="ready")
  check(first.stateTree?.get("one")?.first()?.serialName=="ready" && first.stateTree?.get("one")?.last()==null)
  val stateKnown=ItemDto.StatesInner.READY
  val stateUnknown=ItemDto.StatesInner.Unrecognized("ready")
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",states=listOf(stateUnknown))))}.isFailure==checked)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",states=listOf(ItemDto.StatesInner.Unrecognized("future"))))).values?.size==2)
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="order",states=listOf(stateKnown,ItemDto.StatesInner.DONE)),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="order",states=listOf(stateUnknown,ItemDto.StatesInner.DONE))))}.isFailure==checked)
  check(HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="order",states=listOf(stateKnown,ItemDto.StatesInner.DONE)),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="order",states=listOf(ItemDto.StatesInner.DONE,stateKnown)))).values?.size==2)
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",stateTree=mapOf("one" to listOf(ItemDto.StateTreeValueInner.Unrecognized("ready"),null)))))}.isFailure==checked)

  check(first.nativeInts==listOf(0,null))
  check(first.nativeLongs==listOf(0L,1L))
  check(first.nativeFlags==mapOf("a" to false,"b" to true))
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nativeInts=listOf(0,null))))}.isFailure==checked)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nativeInts=listOf(null,0)))).values?.size==2)
  val flagsA=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="flags",nativeFlags=linkedMapOf("a" to false,"b" to true))
  val flagsB=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="flags",nativeFlags=linkedMapOf("b" to true,"a" to false))
  check(runCatching{HolderDto(values=listOf(flagsA,flagsB))}.isFailure==checked)
  val nativeTreeA=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nativeTree",nativeTree=mapOf("a" to listOf(0.0,1.0)))
  val nativeTreeB=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nativeTree",nativeTree=mapOf("a" to listOf(-0.0,1.0)))
  check(runCatching{HolderDto(values=listOf(nativeTreeA,nativeTreeB))}.isFailure==checked)
  check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="badEnum",numbers=listOf(2f))}.isFailure==checked)
  check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="badEnum",nativeLongs=listOf(2L))}.isFailure==checked)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"bad","nativeTree":{"one":[2]}}""")}.isFailure==checked)

  check(first.nulls==listOf(JsonNull) && first.enumNulls==listOf(null))
  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nulls=listOf(JsonNull))))}.isFailure==checked)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nulls=emptyList()))).values?.size==2)
  check(HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="length",nulls=listOf(JsonNull)),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="length",nulls=listOf(JsonNull,JsonNull)))).values?.size==2)
  val treeA=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nullmap",nullTree=linkedMapOf("a" to listOf(JsonNull),"b" to emptyList()))
  val treeB=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nullmap",nullTree=linkedMapOf("b" to emptyList(),"a" to listOf(JsonNull)))
  check(runCatching{HolderDto(values=listOf(treeA,treeB))}.isFailure==checked)
  val nullableGetter=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",enumNulls=listOf(JsonNull))
  check(HolderDto(values=listOf(first,nullableGetter)).values?.size==2)
  val collapsed=HolderDto(values=listOf(first,nullableGetter))
  check(runCatching{codec.decodeFromString<HolderDto>(codec.encodeToString(collapsed))}.isFailure==checked)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"bad","nulls":["null"]}""")}.isFailure)

  check(runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",numbers=listOf(-0f,1f))))}.isFailure==checked)
  val mapFirst=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="map",lookup=linkedMapOf("a" to 0f,"b" to 1f))
  val mapSame=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="map",lookup=linkedMapOf("b" to 1f,"a" to -0f))
  check(runCatching{HolderDto(values=listOf(mapFirst,mapSame))}.isFailure==checked)
  val nestedFirst=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nested",batches=listOf(linkedMapOf("a" to 0f,"b" to 1f)))
  val nestedSame=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nested",batches=listOf(linkedMapOf("b" to 1f,"a" to -0f)))
  check(runCatching{HolderDto(values=listOf(nestedFirst,nestedSame))}.isFailure==checked)
  check(HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="order",numbers=listOf(0f,1f)),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="order",numbers=listOf(1f,0f)))).values?.size==2)
  check(HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="empty",numbers=null),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="empty",numbers=emptyList()))).values?.size==2)
  val uuidLeft=kotlin.uuid.Uuid.parse("550E8400-E29B-41D4-A716-446655440000")
  val uuidRight=kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="ul",uuids=listOf(uuidLeft)),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="ul",uuids=listOf(uuidRight))))}.isFailure==checked)
  val instantLeft=kotlin.time.Instant.parse("2024-01-01T00:00:00Z")
  val instantRight=kotlin.time.Instant.parse("2024-01-01T01:00:00+01:00")
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="im",instants=mapOf("a" to instantLeft)),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="im",instants=mapOf("a" to instantRight))))}.isFailure==checked)
  if(checked){
   check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad",numbers=listOf(-1f))}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="nan",lookup=mapOf("a" to Float.NaN))))}.isFailure)
  }
  check(first.requiredNull===JsonNull && first.fixedNull===JsonNull && first.typedFixedNull===JsonNull)
  check(first.optionalNull==null && first.nullEnum==null)
  val suppliedNull=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",optionalNull=JsonNull)
  check(HolderDto(values=listOf(first,suppliedNull)).values?.size==2)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"id":"a","requiredNull":"null"}""")}.isFailure)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"id":"a","refRequired":0,"requiredNull":null,"fixedNull":"null"}""")}.isFailure)
  check(first.kind=="fixed" && !first.pinnedFlag && first.pinnedNumber==0.0)
  val changed=codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a","kind":"changed"}""")
  check(changed.kind=="changed")
  check(codec.encodeToJsonElement(changed).toString().contains("changed"))
  check(HolderDto(values=listOf(first,changed)).values?.size==2)
  val sameBody=codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a","kind":"fixed","pinnedFlag":false,"pinnedNumber":-0.0}""")
  check(runCatching{HolderDto(values=listOf(first,sameBody))}.isFailure==checked)
  val changedFlag=codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a","pinnedFlag":true}""")
  check(changedFlag.pinnedFlag)
  check(HolderDto(values=listOf(first,changedFlag)).values?.size==2)
  val changedFloat=codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a","pinnedNumber":1.0}""")
  check(changedFloat.pinnedNumber==1.0)
  check(HolderDto(values=listOf(first,changedFloat)).values?.size==2)
  check(first.state?.serialName=="ready")
  check(codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"default"}""").state?.serialName=="ready")
  val known=ItemDto.State.READY
  val unknown=ItemDto.State.Unrecognized("ready")
  check(known.serialName==unknown.serialName)
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="e",state=known),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="e",state=unknown)))}.isFailure==checked)
  check(HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="e",state=known),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="e",state=ItemDto.State.Unrecognized("future")))).values?.size==2)
  check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="n",state=null),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="n",state=null)))}.isFailure==checked)
  check(HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="n",state=null),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="n",state=known))).values?.size==2)
  if(checked) {
   check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad",version=2)}.isFailure)
   check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad",weight=2f)}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="z",weight=0f),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="z",weight=-0f)))}.isFailure)
  }
  val duplicate=runCatching{HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",version=0)))}
  check(first.list==listOf(0f,1f));check(first.map==mapOf("a" to 0f,"b" to 1f));check(first.deep==listOf(mapOf("a" to listOf(0,null))));check(first.bridge==listOf(true));check(first.borrowed==null);check(first.nullable==null);check(first.nonNullView==emptyList<Float>())
  val mapped=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",list=listOf(-0f,1f),map=linkedMapOf("b" to 1f,"a" to -0f),deep=listOf(mapOf("a" to listOf(0,null))))
  check(runCatching{HolderDto(values=listOf(first,mapped))}.isFailure==checked)
  val different=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",list=listOf(1f,0f))
  check(HolderDto(values=listOf(first,different)).values?.size==2)
  val borrowedTrue=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",borrowed=listOf(true))
  check(HolderDto(values=listOf(first,borrowedTrue)).values?.size==2)
  check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad-list",list=listOf(-1f))}.isFailure==checked)
  check(codec.decodeFromString<ItemDto>(codec.encodeToString(mapped))==mapped)
  check(first.specialUuid?.single()==kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000"))
  check(first.specialInstant?.get("t")==kotlin.time.Instant.parse("2024-01-01T01:00:00+01:00"))
  check(first.specialDays==listOf(kotlinx.datetime.LocalDate(2024,2,29)))
  check(first.specialAlias?.get("t")==kotlin.time.Instant.parse("2024-01-01T00:00:00Z"))
  check(first.specialBorrow==null);check(first.specialBridge==emptyList<kotlin.uuid.Uuid>())
  val mappedSpecial=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",specialUuid=listOf(kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000")),specialInstant=mapOf("t" to kotlin.time.Instant.parse("2024-01-01T01:00:00+01:00")))
  check(runCatching{HolderDto(values=listOf(first,mappedSpecial))}.isFailure==checked)
  check(codec.decodeFromString<ItemDto>(codec.encodeToString(mappedSpecial))==mappedSpecial)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",specialInstant=mapOf("t" to kotlin.time.Instant.parse("2024-01-01T00:00:01Z"))))).values?.size==2)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",specialBorrow=emptyList()))).values?.size==2)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"bad","specialUuid":["invalid"]}""")}.isFailure)
  check(first.nativeRefInts==listOf(0));check(first.nativeRefLongs==listOf(0L));check(first.nativeRefFloats==listOf(0f));check(first.nativeRefDoubles==mapOf("a" to 0.0,"b" to 1.0));check(first.nativeRefFlags==mapOf("a" to true,"b" to true));check(first.nativeRefBorrow==null);check(first.nativeRefBridge==listOf(0f));check(first.nativeRefAlias==listOf(0f))
  val mappedNative=ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nativeRefFloats=listOf(-0f),nativeRefDoubles=linkedMapOf("b" to 1.0,"a" to -0.0),nativeRefFlags=linkedMapOf("b" to true,"a" to true),nativeRefAlias=listOf(-0f))
  check(runCatching{HolderDto(values=listOf(first,mappedNative))}.isFailure==checked)
  check(codec.decodeFromString<ItemDto>(codec.encodeToString(mappedNative))==mappedNative)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nativeRefInts=listOf(1)))).values?.size==2)
  check(HolderDto(values=listOf(first,ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="a",nativeRefBorrow=emptyList()))).values?.size==2)
  check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad-native",nativeRefFloats=listOf(2f))}.isFailure==checked)
  check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad-native",nativeRefLongs=listOf(2L))}.isFailure==checked)
  check(runCatching{ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="bad-native",nativeRefFlags=mapOf("a" to false))}.isFailure==checked)
  check(runCatching{codec.decodeFromString<ItemDto>("""{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"bad","nativeRefInts":[2]}""")}.isFailure==checked)
  val repeatedNull=runCatching{NullableHolderDto(values=listOf(null,null))}
  check(duplicate.isFailure==checked);check(repeatedNull.isFailure==checked)
  val inline=codec.decodeFromString<InlineHolderDto>("""{"values":[{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a"},{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"b"}]}""")
  check(codec.decodeFromString<InlineHolderDto>(codec.encodeToString(inline))==inline)
  check(runCatching{codec.decodeFromString<InlineHolderDto>("""{"values":[{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a"},{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a"}]}""")}.isFailure==checked)

  check(NullableHolderDto(values=listOf(null,first)).values?.size==2)
  val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
  val channels=CopyOnWriteArrayList<ByteChannel>()
  var bad=false;var calls=0;var callbacks=0
  var expectedPatch:JsonObject=JsonObject(emptyMap())
  val engine=DTONativeCollectionReferenceUniqueEngine(MockEngineConfig().apply{addHandler{request->
   calls++;check(calls==callbacks)
   if(request.url.encodedPath=="/patch"){
    check(codec.parseToJsonElement(dtoNativeCollectionReferenceUniqueBody(request.body))==expectedPatch)
    respond("",HttpStatusCode.NoContent)
   }else{
    check(codec.parseToJsonElement(dtoNativeCollectionReferenceUniqueBody(request.body))==codec.encodeToJsonElement(valid))
    val payload=if(bad) """{"values":[{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a"},{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a","version":0,"nulls":[null]}]}""" else codec.encodeToString(valid)
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
   check(runCatching{codec.decodeFromString<HolderDto>("""{"values":[{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a"},{"requiredList":[3],"refRequired":0,"requiredNull":null,"id":"a","version":0,"nulls":[null]}]}""")}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="z",amount=0.0),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="z",amount=-0.0)))}.isFailure)
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="z",weight=Float.NaN)))}.isFailure)
   val left=kotlin.uuid.Uuid.parse("550E8400-E29B-41D4-A716-446655440000")
   val right=kotlin.uuid.Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
   check(runCatching{HolderDto(values=listOf(ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="u",uuid=left),ItemDto(requiredList=listOf(3f),refRequired=0,requiredNull=JsonNull,id="u",uuid=right)))}.isFailure)
  }
 }
}

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
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import org.example.nullablebridge.BridgeClient
import org.example.nullablebridge.models.ConsumerDto
import org.example.nullablebridge.models.ValueDto
import org.example.nullablebridge.models.BridgeDto
import org.example.nullablebridge.models.Patch
import org.example.nullablebridge.api.models.PatchInputDto
import org.example.nullablebridge.api.models.WatchEventDto
private class NullableBridgeIntersectionEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun nullableBridgeIntersectionBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> nullableBridgeIntersectionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected allOf String request content")
}


suspend fun nullableBridgeIntersectionsContracts(checked: Boolean = false) {
 for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  val none: ConsumerDto?=null
  check(codec.encodeToString(none)=="null" && codec.decodeFromString<ConsumerDto?>("null")==null)
  check(codec.decodeFromString<BridgeDto>("null")==null)
  val initial=ConsumerDto()
  check(initial.id=="base" && initial.extra=="extra" && initial.mode?.serialName=="ready")
  val node=ConsumerDto.Node(2)
  val value=ConsumerDto(id="雪🍃",node=node,mode=ConsumerDto.Mode.Unrecognized("future"),next=ValueDto(id="recursive"))
  check(codec.decodeFromString<ConsumerDto>(codec.encodeToString(value))==value)
  check(codec.decodeFromString<ConsumerDto>("{}")==initial)
  check(codec.decodeFromString<BridgeDto>("{}")?.id=="base")
  check(codec.decodeFromString<ValueDto>(codec.encodeToString<BridgeDto>(ValueDto(id="bridge"))).id=="bridge")
  check(codec.encodeToJsonElement(initial).jsonObject.containsKey("id")==defaults)
  if(checked) {
   check(runCatching {ConsumerDto.Node(0)}.isFailure)
   check(runCatching {codec.decodeFromString<ConsumerDto>("""{"node":{"count":0}}""")}.isFailure)
  }
  val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
  val channels=CopyOnWriteArrayList<ByteChannel>()
  var calls=0;var callbacks=0;var bad=false;var nullWire=false
  var patchExpected: JsonObject=JsonObject(emptyMap())
  val engine=NullableBridgeIntersectionEngine(MockEngineConfig().apply{addHandler{request->
   calls++;check(calls==callbacks)
   if(request.url.encodedPath=="/patch") {
    check(codec.parseToJsonElement(nullableBridgeIntersectionBody(request.body))==patchExpected)
    respond("",HttpStatusCode.NoContent)
   } else {
    check(codec.parseToJsonElement(nullableBridgeIntersectionBody(request.body))==(if(nullWire) JsonNull else codec.encodeToJsonElement(value)))
    if(request.url.encodedPath=="/watch") {
     val channel=ByteChannel(autoFlush=true);channels+=channel
     val payload=if(nullWire) "null" else if(bad) """{"node":{"count":0}}""" else codec.encodeToString(value)
     val frame="event: record\r\ndata: $payload\r\n\r\n"
     scope.launch{try{for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte))}finally{channel.close()}}
     respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
    } else respond(if(nullWire) "null" else codec.encodeToString(value),HttpStatusCode.OK,headersOf("Content-Type","application/json"))
   }
  }})
  val http=HttpClient(engine){install(SSE);install(ContentNegotiation){json(codec)}}
  try {
   val api=BridgeClient(http,"http://example.test/",codec).default
   check(api.send(value){callbacks++}==value)
   nullWire=true
   check(api.send(null){callbacks++}==null)
   check((withTimeout(10000){api.watch(input=null){callbacks++}.toList()}.single() as WatchEventDto.Record).data==null)
   nullWire=false
   val before=calls;val flow=api.watch(input=value){callbacks++};check(calls==before)
   check((withTimeout(10000){flow.toList()}.single() as WatchEventDto.Record).data==value)
   for((patch,expected) in listOf(PatchInputDto() to JsonObject(emptyMap()),PatchInputDto(value=Patch.Null) to JsonObject(mapOf("value" to JsonNull)),PatchInputDto(value=Patch.Value(value)) to JsonObject(mapOf("value" to codec.encodeToJsonElement(value))))) {
    patchExpected=expected;api.patchPatch(patch){callbacks++}
   }
   if(checked){bad=true;check(runCatching{withTimeout(10000){api.watch(input=value){callbacks++}.toList()}}.isFailure);bad=false}
   check(channels.all{it.isClosedForRead})
   check(api.send(value){callbacks++}==value)
  } finally{http.close();scope.cancel();channels.forEach{it.close()}}
 }
}

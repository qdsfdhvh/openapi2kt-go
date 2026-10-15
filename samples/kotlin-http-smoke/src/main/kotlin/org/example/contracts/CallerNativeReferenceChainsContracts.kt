@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
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
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import kotlin.time.Instant
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.nativechains.CallerNativeReferenceChainsApiClient
import org.example.nativechains.api.models.SubmitInputDto
import org.example.nativechains.api.models.WatchEventDto
import org.example.nativechains.api.models.WatchNarrowEventDto
import org.example.nativechains.models.MoneyBridgeDto
import org.example.nativechains.models.MoneyFalseBridgeDto
import org.example.nativechains.models.AgainBridgeDto
import org.example.nativechains.models.NativeChoiceDto
import org.example.nativechains.models.HolderDto
import org.example.nativechains.models.Patch
import org.example.scalarvalues.Money

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NativeChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun nativeChainBody(body: OutgoingContent): ByteArray = when (body) {
    is OutgoingContent.ContentWrapper -> nativeChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes()
    else -> error("Unexpected synthetic builtin body")
}

suspend fun callerNativeReferenceChainsContracts() = withTimeout(20000) {
    check(serializer<MoneyBridgeDto>().descriptor.isNullable && serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<MoneyFalseBridgeDto>().descriptor.isNullable && !MoneyBridgeDto.serializer().descriptor.isNullable)
    val ready=MoneyFalseBridgeDto("12.50");val future=MoneyBridgeDto("caller-owned-future")
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        for(value in listOf(null,ready,future)) {
            val wire=codec.encodeToString<MoneyBridgeDto>(value)
            check(codec.parseToJsonElement(wire)==(value?.let {JsonPrimitive(it.decimal)}?:JsonNull))
            check(codec.decodeFromString<MoneyBridgeDto>(wire)==value)
            check(codec.decodeFromString(Money.serializer().nullable,wire)==value)
        }
        check(runCatching {codec.decodeFromString<MoneyBridgeDto>("{}")}.isFailure)
        check(runCatching {codec.decodeFromString<MoneyFalseBridgeDto>("null")}.isFailure)
        val holder=HolderDto(inherited=null,forcedTrue=future,forcedFalse=ready,wrapper=null,wrapperFalse=ready,items=listOf(null,ready),values=mapOf("nil" to null,"ready" to ready),requiredNull=null,nativeLong=Long.MAX_VALUE,nativeBytes=byteArrayOf(-128,-1,0,127),nativeElement=JsonPrimitive("element"),nativeObject=JsonObject(mapOf("future" to JsonNull)),nativeString="text",nativeInstant=Instant.parse("2024-02-29T12:34:56Z"),directMoney=null,directFalse=ready,directWrapped=ready)
        val tree=codec.encodeToJsonElement(holder)
        check(tree.jsonObject["nativeLong"]==JsonPrimitive(Long.MAX_VALUE))
        check(tree.jsonObject["nativeBytes"]==JsonArray(listOf(-128,-1,0,127).map {JsonPrimitive(it)}))
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)))==tree)
        val choice:NativeChoiceDto=NativeChoiceDto.A(ready)
        check(codec.encodeToJsonElement<NativeChoiceDto>(choice).jsonObject["value"]==JsonPrimitive("12.50"))
        check(codec.encodeToJsonElement<NativeChoiceDto>(codec.decodeFromString<NativeChoiceDto>(codec.encodeToString<NativeChoiceDto>(choice)))==codec.encodeToJsonElement<NativeChoiceDto>(choice))
        val patches=listOf(SubmitInputDto(items=emptyList(),values=emptyMap(),requiredNull=null),SubmitInputDto(items=emptyList(),values=emptyMap(),requiredNull=null,inherited=Patch.Null,wrapper=Patch.Null),SubmitInputDto(items=emptyList(),values=emptyMap(),requiredNull=null,inherited=Patch.Value(future),wrapper=Patch.Value(ready),nativeLong=Patch.Value(Long.MAX_VALUE),nativeBytes=Patch.Value(byteArrayOf(-1)),nativeElement=Patch.Value(JsonNull)))
        check("inherited" !in codec.encodeToJsonElement(patches[0]).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["inherited"]==JsonNull)
        check(codec.encodeToJsonElement(patches[2]).jsonObject["inherited"]==JsonPrimitive(future.decimal))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(patches[2])).nativeElement===Patch.Null)
        for(patch in patches)check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(patch)))==codec.encodeToJsonElement(patch))
        var failure="";var expected="null".encodeToByteArray();var rawResponse:ByteArray?=null
        val channels=CopyOnWriteArrayList<ByteChannel>();val writers=CopyOnWriteArrayList<Job>()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=NativeChainStreamingEngine(MockEngineConfig().apply {
            addHandler {request ->
                when(request.url.encodedPath) {
                    "/watch","/watch-narrow" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val narrow=request.url.encodedPath=="/watch-narrow"
                        val name=if(narrow)"narrow" else "money"
                        val data=when(failure) {"null"->"null";"kind"->"{}";"syntax"->"{";else->"\"12.50\""}
                        val frame=if(!narrow && failure.isEmpty())"event: money\r\ndata: null\r\n\r\nevent: money\r\ndata: $data\r\n\r\n" else "event: $name\r\ndata: $data\r\n\r\n"
                        writers.add(scope.launch {try {for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte));awaitCancellation()}catch(closed:ClosedByteChannelException){if(!channel.isClosedForWrite)throw closed}})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    "/optional","/submit" -> {check(nativeChainBody(request.body).contentEquals(expected));respond("",HttpStatusCode.NoContent)}
                    else -> {check(nativeChainBody(request.body).contentEquals(expected));respond(rawResponse?:expected,HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
                }
            }
        })
        val http=HttpClient(engine){install(ContentNegotiation){json(codec)};install(SSE)}
        try {
            val api=CallerNativeReferenceChainsApiClient(http,"https://example.test/",codec).default
            check(api.echo(null)==null);api.optional()
            expected="\"12.50\"".encodeToByteArray();check(api.echo(ready)==ready && api.echo2(ready)==ready);api.optional(ready)
            expected="null".encodeToByteArray();check(api.echoLong(null)==null)
            expected=Long.MAX_VALUE.toString().encodeToByteArray();check(api.echoLong(Long.MAX_VALUE)==Long.MAX_VALUE)
            expected="null".encodeToByteArray();check(api.echoBytes(null)?.contentEquals(expected)==true && api.echoText(null)=="null")
            expected=byteArrayOf(-128,-1,0,127);check(api.echoBytes(expected)?.contentEquals(expected)==true)
            for(raw in listOf("{}","[128]","{")){rawResponse=raw.encodeToByteArray();check(api.echoBytes(expected)?.contentEquals(rawResponse)==true)}
            rawResponse=null;expected="raw-text".encodeToByteArray();check(api.echoText("raw-text")=="raw-text")
            expected="null".encodeToByteArray();check(api.echoElement(null)==null && api.echoObject(null)==null && api.echoInstant(null)==null)
            val raw:JsonElement=JsonArray(listOf(JsonNull,JsonPrimitive(1)))
            expected=codec.encodeToString(JsonElement.serializer(),raw).encodeToByteArray();check(api.echoElement(raw)==raw)
            val obj=JsonObject(mapOf("future" to JsonNull));expected=codec.encodeToString(JsonObject.serializer(),obj).encodeToByteArray();check(api.echoObject(obj)==obj)
            val stamp=Instant.parse("2024-02-29T12:34:56Z");expected="\"2024-02-29T12:34:56Z\"".encodeToByteArray();check(api.echoInstant(stamp)==stamp)
            for(patch in patches){expected=codec.encodeToString(patch).encodeToByteArray();api.submit(patch)}
            expected="\"12.50\"".encodeToByteArray();rawResponse="{}".encodeToByteArray();check(runCatching {api.echo(ready)}.isFailure);rawResponse=null
            val cold=api.watch();check(channels.isEmpty())
            repeat(2){val events=cold.take(2).toList();check((events[0] as WatchEventDto.Money).data==null && (events[1] as WatchEventDto.Money).data==ready);check(channels.last().isClosedForWrite)}
            check((api.watchNarrow().take(1).toList().single() as WatchNarrowEventDto.Narrow).data==ready);check(channels.last().isClosedForWrite)
            for(bad in listOf("null","kind","syntax")){failure=bad;check(runCatching{api.watchNarrow().take(1).toList()}.isFailure);check(channels.last().isClosedForWrite)}
            failure="";check((cold.take(1).toList().single() as WatchEventDto.Money).data==null);check(channels.last().isClosedForWrite)
            check(api.echo2(ready)==ready)
        }finally{http.close();scope.cancel();writers.forEach{it.join()}}
        println("CallerNativeReferenceChains Json defaults=$defaults nulls=$nulls providerAPI-codecs-Patch-nativeLong-rawStringBytes-JSON-Instant-SSEclose-reuse-cold passed")
    }
}

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
import kotlinx.serialization.builtins.ByteArraySerializer
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.byteschains.BytesReferenceChainsApiClient
import org.example.byteschains.api.models.SubmitInputDto
import org.example.byteschains.api.models.WatchEventDto
import org.example.byteschains.models.BytesBridgeDto
import org.example.byteschains.models.BytesFalseBridgeDto
import org.example.byteschains.models.AgainBridgeDto
import org.example.byteschains.models.BytesChoiceDto
import org.example.byteschains.models.HolderDto
import org.example.byteschains.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BytesChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun bytesChainBody(body: OutgoingContent): ByteArray = when (body) {
    is OutgoingContent.ContentWrapper -> bytesChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes()
    else -> error("Unexpected synthetic builtin body")
}

suspend fun bytesReferenceChainsContracts() = withTimeout(20000) {
    check(serializer<BytesBridgeDto>().descriptor.isNullable && serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<BytesFalseBridgeDto>().descriptor.isNullable)
    val ready=byteArrayOf(-128,-1,0,127);val future=byteArrayOf(1,2);val binary=byteArrayOf(4)
    val nullBytes="null".encodeToByteArray()
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        for(value in listOf(null,ready,future)) {
            val wire=codec.encodeToString<BytesBridgeDto>(value)
            check(codec.parseToJsonElement(wire)==(value?.let {JsonArray(it.map {n->JsonPrimitive(n)})}?:JsonNull))
            val back=codec.decodeFromString<BytesBridgeDto>(wire)
            val adapter=codec.decodeFromString(ByteArraySerializer().nullable,wire)
            check(if(value==null)back==null && adapter==null else back?.contentEquals(value)==true && adapter?.contentEquals(value)==true)
        }
        for(bad in listOf("{}","\"base64\"","[128]","[-129]","[null]","{")) check(runCatching {codec.decodeFromString<BytesBridgeDto>(bad)}.isFailure)
        check(runCatching {codec.decodeFromString<BytesFalseBridgeDto>("null")}.isFailure)
        val holder=HolderDto(null,ready,future,null,ready,listOf(null,ready,future),mapOf("nil" to null,"future" to future),null,binary,binary,ready,null,ready,null,ready,null,binary)
        val tree=codec.encodeToJsonElement(holder)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)))==tree)
        val choice:BytesChoiceDto=BytesChoiceDto.A(ready)
        val choiceTree=codec.encodeToJsonElement<BytesChoiceDto>(choice)
        check(choiceTree.jsonObject["value"]==JsonArray(ready.map {JsonPrimitive(it)}))
        check(codec.encodeToJsonElement<BytesChoiceDto>(codec.decodeFromString<BytesChoiceDto>(codec.encodeToString<BytesChoiceDto>(choice)))==choiceTree)
        val patches=listOf(SubmitInputDto(requiredNull=null,requiredFalse=ready),SubmitInputDto(requiredNull=null,requiredFalse=ready,inherited=Patch.Null,wrapper=Patch.Null),SubmitInputDto(requiredNull=null,requiredFalse=ready,inherited=Patch.Value(future),wrapper=Patch.Value(ready),binary=Patch.Value(binary)))
        check("inherited" !in codec.encodeToJsonElement(patches[0]).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["inherited"]==JsonNull)
        check(codec.encodeToJsonElement(patches[2]).jsonObject["inherited"]==JsonArray(future.map {JsonPrimitive(it)}))
        for(patch in patches)check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(patch)))==codec.encodeToJsonElement(patch))
        for(bad in listOf("{}","[128]","[null]"))check(runCatching {codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":[1],\"inherited\":$bad}")}.isFailure)
        var failure="";var expectedRaw=nullBytes;var rawResponse:ByteArray?=null
        var expectedPatch=codec.encodeToJsonElement(patches[0])
        val channels=CopyOnWriteArrayList<ByteChannel>();val writers=CopyOnWriteArrayList<Job>()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=BytesChainStreamingEngine(MockEngineConfig().apply {
            addHandler {request ->
                when(request.url.encodedPath) {
                    "/alias","/direct","/optional","/binary","/binary-direct" -> {
                        check(bytesChainBody(request.body).contentEquals(expectedRaw))
                        respond(rawResponse?:expectedRaw,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> {check(codec.parseToJsonElement(bytesChainBody(request.body).decodeToString())==expectedPatch);respond("",HttpStatusCode.NoContent)}
                    "/events" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val frame=when(failure) {
                            "null"->"event: narrow\r\ndata: null\r\n\r\n"
                            "range"->"event: nullable\r\ndata: [128]\r\n\r\n"
                            "kind"->"event: nullable\r\ndata: {}\r\n\r\n"
                            "element"->"event: binary\r\ndata: [null]\r\n\r\n"
                            "syntax"->"event: nullable\r\ndata: {\r\n\r\n"
                            else->"event: nullable\r\ndata: null\r\n\r\nevent: narrow\r\ndata: [-128,-1,0,127]\r\n\r\nevent: binary\r\ndata: null\r\n\r\nevent: binaryNarrow\r\ndata: [4]\r\n\r\n"
                        }
                        writers.add(scope.launch {try {for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte));awaitCancellation()}catch(closed:ClosedByteChannelException){if(!channel.isClosedForWrite)throw closed}})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else->error("Unexpected synthetic bytes route")
                }
            }
        })
        val http=HttpClient(engine){install(ContentNegotiation){json(codec)};install(SSE)}
        try {
            val api=BytesReferenceChainsApiClient(http,"https://example.test/",codec).default
            check(api.echo(null)?.contentEquals(nullBytes)==true && api.optional()?.contentEquals(nullBytes)==true)
            for(value in listOf(ready,future,nullBytes)){expectedRaw=value;check(api.echo(value)?.contentEquals(value)==true && api.echo2(value).contentEquals(value) && api.optional(value)?.contentEquals(value)==true)}
            expectedRaw=nullBytes;check(api.echoBinary(null)?.contentEquals(nullBytes)==true)
            expectedRaw=binary;check(api.echoBinary(binary)?.contentEquals(binary)==true && api.echoBinary2(binary).contentEquals(binary))
            for(patch in patches){expectedPatch=codec.encodeToJsonElement(patch);api.submit(patch)}
            expectedRaw=ready
            for(raw in listOf("{}","[128]","{")){rawResponse=raw.encodeToByteArray();check(api.echo(ready)?.contentEquals(rawResponse)==true)}
            rawResponse=null
            val cold=api.watch();check(channels.isEmpty())
            repeat(2){
                val events=cold.take(4).toList()
                check((events[0] as WatchEventDto.Nullable).data==null)
                check((events[1] as WatchEventDto.Narrow).data.contentEquals(ready))
                check((events[2] as WatchEventDto.Binary).data==null)
                check((events[3] as WatchEventDto.BinaryNarrow).data.contentEquals(binary))
                check(channels.last().isClosedForWrite)
            }
            for(bad in listOf("null","range","kind","element","syntax")){failure=bad;check(runCatching{cold.take(1).toList()}.isFailure);check(channels.last().isClosedForWrite)}
            failure="";check((cold.take(1).toList().single() as WatchEventDto.Nullable).data==null);check(channels.last().isClosedForWrite)
            check(api.echo2(ready).contentEquals(ready))
        }finally{http.close();scope.cancel();writers.forEach{it.join()}}
        println("BytesReferenceChains Json defaults=$defaults nulls=$nulls signedarray-model-Patch-SSE-rawHTTP-nullASCII-close-reuse-cold passed")
    }
}

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
import org.example.builtinchains.BuiltinReferenceChainsApiClient
import org.example.builtinchains.api.models.PatchBuiltinInputDto
import org.example.builtinchains.api.models.WatchBuiltinEventDto
import org.example.builtinchains.models.AgainBridgeDto
import org.example.builtinchains.models.BuiltinChoiceDto
import org.example.builtinchains.models.CountBridgeDto
import org.example.builtinchains.models.FlagBridgeDto
import org.example.builtinchains.models.HolderDto
import org.example.builtinchains.models.MeasureBridgeDto
import org.example.builtinchains.models.Patch
import org.example.builtinchains.models.RatioBridgeDto
import org.example.builtinchains.models.TextBridgeDto
import org.example.builtinchains.models.TextBridgeFalseDto
import org.example.builtinchains.models.WideBridgeDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BuiltinChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun builtinChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> builtinChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic builtin body")
}

suspend fun builtinReferenceChainsContracts(checked: Boolean = false) = withTimeout(20000) {
    check(serializer<TextBridgeDto>().descriptor.isNullable)
    check(serializer<FlagBridgeDto>().descriptor.isNullable)
    check(serializer<CountBridgeDto>().descriptor.isNullable)
    check(serializer<WideBridgeDto>().descriptor.isNullable)
    check(serializer<RatioBridgeDto>().descriptor.isNullable)
    check(serializer<MeasureBridgeDto>().descriptor.isNullable)
    check(serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<TextBridgeFalseDto>().descriptor.isNullable)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<TextBridgeDto>("null") == null)
        check(codec.decodeFromString<WideBridgeDto>(Long.MAX_VALUE.toString()) == Long.MAX_VALUE)
        check(codec.decodeFromString<CountBridgeDto>(Int.MIN_VALUE.toString()) == Int.MIN_VALUE)
        check(codec.encodeToString<FlagBridgeDto>(true) == "true")
        check(codec.decodeFromString<RatioBridgeDto>("1.5") == 1.5f)
        check(codec.decodeFromString<MeasureBridgeDto>("2.5") == 2.5)
        check(codec.decodeFromString<TextBridgeFalseDto>("\"ok\"") == "ok")
        check(runCatching { codec.decodeFromString<TextBridgeFalseDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<WideBridgeDto>("[]") }.isFailure)
        val holder = HolderDto(null,true,4,4L,1.5f,2.5,"ok","ok",listOf(4L),mapOf("value" to null))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        val choice: BuiltinChoiceDto = BuiltinChoiceDto.Count(4)
        check(codec.decodeFromString<BuiltinChoiceDto>(codec.encodeToString<BuiltinChoiceDto>(choice)) == choice)
        val patches = listOf(PatchBuiltinInputDto(count=null,narrow="ok"),PatchBuiltinInputDto(count=null,narrow="ok",text=Patch.Null,wide=Patch.Null),PatchBuiltinInputDto(count=4,narrow="ok",text=Patch.Value("ok"),wide=Patch.Value(4L)))
        check("text" !in codec.encodeToJsonElement(patches[0]).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["text"] == JsonNull)
        for (patch in patches) check(codec.decodeFromString<PatchBuiltinInputDto>(codec.encodeToString(patch)) == patch)
        if (checked) {
            check(runCatching { HolderDto(null,true,3,4L,1.5f,2.5,"ok",null,emptyList(),emptyMap()) }.isFailure)
            check(runCatching { HolderDto(null,true,4,4L,1.5f,2.5,"x",null,emptyList(),emptyMap()) }.isFailure)
            check(runCatching { HolderDto(null,true,4,4L,1.5f,2.5,"ok",null,listOf(1L),emptyMap()) }.isFailure)
            check(runCatching { PatchBuiltinInputDto(count=4,narrow="ok",text=Patch.Value("x")) }.isFailure)
            check(runCatching { codec.decodeFromString<PatchBuiltinInputDto>("{\"count\":3,\"narrow\":\"ok\"}") }.isFailure)
        }
        var failure = ""
        var expectedPatch = codec.encodeToJsonElement(patches[0])
        var calls = 0
        var queryText: String? = "ok"
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BuiltinChainStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/holder" -> { check(codec.decodeFromString<HolderDto>(builtinChainBody(request.body)) == holder); respond(codec.encodeToString(holder),HttpStatusCode.OK,headersOf("Content-Type","application/json")) }
                    "/text", "/text-again" -> { check(builtinChainBody(request.body) == "ok"); respond(if(failure=="text") "x" else "ok",HttpStatusCode.OK,headersOf("Content-Type","application/json")) }
                    "/count" -> { check(builtinChainBody(request.body) == "4"); respond(when(failure) {"count" -> "3"; "null" -> "null"; else -> "4"},HttpStatusCode.OK,headersOf("Content-Type","application/json")) }
                    "/query" -> { check(request.url.parameters["wide"] == "4"); check(request.url.parameters["text"] == queryText); respond("",HttpStatusCode.NoContent) }
                    "/patch" -> { check(codec.parseToJsonElement(builtinChainBody(request.body)) == expectedPatch); respond("",HttpStatusCode.NoContent) }
                    "/watch" -> {
                        val channel = ByteChannel(autoFlush=true); channels.add(channel)
                        val frames = when(failure) {
                            "count" -> "event: count\r\ndata: 3\r\n\r\n"
                            "null" -> "event: count\r\ndata: null\r\n\r\n"
                            "kind" -> "event: count\r\ndata: []\r\n\r\n"
                            "syntax" -> "event: count\r\ndata: {\r\n\r\n"
                            else -> "event: wide\r\ndata: null\r\n\r\nevent: count\r\ndata: 4\r\n\r\n"
                        }
                        writers.add(scope.launch {
                            try { for(byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() }
                            catch(closed: ClosedByteChannelException) { if(!channel.isClosedForWrite) throw closed }
                        })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic builtin route")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = BuiltinReferenceChainsApiClient(http,"https://example.test/",codec).default
            check(api.echoBuiltinHolder(holder) == holder)
            check(api.echoBuiltin("ok") == "ok" && api.echoBuiltin2("ok") == "ok" && api.echoBuiltin(4) == 4)
            api.queryBuiltin(4L,"ok"); queryText=null; api.queryBuiltin(4L)
            for(patch in patches) { expectedPatch=codec.encodeToJsonElement(patch); api.patchBuiltin(patch) }
            if(checked) {
                val before=calls
                check(runCatching { api.echoBuiltin("x") }.isFailure)
                check(runCatching { api.echoBuiltin(3) }.isFailure)
                check(runCatching { api.queryBuiltin(1L) }.isFailure)
                check(calls==before)
                failure="text";check(runCatching { api.echoBuiltin("ok") }.isFailure)
                failure="count";check(runCatching { api.echoBuiltin(4) }.isFailure)
            }
            failure="null";check(runCatching { api.echoBuiltin(4) }.isFailure);failure=""
            val cold=api.watchBuiltin();check(channels.isEmpty())
            repeat(2) {
                val events=cold.take(2).toList()
                check((events[0] as WatchBuiltinEventDto.Wide).data == null)
                check((events[1] as WatchBuiltinEventDto.Count).data == 4)
                check(channels.last().isClosedForWrite)
            }
            for(bad in if(checked) listOf("count","null","kind","syntax") else listOf("null","kind","syntax")) {
                failure=bad;check(runCatching { cold.take(1).toList() }.isFailure);check(channels.last().isClosedForWrite)
            }
            failure="";check((cold.take(1).toList().single() as WatchBuiltinEventDto.Wide).data == null);check(channels.last().isClosedForWrite)
            check(api.echoBuiltin(4) == 4)
        } finally { http.close();scope.cancel();writers.forEach { it.join() } }
        println("BuiltinReferenceChains Json defaults=$defaults nulls=$nulls checked=$checked native-null-false-Patch-query-erasure-checks-HTTP-liveSSEclose-reuse-cold passed")
    }
}

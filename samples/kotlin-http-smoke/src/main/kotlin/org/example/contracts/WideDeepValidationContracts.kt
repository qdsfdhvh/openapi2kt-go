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
import io.ktor.client.request.forms.FormDataContent
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.http.Parameters
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
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json


import org.example.widedeep.api.DefaultApi
import org.example.widedeep.api.models.SendInputDto
import org.example.widedeep.api.models.WatchEventDto
import org.example.widedeep.models.WideSixteenDto
import org.example.widedeep.models.WideTwelveDto
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class WideDeepValidationStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun wideDeepValidationBytes(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> wideDeepValidationBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    is OutgoingContent.NoContent -> ""
    else -> error("Unexpected synthetic form request body")
}

private fun wideDeepValidationHasNoContent(content: OutgoingContent): Boolean = when (content) {
    is OutgoingContent.ContentWrapper -> wideDeepValidationHasNoContent(content.delegate())
    is OutgoingContent.NoContent -> true
    else -> false
}

suspend fun wideDeepValidationContracts() = withTimeout(30000) {
    val one16 = listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(1))))))))))))))))
    val zero16 = listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(0))))))))))))))))
    val one12 = listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(listOf(1))))))))))))
    val wide16 = WideSixteenDto.default()
    val wide12 = WideTwelveDto.default()
    check(wide16.p199!!.single().single().single().single().single().single().single().single().single().single().single().single().single().single().single().single() == 200)
    check(wide12.p239!!.single().single().single().single().single().single().single().single().single().single().single().single() == 240)
    check(wide16.p199 !== WideSixteenDto.default().p199)
    check(runCatching { WideSixteenDto(p199=zero16) }.isFailure)
    check(runCatching { WideSixteenDto(p198=one16) }.isFailure)
    check(runCatching { WideTwelveDto(p238=one12) }.isFailure)
    check(WideSixteenDto(p199=one16).p198 == null)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.decodeFromString<WideSixteenDto>(codec.encodeToString(wide16)) == wide16)
        check(codec.decodeFromString<WideTwelveDto>(codec.encodeToString(wide12)) == wide12)
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        val input=SendInputDto(p199=one16)
        val response=codec.encodeToString(WideSixteenDto(p199=one16))
        var badEvent=false
        var calls=0
        var callbacks=0
        val engine=WideDeepValidationStreamEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            if (request.url.encodedPath=="/send") {
                check(codec.parseToJsonElement(wideDeepValidationBytes(request.body)) == codec.encodeToJsonElement(SendInputDto.serializer(),input))
                respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            } else {
                check(request.url.encodedPath=="/watch")
                val channel=ByteChannel(autoFlush=true);channels+=channel
                val payload=if (badEvent) "{\"p199\":"+"[[[[[[[[[[[[[[[[0]]]]]]]]]]]]]]]]"+"}" else response
                val frame=": 雪🍃\r\nevent: record\r\ndata: $payload\r\n\r\n"
                writers+=scope.launch {for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()}
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
        } })
        val client=HttpClient(engine) {install(SSE);install(ContentNegotiation){json(codec)}}
        try {
            val api=DefaultApi(client,"https://example.test/",codec)
            val result=api.send(input) {callbacks++}
            check(result.p199!!.single().single().single().single().single().single().single().single().single().single().single().single().single().single().single().single() == 1 && result.p198 == null)
            val before=calls;val beforeCallbacks=callbacks
            check(runCatching {api.send(SendInputDto(p199=zero16)) {callbacks++}}.isFailure)
            check(calls==before && callbacks==beforeCallbacks)
            val flow=api.watch {callbacks++}
            check(calls==before)
            val event=flow.take(1).toList().single() as WatchEventDto.Record
            check(event.data.p199!!.single().single().single().single().single().single().single().single().single().single().single().single().single().single().single().single()==1 && event.data.p198==null)
            check(channels.last().isClosedForRead)
            badEvent=true
            val failure=runCatching {flow.take(1).toList()}.exceptionOrNull()
            check(failure!=null && generateSequence(failure){it.cause}.any {it is IllegalArgumentException})
            check(channels.last().isClosedForRead);badEvent=false
            check(api.send(input) {callbacks++}.p199!!.single().single().single().single().single().single().single().single().single().single().single().single().single().single().single().single()==1)
            check(calls==callbacks)
        } finally {client.close();scope.cancel();writers.forEach{it.cancel()};channels.forEach{it.cancel(null)}}
    }
}

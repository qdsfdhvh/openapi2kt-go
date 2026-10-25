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

import org.example.floatingform.api.DefaultApi
import org.example.floatingform.api.models.SubmitInputDto
import org.example.floatingform.api.models.OptionalInputDto
import org.example.floatingform.api.models.EventsInputDto
import org.example.floatingform.api.models.OptionalEventsInputDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class FloatingFormFieldsStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun floatingFormFieldsBytes(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> floatingFormFieldsBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    is OutgoingContent.NoContent -> ""
    else -> error("Unexpected synthetic form request body")
}

private fun floatingFormFieldsHasNoContent(content: OutgoingContent): Boolean = when (content) {
    is OutgoingContent.ContentWrapper -> floatingFormFieldsHasNoContent(content.delegate())
    is OutgoingContent.NoContent -> true
    else -> false
}

suspend fun floatingFormFieldsMediaContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var expected: String? = "floatValue=-0.0&doubleValue=1.25"
        var badEvent=false
        var cancelled=false
        var calls=0
        var callbacks=0
        val cancellation=CancellationException("synthetic floating form cancellation")
        val engine=FloatingFormFieldsStreamEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            if (cancelled) throw cancellation
            if (expected == null) {
                check(floatingFormFieldsHasNoContent(request.body))
                check(request.body.contentType == null && request.headers["Content-Type"] == null)
            } else {
                check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                check(floatingFormFieldsBytes(request.body) == expected) { floatingFormFieldsBytes(request.body) }
            }
            if (request.url.encodedPath.endsWith("events")) {
                val channel=ByteChannel(autoFlush=true); channels+=channel
                val frame=if (badEvent) "event: message\r\ndata: {\r\n\r\n" else "event: message\r\ndata: \"雪🍃\"\r\n\r\n"
                writers+=scope.launch { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            } else respond("",HttpStatusCode.NoContent)
        } })
        val client=HttpClient(engine) { install(SSE) }
        try {
            val api=DefaultApi(client,"https://example.test/",codec)
            api.submit(SubmitInputDto(-0.0f,1.25)) {callbacks++}
            expected="floatValue=1.25&doubleValue=-0.0&optionalFloat=0.0&optionalDouble=0.0&label=%E9%9B%AA+%26%2B"
            api.submit(SubmitInputDto(1.25f,-0.0,0.0f,0.0,"雪 &+")) {callbacks++}
            expected="floatValue=NaN&doubleValue=Infinity"
            api.submit(SubmitInputDto(Float.NaN,Double.POSITIVE_INFINITY)) {callbacks++}
            expected=null
            api.optional {callbacks++}
            api.optional(input=null) {callbacks++}
            expected="floatValue=-0.0&doubleValue=1.25"
            api.optional(OptionalInputDto(-0.0f,1.25)) {callbacks++}
            expected="override=1.25"
            api.submit(SubmitInputDto(-0.0f,1.25)) {callbacks++;setBody(FormDataContent(Parameters.build { append("override","1.25") }))}
            if (checked) {
                val before=calls;val beforeCallbacks=callbacks
                for (number in listOf(-1.0,11.0,Double.NaN,Double.POSITIVE_INFINITY)) {
                    check(runCatching { api.submit(SubmitInputDto(1.25f,0.0,optionalDouble=number)) {callbacks++} }.isFailure)
                }
                check(calls==before && callbacks==beforeCallbacks)
            }
            expected="floatValue=-0.0&doubleValue=1.25"
            val before=calls
            val flow=api.events(input=EventsInputDto(-0.0f,1.25)) {callbacks++}
            check(calls==before)
            check(flow.take(1).toList().single().let {it is org.example.floatingform.api.models.EventsEventDto.Message && it.data=="雪🍃"})
            check(channels.last().isClosedForRead)
            badEvent=true
            check(runCatching { flow.take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead);badEvent=false
            expected=null
            check(api.optionalEvents {callbacks++}.take(1).toList().size==1)
            expected="floatValue=1.25&doubleValue=2.5&optionalFloat=-0.0"
            check(api.optionalEvents(input=OptionalEventsInputDto(1.25f,2.5,-0.0f)) {callbacks++}.take(1).toList().size==1)
            check(channels.last().isClosedForRead)
            val bare=HttpClient(engine)
            try {
                val direct=DefaultApi(bare,"https://example.test/",codec)
                expected="floatValue=-0.0&doubleValue=1.25"
                direct.submit(SubmitInputDto(-0.0f,1.25)) {callbacks++}
                cancelled=true
                check(runCatching { direct.optional {callbacks++} }.exceptionOrNull() === cancellation)
                cancelled=false;expected=null
                direct.optional {callbacks++}
            } finally {bare.close()}
            check(calls==callbacks)
        } finally {client.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

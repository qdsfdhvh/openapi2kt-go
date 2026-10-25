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

import org.example.enumform.api.DefaultApi
import org.example.enumform.api.models.SubmitInputDto
import org.example.enumform.api.models.OptionalInputDto
import org.example.enumform.api.models.EventsInputDto
import org.example.enumform.api.models.OptionalEventsInputDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class EnumFormFieldsStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun enumFormFieldsBytes(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> enumFormFieldsBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    is OutgoingContent.NoContent -> ""
    else -> error("Unexpected synthetic form request body")
}

private fun enumFormFieldsHasNoContent(content: OutgoingContent): Boolean = when (content) {
    is OutgoingContent.ContentWrapper -> enumFormFieldsHasNoContent(content.delegate())
    is OutgoingContent.NoContent -> true
    else -> false
}

suspend fun enumFormFieldsMediaContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        var expected: String? = "code=a+b%2B%E9%9B%AA&intValue=1&active=true"
        var badEvent=false
        var cancelled=false
        var calls=0
        var callbacks=0
        val cancellation=CancellationException("synthetic enum form cancellation")
        val engine=EnumFormFieldsStreamEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            if (cancelled) throw cancellation
            if (expected == null) {
                check(enumFormFieldsHasNoContent(request.body))
                check(request.body.contentType == null && request.headers["Content-Type"] == null)
            } else {
                check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                check(enumFormFieldsBytes(request.body) == expected) { enumFormFieldsBytes(request.body) }
            }
            if (request.url.encodedPath.endsWith("events")) {
                val channel=ByteChannel(autoFlush=true); channels+=channel
                val frame=if (badEvent) "event: message\r\ndata: {\r\n\r\n" else "event: message\r\ndata: \"雪🍃\"\r\n\r\n"
                writers+=scope.launch { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            } else respond("",HttpStatusCode.NoContent)
        } })
        val client=HttpClient(engine) { install(SSE) }
        val known=SubmitInputDto.Code.fromSerialName("a b+雪")
        val value=SubmitInputDto(code=known,intValue=1,active=true)
        try {
            val api=DefaultApi(client,"https://example.test/",codec)
            api.submit(value) {callbacks++}
            expected="code=future+%26%E9%9B%AA&intValue=2&longValue=9223372036854775807&floatValue=-0.0&doubleValue=2.5&active=false&optionalCode=future&flag=true"
            api.submit(SubmitInputDto(code=SubmitInputDto.Code.Unrecognized("future &雪"),intValue=2,longValue=Long.MAX_VALUE,floatValue=-0.0f,doubleValue=2.5,active=false,optionalCode=SubmitInputDto.OptionalCode.Unrecognized("future"),flag=true)) {callbacks++}
            expected="code=ready&intValue=1&longValue=-9223372036854775808&active=true&optionalCode="
            api.submit(SubmitInputDto(code=SubmitInputDto.Code.fromSerialName("ready"),intValue=1,longValue=Long.MIN_VALUE,active=true,optionalCode=SubmitInputDto.OptionalCode.Unrecognized(""))) {callbacks++}
            if (checked) {
                val before=calls;val beforeCallbacks=callbacks
                check(runCatching { api.submit(SubmitInputDto(code=known,intValue=3,active=true)) {callbacks++} }.isFailure)
                check(runCatching { api.submit(SubmitInputDto(code=known,intValue=1,active=true,flag=false)) {callbacks++} }.isFailure)
                check(runCatching { api.submit(SubmitInputDto(code=known,intValue=1,active=true,floatValue=3.0f)) {callbacks++} }.isFailure)
                check(runCatching { api.submit(SubmitInputDto(code=known,intValue=1,active=true,doubleValue=1.25)) {callbacks++} }.isFailure)
                check(calls==before && callbacks==beforeCallbacks)
            } else {
                expected="code=a+b%2B%E9%9B%AA&intValue=3&floatValue=3.0&doubleValue=1.25&active=true&flag=false"
                api.submit(SubmitInputDto(code=known,intValue=3,floatValue=3.0f,doubleValue=1.25,active=true,flag=false)) {callbacks++}
            }
            expected=null
            api.optional {callbacks++}
            expected="code=ready&intValue=2&active=false"
            api.optional(OptionalInputDto(code=OptionalInputDto.Code.fromSerialName("ready"),intValue=2,active=false)) {callbacks++}
            expected="override=ready"
            api.submit(value) {callbacks++;setBody(FormDataContent(Parameters.build { append("override","ready") }))}
            expected="code=ready&intValue=1&active=true"
            val before=calls
            val flow=api.events(input=EventsInputDto(code=EventsInputDto.Code.fromSerialName("ready"),intValue=1,active=true)) {callbacks++}
            check(calls==before)
            check(flow.take(1).toList().single().let {it is org.example.enumform.api.models.EventsEventDto.Message && it.data=="雪🍃"})
            check(channels.last().isClosedForRead)
            badEvent=true
            check(runCatching { flow.take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead);badEvent=false
            expected=null
            check(api.optionalEvents {callbacks++}.take(1).toList().size==1)
            expected="code=future&intValue=2&active=false"
            check(api.optionalEvents(input=OptionalEventsInputDto(code=OptionalEventsInputDto.Code.Unrecognized("future"),intValue=2,active=false)) {callbacks++}.take(1).toList().size==1)
            check(channels.last().isClosedForRead)
            val bare=HttpClient(engine)
            try {
                val direct=DefaultApi(bare,"https://example.test/",codec)
                expected="code=a+b%2B%E9%9B%AA&intValue=1&active=true"
                direct.submit(value) {callbacks++}
                cancelled=true
                check(runCatching { direct.optional {callbacks++} }.exceptionOrNull() === cancellation)
                cancelled=false;expected=null
                direct.optional {callbacks++}
            } finally {bare.close()}
            check(calls==callbacks)
        } finally {client.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.uuid.ExperimentalUuidApi::class)
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

import kotlin.uuid.Uuid
import kotlin.time.Instant
import kotlinx.datetime.LocalDate

import org.example.standardform.api.DefaultApi
import org.example.standardform.api.models.SubmitInputDto
import org.example.standardform.api.models.OptionalInputDto
import org.example.standardform.api.models.EventsInputDto
import org.example.standardform.api.models.OptionalEventsInputDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StandardFormFieldsStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun standardFormFieldsBytes(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> standardFormFieldsBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    is OutgoingContent.NoContent -> ""
    else -> error("Unexpected synthetic form request body")
}

private fun standardFormFieldsHasNoContent(content: OutgoingContent): Boolean = when (content) {
    is OutgoingContent.ContentWrapper -> standardFormFieldsHasNoContent(content.delegate())
    is OutgoingContent.NoContent -> true
    else -> false
}

suspend fun standardFormFieldsMediaContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val writers=CopyOnWriteArrayList<Job>()
        val uuid=Uuid.parse("123E4567-E89B-12D3-A456-426614174000")
        val day=LocalDate.parse("2026-10-22")
        val instant=Instant.parse("2026-10-22T20:30:40.125+08:00")
        val base="uuidValue=123e4567-e89b-12d3-a456-426614174000&dateValue=2026-10-22"
        var expected: String? = base
        var badEvent=false
        var cancelled=false
        var calls=0
        var callbacks=0
        val cancellation=CancellationException("synthetic standard form cancellation")
        val engine=StandardFormFieldsStreamEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            if (cancelled) throw cancellation
            if (expected == null) {
                check(standardFormFieldsHasNoContent(request.body))
                check(request.body.contentType == null && request.headers["Content-Type"] == null)
            } else {
                check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                check(standardFormFieldsBytes(request.body) == expected) { standardFormFieldsBytes(request.body) }
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
            val initial=SubmitInputDto(uuid,day)
            check(codec.decodeFromString(SubmitInputDto.serializer(),codec.encodeToString(SubmitInputDto.serializer(),initial)) == initial)
            check(codec.encodeToString(SubmitInputDto.serializer(),initial).contains("123e4567-e89b-12d3-a456-426614174000"))
            for (bad in listOf("not-a-date","2026-02-30")) {
                val malformed="""{"uuidValue":"123e4567-e89b-12d3-a456-426614174000","dateValue":"$bad"}"""
                check(runCatching {codec.decodeFromString(SubmitInputDto.serializer(),malformed)}.isFailure)
            }
            api.submit(initial) {callbacks++}
            expected=base+"&optionalInstant=2026-10-22T12%3A30%3A40.125Z&optionalUuid=123e4567-e89b-12d3-a456-426614174000&label=%E9%9B%AA+%26%2B&class=2026-10-22&it=2026-10-22T12%3A30%3A40.125Z"
            api.submit(SubmitInputDto(uuid,day,instant,uuid,"雪 &+",day,instant)) {callbacks++}
            expected=null
            api.optional {callbacks++}
            api.optional(input=null) {callbacks++}
            expected=base
            api.optional(OptionalInputDto(uuid,day)) {callbacks++}
            expected="override=standard"
            api.submit(SubmitInputDto(uuid,day)) {callbacks++;setBody(FormDataContent(Parameters.build { append("override","standard") }))}
            if (checked) {
                val before=calls;val beforeCallbacks=callbacks
                check(runCatching {api.submit(SubmitInputDto(Uuid.parse("00000000-0000-0000-0000-000000000000"),day)) {callbacks++}}.isFailure)
                check(calls==before && callbacks==beforeCallbacks)
            }
            expected=base
            val before=calls
            val flow=api.events(input=EventsInputDto(uuid,day)) {callbacks++}
            check(calls==before)
            check(flow.take(1).toList().single().let {it is org.example.standardform.api.models.EventsEventDto.Message && it.data=="雪🍃"})
            check(channels.last().isClosedForRead)
            badEvent=true
            check(runCatching { flow.take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead);badEvent=false
            expected=null
            check(api.optionalEvents {callbacks++}.take(1).toList().size==1)
            expected=base+"&optionalInstant=2026-10-22T12%3A30%3A40.125Z"
            check(api.optionalEvents(input=OptionalEventsInputDto(uuid,day,instant)) {callbacks++}.take(1).toList().size==1)
            check(channels.last().isClosedForRead)
            val bare=HttpClient(engine)
            try {
                val direct=DefaultApi(bare,"https://example.test/",codec)
                expected=base
                direct.submit(SubmitInputDto(uuid,day)) {callbacks++}
                cancelled=true
                check(runCatching { direct.optional {callbacks++} }.exceptionOrNull() === cancellation)
                cancelled=false;expected=null
                direct.optional {callbacks++}
            } finally {bare.close()}
            check(calls==callbacks)
        } finally {client.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

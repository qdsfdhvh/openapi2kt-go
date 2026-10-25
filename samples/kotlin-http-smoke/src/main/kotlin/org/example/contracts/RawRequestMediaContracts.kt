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
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
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
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import org.example.rawrequest.api.DefaultApi
import org.example.rawrequest.api.models.FormPriorityInputDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RawRequestStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun rawRequestBytes(content: OutgoingContent): ByteArray = when(content) {
    is OutgoingContent.ContentWrapper -> rawRequestBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes()
    is OutgoingContent.NoContent -> byteArrayOf()
    else -> error("Unexpected synthetic raw request body")
}

// The SSE plugin wraps its original content before the engine sees it.
private fun rawRequestHasNoContent(content: OutgoingContent): Boolean = when (content) {
    is OutgoingContent.ContentWrapper -> rawRequestHasNoContent(content.delegate())
    is OutgoingContent.NoContent -> true
    else -> false
}

suspend fun rawRequestMediaContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults=defaults; explicitNulls=nulls }
        val scope = CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val text = "\"quoted\"\n雪 & + / %"
        val binary = byteArrayOf(0,1,127,-128,-1)
        var expectedText=text
        var expectedBytes=binary
        var expectedOptional: ByteArray? = null
        var optionalType: ContentType? = null
        var optionalCallbacks = 0
        var calls=0
        var badEvent=false
        var cancelled=false
        val cancellation=CancellationException("synthetic raw request cancellation")
        val engine=RawRequestStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                if (cancelled) throw cancellation
                val bytes=rawRequestBytes(request.body)
                when(request.url.encodedPath) {
                    "/text", "/raw-priority" -> {
                        check(request.body.contentType?.withoutParameters() == ContentType.Text.Plain)
                        check(bytes.contentEquals(expectedText.encodeToByteArray()))
                        if (request.url.encodedPath == "/text") check(request.url.parameters["io"] == "雪 &+")
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/charset" -> {
                        check(request.body.contentType?.parameter("charset") == "ISO-8859-1")
                        check(bytes.contentEquals(byteArrayOf(99,97,102,-23)))
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/bytes" -> {
                        check(request.body.contentType?.withoutParameters() == ContentType.Application.OctetStream)
                        check(bytes.contentEquals(expectedBytes))
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/priority" -> {
                        check(bytes.decodeToString() == "42")
                        check(request.body.contentType?.withoutParameters() == ContentType.Application.Json)
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/form-priority" -> {
                        check(bytes.decodeToString() == "text=a+b%2B")
                        check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/optional-text", "/optional-bounded", "/optional-bytes", "/optional-charset" -> {
                        check(request.body.contentType == optionalType || request.body.contentType?.withoutParameters() == optionalType)
                        if (expectedOptional == null) {
                            check(rawRequestHasNoContent(request.body))
                            check(request.headers["Content-Type"] == null)
                        } else check(bytes.contentEquals(expectedOptional!!))
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/events", "/optional-events", "/optional-byte-events" -> {
                        if (request.url.encodedPath == "/events") {
                            check(bytes.contentEquals(text.encodeToByteArray()))
                            check(request.body.contentType?.withoutParameters() == ContentType.Text.Plain)
                            check(request.headers["Last-Event-ID"] == "previous")
                        } else {
                            check(request.body.contentType?.withoutParameters() == optionalType)
                            if (expectedOptional == null) {
                                check(rawRequestHasNoContent(request.body))
                                check(request.headers["Content-Type"] == null)
                            } else check(bytes.contentEquals(expectedOptional!!))
                        }
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val frame=if (badEvent) "event: message\r\ndata: {\r\n\r\n" else "event: message\r\ndata: \"received\"\r\n\r\n"
                        writers.add(scope.launch {for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic raw request path")
                }
            }
        })
        val client=HttpClient(engine) {install(ContentNegotiation) {json(codec)};install(SSE)}
        try {
            val api=DefaultApi(client,"https://example.test/",codec)
            api.sendText(io="雪 &+",input=text)
            expectedText="null";api.sendText(io="雪 &+",input="null")
            if (!checked) {expectedText="";api.sendText(io="雪 &+",input="")}
            expectedText=text
            api.rawPriority(text)
            api.sendCharset("café")
            api.sendBytes(binary)
            expectedBytes=byteArrayOf();api.sendBytes(expectedBytes);expectedBytes=binary
            api.jsonPriority(42)
            api.formPriority(FormPriorityInputDto("a b+"))
            expectedText="override";api.sendText(io="雪 &+",input=text) {setBody("override")};expectedText=text
            if (checked) {
                val before=calls;var callbacks=0
                check(runCatching {api.sendText(io="雪 &+",input="") {callbacks++}}.isFailure)
                check(calls==before && callbacks==0)
            }
            // Null/omitted and empty are different raw wire states in every Json mode.
            val optionalStart = calls
            api.optionalText { optionalCallbacks++ }
            api.optionalText(input = null) { optionalCallbacks++ }
            api.optionalBytes { optionalCallbacks++ }
            api.optionalBounded { optionalCallbacks++ }
            expectedOptional = byteArrayOf(); optionalType = ContentType.Text.Plain
            api.optionalText("") { optionalCallbacks++ }
            expectedOptional = "null".encodeToByteArray()
            api.optionalText("null") { optionalCallbacks++ }
            expectedOptional = text.encodeToByteArray()
            api.optionalText(text) { optionalCallbacks++ }
            optionalType = ContentType.Application.OctetStream; expectedOptional = byteArrayOf()
            api.optionalBytes(byteArrayOf()) { optionalCallbacks++ }
            expectedOptional = binary
            api.optionalBytes(binary) { optionalCallbacks++ }
            optionalType = ContentType.parse("TEXT/PLAIN; charset=ISO-8859-1")
            expectedOptional = byteArrayOf(99,97,102,-23)
            api.optionalCharset("café") { optionalCallbacks++ }
            check(calls - optionalStart == optionalCallbacks)
            if (checked) {
                val prior = calls; val priorCallbacks = optionalCallbacks
                check(runCatching { api.optionalBounded("") { optionalCallbacks++ } }.isFailure)
                check(calls == prior && optionalCallbacks == priorCallbacks)
            }
            optionalType = null; expectedOptional = null
            val optionalBefore = calls
            val optionalFlow = api.optionalEvents { optionalCallbacks++ }
            check(calls == optionalBefore)
            check(optionalFlow.take(1).toList().single().let { it is org.example.rawrequest.api.models.OptionalEventsEventDto.Message && it.data == "received" })
            check(channels.last().isClosedForRead)
            expectedOptional = text.encodeToByteArray(); optionalType = ContentType.Text.Plain
            check(api.optionalEvents(input = text).take(1).toList().size == 1)
            expectedOptional = null; optionalType = null
            check(api.optionalByteEvents().take(1).toList().size == 1)
            expectedOptional = byteArrayOf(); optionalType = ContentType.Application.OctetStream
            check(api.optionalByteEvents(input = byteArrayOf()).take(1).toList().size == 1)
            badEvent = true
            check(runCatching { api.optionalByteEvents(input = byteArrayOf()).take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead); badEvent = false
            if (checked) {
                val prior = calls; val priorCallbacks = optionalCallbacks
                val invalidFlow = api.optionalEvents(input = "") { optionalCallbacks++ }
                check(calls == prior)
                check(runCatching { invalidFlow.toList() }.isFailure)
                check(calls == prior && optionalCallbacks == priorCallbacks)
            }
            val before=calls
            val flow=api.events(lastEventId="previous",input=text)
            check(calls==before)
            check(flow.take(1).toList().single().let {it is org.example.rawrequest.api.models.EventsEventDto.Message && it.data=="received"})
            check(channels.last().isClosedForRead)
            badEvent=true;check(runCatching {flow.take(1).toList()}.isFailure);check(channels.last().isClosedForRead);badEvent=false
            check(flow.take(1).toList().size==1)
            cancelled=true;check(runCatching {api.sendBytes(binary)}.exceptionOrNull()===cancellation);cancelled=false
            api.sendBytes(binary)
            val bare=HttpClient(engine)
            try {
                val plain=DefaultApi(bare,"https://example.test/",codec)
                plain.sendText(io="雪 &+",input=text)
                plain.sendBytes(binary)
                optionalType = null; expectedOptional = null
                plain.optionalText()
                plain.optionalBytes(input = null)
                expectedOptional = text.encodeToByteArray(); optionalType = ContentType.Text.Plain
                plain.optionalText(text)
                expectedOptional = binary; optionalType = ContentType.Application.OctetStream
                plain.optionalBytes(binary)
                expectedOptional = "override".encodeToByteArray(); optionalType = ContentType.Text.Plain
                plain.optionalText { setBody("override") }
                cancelled = true
                check(runCatching { plain.optionalBytes() }.exceptionOrNull() === cancellation)
                cancelled = false; expectedOptional = null; optionalType = null
                plain.optionalBytes()
            } finally {bare.close()}
            println("RawRequest checked=$checked defaults=$defaults nulls=$nulls raw-text-UTF8-charset-empty-nulltext-binary-priority-callback-preflight-liveSSE-cancel-reuse PASS")
        } finally {client.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

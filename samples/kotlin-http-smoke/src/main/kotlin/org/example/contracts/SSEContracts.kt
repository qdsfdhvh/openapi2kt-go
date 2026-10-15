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
import io.ktor.client.request.header
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import io.ktor.utils.io.writeStringUtf8
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.joinAll
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.flow.onEach
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.streaming.ApiClient
import org.example.streaming.api.models.EventsEventDto
import org.example.streaming.api.models.EventsInputDto
import org.example.streaming.api.models.HeartbeatEventDto
import org.example.streaming.api.models.StringEventsDataChoiceDto
import org.example.streaming.models.Patch
import java.util.concurrent.CopyOnWriteArrayList

// MockEngine does not advertise SSE or run a response adapter. This test-only
// adapter feeds actual fragmented byte channels through Ktor's own SSE parser;
// it is not a hand-written SSE session or a fake event flow.
@OptIn(io.ktor.utils.io.InternalAPI::class)
internal class StreamingTestEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers,
            response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers,
            response.version, body, response.callContext)
    }
}

// Observe the actual parser read, not merely the request handler/handshake.
private class ObservedReadChannel(
    private val source: ByteReadChannel,
    private val started: CompletableDeferred<Unit>,
) : ByteReadChannel by source {
    override suspend fun awaitContent(min: Int): Boolean {
        started.complete(Unit)
        return source.awaitContent(min)
    }
}

private fun bodyBytes(content: OutgoingContent): ByteArray = when (content) {
    is OutgoingContent.ContentWrapper -> bodyBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes()
    else -> error("Unexpected SSE request body $content")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun sseContracts() = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val producerScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val requests = CopyOnWriteArrayList<HttpRequestData>()
        val firstDelivered = CompletableDeferred<Unit>()
        var eventMode = "all"
        var heartbeatMode = "frame"
        var heartbeatStarted = CompletableDeferred<Unit>()
        val firstFrame = "id: 1\r\nevent: update\r\ndata: {\r\ndata: \"text\":\"🍃\",\"extra\":true}\r\n\r\n"
        val rest = "event: unknown\ndata: malformed ignored\n\n" +
            "event: count\n\n" + // required data missing: skip
            "event: count\ndata: 42\n\n" +
            "event: inline\ndata: {\"value\":\"typed\"}\n\n" +
            "event: choice\ndata: \"open\"\n\n" +
            "event: optional\n\n" + "event: optional\ndata: null\n\n" +
            "event:\ndata: \"default-message\"\n\n" +
            "event: rows\ndata: [{\"value\":\"row\"}]\n\n" +
            "event: lookup\ndata: {\"key\":{\"value\":\"map\"}}\n\n" +
            "id:\nevent: done\n\n" +
            "event: count\ndata: 999" // no terminating blank line: Ktor discards incomplete frame
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                requests += request
                if (request.url.encodedPath == "/ping") {
                    respond("", HttpStatusCode.NoContent)
                } else {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val isEvents = request.url.encodedPath.startsWith("/events/")
                    if (isEvents) {
                        check(request.method == HttpMethod.Post)
                        check(request.url.encodedPath == "/events/a%2Fb")
                        check(request.url.parameters.getAll("labels") == listOf("x", "y"))
                        check(request.headers.getAll("Last-Event-ID")?.size == 1)
                        check(request.headers["Last-Event-ID"] in listOf("cursor", "override"))
                        val body = codec.parseToJsonElement(bodyBytes(request.body).decodeToString()).jsonObject
                        check(body["query"]?.jsonPrimitive?.content == "search")
                        check("note" !in body)
                        val mode = eventMode
                        writers += producerScope.launch {
                            try {
                                for (byte in (": comment\r\n\r\n" + firstFrame).encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                // Proves delivery occurs before EOF/full response buffering.
                                firstDelivered.await()
                                if (mode == "hold") awaitCancellation()
                                for (byte in rest.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            } catch (cause: Throwable) {
                                if (!channel.isClosedForRead) throw cause
                            } finally { channel.close() }
                        }
                    } else {
                        check(request.method == HttpMethod.Get)
                        check(request.headers.getAll("Last-Event-ID")?.size == 1)
                        check(request.headers["Last-Event-ID"] == "resume")
                        if (heartbeatMode == "frame") {
                            writers += producerScope.launch { channel.writeStringUtf8(": comment\n\nretry: 0\n\nevent: heartbeat\n\n"); channel.close() }
                        } else if (heartbeatMode == "incomplete") {
                            writers += producerScope.launch { channel.writeStringUtf8("event: heartbeat"); channel.close() }
                        }
                    }
                    val input = if (!isEvents && heartbeatMode == "block") ObservedReadChannel(channel, heartbeatStarted) else channel
                    respond(input, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream; charset=utf-8"))
                }
            }
        })
        val http = HttpClient(engine) {
            install(SSE) // caller-owned; generated code installs nothing
            install(ContentNegotiation) { json(codec) }
        }
        try {
            val api = ApiClient(http, "http://example.test/", Json(codec) { ignoreUnknownKeys = true }).stream
            val flow = api.events("a/b", listOf("x", "y"), "cursor", EventsInputDto("search", Patch.Absent))
            check(requests.isEmpty()) // creating the Flow performs no IO
            repeat(2) {
                val events = flow.onEach { firstDelivered.complete(Unit) }.toList()
                check(events.size == 10)
                check((events[0] as EventsEventDto.Update).data.text == "🍃")
                check(events[0].id == "1" && events[1].id == "1")
                check((events[1] as EventsEventDto.Count).data == 42)
                check((events[2] as EventsEventDto.Inline).data.value == "typed")
                check(((events[3] as EventsEventDto.Choice).data as StringEventsDataChoiceDto).value == "open")
                check((events[4] as EventsEventDto.Optional).data == null)
                check((events[5] as EventsEventDto.Optional).data == null)
                check((events[6] as EventsEventDto.Message).data == "default-message")
                check((events[7] as EventsEventDto.Rows).data.single().value == "row")
                check((events[8] as EventsEventDto.Lookup).data["key"]!!.value == "map")
                check(events[9] is EventsEventDto.Done && events[9].id == "")
                check(channels.last().isClosedForRead)
            }
            check(requests.size == 2) // every collection opens a fresh session
            eventMode = "hold" // cancellation must close an input not already at EOF
            val one = ApiClient(http, "http://example.test/").stream.events("a/b", listOf("x", "y"), "cursor", EventsInputDto("search")) {
                headers.remove("Last-Event-ID"); header("Last-Event-ID", "override")
            }.take(1).toList()
            check(one.single() is EventsEventDto.Update)
            check(channels.last().isClosedForRead) { "take(1) left an open SSE input" }
            val cancellation = CancellationException("downstream-test")
            try {
                flow.collect { throw cancellation }
                error("Expected downstream cancellation")
            } catch (cause: CancellationException) { check(cause === cancellation) }
            check(channels.last().isClosedForRead)
            // A caller-injected strict codec rejects the extra payload field.
            val strict = ApiClient(http, "http://example.test/", Json).stream
            try {
                strict.events("a/b", listOf("x", "y"), "cursor", EventsInputDto("search")).toList()
                error("Expected payload decode failure")
            } catch (cause: Exception) {
                check(generateSequence(cause as Throwable?) { it.cause }.any { it is SerializationException })
            }
            check(channels.last().isClosedForRead)
            check(api.heartbeat("resume").toList().single() is HeartbeatEventDto.Heartbeat)
            heartbeatMode = "incomplete"
            check(api.heartbeat("resume").toList().isEmpty())
            // Cancel while blocked reading a still-open input, not after a buffered event.
            heartbeatMode = "block"
            heartbeatStarted = CompletableDeferred()
            val waiting = launch { api.heartbeat("resume").collect() }
            heartbeatStarted.await()
            waiting.cancelAndJoin()
            check(waiting.isCancelled && channels.last().isClosedForRead)
            api.ping() // cancelling/finishing a stream never closes the caller's client
            check(requests.size == 9)
            println("SSE contracts passed: defaults=$defaults/nulls=$nulls; fragmented UTF8/CRLF/multiline, live delivery/cold Flow, event/data/id, POST/headers, EOF/decode/cancellation and client ownership; ${requests.size} requests")
        } finally {
            http.close(); engine.close(); producerScope.cancel()
            writers.joinAll()
        }
    }
}

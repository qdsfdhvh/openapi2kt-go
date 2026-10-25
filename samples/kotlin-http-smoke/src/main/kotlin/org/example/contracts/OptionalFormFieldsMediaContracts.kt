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
import org.example.formfields.api.DefaultApi
import org.example.formfields.api.models.EventsInputDto
import org.example.formfields.api.models.SubmitInputDto
import org.example.formfields.api.models.SubmitCopyInputDto
import org.example.formfields.api.models.VariantInputDto
import org.example.formfields.api.models.OptionalSubmitInputDto
import org.example.formfields.api.models.OptionalVariantInputDto
import org.example.formfields.api.models.OptionalEventsInputDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class OptionalFormFieldsStreamEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun optionalFormFieldsBytes(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> optionalFormFieldsBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    is OutgoingContent.NoContent -> ""
    else -> error("Unexpected synthetic form request body")
}

private fun optionalFormFieldsHasNoContent(content: OutgoingContent): Boolean = when (content) {
    is OutgoingContent.ContentWrapper -> optionalFormFieldsHasNoContent(content.delegate())
    is OutgoingContent.NoContent -> true
    else -> false
}

suspend fun optionalFormFieldsMediaContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        val text = "a b+&=/雪%"
        val encoded = "a+b%2B%26%3D%2F%E9%9B%AA%25"
        var expectedFlat: String? = null
        var expectedVariant = "a+b%26%2B%E9%9B%AA=$encoded&empty="
        var expectedEvents = "text=$encoded"
        var expectedOptional: String? = null
        var optionalCallbacks = 0
        var calls = 0
        var override = false
        var badEvent = false
        var cancelled = false
        val cancellation = CancellationException("synthetic form cancellation")
        val engine = OptionalFormFieldsStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                if (cancelled) throw cancellation
                val wire = optionalFormFieldsBytes(request.body)
                if (request.url.encodedPath.startsWith("/optional-")) {
                    if (expectedOptional == null) {
                        check(optionalFormFieldsHasNoContent(request.body))
                        check(request.body.contentType == null && request.headers["Content-Type"] == null)
                    } else {
                        check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                        check(wire == expectedOptional)
                    }
                } else if (request.url.encodedPath != "/priority") {
                    check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                }
                when(request.url.encodedPath) {
                    "/submit", "/submit-copy" -> {
                        check(wire == if (override) "override=$encoded" else expectedFlat ?: "text=$encoded&active=false&count=-2147483648&large=9223372036854775807&class=%26%3D")
                        if (request.url.encodedPath == "/submit" && !override && expectedFlat == null) {
                            check(request.url.parameters["io"] == "雪 &+")
                            check(request.url.parameters["OpenapiKtFormDataImport"] == "alias")
                        }
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/variant" -> {
                        check(wire == expectedVariant)
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/priority" -> {
                        check(wire == "JSON selection")
                        check(request.body.contentType?.withoutParameters() == ContentType.Application.Json)
                        respond("",HttpStatusCode.NoContent)
                    }
                    "/optional-submit", "/optional-variant" -> respond("",HttpStatusCode.NoContent)
                    "/events", "/optional-events" -> {
                        if (request.url.encodedPath == "/events") {
                            check(wire == expectedEvents)
                            check(request.headers["Last-Event-ID"] == "previous")
                        }
                        val channel = ByteChannel(autoFlush = true); channels.add(channel)
                        val frame = if (badEvent) "event: message\r\ndata: {\r\n\r\n" else "event: message\r\ndata: \"received\"\r\n\r\n"
                        writers.add(scope.launch { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); awaitCancellation() })
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected synthetic form URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client,"https://example.test/",codec)
            val value = SubmitInputDto(text,false,Int.MIN_VALUE,Long.MAX_VALUE,"&=")
            api.submit(io = "雪 &+",openapiKtFormDataImport = "alias",input = value)
            api.submitCopy(SubmitCopyInputDto(text,false,Int.MIN_VALUE,Long.MAX_VALUE,"&="))
            api.variant(VariantInputDto(text,""))
            api.priority("JSON selection")
            override = true
            api.submit(input = value) { setBody(FormDataContent(Parameters.build { append("override",text) })) }
            override = false
            if (checked) {
                val before = calls
                check(runCatching { SubmitInputDto("",false,0,0,"allowed") }.isFailure)
                check(calls == before)
            }
            expectedFlat = "text=$encoded"
            api.submit(input = SubmitInputDto(text))
            expectedFlat = "text=$encoded&active=false&count=0&large=0&it="
            api.submit(input = SubmitInputDto(text,active=false,count=0,large=0,it=""))
            expectedFlat = null
            expectedVariant = ""
            api.variant(VariantInputDto())
            expectedVariant = "empty="
            api.variant(VariantInputDto(empty=""))
            expectedVariant = "a+b%26%2B%E9%9B%AA=$encoded&empty="
            val optionalStart = calls
            api.optionalSubmit { optionalCallbacks++ }
            api.optionalSubmit(input = null) { optionalCallbacks++ }
            api.optionalVariant { optionalCallbacks++ }
            expectedOptional = "text=$encoded&active=false&count=-2147483648&large=9223372036854775807&class=%26%3D"
            api.optionalSubmit(OptionalSubmitInputDto(text,false,Int.MIN_VALUE,Long.MAX_VALUE,"&=")) { optionalCallbacks++ }
            expectedOptional = "a+b%26%2B%E9%9B%AA=&empty="
            api.optionalVariant(OptionalVariantInputDto("","")) { optionalCallbacks++ }
            expectedOptional = "override=$encoded"
            api.optionalSubmit { optionalCallbacks++; setBody(FormDataContent(Parameters.build { append("override",text) })) }
            check(calls - optionalStart == optionalCallbacks)
            expectedOptional = "text=$encoded"
            api.optionalSubmit(OptionalSubmitInputDto(text))
            expectedOptional = ""
            api.optionalVariant(OptionalVariantInputDto())
            expectedOptional = null
            val optionalBefore = calls
            val optionalFlow = api.optionalEvents { optionalCallbacks++ }
            check(calls == optionalBefore)
            check(optionalFlow.take(1).toList().single().let { it is org.example.formfields.api.models.OptionalEventsEventDto.Message && it.data == "received" })
            check(channels.last().isClosedForRead)
            expectedOptional = ""
            check(api.optionalEvents(input=OptionalEventsInputDto()).take(1).toList().size == 1)
            expectedOptional = "text=$encoded"
            check(api.optionalEvents(input = OptionalEventsInputDto(text)).take(1).toList().size == 1)
            badEvent = true
            check(runCatching { api.optionalEvents(input = OptionalEventsInputDto(text)).take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead); badEvent = false
            if (checked) {
                val prior = calls; val priorCallbacks = optionalCallbacks
                check(runCatching { api.optionalSubmit(OptionalSubmitInputDto("",false,0,0,"allowed")) { optionalCallbacks++ } }.isFailure)
                check(calls == prior && optionalCallbacks == priorCallbacks)
            }
            val bare = HttpClient(engine)
            try {
                val plain = DefaultApi(bare,"https://example.test/",codec)
                expectedOptional = null; plain.optionalSubmit()
                expectedOptional = "a+b%26%2B%E9%9B%AA=&empty="
                plain.optionalVariant(OptionalVariantInputDto("",""))
                cancelled = true
                check(runCatching { plain.optionalSubmit() }.exceptionOrNull() === cancellation)
                cancelled = false; expectedOptional = null; plain.optionalSubmit()
            } finally { bare.close() }
            expectedEvents = ""
            check(api.events(lastEventId="previous",input=EventsInputDto()).take(1).toList().size == 1)
            expectedEvents = "text=$encoded"
            val before = calls
            val flow = api.events(lastEventId = "previous",input = EventsInputDto(text))
            check(calls == before)
            check(flow.take(1).toList().single().let { it is org.example.formfields.api.models.EventsEventDto.Message && it.data == "received" })
            check(channels.last().isClosedForRead)
            badEvent = true
            check(runCatching { flow.take(1).toList() }.isFailure)
            check(channels.last().isClosedForRead)
            badEvent = false
            check(flow.take(1).toList().size == 1)
            cancelled = true
            check(runCatching { api.variant(VariantInputDto(text,"")) }.exceptionOrNull() === cancellation)
            cancelled = false
            api.variant(VariantInputDto(text,""))
            println("OptionalFormFields checked=$checked defaults=$defaults nulls=$nulls exact-wire-encoding-required-scalars-JSON-priority-callback-liveSSE-cancel-reuse PASS")
        } finally {client.close();scope.cancel();writers.forEach {it.cancel()};channels.forEach {it.cancel(null)}}
    }
}

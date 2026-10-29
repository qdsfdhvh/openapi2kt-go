@file:OptIn(kotlin.uuid.ExperimentalUuidApi::class, kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
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
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.formdefaults.api.DefaultApi
import org.example.formdefaults.api.models.EventsInputDto
import org.example.formdefaults.api.models.OptionalSubmitInputDto
import org.example.formdefaults.api.models.SubmitInputDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class FormDefaultsStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun formDefaultsBytes(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> formDefaultsBytes(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    is OutgoingContent.NoContent -> ""
    else -> error("Unexpected synthetic form content")
}

suspend fun formFieldDefaultsContracts(checked: Boolean = false) = withTimeout(15000) {
    val defaultWire = "requiredValue=required&text=a+b%2B%26%E9%9B%AA&active=false&count=0" +
        "&large=9223372036854775807&ratio=1.25&amount=2.5&code=ready&state=2" +
        "&uuid=123e4567-e89b-12d3-a456-426614174000&date=2026-10-27&instant=2026-10-27T12%3A30%3A40Z"
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var expected: String? = defaultWire
        var calls = 0
        var callbacks = 0
        val engine = FormDefaultsStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                check(formDefaultsBytes(request.body) == (expected ?: ""))
                if (expected == null) {
                    check(request.body.contentType == null && request.headers["Content-Type"] == null)
                } else {
                    check(request.body.contentType?.withoutParameters() == ContentType.Application.FormUrlEncoded)
                }
                if (request.url.encodedPath == "/events") {
                    respond(ByteReadChannel("event: message\ndata: \"ok\"\n\n"),
                        headers = headersOf("Content-Type", "text/event-stream"))
                } else {
                    respond("", HttpStatusCode.NoContent)
                }
            }
        })
        val client = HttpClient(engine) { install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            val input = SubmitInputDto("required")
            check(input.text == "a b+&雪" && input.active == false && input.count == 0)
            check(input.large == Long.MAX_VALUE && input.code?.serialName == "ready")
            check(input.instant.toString() == "2026-10-27T12:30:40Z")
            val decoded = codec.decodeFromString<SubmitInputDto>("""{"requiredValue":"required"}""")
            check(decoded == input)
            check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input)) == input)
            check(runCatching { codec.decodeFromString<SubmitInputDto>("{}") }.isFailure)
            api.submit(input) { callbacks++ }
            api.submit(decoded)
            expected = defaultWire.replace("&text=a+b%2B%26%E9%9B%AA", "")
            api.submit(SubmitInputDto("required", text = null))
            expected = defaultWire.replace("text=a+b%2B%26%E9%9B%AA", "text=")
            if (!checked) api.submit(SubmitInputDto("required", text = ""))
            expected = defaultWire.replace("code=ready", "code=unknown")
            api.submit(codec.decodeFromString<SubmitInputDto>("""{"requiredValue":"required","code":"unknown"}"""))
            expected = null
            api.optionalSubmit()
            api.optionalSubmit(null)
            expected = defaultWire
            api.optionalSubmit(OptionalSubmitInputDto("required"))
            // Both callbacks and Json configuration act on actual values, with
            // form encoding independent of encodeDefaults/explicitNulls.
            expected = "override=caller"
            api.submit(input) {
                callbacks++
                setBody(FormDataContent(Parameters.build { append("override", "caller") }))
            }
            check(callbacks == 2)
            expected = defaultWire
            val beforeStream = calls
            val flow = api.events(input = EventsInputDto("required"))
            check(calls == beforeStream)
            check(flow.take(1).toList().size == 1)
            check(calls == beforeStream + 1)
            api.submit(input) // Client remains usable after stream closure.
            if (checked) {
                val before = calls
                check(runCatching { api.submit(SubmitInputDto("required", text = "")) }.isFailure)
                check(runCatching { api.events(input = EventsInputDto("required", text = "")).toList() }.isFailure)
                check(calls == before)
            }
        } finally {
            client.close()
        }
    }
}

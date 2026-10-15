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
import io.ktor.http.ContentType
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeStringUtf8
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import org.example.namespaces.NamespaceApiClient as OrgClient
import org.example.namespaces.api.models.WatchEventDto as OrgEvent
import frame.example.namespaces.NamespaceApiClient as FrameClient
import frame.example.namespaces.api.models.WatchEventDto as FrameEvent
import input.example.namespaces.NamespaceApiClient as InputClient
import input.example.namespaces.api.models.PostedEventDto as InputEvent
import requestBuilder.example.namespaces.NamespaceApiClient as BuilderClient
import requestBuilder.example.namespaces.api.models.WatchEventDto as BuilderEvent
import value.example.namespaces.NamespaceApiClient as ValueClient
import value.example.namespaces.api.models.WatchEventDto as ValueEvent
import client.example.namespaces.NamespaceApiClient as ClientPrefixClient
import client.example.namespaces.api.models.WatchEventDto as ClientPrefixEvent
import basePath.example.namespaces.NamespaceApiClient as BasePrefixClient
import basePath.example.namespaces.api.models.WatchEventDto as BasePrefixEvent
import json.example.namespaces.NamespaceApiClient as JsonPrefixClient
import json.example.namespaces.api.models.WatchEventDto as JsonPrefixEvent
import httpClient.example.namespaces.NamespaceApiClient as HttpPrefixClient
import httpClient.example.namespaces.api.models.WatchEventDto as HttpPrefixEvent
import shadows.example.namespaces.NamespaceApiClient as PropertyPrefixClient
import shadows.example.namespaces.api.models.WatchEventDto as PropertyPrefixEvent

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NamespaceStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
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

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun namespaceCollisionContracts() = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val requests = mutableListOf<HttpRequestData>()
        val channels = mutableListOf<ByteChannel>()
        val engine = NamespaceStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                requests += request
                check(request.url.host == "example.test")
                if (request.url.encodedPath == "/v1/echo") {
                    check(request.method == HttpMethod.Post)
                    check(request.url.parameters["io"] == "plain-ctor")
                    check(request.body.contentType?.match(ContentType.Application.Json) == true)
                    respond("", HttpStatusCode.NoContent)
                } else if (request.url.encodedPath == "/v1/plain" || request.url.encodedPath.startsWith("/v1/reserved-")) {
                    check(request.method == HttpMethod.Get)
                    respond("", HttpStatusCode.NoContent)
                } else {
                    check(request.url.encodedPath in listOf("/v1/events", "/v1/posted"))
                    check(request.method == if (request.url.encodedPath.endsWith("posted")) HttpMethod.Post else HttpMethod.Get)
                    if (request.url.encodedPath == "/v1/posted") {
                        check(request.body.contentType?.match(ContentType.Application.Json) == true)
                        check(request.url.parameters["io"] == "post-ctor")
                        check(request.url.parameters["kotlinx"] == "post-flow")
                    }
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    channel.writeStringUtf8("event: tick\ndata: \"ready\"\n\n")
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
            }
        })
        val http = HttpClient(engine) { install(SSE) }
        fun parameters() {
            check(requests.last().url.parameters["org"] == "root")
            check(requests.last().url.parameters["io"] == "method")
            check(requests.last().url.parameters["kotlinx"] == "flow")
        }
        try {
            val api = OrgClient(http, json = codec).shadows
            api.plain(org = "root", io = "method", kotlinx = "flow")
            parameters()
            val before = requests.size
            val flow = api.watch(org = "root", io = "method", kotlinx = "flow")
            check(requests.size == before)
            repeat(2) {
                check((flow.take(1).toList().single() as OrgEvent.Tick).data == "ready")
                parameters()
                check(channels.last().isClosedForRead)
            }
            check((FrameClient(http, json = codec).shadows.watch().take(1).toList().single() as FrameEvent.Tick).data == "ready")
            check((BuilderClient(http, json = codec).shadows.watch().take(1).toList().single() as BuilderEvent.Tick).data == "ready")
            check((ValueClient(http, json = codec).shadows.watch().take(1).toList().single() as ValueEvent.Tick).data == "ready")
            api.echo(input = "body", io = "plain-ctor")
            check((InputClient(http, json = codec).shadows.posted(input = "body", io = "post-ctor", kotlinx = "post-flow").take(1).toList().single() as InputEvent.Tick).data == "ready")
            check((ClientPrefixClient(http, json = codec).shadows.watch().take(1).toList().single() as ClientPrefixEvent.Tick).data == "ready")
            check((BasePrefixClient(http, json = codec).shadows.watch().take(1).toList().single() as BasePrefixEvent.Tick).data == "ready")
            check((JsonPrefixClient(http, json = codec).shadows.watch().take(1).toList().single() as JsonPrefixEvent.Tick).data == "ready")
            check((HttpPrefixClient(http, json = codec).shadows.watch().take(1).toList().single() as HttpPrefixEvent.Tick).data == "ready")
            check((PropertyPrefixClient(http, json = codec).shadows.watch().take(1).toList().single() as PropertyPrefixEvent.Tick).data == "ready")
            check(channels.all { it.isClosedForRead })
            api.OpenapiKtHttpMethodImport()
            api.OpenapiKtFlowImport()
            api.OpenapiKtSSEEventImport1()
            api.plain()
            check(requests.last().url.parameters.isEmpty())
        } finally {
            http.close()
            channels.forEach { it.close() }
        }
    }
    println("HTTP namespace contracts passed: unchanged parameters/wire, safe aliases, cold SSE closure and reuse")
}

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.defaultRequest
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.client.request.url
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeStringUtf8
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import org.example.servers.ServerScopeApiClient
import org.example.servers.api.ServersApi
import org.example.servers.api.models.WatchEventDto

// Feed Ktor's actual SSE parser a still-open channel; take(1) must close it.
@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ServerStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
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
suspend fun serverContracts() = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val requests = mutableListOf<HttpRequestData>()
        val channels = mutableListOf<ByteChannel>()
        val engine = ServerStreamingEngine(MockEngineConfig().apply {
            addHandler { request ->
                requests += request
                if (request.url.encodedPath.contains("/events/")) {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    channel.writeStringUtf8("event: tick\ndata: \"ready\"\n\n")
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                } else {
                    respond("", HttpStatusCode.NoContent)
                }
            }
        })
        val http = HttpClient(engine) {
            install(SSE)
            install(ContentNegotiation) { json(codec) }
            defaultRequest { url("https://fallback.example.test/") }
        }
        fun target(host: String, path: String, method: HttpMethod = HttpMethod.Get) {
            val request = requests.last()
            check(request.url.protocol.name == "https")
            check(request.url.host == host) { request.url.toString() }
            check(request.url.encodedPath == path) { request.url.toString() }
            check(request.method == method)
        }
        try {
            val api = ServerScopeApiClient(http, json = codec).servers
            api.root(org = "package-shadow")
            target("api.example.test", "/v2/root")
            check(requests.last().url.parameters["org"] == "package-shadow")
            api.pathScope("a/b?# 雪%")
            target("path.example.test", "/path/scoped/a%2Fb%3F%23%20%E9%9B%AA%25")
            api.operationScope("plain")
            target("operation.example.test", "/operation/scoped/plain", HttpMethod.Post)
            api.relative()
            target("fallback.example.test", "/prefix/relative")
            api.empty()
            target("fallback.example.test", "/empty")
            api.operationScope("before") { url("https://callback.example.test/only") }
            target("callback.example.test", "/only", HttpMethod.Post)

            val override = ServerScopeApiClient(http, "https://override.example.test/custom/", codec).servers
            override.root()
            target("override.example.test", "/custom/root")
            override.pathScope("plain")
            target("override.example.test", "/custom/scoped/plain")
            override.operationScope("plain")
            target("override.example.test", "/custom/scoped/plain", HttpMethod.Post)
            override.empty()
            target("override.example.test", "/custom/empty")
            override.relative()
            target("override.example.test", "/custom/relative")
            val emptyOverride = ServerScopeApiClient(http, "", codec).servers
            emptyOverride.pathScope("empty-override")
            target("fallback.example.test", "/scoped/empty-override")
            // Supplying even the root default explicitly overrides local scopes.
            ServerScopeApiClient(http, "https://api.example.test/v2/", codec).servers.operationScope("same")
            target("api.example.test", "/v2/scoped/same", HttpMethod.Post)
            ServersApi(http, null, codec).pathScope("direct")
            target("path.example.test", "/path/scoped/direct")
            ServersApi(http, "https://direct.example.test/", codec).pathScope("direct")
            target("direct.example.test", "/scoped/direct")

            val count = requests.size
            val flow = api.watch("a/b", org = "package-shadow", lastEventId = "cursor")
            check(requests.size == count)
            repeat(2) {
                val event = flow.take(1).toList().single() as WatchEventDto.Tick
                check(event.data == "ready")
                target("stream.example.test", "/stream/events/a%2Fb")
                check(requests.last().headers["Last-Event-ID"] == "cursor")
                check(requests.last().url.parameters["org"] == "package-shadow")
                check(channels.last().isClosedForRead)
            }
            override.watch("plain").take(1).toList()
            target("override.example.test", "/custom/events/plain")
            check(channels.last().isClosedForRead)
            emptyOverride.watch("empty-override").take(1).toList()
            target("fallback.example.test", "/events/empty-override")
            check(channels.last().isClosedForRead)
            api.watch("callback") { url("https://callback.example.test/events/changed") }.take(1).toList()
            target("callback.example.test", "/events/changed")
            check(channels.last().isClosedForRead)
            api.root()
            target("api.example.test", "/v2/root")
        } finally {
            http.close()
            channels.forEach { it.close() }
        }
    }
    println("HTTP server contracts passed: defaults/scopes/override/relative/path encoding and cold SSE closure/reuse")
}

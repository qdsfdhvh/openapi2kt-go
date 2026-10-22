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
import io.ktor.http.HttpStatusCode
import io.ktor.http.Url
import io.ktor.http.headersOf
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import org.example.parameterstyles.ParameterStyles
import org.example.parameterstyles.api.models.DeepWatchEventDto
import org.example.parameterstyles.models.ChoiceDto
import org.example.parameterstyles.models.CodeDto
import java.util.concurrent.CopyOnWriteArrayList

suspend fun parameterStylesContracts(checked: Boolean = false) {
    val paths = mutableListOf<String>()
    val queries = mutableListOf<String?>()
    val urls = mutableListOf<Url>()
    val engine = MockEngine { request ->
        paths += request.url.encodedPath
        queries += request.url.parameters["filter"]
        urls += request.url
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine)
    try {
        val api = ParameterStyles(http, "http://example.test/").default
        val value = "a/b ?#"
        api.labelscalar(value)
        api.labelscalarExploded(value)
        api.matrixscalar(value)
        api.matrixscalarExploded(value)
        val items = listOf(ChoiceDto.READY_VALUE, null, ChoiceDto.Unrecognized("next/value"))
        api.labelarray(items)
        api.labelarrayExploded(items)
        api.matrixarray(items)
        api.matrixarrayExploded(items)
        check(paths.take(8) == listOf(
            "/labelscalar/.a%2Fb%20%3F%23",
            "/labelscalarExploded/.a%2Fb%20%3F%23",
            "/matrixscalar/;value=a%2Fb%20%3F%23",
            "/matrixscalarExploded/;value=a%2Fb%20%3F%23",
            "/labelarray/.ready%2Fvalue,null,next%2Fvalue",
            "/labelarrayExploded/.ready%2Fvalue.null.next%2Fvalue",
            "/matrixarray/;value=ready%2Fvalue,null,next%2Fvalue",
            "/matrixarrayExploded/;value=ready%2Fvalue;value=null;value=next%2Fvalue"
        )) { paths.toString() }
        api.labelarray(emptyList())
        api.matrixarray(emptyList())
        api.matrixarrayExploded(emptyList())
        check(paths.takeLast(3) == listOf("/labelarray/.", "/matrixarray/;value=", "/matrixarrayExploded/"))
        val entries = linkedMapOf("r/key" to ChoiceDto.READY_VALUE, "empty" to null)
        api.labelmap(entries)
        api.labelmapExploded(entries)
        api.matrixmap(entries)
        api.matrixmapExploded(entries)
        check(paths.takeLast(4) == listOf(
            "/labelmap/.r%2Fkey,ready%2Fvalue,empty,null",
            "/labelmapExploded/.r%2Fkey=ready%2Fvalue.empty=null",
            "/matrixmap/;value=r%2Fkey,ready%2Fvalue,empty,null",
            "/matrixmapExploded/;r%2Fkey=ready%2Fvalue;empty=null"
        )) { paths.toString() }
        api.dictionary(linkedMapOf("a b" to "v/x", "null" to null))
        check(queries.last() == "a b,v/x,null,null")
        api.dictionary(emptyMap())
        check(queries.last() == "")
        api.dictionary()
        check(queries.last() == null)

        val empty = if (checked) "ok" else ""
        api.deepStrings(linkedMapOf("a b" to "v/x ?#&=+%", "x[y]" to empty, "null" to null, "雪" to "值值"))
        check(urls.last().parameters.entries().associate { it.key to it.value.single() } == linkedMapOf(
            "filter[a b]" to "v/x ?#&=+%", "filter[x[y]]" to empty, "filter[雪]" to "值值"
        )) { urls.last().toString() }
        check(urls.last().encodedQuery.contains("%26") && urls.last().encodedQuery.contains("%23"))
        api.deepStrings(emptyMap()); check(urls.last().parameters.isEmpty())
        api.deepStrings(); check(urls.last().parameters.isEmpty())
        api.deepEnums(linkedMapOf("known" to ChoiceDto.READY_VALUE, "future" to ChoiceDto.Unrecognized("next/value"), "null" to null))
        check(urls.last().parameters["filter[known]"] == "ready/value")
        check(urls.last().parameters["filter[future]"] == "next/value")
        check("filter[null]" !in urls.last().parameters.names())
        api.deepEnums(null); check(urls.last().parameters.isEmpty())
        api.deepCodes(linkedMapOf("value" to CodeDto("code/value"), "null" to null))
        check(urls.last().parameters["filter[value]"] == "code/value" && "filter[null]" !in urls.last().parameters.names())
        api.deepNumbers(linkedMapOf("zero" to 0, "negative" to -7))
        check(urls.last().parameters["filter[zero]"] == "0" && urls.last().parameters["filter[negative]"] == "-7")
        api.deepBooleans(linkedMapOf("no" to false, "yes" to true))
        check(urls.last().parameters["filter[no]"] == "false" && urls.last().parameters["filter[yes]"] == "true")
        if (checked) {
            val before = urls.size
            var callback = false
            check(runCatching { api.deepStrings(mapOf("bad" to "x")) { callback = true } }.exceptionOrNull() is IllegalArgumentException)
            check(!callback && urls.size == before)
        }
    } finally { http.close(); engine.close() }
    deepDictionarySSEContracts()
    println("Parameter label/matrix/form-map/deepObject-map wire and SSE contracts passed")
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class ParameterStyleSSEEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun deepDictionarySSEContracts() = withTimeout(10000) {
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    var calls = 0
    val engine = ParameterStyleSSEEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(request.url.parameters["filter[tag]"] == "ready/value")
        check("filter[null]" !in request.url.parameters.names())
        val channel = ByteChannel(autoFlush = true); channels += channel
        scope.launch {
            try {
                for (byte in "event: sample\r\ndata: \"ok\"\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                awaitCancellation()
            } finally { channel.close() }
        }
        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
    } })
    val http = HttpClient(engine) { install(SSE) }
    try {
        val flow = ParameterStyles(http, "http://example.test/").default.deepWatch(linkedMapOf("tag" to ChoiceDto.READY_VALUE, "null" to null))
        check(calls == 0)
        repeat(2) {
            check((flow.take(1).toList().single() as DeepWatchEventDto.Sample).data == "ok")
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
        }
        check(calls == 2)
    } finally { http.close(); scope.cancel(); channels.forEach { it.cancel(null) } }
}

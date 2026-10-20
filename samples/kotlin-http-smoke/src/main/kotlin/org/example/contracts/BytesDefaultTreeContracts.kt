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
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.bytesfielddefaults.BytesFieldDefaults
import org.example.bytesfielddefaults.api.models.TreePatchInputDto
import org.example.bytesfielddefaults.api.models.WatchTreesEventDto
import org.example.bytesfielddefaults.models.Patch
import org.example.bytesfielddefaults.models.TreesDto
import org.example.bytesfielddefaults.models.`FactoryRowsDto default`

internal suspend fun bytesDefaultTreeContracts(codec: Json, checked: Boolean) {
    val value = TreesDto(requiredRows = listOf(byteArrayOf(9)))
    val fresh = TreesDto(requiredRows = listOf(byteArrayOf(9)))
    check(value.rows!![0].contentEquals(byteArrayOf(-128, 0, 127)))
    check(value.rows!![1].isEmpty() && value.nullableRows!![1] == null)
    check(value.bag!!["z"]!!.contentEquals(byteArrayOf(-1)))
    check(value.nested!![0]["bytes"]!!.contentEquals(byteArrayOf(1, 2)))
    check(value.payload!!.blob.contentEquals(byteArrayOf(-128, 127)))
    check(value.payload!!.namedBlob.contentEquals(byteArrayOf(4)))
    check(value.rows!![0] !== fresh.rows!![0] && value.rows!![1] !== fresh.rows!![1])
    check(value.payload!!.blob !== fresh.payload!!.blob)
    fresh.rows!![0][0] = 7
    check(value.rows!![0][0] == (-128).toByte())
    val fromFactory = `FactoryRowsDto default`()
    check(fromFactory.single().contentEquals(byteArrayOf(1)))
    check(fromFactory.single() !== `FactoryRowsDto default`().single())

    val missing = codec.decodeFromString<TreesDto>("{\"requiredRows\":[[9]]}")
    check(missing.rows!![0].contentEquals(byteArrayOf(-128, 0, 127)))
    check(missing.nullRows == null && missing.emptyRows!!.isEmpty())
    val explicit = codec.decodeFromString<TreesDto>("{\"requiredRows\":[[3]],\"rows\":null}")
    check(explicit.rows == null && explicit.requiredRows.single().contentEquals(byteArrayOf(3)))
    check(runCatching { codec.decodeFromString<TreesDto>("{}") }.isFailure)
    val encoded = codec.encodeToJsonElement(TreesDto.serializer(), value) as JsonObject
    check(encoded["rows"] == codec.parseToJsonElement("[[-128,0,127],[]]"))
    check(encoded["bag"] == codec.parseToJsonElement("{\"a\":[],\"z\":[-1]}"))
    check(encoded["nested"] == codec.parseToJsonElement("[{\"bytes\":[1,2]}]"))
    check(encoded.containsKey("emptyRows") == codec.configuration.encodeDefaults)
    check(encoded.containsKey("nullRows") == (codec.configuration.encodeDefaults && codec.configuration.explicitNulls))
    val roundTrip = codec.decodeFromString<TreesDto>(codec.encodeToString(TreesDto.serializer(), value))
    check(codec.encodeToJsonElement(TreesDto.serializer(), roundTrip) == encoded)
    for (bad in listOf("[[128]]", "[[-129]]", "[[1.5]]", "[[null]]", "[\"AAEC\"]")) {
        check(runCatching { codec.decodeFromString<TreesDto>("{\"requiredRows\":[[9]],\"rows\":$bad}") }.isFailure)
    }
    if (checked) check(runCatching { TreesDto(requiredRows = listOf(byteArrayOf(9)), rows = emptyList()) }.exceptionOrNull()?.message?.contains("minItems") == true)

    val patches = listOf(TreePatchInputDto(requiredRows = null), TreePatchInputDto(Patch.Null, null), TreePatchInputDto(Patch.Value(listOf(byteArrayOf(1))), null))
    for (patch in patches) {
        val tree = codec.encodeToJsonElement(TreePatchInputDto.serializer(), patch) as JsonObject
        when (patch.rows) {
            Patch.Absent -> check(!tree.containsKey("rows"))
            Patch.Null -> check(tree["rows"] == JsonNull)
            is Patch.Value -> check(tree["rows"] == codec.parseToJsonElement("[[1]]"))
        }
        check(tree.containsKey("requiredRows") == codec.configuration.explicitNulls)
    }
    val descriptor = TreePatchInputDto.serializer().descriptor
    check(descriptor.isElementOptional(descriptor.getElementIndex("rows")))
    check(!descriptor.isElementOptional(descriptor.getElementIndex("requiredRows")))
    bytesDefaultTreeHttp(codec, value, checked)
}

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BytesTreeEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun bytesTreeBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> bytesTreeBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected bytes tree HTTP body")
}

private suspend fun bytesTreeWireFailure(block: suspend () -> Unit) {
    var failure: Throwable? = null
    try { block() } catch (error: Throwable) { failure = error }
    check(failure != null && generateSequence(failure) { it.cause }.any { it is SerializationException || it is IllegalArgumentException }) { "Missing typed bytes failure: $failure" }
}

private suspend fun bytesDefaultTreeHttp(codec: Json, value: TreesDto, checked: Boolean) {
    var bad = false
    var calls = 0
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val engine = BytesTreeEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            val data = if (bad) "{\"requiredRows\":[[9]],\"rows\":[[128]]}" else "{\"requiredRows\":[[9]]}"
            when (request.url.encodedPath) {
                "/trees" -> {
                    check(bytesTreeBody(request.body) == codec.encodeToString(TreesDto.serializer(), value))
                    respond(data, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/tree-events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val keepOpen = bad
                    producers.launch {
                        try {
                            for (byte in "event: trees\r\ndata: $data\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (keepOpen) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected bytes tree path")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = BytesFieldDefaults(http, "http://example.test/", codec).default
        check(api.trees(value).rows!![0].contentEquals(byteArrayOf(-128, 0, 127)))
        if (checked) {
            val before = calls
            var callback = false
            bytesTreeWireFailure { api.trees(TreesDto(requiredRows = value.requiredRows, rows = emptyList())) { callback = true } }
            check(calls == before && !callback)
        }
        bad = true
        bytesTreeWireFailure { api.trees(value) }
        bad = false
        val flow = api.watchTrees()
        val first = withTimeout(10_000) { flow.toList().single() as WatchTreesEventDto.Trees }
        check(first.data.rows!![0].contentEquals(byteArrayOf(-128, 0, 127)))
        bad = true
        bytesTreeWireFailure { withTimeout(10_000) { api.watchTrees().toList() } }
        check(channels.last().isClosedForRead)
        bad = false
        check(withTimeout(10_000) { api.watchTrees().toList().size } == 1)
        check(api.trees(value).requiredRows.single().contentEquals(byteArrayOf(9)))
    } finally { http.close(); engine.close(); producers.cancel() }
}

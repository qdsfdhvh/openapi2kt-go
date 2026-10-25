@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, io.ktor.utils.io.InternalAPI::class)
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
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import org.example.objectfactory.ObjectFactoryClient
import org.example.objectfactory.api.models.PatchInputDto
import org.example.objectfactory.api.models.WatchEventDto
import org.example.objectfactory.models.HolderDto
import org.example.objectfactory.models.Patch
import org.example.objectfactory.models.RowsDto
import org.example.objectfactory.models.RowsDtoInner
import org.example.objectfactory.models.`AllOfRowsDto default`
import org.example.objectfactory.models.`MapsDto default`
import org.example.objectfactory.models.`MixedDto default`
import org.example.objectfactory.models.`NamedRowsDto default`
import org.example.objectfactory.models.`NarrowRowsDto default`
import org.example.objectfactory.models.`NullRowsDto default`
import org.example.objectfactory.models.`NullableAliasRowsDto default`
import org.example.objectfactory.models.`NullableRowsDto default`
import org.example.objectfactory.models.`RowsDto default`
import org.example.objectfactory.models.`WrappedRowsDto default`

private class ObjectCollectionEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext) ?: return response
        return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
    }
}

private fun objectCollectionBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> objectCollectionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected object collection request content")
}

private suspend fun objectCollectionFailure(block: suspend () -> Unit) {
    try { block() }
    catch (cause: CancellationException) { throw cause }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is SerializationException })
        return
    }
    error("Expected invalid required object member")
}

suspend fun objectCollectionFactoriesContracts(checked: Boolean = false) {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val rows = `RowsDto default`()
        check(rows.single().value == "root" && rows.single().mode?.serialName == "off" && rows.single().flag == false)
        check(rows !== `RowsDto default`() && rows.single() !== `RowsDto default`().single())
        check(RowsDtoInner("manual").mode?.serialName == "on" && RowsDtoInner("manual").flag == true)
        val mapped = `MapsDto default`().getValue("a/~")
        check(mapped.value == "map" && mapped.mode?.serialName == "on" && mapped.flag == true)
        val mixed = `MixedDto default`().single()
        check(mixed.getValue("a").isEmpty() && mixed.getValue("z").single().value == "mixed")
        check(`NamedRowsDto default`().single().value == "named")
        check(`AllOfRowsDto default`().single().let { it.a == "allof" && it.b == 2 })
        val nullable = `NullableRowsDto default`()!!
        check(nullable.first() == null && nullable.last()?.value == "nullable")
        check(`NullRowsDto default`() == null && `NullableAliasRowsDto default`() == null)
        check(`NarrowRowsDto default`().isEmpty() && `WrappedRowsDto default`().single().value == "wrapper")
        check(HolderDto(requiredRows = rows).rows == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredRows":[]}""").rows == null)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<RowsDto>(codec.encodeToString<RowsDto>(rows)) == rows)
        check(PatchInputDto().rows === Patch.Absent)
        objectCollectionWire(codec,rows,checked)
    }
}

private suspend fun objectCollectionWire(codec: Json, rows: RowsDto, checked: Boolean) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var calls = 0
    var callbacks = 0
    var payload = """[{"value":"雪🍃"}]"""
    var expectedPatch = JsonObject(emptyMap())
    val engine = ObjectCollectionEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(calls == callbacks)
        when (request.url.encodedPath) {
            "/send" -> {
                check(codec.parseToJsonElement(objectCollectionBody(request.body)) == codec.encodeToJsonElement<RowsDto>(rows))
                respond("""[{"value":"response"}]""",HttpStatusCode.OK,headersOf("Content-Type","application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(objectCollectionBody(request.body)) == expectedPatch)
                respond("",HttpStatusCode.NoContent)
            }
            "/watch" -> {
                check(codec.parseToJsonElement(objectCollectionBody(request.body)) == codec.encodeToJsonElement<RowsDto>(rows))
                val channel = ByteChannel(autoFlush = true); channels += channel
                val frame = "event: rows\r\ndata: $payload\r\n\r\n"
                scope.launch { try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)) } finally { channel.close() } }
                respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
            }
            else -> error("Unexpected object collection path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ObjectFactoryClient(http,"http://example.test/",codec).collections
        val response = api.send(rows) { callbacks++ }.single()
        check(response.value == "response" && response.mode?.serialName == "on" && response.flag == true)
        val before = calls
        val cold = api.watch(input = rows) { callbacks++ }
        check(calls == before)
        val event = withTimeout(10000) { cold.toList() }.single() as WatchEventDto.Rows
        check(event.data.single().value == "雪🍃" && event.data.single().flag == true)
        for ((patch,expected) in listOf(
            PatchInputDto() to JsonObject(emptyMap()),
            PatchInputDto(rows = Patch.Null) to JsonObject(mapOf("rows" to JsonNull)),
            PatchInputDto(rows = Patch.Value(rows)) to JsonObject(mapOf("rows" to codec.encodeToJsonElement<RowsDto>(rows))),
        )) {
            expectedPatch = expected
            api.patchPatch(patch) { callbacks++ }
        }
        if (checked) {
            val prior = calls; val priorCallbacks = callbacks
            check(runCatching { api.send(listOf(RowsDtoInner(""))) { callbacks++ } }.isFailure)
            check(calls == prior && callbacks == priorCallbacks)
        }
        payload = "[{}]"
        objectCollectionFailure { withTimeout(10000) { api.watch(input = rows) { callbacks++ }.toList() } }
        check(channels.all { it.isClosedForRead })
        check(api.send(rows) { callbacks++ }.single().value == "response")
    } finally { http.close(); scope.cancel(); channels.forEach { it.close() } }
}

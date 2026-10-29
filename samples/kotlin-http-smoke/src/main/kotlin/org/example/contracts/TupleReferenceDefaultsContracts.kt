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
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import org.example.reftupledefaults.api.DefaultApi
import org.example.reftupledefaults.api.models.EventsEventDto
import org.example.reftupledefaults.api.models.UpdateInputDto
import org.example.reftupledefaults.models.HolderDto
import org.example.reftupledefaults.models.RowsDto
import org.example.reftupledefaults.models.ChoiceDto
import org.example.reftupledefaults.models.ItemDto
import org.example.reftupledefaults.models.ComposedDto
import org.example.reftupledefaults.models.Patch
import org.example.reftupledefaults.models.`RowsDto default`
import org.example.reftupledefaults.models.`NullableItemsDto default`
import org.example.reftupledefaults.models.`ObjectsDto default`
import org.example.reftupledefaults.models.`ProductsDto default`
import org.example.reftupledefaults.models.`InternalRowsDto default`
import org.example.reftupledefaults.models.`InternalObjectsDto default`
import org.example.reftupledefaults.models.`InternalProductsDto default`
import org.example.reftupledefaults.models.`DatesDto default`
import org.example.reftupledefaults.models.`InstantsDto default`
import org.example.reftupledefaults.models.`UuidsDto default`
import org.example.reftupledefaults.models.`ChoiceDto default`
import org.example.reftupledefaults.models.`IntsDto default`
import org.example.reftupledefaults.models.`LongsDto default`
import org.example.reftupledefaults.models.`FloatsDto default`
import org.example.reftupledefaults.models.`DoublesDto default`
import org.example.reftupledefaults.models.`FlagsDto default`
import org.example.reftupledefaults.models.`TailDto default`
import org.example.reftupledefaults.models.`MaybeDto default`
import org.example.reftupledefaults.models.`AliasDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class RefDefaultTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun refDefaultTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> refDefaultTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline enum tuple body")
}

suspend fun tupleReferenceDefaultsContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = HolderDto(required = listOf(ItemDto(name = "outer")))
        check(value.rows == listOf(ChoiceDto.fromSerialName("paused")) && value.ints == listOf(2))
        check(value.borrowed == null)
        check(value.internalObjects!!.single().name == "internal" && value.internalObjects!!.single().n == 7)
        check(value.internalProducts!!.single().name == "internal" && value.internalProducts!!.single().n == 7)
        check(`InternalObjectsDto default`().single().name == "internal")
        check(`InternalProductsDto default`().single().name == "internal")
        check(value.rows !== HolderDto(required = value.required).rows)
        check(`RowsDto default`() == listOf(ChoiceDto.fromSerialName("ready")))
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`() == listOf(ChoiceDto.fromSerialName("paused")))
        check(`NullableItemsDto default`() == listOf(null, ItemDto(name = "nullable")))
        check(`ObjectsDto default`() == listOf(ItemDto(name = "root")))
        check(`ProductsDto default`() == listOf(ComposedDto(name = "root")))
        check(value.objects == listOf(ItemDto(name = "field")) && value.objects!!.single().n == 7)
        check(value.products == listOf(ComposedDto(name = "field")) && value.products!!.single().n == 7)
        check(value.objects !== HolderDto(required = value.required).objects)
        check(ItemDto.default().name == "target" && `ChoiceDto default`().serialName == "ready")
        check(`InternalRowsDto default`().single().serialName == "paused")
        check(`DatesDto default`().single().toString() == "2026-10-27")
        check(`InstantsDto default`().single().toString() == "2026-10-27T12:30:00Z")
        check(`UuidsDto default`().single().toString() == "550e8400-e29b-41d4-a716-446655440000")
        check(`IntsDto default`() == listOf(1))
        check(`LongsDto default`() == listOf(9007199254740993L))
        check(`FloatsDto default`() == listOf(0.1f))
        check(`DoublesDto default`() == listOf(2.5))
        check(`FlagsDto default`() == listOf(true))
        check(`TailDto default`().single().serialName == "paused")
        check(`MaybeDto default`() == null)
        val unknown = codec.decodeFromString<RowsDto>("""["future"]""")
        check(unknown.single() == ChoiceDto.fromSerialName("future"))
        check(codec.encodeToString(unknown) == """["future"]""")
        check(codec.decodeFromString<HolderDto>("""{"required":[{"name":"outer"}]}""") == value)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<RowsDto>("[1]") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":[],"rows":null}""").rows == null)
        val encoded = codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(("rows" in encoded) == defaults)
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var bad = false
        var patchMode = 0
        var calls = 0
        val engine = RefDefaultTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(refDefaultTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[{"name":"outer"}],"ints":[3]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/shape-a", "/shape-b" -> respond("""{"rows":["future"]}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val body = codec.parseToJsonElement(refDefaultTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in body)
                            1 -> check(body["optional"] == JsonNull)
                            2 -> check(body["optional"] == JsonArray(listOf(JsonPrimitive("paused"))))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (bad) """{"required":[{"name":"outer"}],"ints":[3]}""" else codec.encodeToString(value)
                        val frame = "event: message\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic inline enum tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(api.shapeA().rows?.single()?.serialName == "future")
            check(api.shapeB().rows?.single()?.serialName == "future")
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Value(listOf(ChoiceDto.fromSerialName("paused")))))
            val event = api.events().take(1).toList().single()
            check(event is EventsEventDto.Message && event.data == value)
            check(channels.last().isClosedForRead)
            check(runCatching { codec.decodeFromString<HolderDto>("""{"required":[{}]}""") }.isFailure)
            check(codec.decodeFromString<HolderDto>("""{"required":[{"name":"outer"}],"rows":["future"]}""").rows?.single()?.serialName == "future")
            if (checked) {
                val before = calls
                check(runCatching { api.echo(HolderDto(required = value.required, ints = listOf(3))) }.isFailure)
                check(calls == before)
                check(runCatching { codec.decodeFromString<HolderDto>("""{"required":[{"name":"outer"}],"ints":[3]}""") }.isFailure)
                bad = true
                check(runCatching { api.echo(value) }.isFailure)
                check(runCatching { api.events().take(1).toList() }.isFailure)
                check(channels.last().isClosedForRead)
                bad = false
                check(api.echo(value) == value && api.events().take(1).toList().size == 1)
            } else {
                check(codec.decodeFromString<HolderDto>("""{"required":[{"name":"outer"}],"ints":[3]}""").ints == listOf(3))
            }
        } finally {
            client.close()
            channels.forEach { it.cancel(null) }
            writers.forEach { it.cancel() }
            scope.cancel()
        }
    }
}

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
import org.example.inlinedtotuples.directional.api.DefaultApi as DirectionalTupleApi
import org.example.inlinedtotuples.directional.api.models.EchoInputDto
import org.example.inlinedtotuples.directional.api.models.EchoPublicResponseDto
import org.example.inlinedtotuples.directional.api.models.EchoPublicResponse2Dto
import org.example.inlinedtotuples.api.DefaultApi
import org.example.inlinedtotuples.api.models.EventsEventDto
import org.example.inlinedtotuples.api.models.UpdateInputDto
import org.example.inlinedtotuples.models.HolderDto
import org.example.inlinedtotuples.models.CountedDto
import org.example.inlinedtotuples.models.RowsDto
import org.example.inlinedtotuples.models.Patch
import org.example.inlinedtotuples.models.`CountedDto default`
import org.example.inlinedtotuples.models.`RowsDto default`
import org.example.inlinedtotuples.models.`AliasDto default`
import org.example.inlinedtotuples.models.`NullableRowsDto default`
import org.example.inlinedtotuples.models.`MaybeDto default`
import org.example.inlinedtotuples.models.`MixedDto default`
import org.example.inlinedtotuples.models.`TailDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class InlineDtoTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun inlineDtoTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> inlineDtoTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline DTO tuple body")
}

suspend fun tupleInlineDtoContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        inlineDtoTupleDirectional(codec)
        val value = HolderDto(
            required = listOf(HolderDto.RequiredInner(name = "outer")),
            nested = listOf(mapOf("key" to listOf(HolderDto.NestedInnerValueInner(name = "nested")))),
        )
        check(value.rows!!.single().name == "field")
        check(value.rows!!.single().n == 7 && value.rows!!.single().state?.serialName == "ready")
        check(value.rows!!.single().child?.label == "inner" && value.borrowed == null)
        val other = HolderDto(required = value.required)
        check(value.rows !== other.rows && value.rows!!.single() !== other.rows!!.single())
        check(value.rows!!.single().child !== other.rows!!.single().child)
        check(`CountedDto default`().single().n == 1)
        val root = `RowsDto default`()
        check(root.single().name == "root" && root.single().n == 7)
        check(root.single().state?.serialName == "ready" && root.single().child?.label == "inner")
        check(root !== `RowsDto default`() && root.single().child !== `RowsDto default`().single().child)
        check(`AliasDto default`().single().name == "alias")
        val nullable = `NullableRowsDto default`()
        check(nullable.first() == null && nullable.last()?.name == "nullable")
        check(`MaybeDto default`() == null)
        check(`TailDto default`().single().name == "tail")
        check(`MixedDto default`() == codec.decodeFromString<List<JsonElement>>("""[null,{"default":"data"}]"""))
        val unknown = codec.decodeFromString<RowsDto>("""[{"name":"future","state":"future"}]""")
        check(unknown.single().state?.serialName == "future")
        check(codec.decodeFromString<RowsDto>(codec.encodeToString(unknown)) == unknown)
        // Lists intentionally do not enforce tuple positional arity or closed-tail length.
        check(codec.decodeFromString<RowsDto>("[]").isEmpty())
        check(codec.decodeFromString<RowsDto>("""[{"name":"a"},{"name":"b"}]""").size == 2)
        check(runCatching { codec.decodeFromString<RowsDto>("[null]") }.isFailure)
        check(runCatching { codec.decodeFromString<RowsDto>("[{}]") }.isFailure)
        check(runCatching { codec.decodeFromString<RowsDto>("[1]") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(value)) == value)
        check(codec.decodeFromString<HolderDto>("""{"required":[],"rows":null}""").rows == null)
        val encoded = codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(("rows" in encoded) == defaults)
        check(UpdateInputDto(required = null).optional == Patch.Absent)
        if (nulls) {
            check(runCatching { codec.decodeFromString<UpdateInputDto>("{}") }.isFailure)
        } else {
            check(codec.decodeFromString<UpdateInputDto>("{}").required == null)
        }
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val writers = CopyOnWriteArrayList<Job>()
        var bad = false
        var patchMode = 0
        var calls = 0
        var expectedPatch: JsonElement? = null
        val engine = InlineDtoTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(inlineDtoTupleBody(request.body)) == value)
                        respond(if (bad) """{"required":[{"name":""}]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/inline" -> respond("""[{"name":"direct","state":"future","extra":7},{}]""", headers = headersOf("Content-Type", "application/json"))
                    "/shape-a", "/shape-b" -> respond("""{"name":"hoisted"}""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val body = codec.parseToJsonElement(inlineDtoTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in body)
                            1 -> check(body["optional"] == JsonNull)
                            2 -> check(body["optional"] == expectedPatch)
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (bad) """{"required":[{"name":""}]}""" else codec.encodeToString(value)
                        val frame = "event: message\r\ndata: $payload\r\n\r\n"
                        writers.add(scope.launch {
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            awaitCancellation()
                        })
                        respond(channel, headers = headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected synthetic inline DTO tuple URL")
                }
            }
        })
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val api = DefaultApi(client, "https://example.test/", codec)
            check(api.echo(value) == value)
            check(api.shapeA().name == "hoisted" && api.shapeB().name == "hoisted")
            val projected = api.inlineRows()
            check(projected.first()["name"] == JsonPrimitive("direct"))
            check(projected.first()["state"] == JsonPrimitive("future"))
            check(projected.first()["extra"] == JsonPrimitive(7) && projected.last().isEmpty())
            api.updateUpdate(UpdateInputDto(required = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, optional = Patch.Null))
            patchMode = 2
            val patchValue = codec.decodeFromString<UpdateInputDto.OptionalInner>("""{"name":"patch","state":null,"n":null,"child":null}""")
            // Use explicit nullable leaf states; encodeDefaults/explicitNulls still apply inside the DTO.
            val patch = UpdateInputDto(required = null, optional = Patch.Value(listOf(patchValue)))
            val patchBody = codec.parseToJsonElement(codec.encodeToString(patch)).jsonObject
            check(patchBody["optional"] is JsonArray)
            expectedPatch = patchBody["optional"]
            api.updateUpdate(patch)
            val event = api.events().take(1).toList().single()
            check(event is EventsEventDto.Message && event.data == value)
            check(channels.last().isClosedForRead)
            if (checked) {
                check(runCatching { codec.decodeFromString<CountedDto>("""[{"name":"count"}]""") }.isFailure)
                val before = calls
                check(runCatching { api.echo(HolderDto(required = listOf(HolderDto.RequiredInner(name = "")))) }.isFailure)
                check(calls == before)
                check(runCatching { codec.decodeFromString<RowsDto>("""[{"name":""}]""") }.isFailure)
                check(runCatching { codec.decodeFromString<RowsDto>("""[{"name":"ok","n":-1}]""") }.isFailure)
                bad = true
                check(runCatching { api.echo(value) }.isFailure)
                check(runCatching { api.events().take(1).toList() }.isFailure)
                check(channels.last().isClosedForRead)
                bad = false
                check(api.echo(value) == value && api.events().take(1).toList().size == 1)
            } else {
                check(codec.decodeFromString<CountedDto>("""[{"name":"count"}]""").single().n == 7)
                check(codec.decodeFromString<RowsDto>("""[{"name":"","n":-1}]""").single().n == -1)
            }
        } finally {
            client.close()
            channels.forEach { it.cancel(null) }
            writers.forEach { it.cancel() }
            scope.cancel()
        }
    }
}

private suspend fun inlineDtoTupleDirectional(codec: Json) {
    val request = EchoInputDto(rows = listOf(EchoInputDto.RowsInner(name = "input", secret = "synthetic")))
    val response = EchoPublicResponseDto(rows = listOf(EchoPublicResponse2Dto(name = "output", id = 1)))
    val decoded = codec.decodeFromString<EchoInputDto>(codec.encodeToString(request))
    check(decoded.rows.single().id == org.example.inlinedtotuples.directional.models.Patch.Absent)
    val engine = MockEngine { call ->
        val json = codec.parseToJsonElement(inlineDtoTupleBody(call.body)).jsonObject
        val row = (json.getValue("rows") as JsonArray).single().jsonObject
        check(row["secret"] == JsonPrimitive("synthetic") && "id" !in row)
        respond("""{"rows":[{"name":"output","id":1}]}""", headers = headersOf("Content-Type", "application/json"))
    }
    val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
    try {
        val api = DirectionalTupleApi(client, "https://example.test/")
        val actual = api.echo(request)
        check(actual == response && actual.rows.single().id == 1)
        val publicRow = (codec.parseToJsonElement(codec.encodeToString(actual)).jsonObject.getValue("rows") as JsonArray).single().jsonObject
        check("secret" !in publicRow)
        check(runCatching { codec.decodeFromString<EchoPublicResponseDto>("""{"rows":[{"name":"output","id":1,"secret":"extra"}]}""") }.isFailure)
        check(codec.decodeFromString<EchoInputDto>("""{"rows":[{"name":"input","secret":"synthetic","id":2}]}""").rows.single().id == org.example.inlinedtotuples.directional.models.Patch.Value(2))
    } finally {
        client.close()
    }
}

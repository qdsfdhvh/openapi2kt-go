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
import kotlinx.coroutines.yield
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import org.example.inlinetuples.api.DefaultApi
import org.example.inlinetuples.api.models.EventsEventDto
import org.example.inlinetuples.api.models.UpdateInputDto
import kotlin.time.Instant
import kotlin.uuid.Uuid
import org.example.scalarvalues.Money
import kotlinx.datetime.LocalDate
import org.example.inlinetuples.models.OrdinaryMetadataDto
import org.example.inlinetuples.models.MetaOwnerDto
import org.example.inlinetuples.models.MetaInternalHolderDto
import org.example.inlinetuples.models.MetaInternalObjectDto
import org.example.inlinetuples.models.`MetaInternalDefaultDto default`
import org.example.inlinetuples.models.`MetaInternalMaybeDto default`
import org.example.inlinetuples.models.`MetaInternalNarrowDto default`
import org.example.inlinetuples.models.`PlainInternalDefaultDto default`
import org.example.inlinetuples.models.`MetaListDefaultDto default`
import org.example.inlinetuples.models.`MetaDateDto default`
import org.example.inlinetuples.models.`MetaDateAliasDto default`
import org.example.inlinetuples.models.`MetaMaybeDto default`
import org.example.inlinetuples.models.`MetaNarrowDto default`
import org.example.inlinetuples.models.`MetaIntrinsicDto default`
import org.example.inlinetuples.models.`MetaIntrinsicPeerDto default`
import org.example.inlinetuples.models.`OrdinaryRowsDto default`
import org.example.inlinetuples.models.OrdinaryObjectDto
import org.example.inlinetuples.models.MetadataDto
import org.example.inlinetuples.models.MetaDateDto
import org.example.inlinetuples.models.MetaDateInlineRowsDto
import org.example.inlinetuples.models.MetaDateNamedRowsDto
import org.example.inlinetuples.models.`MetaDateInlineRowsDto default`
import org.example.inlinetuples.models.`MetaDateNamedRowsDto default`
import org.example.inlinetuples.models.`MetaAliasRowsDto default`
import org.example.inlinetuples.models.HolderDto
import org.example.inlinetuples.models.RowsDto
import org.example.inlinetuples.models.RowsDtoInner
import org.example.inlinetuples.models.NullableRowsDto
import org.example.inlinetuples.models.NullableRowsDtoInner
import org.example.inlinetuples.models.Patch
import org.example.inlinetuples.models.`RowsDto default`
import org.example.inlinetuples.models.`NullableRowsDto default`
import org.example.inlinetuples.models.`IntsDto default`
import org.example.inlinetuples.models.`LongsDto default`
import org.example.inlinetuples.models.`FloatsDto default`
import org.example.inlinetuples.models.`DoublesDto default`
import org.example.inlinetuples.models.`FlagsDto default`
import org.example.inlinetuples.models.`TailDto default`
import org.example.inlinetuples.models.`MixedDto default`
import org.example.inlinetuples.models.`MaybeDto default`
import org.example.inlinetuples.models.`AliasDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class InlineEnumTupleStreamEngine(config: MockEngineConfig) : MockEngine(config) {
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

private fun inlineEnumTupleBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> inlineEnumTupleBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic inline enum tuple body")
}

suspend fun tupleInlineEnumsContracts(checked: Boolean = false) = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val ordinary = codec.decodeFromString<OrdinaryMetadataDto>("""{"required":"ready"}""")
        check(ordinary == OrdinaryMetadataDto(required = ordinary.required))
        check(ordinary.state?.serialName == "paused" && ordinary.named?.serialName == "ready")
        check(ordinary.rows?.single()?.serialName == "ready")
        check(ordinary.map?.get("first")?.serialName == "ready" && ordinary.map?.get("second") == null)
        check(ordinary.nested?.get("first")?.single()?.serialName == "ready")
        check(ordinary.refRows?.single()?.serialName == "paused")
        check(ordinary.borrowed == MetaOwnerDto.Value.fromSerialName("ready"))
        check(ordinary.maybe == null && ordinary.unannotated == null)
        check(ordinary.narrow?.serialName == "ready")
        check(`MetaDateDto default`().serialName == "ready")
        check(`MetaDateAliasDto default`().serialName == "paused")
        check(`MetaMaybeDto default`() == null && `MetaIntrinsicDto default`() == null && `MetaIntrinsicPeerDto default`() == null)
        check(`MetaNarrowDto default`().serialName == "paused")
        check(`MetaInternalDefaultDto default`().serialName == "paused")
        check(`MetaInternalMaybeDto default`() == null)
        check(`MetaInternalNarrowDto default`().serialName == "ready")
        check(`PlainInternalDefaultDto default`().serialName == "paused")
        check(`MetaListDefaultDto default`().serialName == "ready")
        check(MetaOwnerDto().value?.serialName == "ready")
        check(MetaInternalObjectDto.default().state.serialName == "ready")
        val internal = codec.decodeFromString<MetaInternalHolderDto>("""{"required":"ready"}""")
        check(internal.required.serialName == "ready" && internal.state?.serialName == "ready")
        check(internal.rows?.map { it.serialName } == listOf("ready", "paused"))
        check(internal.unannotated == null && internal.maybe == null)
        check(runCatching { codec.decodeFromString<MetaInternalHolderDto>("{}").required }.isFailure)
        check(codec.decodeFromString<MetaInternalHolderDto>("""{"required":"future","state":null}""").required.serialName == "future")
        check(`OrdinaryRowsDto default`().single().serialName == "ready")
        check(OrdinaryObjectDto.default().state.serialName == "ready")
        check(runCatching { codec.decodeFromString<OrdinaryMetadataDto>("{}") }.isFailure)
        check(codec.decodeFromString<OrdinaryMetadataDto>(codec.encodeToString(ordinary)) == ordinary)
        val unknownOrdinary = codec.decodeFromString<OrdinaryMetadataDto>("""{"required":"future","borrowed":"unknown-byte-word","state":"not-an-instant"}""")
        check(unknownOrdinary.borrowed?.serialName == "unknown-byte-word" && unknownOrdinary.state?.serialName == "not-an-instant")
        check(ordinary.uniqueRows?.map { it.serialName } == listOf("ready", "paused"))
        val duplicateRows = """{"required":"ready","uniqueRows":["ready","ready"]}"""
        val duplicateMaps = """{"required":"ready","uniqueMaps":[{"a":"ready","b":"paused"},{"b":"paused","a":"ready"}]}"""
        val duplicateObjects = """{"required":"ready","uniqueObjects":[{"state":"ready"},{"state":"ready"}]}"""
        if (checked) {
            check(runCatching { codec.decodeFromString<OrdinaryMetadataDto>(duplicateRows) }.isFailure)
            check(runCatching { codec.decodeFromString<OrdinaryMetadataDto>(duplicateObjects) }.isFailure)
            check(runCatching { codec.decodeFromString<OrdinaryMetadataDto>(duplicateMaps) }.isFailure)
        } else {
            check(codec.decodeFromString<OrdinaryMetadataDto>(duplicateRows).uniqueRows?.size == 2)
            check(codec.decodeFromString<OrdinaryMetadataDto>(duplicateObjects).uniqueObjects?.size == 2)
            check(codec.decodeFromString<OrdinaryMetadataDto>(duplicateMaps).uniqueMaps?.size == 2)
        }
        val lexicalUuid = codec.decodeFromString<OrdinaryMetadataDto>("""{"required":"ready","uniqueRows":["a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","A0EEBC99-9C0B-4EF8-BB6D-6BB9BD380A11"]}""")
        check(lexicalUuid.uniqueRows?.size == 2)
        val metadata = MetadataDto()
        val metadataJson = codec.parseToJsonElement(codec.encodeToString(metadata)).jsonObject
        if (defaults) {
            check(metadataJson.filterKeys { it != "nullableWords" }.values.all { it == JsonArray(listOf(JsonPrimitive("ready"))) })
            check(metadataJson["nullableWords"] == JsonArray(listOf(JsonNull, JsonPrimitive("ready"))))
        }
        check(metadata.nullableWords?.first() == null && metadata.nullableWords?.last()?.serialName == "ready")
        check(codec.decodeFromString<MetadataDto>(codec.encodeToString(metadata)) == metadata)
        check(`MetaDateInlineRowsDto default`().single().serialName == "ready")
        check(`MetaDateNamedRowsDto default`() == listOf(MetaDateDto.fromSerialName("ready")))
        check(`MetaAliasRowsDto default`() == listOf(MetaDateDto.fromSerialName("paused")))
        check(`MetaDateNamedRowsDto default`() !== `MetaDateNamedRowsDto default`())
        check(codec.decodeFromString<MetaDateInlineRowsDto>("""["not-a-date"]""").single().serialName == "not-a-date")
        check(codec.decodeFromString<MetaDateNamedRowsDto>("""["future"]""") == listOf(MetaDateDto.fromSerialName("future")))
        check(runCatching { codec.decodeFromString<MetaDateNamedRowsDto>("[7]") }.isFailure)
        val value = HolderDto(required = listOf(HolderDto.RequiredInner.READY), metadata = metadata, ordinary = ordinary, internalDefaults = internal)
        check(value.rows == listOf(HolderDto.RowsInner.PAUSED) && value.ints == listOf(2))
        check(value.borrowed == null)
        check(value.rows !== HolderDto(required = value.required).rows)
        check(`RowsDto default`() == listOf(RowsDtoInner.fromSerialName("ready")))
        check(`RowsDto default`() !== `RowsDto default`())
        check(`AliasDto default`() == listOf(RowsDtoInner.fromSerialName("paused")))
        check(`NullableRowsDto default`() == listOf(null, NullableRowsDtoInner.fromSerialName("ready")))
        check(`IntsDto default`() == listOf(1))
        check(`LongsDto default`() == listOf(9007199254740993L))
        check(`FloatsDto default`() == listOf(0.1f))
        check(`DoublesDto default`() == listOf(2.5))
        check(`FlagsDto default`() == listOf(true))
        check(`TailDto default`().single().serialName == "paused")
        check(`MaybeDto default`() == null)
        check(`MixedDto default`() == codec.parseToJsonElement("""[null,{"enum":"data"}]"""))
        val unknown = codec.decodeFromString<RowsDto>("""["future"]""")
        check(unknown.single() == RowsDtoInner.fromSerialName("future"))
        check(codec.encodeToString(unknown) == """["future"]""")
        check(codec.decodeFromString<NullableRowsDto>("""[null,"null"]""") == listOf(null, NullableRowsDtoInner.fromSerialName("null")))
        check(codec.decodeFromString<HolderDto>("""{"required":["ready"],"metadata":{},"ordinary":{"required":"ready"},"internalDefaults":{"required":"ready"}}""") == value)
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
        var badMetadata = false
        var patchMode = 0
        var calls = 0
        val engine = InlineEnumTupleStreamEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/echo" -> {
                        check(codec.decodeFromString<HolderDto>(inlineEnumTupleBody(request.body)) == value)
                        respond(if (badMetadata) """{"required":["ready"],"metadata":{"dateNamed":[7]}}""" else if (bad) """{"required":["ready"],"ints":[3]}""" else codec.encodeToString(value), headers = headersOf("Content-Type", "application/json"))
                    }
                    "/shape-a", "/shape-b" -> respond("""{"rows":["future"]}""", headers = headersOf("Content-Type", "application/json"))
                    "/inline" -> respond("""["2026-01-02"]""", headers = headersOf("Content-Type", "application/json"))
                    "/meta-uuid" -> respond("""["a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"]""", headers = headersOf("Content-Type", "application/json"))
                    "/meta-timestamp" -> respond("""["2026-01-02T03:04:05Z"]""", headers = headersOf("Content-Type", "application/json"))
                    "/meta-provider" -> respond("""["12.50"]""", headers = headersOf("Content-Type", "application/json"))
                    "/meta-byte" -> respond("[[1,-128,127]]", headers = headersOf("Content-Type", "application/json"))
                    "/meta-native" -> respond("[7]", headers = headersOf("Content-Type", "application/json"))
                    "/meta-named" -> respond("""["future"]""", headers = headersOf("Content-Type", "application/json"))
                    "/patch" -> {
                        val body = codec.parseToJsonElement(inlineEnumTupleBody(request.body)).jsonObject
                        when (patchMode) {
                            0 -> check("optional" !in body && "ordinaryOptional" !in body)
                            1 -> check(body["optional"] == JsonNull)
                            2 -> check(body["optional"] == JsonArray(listOf(JsonPrimitive("paused"))))
                            3 -> check("optional" !in body && body["ordinaryOptional"] == JsonPrimitive("paused"))
                        }
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels.add(channel)
                        val payload = if (badMetadata) """{"required":["ready"],"metadata":{"dateNamed":[7]}}""" else if (bad) """{"required":["ready"],"ints":[3]}""" else codec.encodeToString(value)
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
            check(api.inlineRows() == listOf(LocalDate(2026, 1, 2)))
            check(api.metaByte().single().contentEquals(byteArrayOf(1, -128, 127)))
            check(api.metaNative() == listOf(7L))
            check(api.metaUuid() == listOf(Uuid.parse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")))
            check(api.metaTimestamp() == listOf(Instant.parse("2026-01-02T03:04:05Z")))
            check(api.metaProvider() == listOf(Money("12.50")))
            check(api.metaNamed() == listOf(MetaDateDto.fromSerialName("future")))
            check(api.shapeA().rows?.single()?.serialName == "future")
            check(api.shapeB().rows?.single()?.serialName == "future")
            api.updateUpdate(UpdateInputDto(required = null, ordinaryRequired = null))
            patchMode = 1
            api.updateUpdate(UpdateInputDto(required = null, ordinaryRequired = null, optional = Patch.Null))
            patchMode = 2
            api.updateUpdate(UpdateInputDto(required = null, ordinaryRequired = null, optional = Patch.Value(listOf(UpdateInputDto.OptionalInner.PAUSED))))
            patchMode = 3
            api.updateUpdate(UpdateInputDto(required = null, ordinaryRequired = null, ordinaryOptional = Patch.Value(UpdateInputDto.OrdinaryOptional.fromSerialName("paused"))))
            val event = api.events().take(1).toList().single()
            check(event is EventsEventDto.Message && event.data == value)
            withTimeout(2000) { while (!channels.last().isClosedForRead) yield() }
            badMetadata = true
            check(runCatching { api.echo(value) }.isFailure)
            check(runCatching { api.events().take(1).toList() }.isFailure)
            withTimeout(2000) { while (!channels.last().isClosedForRead) yield() }
            badMetadata = false
            check(api.echo(value) == value)
            val reused = api.events().take(1).toList().single()
            check(reused is EventsEventDto.Message && reused.data == value)
            withTimeout(2000) { while (!channels.last().isClosedForRead) yield() }
            if (checked) {
                val before = calls
                check(runCatching { api.echo(HolderDto(required = value.required, ints = listOf(3))) }.isFailure)
                check(calls == before)
                check(runCatching { codec.decodeFromString<HolderDto>("""{"required":["ready"],"ints":[3]}""") }.isFailure)
                bad = true
                check(runCatching { api.echo(value) }.isFailure)
                check(runCatching { api.events().take(1).toList() }.isFailure)
            } else {
                check(codec.decodeFromString<HolderDto>("""{"required":["ready"],"ints":[3]}""").ints == listOf(3))
            }
        } finally {
            client.close()
            channels.forEach { it.cancel(null) }
            writers.forEach { it.cancel() }
            scope.cancel()
        }
    }
}

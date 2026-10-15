@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
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
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.rootaliases.RootAliasApiClient
import org.example.rootaliases.api.models.SubmitInputDto
import org.example.rootaliases.api.models.WatchEventDto
import org.example.rootaliases.models.ARecordDto
import org.example.rootaliases.models.BRecordDto
import org.example.rootaliases.models.HolderDto
import org.example.rootaliases.models.Patch
import org.example.rootaliases.models.ZRecordDto
import org.example.rootaliasprobes.models.ADeepDto
import org.example.rootaliasprobes.models.AWideDto
import org.example.rootaliasprobes.models.CheckedAliasDto
import org.example.rootaliasprobes.models.ComposedAliasDto
import org.example.rootaliasprobes.models.ZComposedDto

private fun rootAliasReject(block: () -> Unit) {
    try { block(); error("Missing canonical root-alias validation") }
    catch (cause: IllegalArgumentException) { check(cause.message?.contains("minLength") == true) }
}

private suspend fun rootAliasWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing canonical alias wire failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is SerializationException || it is IllegalArgumentException) && it.message?.contains(message) == true
        }) { "Wrong root alias failure: $cause" }
    }
}

private fun rootAliasBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> rootAliasBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected root alias body $body")
}

suspend fun rootObjectAliasContracts() {
    // Both constructors and companion serializers resolve to the original DTO.
    val alias: ARecordDto = ARecordDto("parent", BRecordDto("child"))
    val second: BRecordDto = alias
    val original: ZRecordDto = second
    check(ARecordDto.serializer().descriptor == ZRecordDto.serializer().descriptor)
    check(BRecordDto.serializer().descriptor == ZRecordDto.serializer().descriptor)
    check(alias.copy(id = "next") is ZRecordDto)
    val holder = HolderDto(alias, second, list = listOf(alias), entries = mapOf("k" to alias))
    rootAliasReject { CheckedAliasDto("") }
    val checked = CheckedAliasDto("valid")
    val composed = ComposedAliasDto("valid", true)
    check(ComposedAliasDto.serializer().descriptor == ZComposedDto.serializer().descriptor)
    rootAliasReject { ComposedAliasDto("") }
    rootAliasReject { composed.copy(id = "") }
    rootAliasReject { checked.copy(id = "") }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val streamCodec = Json(codec) { ignoreUnknownKeys = false }
        val wire = codec.encodeToString(ARecordDto.serializer(), alias)
        check(codec.decodeFromString<ARecordDto>(wire) == original)
        check(codec.decodeFromString<BRecordDto>(wire) == original)
        check(codec.decodeFromString(ARecordDto.serializer(), wire) == original)
        check(codec.decodeFromString(HolderDto.serializer(), codec.encodeToString(HolderDto.serializer(), holder)) == holder)
        rootAliasReject { codec.decodeFromString<CheckedAliasDto>("{\"id\":\"\"}") }
        check(codec.decodeFromString<ComposedAliasDto>(codec.encodeToString(composed)) == composed)
        val absentFlag: ComposedAliasDto = ComposedAliasDto("valid")
        check(codec.decodeFromString<ComposedAliasDto>(codec.encodeToString(absentFlag)) == absentFlag)
        rootAliasReject { codec.decodeFromString<ComposedAliasDto>("{\"id\":\"\"}") }
        val wide = AWideDto(p0 = checked)
        check(codec.decodeFromString(AWideDto.serializer(), codec.encodeToString(AWideDto.serializer(), wide)) == wide)
        val deep = ADeepDto(checked)
        check(codec.decodeFromString(ADeepDto.serializer(), codec.encodeToString(ADeepDto.serializer(), deep)) == deep)
        var calls = 0
        var aliasCalls = 0
        var directCalls = 0
        var mode = 0
        var expectedPatch = ""
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct" -> {
                        if (request.url.encodedPath == "/alias") aliasCalls++ else directCalls++
                        check(rootAliasBody(request.body) == wire)
                        respond(if (mode == 1) "{}" else wire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(rootAliasBody(request.body) == expectedPatch)
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val payload = when (current) {
                                    1 -> "{}"
                                    2 -> "{\"id\":\"valid\",\"extra\":true}"
                                    else -> wire
                                }
                                val frame = "event: record\r\nid: alias-id\r\ndata: $payload\r\n\r\n"
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current != 0) awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected alias route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootAliasApiClient(http, "http://example.test/", streamCodec).default
            check(api.echo(alias) == alias)
            check(api.echo2(alias) == alias)
            check(aliasCalls == 1 && directCalls == 1)
            mode = 1
            rootAliasWireReject("id") { api.echo(alias) }
            mode = 0
            for (state in listOf<Patch<ZRecordDto>>(Patch.Absent, Patch.Null, Patch.Value(alias))) {
                val input = SubmitInputDto(optional = state, requiredNull = null)
                val encoded = codec.encodeToJsonElement(SubmitInputDto.serializer(), input) as JsonObject
                check(when (state) {
                    Patch.Absent -> "optional" !in encoded
                    Patch.Null -> encoded["optional"] == JsonNull
                    is Patch.Value -> encoded["optional"] == codec.parseToJsonElement(wire)
                })
                check(encoded.containsKey("requiredNull") == nulls)
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(), encoded) == input)
                expectedPatch = codec.encodeToString(SubmitInputDto.serializer(), input)
                api.submit(input)
            }
            val before = calls
            val cold = api.watch()
            check(calls == before)
            repeat(2) {
                val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
                check(event.data == alias && event.id == "alias-id")
            }
            check(calls == before + 2)
            for ((bad, message) in listOf(1 to "id", 2 to "extra")) {
                mode = bad
                rootAliasWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
                check(channels.last().isClosedForRead) { "Alias payload failure left still-open stream alive" }
            }
            mode = 0
            check(withTimeout(10_000) { api.watch().toList().size } == 1)
            check(api.echo(alias) == alias)
        } finally { http.close(); engine.close(); producers.cancel() }
        println("Root object alias contracts passed: defaults=$defaults/nulls=$nulls; alias constructors/serializer/canonical recursion/240-wide/48-source-hop/Patch/HTTP/live SSE close-reuse-cold")
    }
}

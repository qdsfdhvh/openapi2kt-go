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
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import org.example.runtimealpha.SharedRuntimeExampleApiClient as AlphaClient
import org.example.runtimebeta.SharedRuntimeExampleApiClient as BetaClient
import org.example.runtimealpha.api.models.SubmitInputDto as AlphaInput
import org.example.runtimebeta.api.models.SubmitInputDto as BetaInput
import org.example.runtimealpha.api.models.WatchEventDto as AlphaEvent
import org.example.runtimebeta.api.models.WatchEventDto as BetaEvent
import org.example.runtimealpha.models.RecordDto as AlphaRecord
import org.example.runtimebeta.models.RecordDto as BetaRecord
import org.example.shared.runtime.Patch
import org.example.runtimecollisions.models.Patch as ScopedModelPatch
import org.example.runtimecollisions.models.PatchSerializer as ScopedModelPatchSerializer
import org.example.runtimecollisions.models.LongAsStringSerializer as ScopedModelLongSerializer

private fun sharedBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> sharedBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected shared runtime body")
}
private fun sharedReject(block: () -> Unit) {
    try { block(); error("Invalid shared runtime input accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException || it is SerializationException }) { "Wrong shared-runtime failure $cause" } }
}
suspend fun runtimePackageContracts() = withTimeout(15000) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val scopedModel = ScopedModelPatch("model")
        check(scopedModel.copy(id = "copy").id == "copy")
        check(codec.decodeFromString<ScopedModelPatch>(codec.encodeToString(scopedModel)) == scopedModel)
        check(codec.decodeFromString<ScopedModelPatchSerializer>("{\"id\":\"codec-name\"}").id == "codec-name")
        check(codec.decodeFromString<ScopedModelLongSerializer>("{\"id\":\"bigint-name\"}").id == "bigint-name")
        val shared: Patch<Long> = Patch.Value(17L)
        val a = AlphaInput("ok", shared)
        val b = BetaInput("ok", shared)
        val expected = JsonObject(mapOf("id" to JsonPrimitive("ok"), "amount" to JsonPrimitive("17")))
        check(codec.encodeToJsonElement(a) == expected && codec.encodeToJsonElement(b) == expected)
        check(codec.decodeFromString<AlphaInput>(codec.encodeToString(a)) == a)
        check(codec.decodeFromString<BetaInput>(codec.encodeToString(b)) == b)
        check(Patch.ofNullable<Long>(null) == Patch.Null && Patch.orAbsent<Long>(null) == Patch.Absent)
        check(codec.decodeFromString<BetaInput>("{\"id\":\"ok\",\"amount\":null}").amount == Patch.Null)
        check(codec.decodeFromString<AlphaInput>("{\"id\":\"ok\"}").amount == Patch.Absent)
        sharedReject { AlphaInput("x") }
        sharedReject { b.copy(id = "x") }
        sharedReject { codec.decodeFromString<AlphaInput>("{\"id\":\"x\"}") }
        sharedReject { codec.decodeFromString<BetaInput>("{\"id\":\"ok\",\"amount\":17}") }
        val record = AlphaRecord(23L, listOf(23L))
        val recordJSON = "{\"amount\":\"23\",\"values\":[23]}"
        check(codec.encodeToJsonElement(record) == codec.parseToJsonElement(recordJSON))
        check(codec.decodeFromString<AlphaRecord>(recordJSON) == record)
        check(codec.decodeFromString<BetaRecord>(recordJSON) == BetaRecord(23L, listOf(23L)))
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val channels = CopyOnWriteArrayList<ByteChannel>()
        var mode = "ok"
        var submits = 0
        var streams = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                when (request.url.encodedPath) {
                    "/submit" -> {
                        val tree = codec.parseToJsonElement(sharedBody(request.body)) as JsonObject
                        check(tree["id"] == JsonPrimitive("ok"))
                        when (submits++ % 3) {
                            0 -> check("amount" !in tree)
                            1 -> check(tree["amount"] == JsonNull)
                            2 -> check(tree["amount"] == JsonPrimitive("17"))
                        }
                        respond(recordJSON, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/amount" -> {
                        check(sharedBody(request.body) == "17")
                        respond("23", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/watch" -> {
                        streams++
                        val channel = ByteChannel(autoFlush = true)
                        channels += channel
                        val data = if (mode == "ok") recordJSON else "{\"amount\":false,\"values\":[23]}"
                        scope.launch {
                            try {
                                for (byte in "id: shared\r\ndata: $data\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                awaitCancellation()
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected shared runtime route")
                }
            }
        })
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) }; install(SSE) }
        try {
            val alpha = AlphaClient(http, "https://api.example.test/", codec)
            val beta = BetaClient(http, "https://api.example.test/", codec)
            for (patch in listOf<Patch<Long>>(Patch.Absent, Patch.Null, shared)) check(alpha.default.submit(AlphaInput("ok", patch)) == record)
            for (patch in listOf<Patch<Long>>(Patch.Absent, Patch.Null, shared)) check(beta.default.submit(BetaInput("ok", patch)) == BetaRecord(23L, listOf(23L)))
            check(alpha.default.amount(17L) == 23L && beta.default.amount(17L) == 23L)
            val cold = alpha.default.watch()
            check(streams == 0)
            repeat(2) { check((cold.take(1).toList().single() as AlphaEvent.Message).data == record) }
            check((beta.default.watch().take(1).toList().single() as BetaEvent.Message).data == BetaRecord(23L, listOf(23L)))
            mode = "bad"
            try { alpha.default.watch().take(1).toList(); error("Invalid bigint SSE accepted") }
            catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) }
            check(channels.all { it.isClosedForRead || it.isClosedForWrite })
            check(alpha.default.amount(17L) == 23L)
            check(submits == 6 && streams == 4)
        } finally { http.close(); scope.cancel() }
        println("SharedRuntime Json defaults=$defaults nulls=$nulls Patch-identity-two-specs-bigint-fields-root-list-HTTP-fragmentedSSE-close-reuse-cold checks passed")
    }
}

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
import kotlinx.serialization.MissingFieldException
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import org.example.pointerfollowup.PointerFollowupApiClient as FollowupClient
import org.example.pointerfollowup.api.models.SubmitInputDto as FollowupInput
import org.example.pointerfollowup.models.LeavesDto as FollowupLeaves
import org.example.pointerfollowup.models.Patch as FollowupPatch
import org.example.pointerfollowup.models.SharedDto as FollowupShared
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.http.limits.ApiClient as PointerLimitClient
import org.example.http.limits.models.PointerEncodeLimit
import org.example.pointerhelpers.ApiClient as PointerHelperClient
import org.example.pointerhelpers.api.models.HelperWatchEventDto
import org.example.pointerhelpers.api.models.ProbeInputDto
import org.example.pointerhelpers.models.Patch as HelperPatch
import org.example.pointerhelpers.models.ProbeDto
import org.example.pointers.SchemaPointerApiClient
import org.example.pointers.api.models.PatchInputDto
import org.example.pointers.api.models.WatchEventDto
import org.example.pointers.models.Patch
import org.example.pointers.models.RecordDto
import org.example.pointers.models.SourceDto

private var pointerAssertions = 0
private fun pointerCheck(value: Boolean, label: String) {
    pointerAssertions++
    check(value) { label }
}
private fun pointerReject(block: () -> Unit) {
    pointerAssertions++
    try { block(); error("Missing borrowed terminal validation") }
    catch (cause: IllegalArgumentException) { check(cause.message?.startsWith("Field value") == true) { "$cause" } }
}
private suspend fun pointerWireReject(message: String, block: suspend () -> Unit) {
    pointerAssertions++
    try { block(); error("Missing typed borrowed terminal failure") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message) == true
        }) { "Wrong borrowed terminal cause: $cause" }
    }
}
private fun pointerBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> pointerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected terminal body $body")
}
private fun pointerOrdinary() = RecordDto("ok", 2, true, 2L, 2f, 2.0, nonNull = "ok")

suspend fun schemaInternalScalarPointerContracts() {
    val ordinary = pointerOrdinary()
    val shared = RecordDto.serializer().descriptor
    pointerCheck(shared === RecordDto.generatedSerializer().descriptor, "retained ordinary count delegate identity")
    pointerCheck(!shared.getElementDescriptor(7).isNullable && shared.getElementDescriptor(6).isNullable, "explicit false vs inherited terminal null")
    val request = PatchInputDto.serializer().descriptor
    pointerCheck(!request.isElementOptional(1) && request.getElementDescriptor(1).isNullable, "inline required nullable no default")
    pointerCheck(RecordDto.Nested.serializer().descriptor === RecordDto.Nested.generatedSerializer().descriptor, "nested internal leaf is NOT catalog root")
    pointerReject { ordinary.copy(value = "a") }
    pointerReject { RecordDto("a", 2, true, 2L, 2f, 2.0, nonNull = "ok") }
    pointerReject { ordinary.copy(integer = 0) }
    pointerReject { ordinary.copy(long = 10L) }
    pointerReject { ordinary.copy(float = Float.NaN) }
    pointerReject { ordinary.copy(double = 10.0) }
    pointerReject { ordinary.copy(nonNull = "a") }
    pointerReject { ordinary.copy(forceNull = "a") }
    pointerReject { ordinary.copy(list = listOf(0)) }
    pointerReject { ordinary.copy(list = listOf(2, 2)) }
    pointerReject { ordinary.copy(list = emptyList()) }
    pointerReject { ordinary.copy(map = mapOf("key" to "a")) }
    pointerReject { ordinary.copy(one = "a") }
    pointerReject { ordinary.copy(any = 0L) }
    pointerReject { ordinary.copy(composed = "a") }
    pointerReject { PatchInputDto(PatchInputDto.Payload("ok"), "ok", Patch.Value("a")) }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val strict = Json(codec) { ignoreUnknownKeys = false }
        val value = ordinary.copy(list = listOf(1, 2), map = mapOf("key" to "ok"), one = "ok", any = 2L,
            composed = "ok", nested = RecordDto.Nested("ok"))
        val text = codec.encodeToString(RecordDto.serializer(), value)
        pointerCheck(text == codec.encodeToString(RecordDto.generatedSerializer(), value), "unchanged actual delegate bytes")
        pointerCheck(codec.decodeFromString(RecordDto.serializer(), text) == value, "four Json full terminal graph roundtrip")
        pointerCheck(codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(text)) == value, "element serde")
        pointerReject { codec.decodeFromString(RecordDto.serializer(), text.replace("\"value\":\"ok\"", "\"value\":\"a\"")) }
        pointerReject { codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement(text.replace("\"integer\":2", "\"integer\":0"))) }
        val source = SourceDto(aBC = "ok", n1 = 2, one = listOf(null, "ok"), any = listOf(null, 2L))
        pointerCheck(codec.decodeFromString(SourceDto.serializer(), codec.encodeToString(SourceDto.serializer(), source)) == source, "escaped key/~01, nullable collection leaf independent of container")
        val nullable = ordinary.copy(nullable = null, forceNull = null, nested = null)
        val encoded = codec.encodeToJsonElement(RecordDto.serializer(), nullable) as JsonObject
        pointerCheck(encoded.containsKey("nullable") == (defaults && nulls), "shared inherited-null default follows caller Json")
        val nestedNullText = codec.encodeToString(RecordDto.serializer(), ordinary)
        pointerWireReject("Object violates minProperties.") {
            codec.decodeFromString(RecordDto.serializer(), nestedNullText.dropLast(1) + ",\"nested\":{}}")
        }
        pointerWireReject("Object violates minProperties.") { codec.decodeFromString(RecordDto.serializer(), "{}") }
        val nestedRaw = """{"value":"ok","unknown":null}"""
        pointerCheck(codec.decodeFromString(RecordDto.Nested.serializer(), nestedRaw) == RecordDto.Nested("ok"), "raw incoming count includes unknown null before loss")
        pointerWireReject("Object violates maxProperties.") { codec.decodeFromString(RecordDto.Nested.serializer(), """{"value":"ok","x":null,"y":null}""") }
        val payload = PatchInputDto.Payload("ok")
        for (state in listOf<Patch<String>>(Patch.Absent, Patch.Null, Patch.Value("ok"))) {
            val patch = PatchInputDto(payload, "ok", state)
            val element = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            pointerCheck(when (state) { Patch.Absent -> "note" !in element; Patch.Null -> element["note"] == JsonNull; is Patch.Value -> element["note"] == JsonPrimitive("ok") }, "Patch Absent/Null/Value never dropped")
            pointerCheck(codec.decodeFromJsonElement(PatchInputDto.serializer(), element) == patch, "Patch terminal serde")
            val nestedPatch = patch.copy(payload = payload.copy(note = state))
            pointerCheck(codec.decodeFromString(PatchInputDto.serializer(), codec.encodeToString(PatchInputDto.serializer(), nestedPatch)) == nestedPatch, "nested terminal Patch uses same checks")
        }
        val requiredNull = PatchInputDto(payload, null, nonNull = "ok")
        val requiredObj = codec.encodeToJsonElement(PatchInputDto.serializer(), requiredNull) as JsonObject
        pointerCheck(requiredObj.containsKey("requiredNull") == nulls, "required nullable caller Json, explicit false optional is NOT Patch")
        val missing = """{"payload":{"value":"ok"},"unknown":null}"""
        if (nulls) {
            try { codec.decodeFromString(PatchInputDto.serializer(), missing); error("Missing required nullable accepted") }
            catch (cause: MissingFieldException) { pointerCheck(cause.missingFields == listOf("requiredNull"), "original required nullable diagnostic") }
        } else pointerCheck(codec.decodeFromString(PatchInputDto.serializer(), missing).requiredNull == null, "existing implicit nullable inference")
        pointerWire(codec, strict, value)
        pointerFullRuleHelpers(codec, strict)
        pointerPresenceNullFollowup(codec, defaults, nulls)
        println("Schema internal scalar Pointer contracts passed: defaults=$defaults/nulls=$nulls; typed ctor-copy-serde/Patch/nullable/counts/HTTP/live SSE")
    }
    println("Schema internal scalar Pointer assertions=$pointerAssertions; repeated-terminal120LongDouble/240mixed compilation-only")
}

private suspend fun pointerWire(codec: Json, strict: Json, ordinary: RecordDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var mode = 0
    var calls = 0
    var expectedPatch = ""
    val body = codec.encodeToString(RecordDto.serializer(), ordinary)
    val engine = StreamingTestEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/records" -> {
                    pointerCheck(request.url.parameters["limit"] == "2", "typed parameter wire")
                    pointerCheck(pointerBody(request.body) == body, "actual typed body exact caller Json")
                    respond(if (mode == 1) body.replace("\"value\":\"ok\"", "\"value\":\"a\"") else body,
                        HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> { pointerCheck(pointerBody(request.body) == expectedPatch, "typed inline Patch body"); respond("", HttpStatusCode.NoContent) }
                "/raw" -> {
                    pointerCheck(pointerBody(request.body) == "ok", "terminal String body remains RAW, not JSON quoted")
                    respond(if (mode == 2) "\"ok\"" else "ok", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val currentMode = mode
                    producers.launch {
                        try {
                            val data = when (currentMode) {
                                1 -> body.replace("\"value\":\"ok\"", "\"value\":\"a\"")
                                2 -> "{}"
                                3 -> body.dropLast(1) + ",\"unknown\":null}"
                                else -> body
                            }
                            val frame = "event: record\r\nid: pointer-id\r\ndata: $data\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (currentMode != 0) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected borrowed terminal dispatch")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = SchemaPointerApiClient(http, "http://example.test/", strict).pointers
        val before = calls
        var callback = false
        pointerWireReject("Field value is below minimum.") { api.send(0, ordinary) { callback = true } }
        pointerCheck(!callback && calls == before, "invalid typed parameter stops callback AND dispatch")
        pointerWireReject("Field value is shorter than minLength.") { api.raw("a") { callback = true } }
        pointerCheck(!callback && calls == before, "invalid raw String preflight stops callback/dispatch")
        pointerCheck(api.send(2, ordinary) == ordinary, "typed response retains borrowed constraints")
        pointerCheck(api.raw("ok") == "ok", "raw String response not JSON decoded")
        mode = 2
        pointerWireReject("Field value is longer than maxLength.") { api.raw("ok") }
        mode = 1
        pointerWireReject("Field value is shorter than minLength.") { api.send(2, ordinary) }
        mode = 0
        val patch = PatchInputDto(PatchInputDto.Payload("ok", Patch.Null), "ok", Patch.Null)
        expectedPatch = codec.encodeToString(PatchInputDto.serializer(), patch)
        api.patchPatch(patch)
        val limitApi = PointerLimitClient(http, "http://example.test/").default
        val beforeCount = calls
        callback = false
        pointerWireReject("Object violates maxProperties.") { limitApi.pointerCount(PointerEncodeLimit("ok", "ok")) { callback = true } }
        pointerCheck(callback && calls == beforeCount, "count encoding callback runs but NO invalid engine dispatch")
        val beforeCold = calls
        val cold = api.watch()
        pointerCheck(calls == beforeCold, "cold SSE does not dispatch")
        val event = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        pointerCheck(event.data == ordinary && event.id == "pointer-id", "actual fragmented SSE typed payload")
        pointerCheck(withTimeout(10_000) { cold.toList().size } == 1 && calls == beforeCold + 2, "cold recollection opens new session")
        for ((bad, message) in listOf(1 to "Field value is shorter than minLength.", 2 to "Object violates minProperties.", 3 to "unknown")) {
            mode = bad
            pointerWireReject(message) { withTimeout(10_000) { api.watch().toList() } }
            pointerCheck(channels.last().isClosedForRead, "known bad primitive/count/distinct strict Json closes STILL OPEN fragmented input")
        }
        mode = 0
        pointerCheck(withTimeout(10_000) { api.watch().toList().size } == 1, "SSE client reuse after failure")
        pointerCheck(api.send(2, ordinary) == ordinary, "caller HTTP client remains usable")
    } finally { http.close(); engine.close(); producers.cancel() }
}

private suspend fun pointerFullRuleHelpers(codec: Json, strict: Json) {
    val probe = ProbeDto("ok", 2L, 2.0, floatTie = 0.4f, doubleTie = 0.4,
        floatRemainder = 0.2f, doubleRemainder = 0.2, list = listOf(2L, 4L),
        map = mapOf("key" to 2.0), alias = mapOf("key" to 2L))
    for (bad in listOf(-2L, 10L, 0L, 8L, 3L)) pointerReject { probe.copy(n = bad) }
    for (bad in listOf(-2.0, 10.0, 0.0, 8.0, 3.0, Double.NaN)) pointerReject { probe.copy(d = bad) }
    for (bad in listOf("a", "abcd", "12")) pointerReject { probe.copy(s = bad) }
    pointerReject { ProbeDto("12", 2L, 2.0) }
    pointerReject { probe.copy(floatTie = 0.3f) }
    pointerReject { probe.copy(doubleTie = 0.3) }
    pointerReject { probe.copy(floatRemainder = 0.3f) }
    pointerReject { probe.copy(doubleRemainder = 0.3) }
    pointerReject { probe.copy(list = listOf(2L, 2L)) }
    pointerReject { probe.copy(list = emptyList()) }
    pointerReject { probe.copy(list = listOf(0L)) }
    pointerReject { probe.copy(map = emptyMap()) }
    pointerReject { probe.copy(map = mapOf("key" to 3.0)) }
    pointerReject { probe.copy(alias = mapOf("key" to 3L)) }
    val text = codec.encodeToString(ProbeDto.serializer(), probe)
    pointerCheck(text == codec.encodeToString(ProbeDto.generatedSerializer(), probe), "typed helper leaves retained automatic count delegate bytes")
    pointerCheck(codec.decodeFromString(ProbeDto.serializer(), text) == probe, "ALL primitive rules and alias/list/map helper serde")
    for ((from, to) in listOf("\"n\":2" to "\"n\":3", "\"floatTie\":0.4" to "\"floatTie\":0.3", "\"doubleRemainder\":0.2" to "\"doubleRemainder\":0.3")) {
        pointerReject { codec.decodeFromString(ProbeDto.serializer(), text.replace(from, to)) }
        pointerReject { codec.decodeFromJsonElement(ProbeDto.serializer(), codec.parseToJsonElement(text.replace(from, to))) }
    }
    val value = ProbeInputDto(HelperPatch.Value(2L), HelperPatch.Value(listOf(2L, 4L)), HelperPatch.Value(mapOf("key" to 2.0)))
    for (p in listOf(ProbeInputDto(), ProbeInputDto(HelperPatch.Null, HelperPatch.Null, HelperPatch.Null), value)) {
        val element = codec.encodeToJsonElement(ProbeInputDto.serializer(), p) as JsonObject
        pointerCheck(codec.decodeFromJsonElement(ProbeInputDto.serializer(), element) == p, "full typed property helper preserves Patch/list/map Null/Absent/Value")
    }
    pointerReject { value.copy(n = HelperPatch.Value(0L)) }
    pointerReject { value.copy(list = HelperPatch.Value(listOf(2L, 2L))) }
    pointerReject { value.copy(map = HelperPatch.Value(mapOf("key" to 3.0))) }
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var bad = false
    var calls = 0
    val engine = StreamingTestEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/probe" -> {
                    pointerCheck(pointerBody(request.body) == codec.encodeToString(ProbeInputDto.serializer(), value), "full-rule helper HTTP typed Patch body exact Json")
                    respond("", HttpStatusCode.NoContent)
                }
                "/helper-events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val invalid = bad
                    producers.launch {
                        try {
                            val payload = if (invalid) text.replace("\"n\":2", "\"n\":3") else text
                            for (byte in "event: probe\r\ndata: $payload\r\n\r\n".encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (invalid) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected full-rule helper request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = PointerHelperClient(http, "http://example.test/", strict).default
        api.probe(value)
        val before = calls
        val cold = api.helperWatch()
        pointerCheck(calls == before, "full-rule helper stream is cold")
        pointerCheck((withTimeout(10_000) { cold.toList().single() } as HelperWatchEventDto.Probe).data == probe, "full-rule helper live SSE serde")
        bad = true
        pointerWireReject("multipleOf") { withTimeout(10_000) { api.helperWatch().toList() } }
        pointerCheck(channels.last().isClosedForRead, "full-rule numeric helper failure closes still-open fragmented input")
        bad = false
        pointerCheck(withTimeout(10_000) { cold.toList().size } == 1, "full-rule helper cold reuse after failure")
        api.probe(value)
    } finally { http.close(); engine.close(); producers.cancel() }
}

private var pointerFollowupAssertions = 0
private fun followupCheck(value: Boolean, label: String) {
    pointerFollowupAssertions++
    check(value) { label }
}
private suspend fun followupReject(block: suspend () -> Unit) {
    pointerFollowupAssertions++
    try { block(); error("Missing follow-up borrowed validation") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException || it is SerializationException }) { "$cause" }
    }
}

private suspend fun pointerPresenceNullFollowup(codec: Json, defaults: Boolean, nulls: Boolean) {
    val list: List<String?> = listOf(null, "ok")
    val map: Map<String, String?> = linkedMapOf("nil" to null, "valid" to "ok")
    val leaves = FollowupLeaves(list, map, listOf(null, 2L), list, map, listOf("ok"), mapOf("valid" to "ok"))
    val descriptor = FollowupLeaves.serializer().descriptor
    followupCheck(!descriptor.getElementDescriptor(0).isNullable && descriptor.getElementDescriptor(0).getElementDescriptor(0).isNullable, "nonnull list, truly borrowed nullable item")
    followupCheck(!descriptor.getElementDescriptor(1).isNullable && descriptor.getElementDescriptor(1).getElementDescriptor(1).isNullable, "nonnull map, truly borrowed nullable value")
    followupCheck(descriptor.getElementDescriptor(3).isNullable && descriptor.getElementDescriptor(4).isNullable, "nullable containers independent of nullable leaves")
    followupCheck(!descriptor.getElementDescriptor(5).getElementDescriptor(0).isNullable && !descriptor.getElementDescriptor(6).getElementDescriptor(1).isNullable, "explicit false leaf override")
    val encoded = codec.encodeToJsonElement(FollowupLeaves.serializer(), leaves) as JsonObject
    followupCheck(encoded["list"] == JsonArray(listOf(JsonNull, JsonPrimitive("ok"))) && (encoded["map"] as JsonObject)["nil"] == JsonNull, "null borrowed list/map entries survive every caller Json")
    followupCheck(codec.decodeFromString(FollowupLeaves.serializer(), codec.encodeToString(FollowupLeaves.serializer(), leaves)) == leaves, "real borrowed nullable leaves string serde")
    followupCheck(codec.decodeFromJsonElement(FollowupLeaves.serializer(), encoded) == leaves, "real borrowed nullable leaves element serde")
    val nullContainers = leaves.copy(nullableList = null, nullableMap = null)
    followupCheck(nullContainers.list.first() == null && nullContainers.map["nil"] == null, "copy accepts null leaf vs null containers")
    followupCheck(codec.decodeFromString(FollowupLeaves.serializer(), codec.encodeToString(FollowupLeaves.serializer(), nullContainers)) == nullContainers, "null containers string serde")
    followupCheck(codec.decodeFromJsonElement(FollowupLeaves.serializer(), codec.encodeToJsonElement(FollowupLeaves.serializer(), nullContainers)) == nullContainers, "null containers element serde")
    followupCheck(leaves.copy(list = listOf(null), map = mapOf("nil" to null), numbers = listOf(null)).list == listOf(null), "valid null leaves skip primitive checks but meet counts")
    followupReject { FollowupLeaves(listOf("a"), map, listOf(2L), null, null, listOf("ok"), mapOf("valid" to "ok")) }
    followupReject { FollowupLeaves(list, mapOf("bad" to "12"), listOf(2L), null, null, listOf("ok"), mapOf("valid" to "ok")) }
    for (bad in listOf("a", "abcd", "12")) {
        followupReject { leaves.copy(list = listOf(null, bad)) }
        followupReject { leaves.copy(map = mapOf("nil" to null, "bad" to bad)) }
        for (field in listOf("list", "map")) {
            val badElement = if (field == "list") JsonArray(listOf(JsonNull, JsonPrimitive(bad))) else JsonObject(mapOf("nil" to JsonNull, "bad" to JsonPrimitive(bad)))
            val invalid = JsonObject(encoded + (field to badElement))
            followupReject { codec.decodeFromString(FollowupLeaves.serializer(), invalid.toString()) }
            followupReject { codec.decodeFromJsonElement(FollowupLeaves.serializer(), invalid) }
        }
    }
    for (bad in listOf(-2L, 0L, 3L, 8L, 10L)) {
        followupReject { leaves.copy(numbers = listOf(null, bad)) }
        val invalid = JsonObject(encoded + ("numbers" to JsonArray(listOf(JsonNull, JsonPrimitive(bad)))))
        followupReject { codec.decodeFromString(FollowupLeaves.serializer(), invalid.toString()) }
        followupReject { codec.decodeFromJsonElement(FollowupLeaves.serializer(), invalid) }
    }
    followupReject { leaves.copy(list = emptyList()) }
    followupReject { leaves.copy(map = emptyMap()) }
    followupReject { leaves.copy(list = listOf(null, null)) }
    followupReject { leaves.copy(nullableList = emptyList()) }
    followupReject { leaves.copy(nullableMap = emptyMap()) }
    for ((field, bad) in listOf("nonNullList" to JsonArray(listOf(JsonNull)), "nonNullMap" to JsonObject(mapOf("nil" to JsonNull)), "list" to JsonNull, "map" to JsonNull)) {
        val invalid = JsonObject(encoded + (field to bad))
        followupReject { codec.decodeFromString(FollowupLeaves.serializer(), invalid.toString()) }
        followupReject { codec.decodeFromJsonElement(FollowupLeaves.serializer(), invalid) }
    }
    val inputDescriptor = FollowupInput.serializer().descriptor
    followupCheck(inputDescriptor.isElementOptional(0) && inputDescriptor.isElementOptional(1), "inherited/member true Patch optional")
    followupCheck(!inputDescriptor.isElementOptional(7) && inputDescriptor.getElementDescriptor(7).isNullable && !inputDescriptor.getElementDescriptor(8).isNullable, "required null/explicit false controls")
    val shared = FollowupShared(inherited = null, memberTrue = null)
    followupCheck(codec.decodeFromString(FollowupShared.serializer(), codec.encodeToString(FollowupShared.serializer(), shared)) == shared, "shared wrappers retain ordinary nullable defaults, not Patch")
    val base = FollowupInput(requiredNull = "ok", requiredFalse = "ok")
    val missingNull = JsonObject((codec.encodeToJsonElement(FollowupInput.serializer(), base) as JsonObject) - "requiredNull")
    if (nulls) followupReject { codec.decodeFromJsonElement(FollowupInput.serializer(), missingNull) }
    else followupCheck(codec.decodeFromJsonElement(FollowupInput.serializer(), missingNull).requiredNull == null, "existing required nullable inference with explicitNulls false")
    val missingFalse = JsonObject((codec.encodeToJsonElement(FollowupInput.serializer(), base) as JsonObject) - "requiredFalse")
    followupReject { codec.decodeFromJsonElement(FollowupInput.serializer(), missingFalse) }
    var requests = 0
    var expected = ""
    val engine = StreamingTestEngine(MockEngineConfig().apply {
        addHandler { request ->
            requests++
            followupCheck(request.url.encodedPath == "/submit" && pointerBody(request.body) == expected, "actual inherited/member true wrapper+nullable leaf Patch HTTP body")
            respond("", HttpStatusCode.NoContent)
        }
    })
    val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
    try {
        val api = FollowupClient(http, "http://example.test/").default
        for (state in listOf<FollowupPatch<String>>(FollowupPatch.Absent, FollowupPatch.Null, FollowupPatch.Value("ok"))) {
            val collectionState = when (state) {
                FollowupPatch.Absent -> FollowupPatch.Absent
                FollowupPatch.Null -> FollowupPatch.Null
                is FollowupPatch.Value -> FollowupPatch.Value(list)
            }
            val mapState = when (state) {
                FollowupPatch.Absent -> FollowupPatch.Absent
                FollowupPatch.Null -> FollowupPatch.Null
                is FollowupPatch.Value -> FollowupPatch.Value(map)
            }
            val input = base.copy(inherited = state, memberTrue = state, annotationTrue = state, outerTrue = state,
                nested = FollowupInput.Nested(inherited = state, memberTrue = state), patchList = collectionState, patchMap = mapState)
            val objectWire = codec.encodeToJsonElement(FollowupInput.serializer(), input) as JsonObject
            for (key in listOf("inherited", "memberTrue", "annotationTrue", "outerTrue")) {
                followupCheck(when (state) { FollowupPatch.Absent -> key !in objectWire; FollowupPatch.Null -> objectWire[key] == JsonNull; is FollowupPatch.Value -> objectWire[key] == JsonPrimitive("ok") }, "wrapper Absent/Null/bare Value exact wire even defaults=true/nulls=false")
            }
            val nestedWire = objectWire["nested"] as JsonObject
            followupCheck(when (state) { FollowupPatch.Absent -> "inherited" !in nestedWire && "memberTrue" !in nestedWire; FollowupPatch.Null -> nestedWire["inherited"] == JsonNull && nestedWire["memberTrue"] == JsonNull; is FollowupPatch.Value -> nestedWire["inherited"] == JsonPrimitive("ok") && nestedWire["memberTrue"] == JsonPrimitive("ok") }, "nested borrowed wrapper uses same effective presence")
            for (key in listOf("memberFalse", "annotationFalse", "outerFalse", "named")) followupCheck((key in objectWire) == (defaults && nulls), "false/named controls stay ordinary optional null")
            followupCheck(when (state) { FollowupPatch.Absent -> "patchList" !in objectWire && "patchMap" !in objectWire; FollowupPatch.Null -> objectWire["patchList"] == JsonNull && objectWire["patchMap"] == JsonNull; is FollowupPatch.Value -> objectWire["patchList"] == encoded["list"] && objectWire["patchMap"] == encoded["map"] }, "Patch.Null container distinct from Value containing null leaves")
            followupCheck(codec.decodeFromJsonElement(FollowupInput.serializer(), objectWire) == input, "wrapper/container Patch element serde")
            expected = codec.encodeToString(FollowupInput.serializer(), input)
            followupCheck(codec.decodeFromString(FollowupInput.serializer(), expected) == input, "wrapper/container Patch string serde")
            api.submit(input)
        }
        followupCheck(requests == 3, "all three wire states reached actual engine")
        val before = requests
        var callback = false
        followupReject { api.submit(base.copy(inherited = FollowupPatch.Value("a"))) { callback = true } }
        followupReject { api.submit(base.copy(memberTrue = FollowupPatch.Value("12"))) { callback = true } }
        followupReject { base.copy(patchList = FollowupPatch.Value(listOf(null, "a"))) }
        followupReject { base.copy(patchMap = FollowupPatch.Value(mapOf("nil" to null, "bad" to "12"))) }
        followupCheck(requests == before && !callback, "invalid typed constructor/copy never dispatches/calls callback")
        val invalid = JsonObject((codec.encodeToJsonElement(FollowupInput.serializer(), base) as JsonObject) + ("memberTrue" to JsonPrimitive("a")))
        followupReject { codec.decodeFromString(FollowupInput.serializer(), invalid.toString()) }
        followupReject { codec.decodeFromJsonElement(FollowupInput.serializer(), invalid) }
        val requiredNull = base.copy(requiredNull = null)
        val nullWire = codec.encodeToJsonElement(FollowupInput.serializer(), requiredNull) as JsonObject
        followupCheck(("requiredNull" in nullWire) == nulls, "required nullable preserves caller Json behavior")
        followupCheck(codec.decodeFromJsonElement(FollowupInput.serializer(), nullWire) == requiredNull, "required nullable roundtrip")
    } finally { http.close(); engine.close() }
    println("Schema internal scalar Pointer follow-up passed: defaults=$defaults/nulls=$nulls; borrowed nullable List/Map leaves, wrapper Patch and actual HTTP; assertions=$pointerFollowupAssertions")
}

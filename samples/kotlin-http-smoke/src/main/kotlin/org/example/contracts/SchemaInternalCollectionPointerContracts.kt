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
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.collectionpointers.CollectionPointerApiClient
import org.example.collectionpointers.api.models.PatchCollectionsInputDto
import org.example.collectionpointers.api.models.WatchListEventDto
import org.example.collectionpointers.api.models.WatchMapEventDto
import org.example.collectionpointers.models.Patch
import org.example.collectionpointers.models.RecordDto
import org.example.collectionpointerprobes.ApiClient as ProbeClient
import org.example.collectionpointerprobes.models.CountCodecProbeDto
import org.example.collectionpointerprobes.models.CountOnlyProbeDto
import org.example.collectionpointerprobes.models.FullRuleProbeDto

private var collectionPointerAssertions = 0
private fun collectionPointerCheck(value: Boolean, label: String) {
    collectionPointerAssertions++
    check(value) { label }
}
private fun collectionPointerReject(message: String, block: () -> Unit) {
    collectionPointerAssertions++
    try { block(); error("Missing collection barrier: $message") }
    catch (cause: IllegalArgumentException) { check(cause.message?.contains(message) == true) { "$cause" } }
}
private suspend fun collectionPointerWireReject(message: String, block: suspend () -> Unit) {
    collectionPointerAssertions++
    try { block(); error("Missing collection wire barrier: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message) == true
        }) { "Wrong collection failure: $cause" }
    }
}
private fun collectionPointerBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> collectionPointerBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected collection body $body")
}

suspend fun schemaInternalCollectionPointerContracts() {
    val descriptor = RecordDto.serializer().descriptor
    collectionPointerCheck(descriptor === RecordDto.generatedSerializer().descriptor, "retained DTO count delegate")
    collectionPointerCheck(!descriptor.getElementDescriptor(0).isNullable && descriptor.getElementDescriptor(0).getElementDescriptor(0).isNullable,
        "explicit false root is required non-null List with independently nullable String leaf")
    collectionPointerCheck(descriptor.getElementDescriptor(2).isNullable && descriptor.getElementDescriptor(4).isNullable,
        "inherited list root and explicit true map root")
    val request = PatchCollectionsInputDto.serializer().descriptor
    collectionPointerCheck(!request.isElementOptional(2) && request.getElementDescriptor(2).isNullable, "required nullable no default")
    collectionPointerCheck(request.isElementOptional(0) && !request.getElementDescriptor(0).isNullable && request.getElementDescriptor(3).isNullable,
        "Patch delegates collection descriptor, false override ordinary optional nullable descriptor")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true; isLenient = true }
        val strict = Json(codec) { ignoreUnknownKeys = false; isLenient = false }
        val value = RecordDto(listOf("ok", null), mapOf("key" to 2, "null" to null), inherited = null,
            falseOuter = listOf(null), trueOuter = null, wrapped = listOf("ok"), folded = listOf(null),
            mixed = listOf(null, mapOf("key" to listOf("ok"))))
        val text = codec.encodeToString(RecordDto.serializer(), value)
        val element = codec.encodeToJsonElement(RecordDto.serializer(), value) as JsonObject
        collectionPointerCheck(text == codec.encodeToString(RecordDto.generatedSerializer(), value), "count codec unchanged actual bytes")
        collectionPointerCheck(codec.decodeFromString(RecordDto.serializer(), text) == value, "four Json string roundtrip")
        collectionPointerCheck(codec.decodeFromJsonElement(RecordDto.serializer(), element) == value, "four Json element roundtrip")
        collectionPointerCheck((element["values"] as JsonArray)[1] == JsonNull && (element["entries"] as JsonObject)["null"] == JsonNull,
            "leaf/entry JSON null survives explicitNulls=false")
        collectionPointerCheck(element.containsKey("inherited") == (defaults && nulls), "shared root null obeys ordinary caller defaults")
        collectionPointerCheck((element["mixed"] as JsonArray)[0] == JsonNull, "nested nullable container distinct from scalar null")
        collectionPointerReject("minItems") { RecordDto(emptyList(), mapOf("key" to 2)) }
        collectionPointerReject("minLength") { RecordDto(listOf("a"), mapOf("key" to 2)) }
        for ((bad, reason) in listOf(emptyList<String?>() to "minItems", listOf("a") to "minLength", listOf("abcd") to "maxLength",
            listOf("12") to "pattern", listOf("ok", "ok") to "uniqueItems", listOf(null, null) to "uniqueItems",
            listOf("ok", "ab", "cd", "ef") to "maxItems")) {
            collectionPointerReject(reason) { value.copy(values = bad) }
            val invalid = JsonObject(element + ("values" to JsonArray(bad.map { if (it == null) JsonNull else JsonPrimitive(it) })))
            collectionPointerReject(reason) { codec.decodeFromString(RecordDto.serializer(), invalid.toString()) }
            collectionPointerReject(reason) { codec.decodeFromJsonElement(RecordDto.serializer(), invalid) }
        }
        collectionPointerReject("minProperties") { value.copy(entries = emptyMap()) }
        collectionPointerReject("maximum") { value.copy(entries = mapOf("key" to 10)) }
        collectionPointerReject("minItems") { value.copy(mixed = listOf(mapOf("key" to emptyList()))) }
        collectionPointerReject("minLength") { value.copy(mixed = listOf(mapOf("key" to listOf("a")))) }
        collectionPointerCheck(value.copy(inherited = null, wrapped = null, folded = null, mixed = null).values == value.values,
            "null root skips contents, unlike non-null collection containing invalid leaf")
        collectionPointerReject("minItems") { value.copy(wrapped = emptyList()) }
        collectionPointerWireReject("Object violates minProperties.") { codec.decodeFromString(RecordDto.serializer(), "{}") }
        collectionPointerWireReject("Object violates maxProperties.") {
            codec.decodeFromString(RecordDto.serializer(), """{"values":["ok"],"entries":{"key":2},"x":null,"y":null,"z":null,"a":null,"b":null,"c":null,"d":null}""")
        }
        val states = listOf<Patch<List<String?>>>(Patch.Absent, Patch.Null, Patch.Value(listOf(null, "ok")))
        for (state in states) {
            val patch = PatchCollectionsInputDto(values = state, requiredNull = null, nested = PatchCollectionsInputDto.Nested(state))
            val obj = codec.encodeToJsonElement(PatchCollectionsInputDto.serializer(), patch) as JsonObject
            collectionPointerCheck(when (state) { Patch.Absent -> "values" !in obj; Patch.Null -> obj["values"] == JsonNull; is Patch.Value -> obj["values"] == JsonArray(listOf(JsonNull, JsonPrimitive("ok"))) },
                "wrapper inherited Patch Absent/Null/Value not dropped")
            collectionPointerCheck(obj.containsKey("requiredNull") == nulls, "required nullable follows caller explicitNulls")
            collectionPointerCheck(codec.decodeFromJsonElement(PatchCollectionsInputDto.serializer(), obj) == patch, "nested wrapper Patch element roundtrip")
            collectionPointerCheck(codec.decodeFromString(PatchCollectionsInputDto.serializer(), obj.toString()) == patch, "nested wrapper Patch string roundtrip")
        }
        collectionPointerReject("minLength") { PatchCollectionsInputDto(Patch.Value(listOf("a")), requiredNull = null) }
        collectionPointerReject("minItems") { PatchCollectionsInputDto(requiredNull = null).copy(values = Patch.Value(emptyList())) }
        collectionPointerReject("minimum") { PatchCollectionsInputDto(requiredNull = null, entries = Patch.Value(mapOf("key" to 0))) }
        if (nulls) {
            try { codec.decodeFromString(PatchCollectionsInputDto.serializer(), "{}"); error("required nullable missing") }
            catch (cause: MissingFieldException) { collectionPointerCheck(cause.missingFields == listOf("requiredNull"), "required missing unchanged") }
        } else collectionPointerCheck(codec.decodeFromString(PatchCollectionsInputDto.serializer(), "{}").requiredNull == null, "existing implicit required nullable inference")
        collectionPointerFullRules(codec)
        collectionPointerWire(codec, strict, value)
        println("Schema internal collection Pointer contracts passed: defaults=$defaults/nulls=$nulls; ctor-copy-string-element-serde/null/Patch/typed HTTP/live SSE")
    }
    println("Schema internal collection Pointer assertions=$collectionPointerAssertions; independent240shallow/full-count-only andsingle16mixed compilation-only; combined wide-deep compiler resources remain OPEN")
}

private fun collectionPointerFullRules(codec: Json) {
    val probe = FullRuleProbeDto(stringList = listOf("ok", null), stringMap = mapOf("key" to "ok"), longList = listOf(2L, 4L),
        doubleMap = mapOf("key" to 2.0), floatList = listOf(2f))
    val encoded = codec.encodeToJsonElement(FullRuleProbeDto.serializer(), probe) as JsonObject
    collectionPointerCheck(codec.decodeFromString(FullRuleProbeDto.serializer(), encoded.toString()) == probe, "all leaf numeric categories retain full rules")
    for ((bad, reason) in listOf(-4L to "minimum", 12L to "maximum", 0L to "exclusiveMinimum", 8L to "exclusiveMaximum", 3L to "multipleOf")) {
        collectionPointerReject(reason) { probe.copy(longList = listOf(bad)) }
        val invalid = JsonObject(encoded + ("longList" to JsonArray(listOf(JsonPrimitive(bad)))))
        collectionPointerReject(reason) { codec.decodeFromString(FullRuleProbeDto.serializer(), invalid.toString()) }
        collectionPointerReject(reason) { codec.decodeFromJsonElement(FullRuleProbeDto.serializer(), invalid) }
    }
    collectionPointerReject("finite") { probe.copy(doubleMap = mapOf("key" to Double.NaN)) }
    collectionPointerReject("multipleOf") { probe.copy(doubleMap = mapOf("key" to 3.0)) }
    collectionPointerReject("exclusiveMaximum") { probe.copy(floatList = listOf(8f)) }
    collectionPointerReject("uniqueItems") { probe.copy(longList = listOf(2L, 2L)) }
    val counts = CountOnlyProbeDto(list = listOf(null), map = mapOf("key" to null))
    collectionPointerCheck(codec.decodeFromJsonElement(CountOnlyProbeDto.serializer(), codec.encodeToJsonElement(CountOnlyProbeDto.serializer(), counts)) == counts,
        "count-only small runtime independent of primitive hints")
    collectionPointerCheck(counts.copy(list = listOf(null, null, "ok")).list?.size == 3, "uniqueItems:false is count-only validated noop, duplicates permitted")
    collectionPointerReject("minItems") { counts.copy(list = emptyList()) }
    collectionPointerReject("maxProperties") { counts.copy(map = mapOf("a" to null, "b" to null, "c" to null, "d" to null)) }
}

private suspend fun collectionPointerWire(codec: Json, strict: Json, value: RecordDto) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var mode = 0
    var calls = 0
    var expectedPatch = ""
    val engine = StreamingTestEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/parameters" -> {
                    collectionPointerCheck(request.url.parameters.getAll("values") == listOf("ok") && request.url.parameters["key"] == "2" && "null" !in request.url.parameters.names(),
                        "actual List repeat/Map entries nullable omission wire")
                    respond("", HttpStatusCode.NoContent)
                }
                "/list" -> {
                    collectionPointerCheck(collectionPointerBody(request.body) == "[\"ok\",null]", "typed root List exact body")
                    val body = when (mode) { 1 -> "[\"a\"]"; 2 -> "[]"; 3 -> "[ok]"; else -> "[\"ok\",null]" }
                    respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/map" -> {
                    collectionPointerCheck(collectionPointerBody(request.body) == "{\"key\":2,\"null\":null}", "typed root Map exact body")
                    respond(if (mode == 1) "{\"key\":0}" else "{\"key\":2,\"null\":null}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/record" -> respond(codec.encodeToString(RecordDto.serializer(), value), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/patch" -> { collectionPointerCheck(collectionPointerBody(request.body) == expectedPatch, "actual inline collection Patch wire"); respond("", HttpStatusCode.NoContent) }
                "/events/list", "/events/map" -> {
                    val map = request.url.encodedPath.endsWith("map")
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val current = mode
                    producers.launch {
                        try {
                            val data = if (map) { if (current == 1) "{\"key\":0}" else "{\"key\":2,\"null\":null}" }
                                else when (current) { 1 -> "[\"a\"]"; 2 -> "[]"; 3 -> "[ok]"; 4 -> "null"; else -> "[\"ok\",null]" }
                            val frame = "event: values\r\nid: collection-id\r\ndata: $data\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (current in 1..3) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected collection dispatch ${request.url}")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = CollectionPointerApiClient(http, "http://example.test/", strict).collectionPointers
        val before = calls
        var callback = false
        collectionPointerWireReject("minLength") { api.parameters(listOf("a"), mapOf("key" to 2)) { callback = true } }
        collectionPointerWireReject("minItems") { api.sendList(emptyList()) { callback = true } }
        collectionPointerWireReject("minimum") { api.sendMap(mapOf("key" to 0)) { callback = true } }
        collectionPointerCheck(!callback && calls == before, "typed leaf/count preflight stops callback AND engine")
        api.parameters(listOf("ok", null), mapOf("key" to 2, "null" to null))
        collectionPointerCheck(api.sendList(listOf("ok", null)) == listOf("ok", null), "root List response leaf null")
        collectionPointerCheck(api.sendMap(mapOf("key" to 2, "null" to null)) == mapOf("key" to 2, "null" to null), "root Map response entry null")
        collectionPointerCheck(api.sendRecord(value) == value, "shared response not request Patch")
        val patch = PatchCollectionsInputDto(values = Patch.Null, requiredNull = listOf(null), entries = Patch.Value(mapOf("key" to null)))
        expectedPatch = codec.encodeToString(PatchCollectionsInputDto.serializer(), patch)
        api.patchCollections(patch)
        // DTO wire counts have a later codec-owned boundary than typed preflight.
        callback = false
        val beforeCount = calls
        collectionPointerWireReject("Object violates maxProperties.") {
            ProbeClient(http, "http://example.test/").default.codec(CountCodecProbeDto(listOf("ok"), listOf("ok"))) { callback = true }
        }
        collectionPointerCheck(callback && calls == beforeCount, "DTO count codec callback runs but engine suppressed independently")
        mode = 1
        collectionPointerWireReject("minLength") { api.sendList(listOf("ok", null)) }
        collectionPointerWireReject("minimum") { api.sendMap(mapOf("key" to 2, "null" to null)) }
        mode = 2
        collectionPointerWireReject("minItems") { api.sendList(listOf("ok", null)) }
        mode = 0
        val beforeCold = calls
        val cold = api.watchList()
        collectionPointerCheck(calls == beforeCold, "collection SSE genuinely cold")
        val event = withTimeout(10_000) { cold.toList().single() as WatchListEventDto.Values }
        collectionPointerCheck(event.data == listOf("ok", null) && event.id == "collection-id", "fragmented actual typed List SSE")
        collectionPointerCheck(withTimeout(10_000) { cold.toList().size } == 1 && calls == beforeCold + 2, "cold recollection opens another stream")
        val mapEvent = withTimeout(10_000) { api.watchMap().toList().single() as WatchMapEventDto.Values }
        collectionPointerCheck(mapEvent.data == mapOf("key" to 2, "null" to null), "fragmented actual typed Map SSE")
        for ((bad, reason) in listOf(1 to "minLength", 2 to "minItems")) {
            mode = bad
            collectionPointerWireReject(reason) { withTimeout(10_000) { cold.toList() } }
            collectionPointerCheck(channels.last().isClosedForRead, "independent bad leaf/container closes STILL OPEN input")
        }
        mode = 1
        collectionPointerWireReject("minimum") { withTimeout(10_000) { api.watchMap().toList() } }
        collectionPointerCheck(channels.last().isClosedForRead, "bad Map leaf closes still-open input")
        mode = 3
        collectionPointerCheck(api.sendList(listOf("ok", null)) == listOf("ok"), "HTTP lenient Json control")
        collectionPointerWireReject("Unexpected JSON token") { withTimeout(10_000) { cold.toList() } }
        collectionPointerCheck(channels.last().isClosedForRead, "independent strict SSE Json, not HTTP codec")
        mode = 4
        collectionPointerCheck((withTimeout(10_000) { cold.toList().single() } as WatchListEventDto.Values).data == null, "nullable data root distinct from nullable list leaf")
        mode = 0
        collectionPointerCheck(withTimeout(10_000) { cold.toList().size } == 1, "stream reuse after all failures")
        collectionPointerCheck(api.sendList(listOf("ok", null)) == listOf("ok", null), "caller HTTP client still usable")
    } finally { http.close(); engine.close(); producers.cancel() }
}

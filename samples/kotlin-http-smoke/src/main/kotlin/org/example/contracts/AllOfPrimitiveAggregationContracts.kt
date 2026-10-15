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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.aggregation.ApiClient
import org.example.aggregation.api.models.PatchInputDto
import org.example.aggregation.api.models.WatchEventDto
import org.example.aggregation.models.Patch
import org.example.aggregation.models.RecordDto
import org.example.aggregation.models.ZNullableDto
import org.example.aggregation.models.ZNumbersDto
import org.example.http.limits.models.AggregateFloatRemainder
import org.example.http.limits.models.AggregateUpperTies

private var aggregateAssertions = 0
private fun aggregateVerify(value: Boolean, label: String) {
    aggregateAssertions++
    check(value) { label }
}
private fun aggregateReject(keyword: String, block: () -> Unit) {
    aggregateAssertions++
    val error = runCatching(block).exceptionOrNull()
    check(error != null && generateSequence(error) { it.cause }.any {
        it is IllegalArgumentException && it.message?.startsWith("Field value") == true && it.message?.contains(keyword) == true
    }) { "Missing generated safe primitive failure $keyword: $error" }
}
private suspend fun aggregateWireReject(keyword: String, block: suspend () -> Unit) {
    aggregateAssertions++
    var failure: Throwable? = null
    try { block() } catch (error: Throwable) { failure = error }
    check(failure != null && generateSequence(failure) { it.cause }.any {
        it is IllegalArgumentException && it.message?.startsWith("Field value") == true && it.message?.contains(keyword) == true
    }) { "Missing generated wire cause $keyword: $failure" }
}
private fun aggregateBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> aggregateBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected aggregation body")
}

suspend fun allOfPrimitiveAggregationContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls; ignoreUnknownKeys = true }
        val permissive = Json(codec) { allowSpecialFloatingPointValues = true }
        val record = RecordDto("ab")
        aggregateVerify(record.copy(value = "abc") == RecordDto("abc"), "ctor/copy conjunction")
        aggregateVerify(codec.encodeToString(RecordDto.serializer(), record) == "{\"value\":\"ab\"}", "primitive wire unchanged")
        aggregateVerify(codec.decodeFromString<RecordDto>("{\"value\":\"ab\"}") == record, "automatic serde")
        for ((value, keyword) in listOf("a" to "minLength", "abcd" to "maxLength", "12" to "pattern")) {
            aggregateReject(keyword) { RecordDto(value) }
            aggregateReject(keyword) { record.copy(value = value) }
            aggregateReject(keyword) { codec.decodeFromString<RecordDto>("{\"value\":\"$value\"}") }
            aggregateReject(keyword) { codec.decodeFromJsonElement(RecordDto.serializer(), codec.parseToJsonElement("{\"value\":\"$value\"}")) }
        }
        aggregateVerify(codec.decodeFromString<ZNullableDto>(codec.encodeToString(ZNullableDto.serializer(), ZNullableDto(null))) == ZNullableDto(null), "shared nullable defaults unchanged")
        aggregateVerify(codec.decodeFromString<ZNullableDto>("{\"value\":\"ab\"}") == ZNullableDto("ab"), "nullable non-null conjunction")
        aggregateReject("minLength") { ZNullableDto("a") }
        aggregateReject("maxLength") { codec.decodeFromString<ZNullableDto>("{\"value\":\"abcd\"}") }
        val payloads = listOf(
            PatchInputDto.Payload(requiredNull = null),
            PatchInputDto.Payload(Patch.Null, null),
            PatchInputDto.Payload(Patch.Value("ab"), "abc"),
        )
        for (payload in payloads) {
            val element = codec.encodeToJsonElement(PatchInputDto.Payload.serializer(), payload) as JsonObject
            aggregateVerify(element.containsKey("requiredNull") == (nulls || payload.requiredNull != null), "required nullable no default follows explicitNulls")
            when (payload.value) {
                Patch.Absent -> aggregateVerify(!element.containsKey("value"), "Patch Absent omitted under four Json")
                Patch.Null -> aggregateVerify(element["value"] == JsonNull, "Patch Null always present")
                is Patch.Value -> aggregateVerify(element["value"] == JsonPrimitive("ab"), "Patch Value bare wire")
            }
            aggregateVerify(codec.decodeFromJsonElement(PatchInputDto.Payload.serializer(), element) == payload, "Patch automatic serde")
        }
        val descriptor = PatchInputDto.Payload.serializer().descriptor
        aggregateVerify(descriptor.isElementOptional(0) && !descriptor.isElementOptional(1) && descriptor.getElementDescriptor(1).isNullable, "winning presence/signature")
        aggregateReject("minLength") { payloads[0].copy(value = Patch.Value("a")) }
        aggregateReject("maxLength") { PatchInputDto.Payload(requiredNull = "abcd") }
        aggregateReject("minLength") { codec.decodeFromString<PatchInputDto.Payload>("{\"requiredNull\":null,\"value\":\"a\"}") }
        val numbers = ZNumbersDto("shadow", Long.MIN_VALUE, 2, 0.5, 0.4f, 0.4)
        for (long in listOf(Long.MIN_VALUE, Long.MAX_VALUE)) {
            val wide = numbers.copy(long = long)
            aggregateVerify(codec.decodeFromString<ZNumbersDto>(codec.encodeToString(ZNumbersDto.serializer(), wide)) == wide, "Long extrema safe kotlin shadow constructor/serde")
        }
        for (step in listOf(0, 8)) aggregateReject(if (step == 0) "exclusiveMinimum" else "exclusiveMaximum") { numbers.copy(steps = step) }
        aggregateReject("multipleOf") { numbers.copy(steps = 3) }
        aggregateReject("exclusiveMinimum") { numbers.copy(floatTie = 0.3f) }
        aggregateReject("exclusiveMinimum") { numbers.copy(doubleTie = 0.3) }
        aggregateReject("exclusiveMinimum") { codec.decodeFromString<ZNumbersDto>(codec.encodeToString(ZNumbersDto.serializer(), numbers).replace("\"floatTie\":0.4", "\"floatTie\":0.3")) }
        aggregateReject("exclusiveMinimum") { codec.decodeFromString<ZNumbersDto>(codec.encodeToString(ZNumbersDto.serializer(), numbers).replace("\"doubleTie\":0.4", "\"doubleTie\":0.3")) }
        for (fraction in listOf(0.0, 1.0)) aggregateReject(if (fraction == 0.0) "exclusiveMinimum" else "exclusiveMaximum") { numbers.copy(fraction = fraction) }
        aggregateReject("multipleOf") { numbers.copy(fraction = 0.3) }
        aggregateReject("multipleOf") { numbers.copy(tenths = 0.3) }
        aggregateVerify(numbers.copy(tenths = 0.2).tenths == 0.2, "strict IEEE remainder positive control")
        aggregateVerify(codec.decodeFromString<AggregateFloatRemainder>("{\"value\":0.2}") == AggregateFloatRemainder(0.2f), "Float strict remainder positive")
        aggregateReject("multipleOf") { AggregateFloatRemainder(0.3f) }
        aggregateReject("multipleOf") { codec.decodeFromString<AggregateFloatRemainder>("{\"value\":0.3}") }
        for (value in listOf(Double.NaN, Double.POSITIVE_INFINITY, Double.NEGATIVE_INFINITY)) aggregateReject("finite") { numbers.copy(fraction = value) }
        aggregateReject("finite") { numbers.copy(floatTie = Float.NaN) }
        aggregateReject("finite") { permissive.decodeFromString<ZNumbersDto>(codec.encodeToString(ZNumbersDto.serializer(), numbers).replace("\"fraction\":0.5", "\"fraction\":NaN")) }
        aggregateUpperTies(codec, defaults, nulls)
        aggregateHttp(codec, record, payloads.map { PatchInputDto(it) })
    }
    println("AllOf primitive aggregation contracts passed: $aggregateAssertions assertions, fresh four-Json constructor/copy/serde/Patch/Float+Double ties/HTTP/live SSE")
}

private fun aggregateUpperTies(codec: Json, defaults: Boolean, nulls: Boolean) {
    val before = aggregateAssertions
    val upper = AggregateUpperTies(0.2f, 0.2)
    aggregateVerify(upper.floatUpper == 0.2f && upper.doubleUpper == 0.2, "upper tie legal constructor control")
    val copied = upper.copy(floatUpper = 0.2f, doubleUpper = 0.2)
    aggregateVerify(copied == upper, "upper tie legal copy control")
    val encoded = codec.encodeToString(AggregateUpperTies.serializer(), copied)
    aggregateVerify(encoded == "{\"floatUpper\":0.2,\"doubleUpper\":0.2}", "upper tie exact legal wire")
    aggregateVerify(codec.decodeFromString<AggregateUpperTies>(encoded) == upper, "upper tie automatic string roundtrip")
    aggregateVerify(codec.decodeFromJsonElement(AggregateUpperTies.serializer(), codec.encodeToJsonElement(AggregateUpperTies.serializer(), copied)) == upper, "upper tie automatic element roundtrip")
    // Test each mapped endpoint independently; the other field stays valid.
    aggregateReject("exclusiveMaximum") { AggregateUpperTies(0.3f, 0.2) }
    aggregateReject("exclusiveMaximum") { upper.copy(floatUpper = 0.3f) }
    val badFloat = "{\"floatUpper\":0.3,\"doubleUpper\":0.2}"
    aggregateReject("exclusiveMaximum") { codec.decodeFromString<AggregateUpperTies>(badFloat) }
    aggregateReject("exclusiveMaximum") { codec.decodeFromJsonElement(AggregateUpperTies.serializer(), codec.parseToJsonElement(badFloat)) }
    aggregateReject("exclusiveMaximum") { AggregateUpperTies(0.2f, 0.3) }
    aggregateReject("exclusiveMaximum") { upper.copy(doubleUpper = 0.3) }
    val badDouble = "{\"floatUpper\":0.2,\"doubleUpper\":0.3}"
    aggregateReject("exclusiveMaximum") { codec.decodeFromString<AggregateUpperTies>(badDouble) }
    aggregateReject("exclusiveMaximum") { codec.decodeFromJsonElement(AggregateUpperTies.serializer(), codec.parseToJsonElement(badDouble)) }
    println("Upper Float/Double mapped ties passed: defaults=$defaults/nulls=$nulls; ${aggregateAssertions - before} fresh generated assertions, independent ctor/copy/automatic decode rejection and legal wire controls")
}

private suspend fun aggregateHttp(codec: Json, record: RecordDto, patches: List<PatchInputDto>) {
    var calls = 0
    var badResponse = false
    var badSSE = false
    var expectedPatch = ""
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val engine = StreamingTestEngine(MockEngineConfig().apply {
        addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/records" -> {
                    aggregateVerify(aggregateBody(request.body) == codec.encodeToString(RecordDto.serializer(), record), "HTTP exact legal wire")
                    respond(if (badResponse) "{\"value\":\"a\"}" else "{\"value\":\"ab\"}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/patch" -> {
                    aggregateVerify(aggregateBody(request.body) == expectedPatch, "HTTP Patch exact wire")
                    respond("", HttpStatusCode.NoContent)
                }
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true)
                    channels += channel
                    val bad = badSSE
                    producers.launch {
                        try {
                            val frame = "event: record\r\nid: aggregate-id\r\ndata: {\"value\":\"${if (bad) "a" else "ab"}\"}\r\n\r\n"
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (bad) awaitCancellation()
                        } finally { channel.close() }
                    }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                else -> error("Unexpected aggregation request")
            }
        }
    })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = ApiClient(http, "http://example.test/", codec).bounds
        aggregateVerify(api.send(record) == record, "HTTP nominal response conjunction")
        for (patch in patches) { expectedPatch = codec.encodeToString(PatchInputDto.serializer(), patch); api.patchPatch(patch) }
        val beforeBad = calls
        var callback = false
        aggregateWireReject("minLength") { api.send(RecordDto("a")) { callback = true } }
        aggregateWireReject("minLength") { api.patchPatch(PatchInputDto(patches[0].payload.copy(value = Patch.Value("a")))) { callback = true } }
        aggregateVerify(!callback && calls == beforeBad, "typed invalid constructor before callback/dispatch, count codec policy unchanged")
        badResponse = true
        aggregateWireReject("minLength") { api.send(record) }
        badResponse = false
        val before = calls
        val cold = api.watch()
        aggregateVerify(calls == before, "cold Flow no request")
        val first = withTimeout(10_000) { cold.toList().single() as WatchEventDto.Record }
        aggregateVerify(first.data == record && first.id == "aggregate-id", "fresh SSE typed payload/transport")
        aggregateVerify(withTimeout(10_000) { cold.toList().size } == 1 && calls == before + 2, "cold independent recollection")
        badSSE = true
        aggregateWireReject("minLength") { withTimeout(10_000) { api.watch().toList() } }
        aggregateVerify(channels.last().isClosedForRead, "bad known payload closes STILL OPEN fragmented input")
        badSSE = false
        aggregateVerify(withTimeout(10_000) { api.watch().toList().size } == 1, "SSE reusable after rejection")
        aggregateVerify(api.send(record) == record, "HTTP caller client reusable")
    } finally { http.close(); engine.close(); producers.cancel() }
}

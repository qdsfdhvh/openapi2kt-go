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
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CompletableDeferred
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
import org.example.nestednativeenum.NestedNativeEnum
import org.example.nestednativeenum.api.models.PatchInputDto
import org.example.nestednativeenum.api.models.WatchEventDto
import org.example.nestednativeenum.models.HolderDto
import org.example.nestednativeenum.models.Patch
import org.example.nestednativeenum.models.RequiredNullableDto
import org.example.nestednativeenum.models.RequiredTreesDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NestedNativeEnumEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nestedNativeEnumReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nestedNativeEnumBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nestedNativeEnumBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected native tree request body $body")
}

// These generic arguments are deliberate ABI checks, not JSON-only assertions.
private fun <T> nestedNativeEnumTrees(
    scalar: T, rows: List<List<T?>?>?, listMap: List<Map<String, T?>?>?,
    mapList: Map<String, List<T?>?>?, mapMap: Map<String, Map<String, T?>?>?,
    mixed: List<Map<String, List<Map<String, T>>>>?
) {
    check(rows == listOf(listOf(scalar, null), null, emptyList()))
    check(listMap == listOf(mapOf("\$ref" to scalar, "a/~" to null), null, emptyMap()))
    check(mapList == mapOf("a" to listOf(scalar, null), "nil" to null, "empty" to emptyList()))
    check(mapMap == mapOf("a" to mapOf("\$ref" to scalar, "a/~" to null), "nil" to null, "empty" to emptyMap()))
    check(mixed == listOf(mapOf("\$ref" to listOf(mapOf("a/~" to scalar)))))
}

private fun nestedNativeEnumFields(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "intRows" to value.intRows, "longRows" to value.longRows, "floatRows" to value.floatRows, "doubleRows" to value.doubleRows, "boolRows" to value.boolRows,
    "intListMap" to value.intListMap, "longListMap" to value.longListMap, "floatListMap" to value.floatListMap, "doubleListMap" to value.doubleListMap, "boolListMap" to value.boolListMap,
    "intMapList" to value.intMapList, "longMapList" to value.longMapList, "floatMapList" to value.floatMapList, "doubleMapList" to value.doubleMapList, "boolMapList" to value.boolMapList,
    "intMapMap" to value.intMapMap, "longMapMap" to value.longMapMap, "floatMapMap" to value.floatMapMap, "doubleMapMap" to value.doubleMapMap, "boolMapMap" to value.boolMapMap,
    "intMixed" to value.intMixed, "longMixed" to value.longMixed, "floatMixed" to value.floatMixed, "doubleMixed" to value.doubleMixed, "boolMixed" to value.boolMixed,
    "longEnds" to value.longEnds, "rounded" to value.rounded, "empty" to value.empty, "emptyMap" to value.emptyMap, "nil" to value.nil
)

private fun nestedNativeEnumFresh(left: Any?, right: Any?) {
    check(left == right)
    when (left) {
        is List<*> -> { check(left !== right); right as List<*>; left.indices.forEach { nestedNativeEnumFresh(left[it], right[it]) } }
        is Map<*, *> -> { check(left !== right); right as Map<*, *>; left.keys.forEach { nestedNativeEnumFresh(left[it], right[it]) } }
    }
}

suspend fun nestedNativeEnumCollectionDefaultsContracts(checked: Boolean) {
    val required = listOf(listOf(2))
    val value = HolderDto(required = required, kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow")
    nestedNativeEnumTrees<Int>(2, value.intRows, value.intListMap, value.intMapList, value.intMapMap, value.intMixed)
    nestedNativeEnumTrees<Long>(2L, value.longRows, value.longListMap, value.longMapList, value.longMapMap, value.longMixed)
    nestedNativeEnumTrees<Float>(1.0f, value.floatRows, value.floatListMap, value.floatMapList, value.floatMapMap, value.floatMixed)
    nestedNativeEnumTrees<Double>(1.0, value.doubleRows, value.doubleListMap, value.doubleMapList, value.doubleMapMap, value.doubleMixed)
    nestedNativeEnumTrees<Boolean>(true, value.boolRows, value.boolListMap, value.boolMapList, value.boolMapMap, value.boolMixed)
    check(value.longEnds == mapOf("type" to listOf(Long.MIN_VALUE, Long.MAX_VALUE)))
    check(value.rounded == listOf(mapOf("type" to 1.0f)))
    check(value.empty == emptyList<List<Boolean>>() && value.emptyMap == emptyMap<String, Map<String, Int>>() && value.nil == null)
    check(HolderDto.Nested().value == listOf(mapOf("type" to true)))
    nestedNativeEnumFresh(nestedNativeEnumFields(value), nestedNativeEnumFields(HolderDto(required = required)))
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    check(!PatchInputDto.serializer().descriptor.isElementOptional(5))
    for (index in 0 until 5) check(!RequiredTreesDto.serializer().descriptor.isElementOptional(index))
    check(HolderDto.serializer().descriptor.serialName == "org.example.nestednativeenum.models.HolderDto")
    val requiredTrees = RequiredTreesDto(rows = required, listMap = listOf(mapOf("a" to 2L)), mapList = mapOf("a" to listOf(1.0f)), mapMap = mapOf("a" to mapOf("b" to 1.0)), mixed = listOf(mapOf("a" to listOf(mapOf("b" to true)))))
    // The raw Float token 1.000000001 maps to native 1.0f, not a token wrapper.
    check(HolderDto(required = required, rounded = listOf(mapOf("mapped" to 1.0f))).rounded?.single()?.get("mapped") == 1.0f)
    val constructors: List<() -> HolderDto> = listOf(
        { HolderDto(required = required, intRows = listOf(listOf(9))) },
        { HolderDto(required = required, longListMap = listOf(mapOf("a" to 9L))) },
        { HolderDto(required = required, floatMapList = mapOf("a" to listOf(9.0f))) },
        { HolderDto(required = required, doubleMapMap = mapOf("a" to mapOf("b" to 9.0))) },
        { HolderDto(required = required, boolMixed = listOf(mapOf("a" to listOf(mapOf("b" to false))))) }
    )
    for (construct in constructors) {
        if (checked) nestedNativeEnumReject("enum") { construct() } else check(construct().required == required)
    }
    if (checked) {
        nestedNativeEnumReject("minimum") { HolderDto(required = listOf(listOf(1))) }
        nestedNativeEnumReject("minimum") { HolderDto(required = required, longRows = listOf(listOf(1L))) }
        nestedNativeEnumReject("maximum") { HolderDto(required = required, floatListMap = listOf(mapOf("a" to 1.5f))) }
        nestedNativeEnumReject("maximum") { HolderDto(required = required, doubleMapList = mapOf("a" to listOf(1.5))) }
        nestedNativeEnumReject("minItems") { HolderDto(required = emptyList()) }
        nestedNativeEnumReject("minItems") { HolderDto(required = listOf(emptyList())) }
        nestedNativeEnumReject("maxItems") { HolderDto(required = required, intRows = List(4) { null }) }
        nestedNativeEnumReject("maxItems") { HolderDto(required = required, longRows = listOf(List(3) { null })) }
        nestedNativeEnumReject("maxItems") { HolderDto(required = required, floatListMap = List(4) { null }) }
        nestedNativeEnumReject("maxProperties") { HolderDto(required = required, doubleListMap = listOf(mapOf("a" to null, "b" to null, "c" to null))) }
        nestedNativeEnumReject("maxProperties") { HolderDto(required = required, boolMapList = mapOf("a" to null, "b" to null, "c" to null, "d" to null)) }
        nestedNativeEnumReject("maxItems") { HolderDto(required = required, intMapList = mapOf("a" to List(3) { null })) }
        nestedNativeEnumReject("maxProperties") { HolderDto(required = required, longMapMap = mapOf("a" to null, "b" to null, "c" to null, "d" to null)) }
        nestedNativeEnumReject("maxProperties") { HolderDto(required = required, floatMapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null))) }
        nestedNativeEnumReject("maxProperties") { HolderDto(required = required, boolMixed = listOf(mapOf("a" to listOf(mapOf("a" to true, "b" to true))))) }
        nestedNativeEnumReject("minItems") { RequiredTreesDto(rows = listOf(emptyList()), listMap = requiredTrees.listMap, mapList = requiredTrees.mapList, mapMap = requiredTrees.mapMap, mixed = requiredTrees.mixed) }
        nestedNativeEnumReject("minProperties") { RequiredTreesDto(rows = requiredTrees.rows, listMap = listOf(emptyMap()), mapList = requiredTrees.mapList, mapMap = requiredTrees.mapMap, mixed = requiredTrees.mixed) }
        nestedNativeEnumReject("minItems") { RequiredTreesDto(rows = requiredTrees.rows, listMap = requiredTrees.listMap, mapList = mapOf("a" to emptyList()), mapMap = requiredTrees.mapMap, mixed = requiredTrees.mixed) }
        nestedNativeEnumReject("minProperties") { RequiredTreesDto(rows = requiredTrees.rows, listMap = requiredTrees.listMap, mapList = requiredTrees.mapList, mapMap = mapOf("a" to emptyMap()), mixed = requiredTrees.mixed) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = codec.decodeFromString<HolderDto>("""{"required":[[2]]}""")
        nestedNativeEnumFresh(nestedNativeEnumFields(value), nestedNativeEnumFields(missing))
        nestedNativeEnumReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":[[2]],"intRows":null,"longListMap":null,"floatMapList":null,"doubleMapMap":null,"boolMixed":null}""").let { it.intRows == null && it.longListMap == null && it.floatMapList == null && it.doubleMapMap == null && it.boolMixed == null })
        val supplied = codec.decodeFromString<HolderDto>("""{"required":[[2]],"intRows":[null,[],[null,3]],"longListMap":[{"new":3}],"floatMapList":{"new":[0.5]},"doubleMapMap":{"new":{"new":0.5}},"boolMixed":[{"new":[{"new":true}]}]}""")
        check(supplied.intRows == listOf(null, emptyList(), listOf(null, 3)))
        check(supplied.longListMap == listOf(mapOf("new" to 3L)) && supplied.floatMapList == mapOf("new" to listOf(0.5f)))
        check(supplied.doubleMapMap == mapOf("new" to mapOf("new" to 0.5)) && supplied.boolMixed == listOf(mapOf("new" to listOf(mapOf("new" to true)))))
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        for (field in nestedNativeEnumFields(missing).keys - "nil") check(encoded.containsKey(field) == defaults) { "Unexpected default emission for $field" }
        if (defaults) {
            check(encoded["intRows"] == codec.parseToJsonElement("[[2,null],null,[]]"))
            check(encoded["longListMap"] == codec.parseToJsonElement("""[{"${'$'}ref":2,"a/~":null},null,{}]"""))
            check(encoded["floatMapList"] == codec.parseToJsonElement("""{"a":[1.0,null],"empty":[],"nil":null}"""))
            check(encoded["doubleMapMap"] == codec.parseToJsonElement("""{"a":{"${'$'}ref":1.0,"a/~":null},"empty":{},"nil":null}"""))
            check(encoded["boolMixed"] == codec.parseToJsonElement("""[{"${'$'}ref":[{"a/~":true}]}]"""))
        }
        check(nestedNativeEnumFields(codec.decodeFromJsonElement(HolderDto.serializer(), encoded)) == nestedNativeEnumFields(missing))
        val nested = HolderDto.Nested()
        check(codec.decodeFromJsonElement(HolderDto.Nested.serializer(), codec.encodeToJsonElement(HolderDto.Nested.serializer(), nested)).value == nested.value)
        val requiredWire = codec.encodeToJsonElement(RequiredTreesDto.serializer(), requiredTrees) as JsonObject
        check(requiredWire == codec.parseToJsonElement("""{"rows":[[2]],"listMap":[{"a":2}],"mapList":{"a":[1.0]},"mapMap":{"a":{"b":1.0}},"mixed":[{"a":[{"b":true}]}]}"""))
        for (field in requiredWire.keys) nestedNativeEnumReject { codec.decodeFromJsonElement(RequiredTreesDto.serializer(), JsonObject(requiredWire - field)) }
        val requiredDecoded = codec.decodeFromJsonElement(RequiredTreesDto.serializer(), requiredWire)
        check(requiredDecoded.rows == requiredTrees.rows && requiredDecoded.listMap == requiredTrees.listMap && requiredDecoded.mapList == requiredTrees.mapList && requiredDecoded.mapMap == requiredTrees.mapMap && requiredDecoded.mixed == requiredTrees.mixed)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), RequiredNullableDto(value = null)) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire).value == null)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.required == null && absent.rows == Patch.Absent && absent.listMap == Patch.Absent && absent.mapList == Patch.Absent && absent.mapMap == Patch.Absent && absent.mixed == Patch.Absent)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it in listOf("rows", "listMap", "mapList", "mapMap", "mixed") })
        val update = PatchInputDto(required = required, rows = Patch.Value(required), listMap = Patch.Value(listOf(mapOf("a" to 2L))), mapList = Patch.Value(mapOf("a" to listOf(1.0f))), mapMap = Patch.Value(mapOf("a" to mapOf("b" to 1.0))), mixed = Patch.Value(listOf(mapOf("a" to listOf(mapOf("b" to true))))))
        val updateWire = codec.encodeToJsonElement(PatchInputDto.serializer(), update) as JsonObject
        check(updateWire == JsonObject(requiredWire + ("required" to codec.parseToJsonElement("[[2]]"))))
        val decodedUpdate = codec.decodeFromJsonElement(PatchInputDto.serializer(), updateWire)
        check(decodedUpdate.rows == update.rows && decodedUpdate.listMap == update.listMap && decodedUpdate.mapList == update.mapList && decodedUpdate.mapMap == update.mapMap && decodedUpdate.mixed == update.mixed)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"listMap":null,"mapList":null,"mapMap":null,"mixed":null}""")
        check(cleared.rows == Patch.Null && cleared.listMap == Patch.Null && cleared.mapList == Patch.Null && cleared.mapMap == Patch.Null && cleared.mixed == Patch.Null)
        val clearWire = codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject
        for (field in listOf("rows", "listMap", "mapList", "mapMap", "mixed")) check(clearWire[field] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        if (checked) {
            nestedNativeEnumReject("enum") { PatchInputDto(required = null, mixed = Patch.Value(listOf(mapOf("a" to listOf(mapOf("b" to false)))))) }
            nestedNativeEnumReject("minimum") { PatchInputDto(required = null, listMap = Patch.Value(listOf(mapOf("a" to 1L)))) }
            nestedNativeEnumReject("maximum") { PatchInputDto(required = null, mapList = Patch.Value(mapOf("a" to listOf(1.5f)))) }
            nestedNativeEnumReject("minItems") { PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList()))) }
            nestedNativeEnumReject("maxProperties") { PatchInputDto(required = null, mapMap = Patch.Value(mapOf("a" to mapOf("a" to 1.0, "b" to 1.0, "c" to 1.0)))) }
            nestedNativeEnumReject("minItems") { codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":[[]]}""") }
        } else {
            check(PatchInputDto(required = null, mixed = Patch.Value(listOf(mapOf("a" to listOf(mapOf("b" to false)))))).mixed is Patch.Value)
        }
        for ((bad, rule) in nestedNativeEnumInvalidPayloads(checked)) nestedNativeEnumReject(rule) { codec.decodeFromString<HolderDto>(bad) }
        for (bad in nestedNativeEnumUnknownPayloads()) {
            if (checked) nestedNativeEnumReject("enum") { codec.decodeFromString<HolderDto>(bad) }
            else check(codec.decodeFromString<HolderDto>(bad).required == required)
        }
        nestedNativeEnumWire(codec, missing, checked)
        println("Nested native enum defaults Json defaults=$defaults nulls=$nulls checked=$checked native5-LL-LM-ML-MM-LMLM-constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private fun nestedNativeEnumUnknownPayloads(): List<String> = listOf(
    """{"required":[[2]],"intRows":[[9]]}""",
    """{"required":[[2]],"longListMap":[{"a":9}]}""",
    """{"required":[[2]],"floatMapList":{"a":[9.0]}}""",
    """{"required":[[2]],"doubleMapMap":{"a":{"b":9.0}}}""",
    """{"required":[[2]],"boolMixed":[{"a":[{"b":false}]}]}"""
)

private fun nestedNativeEnumInvalidPayloads(checked: Boolean): List<Pair<String, String?>> {
    val invalid = mutableListOf<Pair<String, String?>>(
        """{"required":[[null]]}""" to null,
        """{"required":[[2]],"intRows":[{}]}""" to null,
        """{"required":[[2]],"longListMap":[{"bad":true}]}""" to null,
        """{"required":[[2]],"floatMapList":{"a":[false]}}""" to null,
        """{"required":[[2]],"doubleMapMap":{"a":{"b":[]}}}""" to null,
        """{"required":[[2]],"boolMixed":[{"a":[{"b":null}]}]}""" to null,
        "{" to null
    )
    if (checked) invalid += nestedNativeEnumUnknownPayloads().map { it to "enum" } + listOf(
        """{"required":[[1]]}""" to "minimum",
        """{"required":[[2]],"longRows":[[1]]}""" to "minimum",
        """{"required":[[2]],"floatListMap":[{"a":1.5}]}""" to "maximum",
        """{"required":[[2]],"doubleMapList":{"a":[1.5]}}""" to "maximum",
        """{"required":[]}""" to "minItems",
        """{"required":[[]]}""" to "minItems",
        """{"required":[[2]],"intRows":[null,null,null,null]}""" to "maxItems",
        """{"required":[[2]],"longRows":[[null,null,null]]}""" to "maxItems",
        """{"required":[[2]],"floatListMap":[null,null,null,null]}""" to "maxItems",
        """{"required":[[2]],"doubleListMap":[{"a":null,"b":null,"c":null}]}""" to "maxProperties",
        """{"required":[[2]],"boolMapList":{"a":null,"b":null,"c":null,"d":null}}""" to "maxProperties",
        """{"required":[[2]],"intMapList":{"a":[null,null,null]}}""" to "maxItems",
        """{"required":[[2]],"longMapMap":{"a":null,"b":null,"c":null,"d":null}}""" to "maxProperties",
        """{"required":[[2]],"floatMapMap":{"a":{"a":null,"b":null,"c":null}}}""" to "maxProperties",
        """{"required":[[2]],"boolMixed":[{"a":[{"a":true,"b":true}]}]}""" to "maxProperties"
    )
    return invalid
}

private suspend fun nestedNativeEnumWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    val good = """{"required":[[2]]}"""
    val native = listOf(mapOf("a" to listOf(mapOf("b" to 2))))
    var payload = good
    var nativePayload = """[{"a":[{"b":2}]}]"""
    var expectedNativeBody = nativePayload
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NestedNativeEnumEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nestedNativeEnumBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/inline" -> respond("{}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            "/native" -> {
                check(codec.parseToJsonElement(nestedNativeEnumBody(request.body)) == codec.parseToJsonElement(expectedNativeBody))
                respond(nativePayload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nestedNativeEnumBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                val written = CompletableDeferred<Unit>()
                channels += channel
                framesWritten += written
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    written.complete(Unit)
                    if (holdOpen) awaitCancellation()
                } catch (cause: Throwable) { written.completeExceptionally(cause); throw cause } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected nested native enum path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NestedNativeEnum(http, "http://example.test/", codec).defaults
        check(nestedNativeEnumFields(api.send(value) { callbacks++ }) == nestedNativeEnumFields(value))
        check(api.fetch { callbacks++ }.value == listOf(mapOf("a" to 2L)))
        check(api.applyNative(native) { callbacks++ } == native)
        for (patch in listOf(
            PatchInputDto(required = null),
            PatchInputDto(required = null, rows = Patch.Null, listMap = Patch.Null, mapList = Patch.Null, mapMap = Patch.Null, mixed = Patch.Null),
            PatchInputDto(required = listOf(listOf(2)), rows = Patch.Value(listOf(listOf(2))), listMap = Patch.Value(listOf(mapOf("a" to 2L))), mapList = Patch.Value(mapOf("a" to listOf(1.0f))), mapMap = Patch.Value(mapOf("a" to mapOf("b" to 1.0))), mixed = Patch.Value(listOf(mapOf("a" to listOf(mapOf("b" to true))))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch) { callbacks++ }
        }
        if (checked) {
            val beforeCalls = calls
            val beforeCallbacks = callbacks
            // Bare native trees bypass DTO construction: this exercises actual operation preflight.
            nestedNativeEnumReject("enum") { api.applyNative(listOf(mapOf("a" to listOf(mapOf("b" to 9))))) { callbacks++ } }
            nestedNativeEnumReject("minimum") { api.applyNative(listOf(mapOf("a" to listOf(mapOf("b" to 1))))) { callbacks++ } }
            nestedNativeEnumReject("minItems") { api.applyNative(emptyList()) { callbacks++ } }
            check(calls == beforeCalls && callbacks == beforeCallbacks)
            for ((bad, rule) in listOf("""[{"a":[{"b":9}]}]""" to "enum", """[{"a":[{"b":1}]}]""" to "minimum", "[]" to "minItems")) {
                nativePayload = bad
                val beforeResult = calls
                nestedNativeEnumReject(rule) { api.applyNative(native) { callbacks++ } }
                check(calls == beforeResult + 1 && callbacks == calls)
                nativePayload = expectedNativeBody
                check(api.applyNative(native) { callbacks++ } == native)
            }
        } else {
            val unknown = listOf(mapOf("a" to listOf(mapOf("b" to 9))))
            nativePayload = """[{"a":[{"b":9}]}]"""
            expectedNativeBody = nativePayload
            check(api.applyNative(unknown) { callbacks++ } == unknown)
            nativePayload = """[{"a":[{"b":2}]}]"""
            expectedNativeBody = nativePayload
        }
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watch { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        repeat(2) {
            val event = (withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data
            check(nestedNativeEnumFields(event) == nestedNativeEnumFields(value))
            check(calls == before + it + 1 && callbacks == beforeCallbacks + it + 1)
        }
        if (!checked) for (unknown in nestedNativeEnumUnknownPayloads()) {
            payload = unknown
            val result = api.send(value) { callbacks++ }
            val event = (withTimeout(10000) { api.watch { callbacks++ }.toList() }.single() as WatchEventDto.Holder).data
            val expected = codec.decodeFromString<HolderDto>(unknown)
            check(nestedNativeEnumFields(result) == nestedNativeEnumFields(expected) && nestedNativeEnumFields(event) == nestedNativeEnumFields(expected))
        }
        for ((bad, rule) in nestedNativeEnumInvalidPayloads(checked)) {
            payload = bad
            failing = true
            val beforeFailure = calls
            nestedNativeEnumReject(rule) { api.send(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            nestedNativeEnumReject(rule) { withTimeout(10000) { api.watch { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead) { "Failed full-frame still-open stream was not closed" }
            payload = good
            failing = false
            check(nestedNativeEnumFields(api.send(value) { callbacks++ }) == nestedNativeEnumFields(value))
            val recovered = (withTimeout(10000) { api.watch { callbacks++ }.toList() }.single() as WatchEventDto.Holder).data
            check(nestedNativeEnumFields(recovered) == nestedNativeEnumFields(value))
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

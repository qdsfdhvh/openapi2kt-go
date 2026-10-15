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
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import org.example.nestedstringenum.NestedStringEnum
import org.example.nestedstringenum.api.models.FetchResponseDto
import org.example.nestedstringenum.api.models.PatchInputDto
import org.example.nestedstringenum.api.models.WatchEventDto
import org.example.nestedstringenum.models.HolderDto
import org.example.nestedstringenum.models.Patch
import org.example.nestedstringenum.models.RequiredNullableDto
import org.example.nestedstringenum.models.RequiredTreesDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NestedStringEnumEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nestedStringEnumReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nestedStringEnumBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nestedStringEnumBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

private fun nestedStringEnumSameDefaults(left: HolderDto, right: HolderDto) {
    check(left.rows == right.rows && left.listMap == right.listMap && left.mapList == right.mapList && left.mapMap == right.mapMap)
    check(left.mixed == right.mixed && left.special == right.special && left.empty == right.empty && left.emptyMap == right.emptyMap && left.nil == right.nil)
}

suspend fun nestedStringEnumCollectionDefaultsContracts(checked: Boolean) {
    val required = listOf(listOf(HolderDto.RequiredInnerInner.TWO))
    val value = HolderDto(required = required, kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow")
    val rows: List<List<HolderDto.RowsInnerInner?>?>? = value.rows
    val listMap: List<Map<String, HolderDto.ListMapInnerValue?>?>? = value.listMap
    val mapList: Map<String, List<HolderDto.MapListValueInner?>?>? = value.mapList
    val mapMap: Map<String, Map<String, HolderDto.MapMapValueValue?>>? = value.mapMap
    val mixed: List<Map<String, List<Map<String, HolderDto.MixedInnerValueInnerValue>>>>? = value.mixed
    check(rows == listOf(listOf(HolderDto.RowsInnerInner.ONE, null, HolderDto.RowsInnerInner.TWO, HolderDto.RowsInnerInner.NULL), null, emptyList()))
    check(listMap == listOf(mapOf("z" to null, "a/~" to HolderDto.ListMapInnerValue.YES, "\$ref" to HolderDto.ListMapInnerValue.NO), null, emptyMap()))
    check(mapList == mapOf("z" to null, "b" to emptyList(), "a" to listOf(HolderDto.MapListValueInner.UP, null)))
    check(mapMap == mapOf("z" to emptyMap(), "a" to mapOf("z" to null, "a" to HolderDto.MapMapValueValue.ON, "b" to HolderDto.MapMapValueValue.OFF)))
    check(mixed == listOf(mapOf("a" to listOf(mapOf("b" to HolderDto.MixedInnerValueInnerValue.DEEP)))))
    check(value.special?.single()?.map { it.serialName } == listOf("quote\"\$😀", "\n", ""))
    check(value.empty == emptyList<List<HolderDto.EmptyInnerInner>>() && value.emptyMap == emptyMap<String, Map<String, HolderDto.EmptyMapValueValue>>() && value.nil == null)
    check(HolderDto.Nested().value == listOf(mapOf("one" to HolderDto.Nested.ValueInnerValue.NESTED)))
    check(HolderDto.RowsInnerInner.entries.map { it.serialName } == listOf("one", "two", "null"))
    check(HolderDto.ListMapInnerValue.entries.map { it.serialName } == listOf("yes", "no"))
    check(HolderDto.MapListValueInner.entries.map { it.serialName } == listOf("up", "down"))
    check(HolderDto.MapMapValueValue.entries.map { it.serialName } == listOf("on", "off"))
    check(HolderDto.RowsInnerInner.fromSerialName("one") === HolderDto.RowsInnerInner.ONE)
    check(HolderDto.ListMapInnerValue.Serializer.fromSerialName("yes") === HolderDto.ListMapInnerValue.YES)
    check(HolderDto.MapListValueInner.fromSerialName("up") === HolderDto.MapListValueInner.UP)
    check(HolderDto.MapMapValueValue.Serializer.fromSerialName("off") === HolderDto.MapMapValueValue.OFF)
    check(HolderDto.RowsInnerInner.fromSerialName("future😀") == HolderDto.RowsInnerInner.Unrecognized("future😀"))
    check(HolderDto.ListMapInnerValue.Serializer.fromSerialName("future") == HolderDto.ListMapInnerValue.Unrecognized("future"))
    val second = HolderDto(required = required)
    nestedStringEnumSameDefaults(value, second)
    check(value.rows !== second.rows && value.rows?.first() !== second.rows?.first())
    check(value.listMap !== second.listMap && value.listMap?.first() !== second.listMap?.first())
    check(value.mapList !== second.mapList && value.mapList?.get("a") !== second.mapList?.get("a"))
    check(value.mapMap !== second.mapMap && value.mapMap?.get("a") !== second.mapMap?.get("a"))
    check(value.empty !== second.empty && value.emptyMap !== second.emptyMap)
    check(value.rows?.last() !== second.rows?.last() && value.listMap?.last() !== second.listMap?.last())
    check(value.mapList?.get("b") !== second.mapList?.get("b") && value.mapMap?.get("z") !== second.mapMap?.get("z"))
    check(value.mixed !== second.mixed && value.mixed?.first() !== second.mixed?.first())
    check(value.mixed?.first()?.get("a") !== second.mixed?.first()?.get("a"))
    check(value.mixed?.first()?.get("a")?.first() !== second.mixed?.first()?.get("a")?.first())
    check(!HolderDto.serializer().descriptor.isElementOptional(0))
    check(!RequiredNullableDto.serializer().descriptor.isElementOptional(0))
    check(!PatchInputDto.serializer().descriptor.isElementOptional(4))
    for (index in 0 until 4) check(!RequiredTreesDto.serializer().descriptor.isElementOptional(index))
    check(HolderDto.serializer().descriptor.serialName == "org.example.nestedstringenum.models.HolderDto")
    check(!HolderDto.RowsInnerInner.serializer().descriptor.isNullable)
    check(HolderDto.RowsInnerInner.serializer().nullable.descriptor.isNullable)
    check(HolderDto.RowsInnerInner.serializer().descriptor.serialName == "org.example.nestedstringenum.models.HolderDto.RowsInnerInner")
    check(HolderDto.ListMapInnerValue.serializer().descriptor.serialName == "org.example.nestedstringenum.models.HolderDto.ListMapInnerValue")
    check(HolderDto.MapListValueInner.serializer().descriptor.serialName == "org.example.nestedstringenum.models.HolderDto.MapListValueInner")
    check(HolderDto.MapMapValueValue.serializer().descriptor.serialName == "org.example.nestedstringenum.models.HolderDto.MapMapValueValue")
    val future = HolderDto(
        required = listOf(listOf(HolderDto.RequiredInnerInner.Unrecognized("future"))),
        rows = listOf(listOf(HolderDto.RowsInnerInner.Unrecognized("future😀"), null)),
        listMap = listOf(mapOf("future" to HolderDto.ListMapInnerValue.Unrecognized("future"))),
        mapList = mapOf("future" to listOf(HolderDto.MapListValueInner.Unrecognized("future"))),
        mapMap = mapOf("future" to mapOf("future" to HolderDto.MapMapValueValue.Unrecognized("future"))),
        mixed = listOf(mapOf("future" to listOf(mapOf("future" to HolderDto.MixedInnerValueInnerValue.Unrecognized("future")))))
    )
    val requiredTrees = RequiredTreesDto(
        rows = listOf(listOf(RequiredTreesDto.RowsInnerInner.TWO)),
        listMap = listOf(mapOf("a" to RequiredTreesDto.ListMapInnerValue.TWO)),
        mapList = mapOf("a" to listOf(RequiredTreesDto.MapListValueInner.TWO)),
        mapMap = mapOf("a" to mapOf("b" to RequiredTreesDto.MapMapValueValue.TWO)))
    if (checked) {
        nestedStringEnumReject("minItems") { RequiredTreesDto(rows = listOf(emptyList()), listMap = requiredTrees.listMap, mapList = requiredTrees.mapList, mapMap = requiredTrees.mapMap) }
        nestedStringEnumReject("minProperties") { RequiredTreesDto(rows = requiredTrees.rows, listMap = listOf(emptyMap()), mapList = requiredTrees.mapList, mapMap = requiredTrees.mapMap) }
        nestedStringEnumReject("minItems") { RequiredTreesDto(rows = requiredTrees.rows, listMap = requiredTrees.listMap, mapList = mapOf("a" to emptyList()), mapMap = requiredTrees.mapMap) }
        nestedStringEnumReject("minProperties") { RequiredTreesDto(rows = requiredTrees.rows, listMap = requiredTrees.listMap, mapList = requiredTrees.mapList, mapMap = mapOf("a" to emptyMap())) }
        nestedStringEnumReject("minItems") { HolderDto(required = emptyList()) }
        nestedStringEnumReject("minItems") { HolderDto(required = listOf(emptyList())) }
        nestedStringEnumReject("maxItems") { HolderDto(required = required, rows = List(4) { null }) }
        nestedStringEnumReject("maxItems") { HolderDto(required = required, rows = listOf(List(5) { HolderDto.RowsInnerInner.ONE })) }
        nestedStringEnumReject("maxItems") { HolderDto(required = required, listMap = List(4) { null }) }
        nestedStringEnumReject("maxProperties") { HolderDto(required = required, listMap = listOf(mapOf("a" to null, "b" to null, "c" to null, "d" to null))) }
        nestedStringEnumReject("maxProperties") { HolderDto(required = required, mapList = mapOf("a" to null, "b" to null, "c" to null, "d" to null)) }
        nestedStringEnumReject("maxItems") { HolderDto(required = required, mapList = mapOf("a" to List(3) { null })) }
        nestedStringEnumReject("maxProperties") { HolderDto(required = required, mapMap = mapOf("a" to emptyMap(), "b" to emptyMap(), "c" to emptyMap())) }
        nestedStringEnumReject("maxProperties") { HolderDto(required = required, mapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null, "d" to null))) }
        nestedStringEnumReject("maxProperties") { HolderDto(required = required, mixed = listOf(mapOf("a" to listOf(mapOf("a" to HolderDto.MixedInnerValueInnerValue.DEEP, "b" to HolderDto.MixedInnerValueInnerValue.DEEP))))) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.encodeToString(HolderDto.RowsInnerInner.serializer(), HolderDto.RowsInnerInner.NULL) == "\"null\"")
        check(codec.decodeFromString(HolderDto.RowsInnerInner.serializer().nullable, "null") == null)
        nestedStringEnumReject { codec.decodeFromString(HolderDto.RowsInnerInner.serializer(), "null") }
        check(codec.decodeFromString(HolderDto.RowsInnerInner.serializer(), "\"future😀\"") == HolderDto.RowsInnerInner.Unrecognized("future😀"))
        check(codec.decodeFromString(HolderDto.ListMapInnerValue.serializer(), "\"yes\"") === HolderDto.ListMapInnerValue.YES)
        check(codec.decodeFromString(HolderDto.MapListValueInner.serializer(), "\"up\"") === HolderDto.MapListValueInner.UP)
        check(codec.decodeFromString(HolderDto.MapMapValueValue.serializer(), "\"off\"") === HolderDto.MapMapValueValue.OFF)
        val requiredWire = codec.encodeToJsonElement(RequiredTreesDto.serializer(), requiredTrees) as JsonObject
        check(requiredWire == codec.parseToJsonElement("""{"rows":[["two"]],"listMap":[{"a":"two"}],"mapList":{"a":["two"]},"mapMap":{"a":{"b":"two"}}}"""))
        val requiredDecoded = codec.decodeFromJsonElement(RequiredTreesDto.serializer(), requiredWire)
        check(requiredDecoded.rows == requiredTrees.rows && requiredDecoded.listMap == requiredTrees.listMap && requiredDecoded.mapList == requiredTrees.mapList && requiredDecoded.mapMap == requiredTrees.mapMap)
        for (field in listOf("rows", "listMap", "mapList", "mapMap")) {
            nestedStringEnumReject { codec.decodeFromJsonElement(RequiredTreesDto.serializer(), JsonObject(requiredWire - field)) }
        }
        if (checked) {
            for ((field, bad, rule) in listOf(
                Triple("rows", "[[]]", "minItems"), Triple("listMap", "[{}]", "minProperties"),
                Triple("mapList", "{\"a\":[]}", "minItems"), Triple("mapMap", "{\"a\":{}}", "minProperties"))) {
                nestedStringEnumReject(rule) { codec.decodeFromJsonElement(RequiredTreesDto.serializer(), JsonObject(requiredWire + (field to codec.parseToJsonElement(bad)))) }
            }
        }
        val requiredNull = RequiredNullableDto(value = null)
        val nullWire = codec.encodeToJsonElement(RequiredNullableDto.serializer(), requiredNull) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableDto.serializer(), nullWire).value == null)
        check(runCatching { codec.decodeFromString<RequiredNullableDto>("{}") }.isSuccess == !nulls)
        val missing = codec.decodeFromString<HolderDto>("""{"required":[["two"]]}""")
        nestedStringEnumSameDefaults(missing, value)
        nestedStringEnumReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":[["two"]],"rows":null,"listMap":null,"mapList":null,"mapMap":null}""").let { it.rows == null && it.listMap == null && it.mapList == null && it.mapMap == null })
        val supplied = codec.decodeFromString<HolderDto>("""{"required":[["two"]],"rows":[null,[],[null,"two"]],"listMap":[{"new":"no"}],"mapList":{"new":["down"]},"mapMap":{"new":{"new":"off"}}}""")
        check(supplied.rows == listOf(null, emptyList(), listOf(null, HolderDto.RowsInnerInner.TWO)))
        check(supplied.listMap == listOf(mapOf("new" to HolderDto.ListMapInnerValue.NO)))
        check(supplied.mapList == mapOf("new" to listOf(HolderDto.MapListValueInner.DOWN)))
        check(supplied.mapMap == mapOf("new" to mapOf("new" to HolderDto.MapMapValueValue.OFF)))
        val futureWire = codec.encodeToJsonElement(HolderDto.serializer(), future) as JsonObject
        check(futureWire["rows"] == codec.parseToJsonElement("""[["future😀",null]]"""))
        check(futureWire["listMap"] == codec.parseToJsonElement("""[{"future":"future"}]"""))
        nestedStringEnumSameDefaults(codec.decodeFromJsonElement(HolderDto.serializer(), futureWire), future)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        for (field in listOf("rows", "listMap", "mapList", "mapMap", "mixed", "special", "empty", "emptyMap")) check(encoded.containsKey(field) == defaults)
        if (defaults) {
            check(encoded["rows"] == codec.parseToJsonElement("""[["one",null,"two","null"],null,[]]"""))
            check(encoded["listMap"] == codec.parseToJsonElement("""[{"${'$'}ref":"no","a/~":"yes","z":null},null,{}]"""))
            check(encoded["mapList"] == codec.parseToJsonElement("""{"a":["up",null],"b":[],"z":null}"""))
            check(encoded["mapMap"] == codec.parseToJsonElement("""{"a":{"a":"on","b":"off","z":null},"z":{}}"""))
        }
        nestedStringEnumSameDefaults(codec.decodeFromJsonElement(HolderDto.serializer(), encoded), missing)
        val nested = HolderDto.Nested()
        check(codec.decodeFromJsonElement(HolderDto.Nested.serializer(), codec.encodeToJsonElement(HolderDto.Nested.serializer(), nested)).value == nested.value)
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.rows == Patch.Absent && absent.listMap == Patch.Absent && absent.mapList == Patch.Absent && absent.mapMap == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it in listOf("rows", "listMap", "mapList", "mapMap") })
        val update = PatchInputDto(
            required = listOf(listOf(PatchInputDto.RequiredInnerInner.TWO)),
            rows = Patch.Value(listOf(listOf(PatchInputDto.RowsInnerInner.Unrecognized("future")))),
            listMap = Patch.Value(listOf(mapOf("a" to PatchInputDto.ListMapInnerValue.YES))),
            mapList = Patch.Value(mapOf("a" to listOf(PatchInputDto.MapListValueInner.UP))),
            mapMap = Patch.Value(mapOf("a" to mapOf("b" to PatchInputDto.MapMapValueValue.Unrecognized("future")))))
        val updateWire = codec.encodeToJsonElement(PatchInputDto.serializer(), update) as JsonObject
        check(updateWire["rows"] == codec.parseToJsonElement("""[["future"]]"""))
        check(updateWire["mapMap"] == codec.parseToJsonElement("""{"a":{"b":"future"}}"""))
        val decodedUpdate = codec.decodeFromJsonElement(PatchInputDto.serializer(), updateWire)
        check(decodedUpdate.rows == update.rows && decodedUpdate.listMap == update.listMap && decodedUpdate.mapList == update.mapList && decodedUpdate.mapMap == update.mapMap)
        val cleared = codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":null,"listMap":null,"mapList":null,"mapMap":null}""")
        check(cleared.rows == Patch.Null && cleared.listMap == Patch.Null && cleared.mapList == Patch.Null && cleared.mapMap == Patch.Null)
        val clearedWire = codec.encodeToJsonElement(PatchInputDto.serializer(), cleared) as JsonObject
        for (field in listOf("rows", "listMap", "mapList", "mapMap")) check(clearedWire[field] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        for ((bad, rule) in nestedStringEnumInvalidPayloads(checked)) nestedStringEnumReject(rule) { codec.decodeFromString<HolderDto>(bad) }
        if (checked) {
            nestedStringEnumReject("minItems") { PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList()))) }
            nestedStringEnumReject("maxProperties") { PatchInputDto(required = null, listMap = Patch.Value(listOf(mapOf("a" to PatchInputDto.ListMapInnerValue.YES, "b" to PatchInputDto.ListMapInnerValue.YES, "c" to PatchInputDto.ListMapInnerValue.YES)))) }
            nestedStringEnumReject("maxItems") { PatchInputDto(required = null, mapList = Patch.Value(mapOf("a" to List(3) { PatchInputDto.MapListValueInner.UP }))) }
            nestedStringEnumReject("maxProperties") { PatchInputDto(required = null, mapMap = Patch.Value(mapOf("a" to mapOf("a" to PatchInputDto.MapMapValueValue.ON, "b" to PatchInputDto.MapMapValueValue.ON, "c" to PatchInputDto.MapMapValueValue.ON)))) }
            nestedStringEnumReject("minItems") { codec.decodeFromString<PatchInputDto>("""{"required":null,"rows":[[]]}""") }
        }
        nestedStringEnumWire(codec, missing, checked)
        println("Nested String enum defaults Json defaults=$defaults nulls=$nulls checked=$checked lexical-open-enum-constructor-serde-Patch-HTTP-liveSSE passed")
    }
}

private fun nestedStringEnumInvalidPayloads(checked: Boolean): List<Pair<String, String?>> {
    val invalid = mutableListOf<Pair<String, String?>>(
        """{"required":[[null]]}""" to null,
        """{"required":[["two"]],"rows":[{}]}""" to null,
        """{"required":[["two"]],"listMap":[{"bad":1}]}""" to null,
        """{"required":[["two"]],"mapList":{"a":[false]}}""" to null,
        """{"required":[["two"]],"mapMap":{"a":{"b":[]}}}""" to null,
        "{" to null)
    if (checked) invalid += listOf(
        """{"required":[]}""" to "minItems",
        """{"required":[[]]}""" to "minItems",
        """{"required":[["two"]],"rows":[null,null,null,null]}""" to "maxItems",
        """{"required":[["two"]],"rows":[["one","one","one","one","one"]]}""" to "maxItems",
        """{"required":[["two"]],"listMap":[null,null,null,null]}""" to "maxItems",
        """{"required":[["two"]],"listMap":[{"a":null,"b":null,"c":null,"d":null}]}""" to "maxProperties",
        """{"required":[["two"]],"mapList":{"a":null,"b":null,"c":null,"d":null}}""" to "maxProperties",
        """{"required":[["two"]],"mapList":{"a":[null,null,null]}}""" to "maxItems",
        """{"required":[["two"]],"mapMap":{"a":{},"b":{},"c":{}}}""" to "maxProperties",
        """{"required":[["two"]],"mapMap":{"a":{"a":null,"b":null,"c":null,"d":null}}}""" to "maxProperties",
        """{"required":[["two"]],"mixed":[{"a":[{"a":"deep","b":"deep"}]}]}""" to "maxProperties")
    return invalid
}

private suspend fun nestedStringEnumWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val good = """{"required":[["two"]]}"""
    var payload = good
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NestedStringEnumEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Request callback must run once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nestedStringEnumBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/inline" -> respond("{}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            "/patch" -> {
                check(codec.parseToJsonElement(nestedStringEnumBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected nested String enum path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NestedStringEnum(http, "http://example.test/", codec).defaults
        nestedStringEnumSameDefaults(api.send(value) { callbacks++ }, value)
        check(api.fetch { callbacks++ }.listMap == listOf(mapOf("a" to FetchResponseDto.ListMapInnerValue.ONE)))
        for (patch in listOf(
            PatchInputDto(required = null),
            PatchInputDto(required = null, rows = Patch.Null, listMap = Patch.Null, mapList = Patch.Null, mapMap = Patch.Null),
            PatchInputDto(required = listOf(listOf(PatchInputDto.RequiredInnerInner.TWO)), rows = Patch.Value(listOf(listOf(PatchInputDto.RowsInnerInner.Unrecognized("future")))), listMap = Patch.Value(listOf(mapOf("a" to PatchInputDto.ListMapInnerValue.YES))), mapList = Patch.Value(mapOf("a" to listOf(PatchInputDto.MapListValueInner.UP))), mapMap = Patch.Value(mapOf("a" to mapOf("b" to PatchInputDto.MapMapValueValue.ON))))
        )) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch) { callbacks++ }
        }
        if (checked) {
            val beforeCalls = calls
            val beforeCallbacks = callbacks
            nestedStringEnumReject("minItems") { api.send(HolderDto(required = listOf(emptyList()))) { callbacks++ } }
            nestedStringEnumReject("minItems") { api.patchPatch(PatchInputDto(required = null, rows = Patch.Value(listOf(emptyList())))) { callbacks++ } }
            check(calls == beforeCalls && callbacks == beforeCallbacks)
        }
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watch { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        repeat(2) {
            nestedStringEnumSameDefaults((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data, value)
            check(calls == before + it + 1 && callbacks == beforeCallbacks + it + 1)
        }
        payload = """{"required":[["future"]],"rows":[["future😀",null]],"listMap":[{"a":"future"}],"mapList":{"a":["future"]},"mapMap":{"a":{"b":"future"}}}"""
        val future = api.send(value) { callbacks++ }
        check(future.required.single().single() == HolderDto.RequiredInnerInner.Unrecognized("future"))
        check(future.rows?.single()?.first() == HolderDto.RowsInnerInner.Unrecognized("future😀"))
        check(future.listMap?.single()?.get("a") == HolderDto.ListMapInnerValue.Unrecognized("future"))
        check(future.mapList?.get("a")?.single() == HolderDto.MapListValueInner.Unrecognized("future"))
        check(future.mapMap?.get("a")?.get("b") == HolderDto.MapMapValueValue.Unrecognized("future"))
        val event = (withTimeout(10000) { api.watch { callbacks++ }.toList() }.single() as WatchEventDto.Holder).data
        check(event.rows == future.rows && event.listMap == future.listMap && event.mapList == future.mapList && event.mapMap == future.mapMap)
        for ((bad, rule) in nestedStringEnumInvalidPayloads(checked)) {
            payload = bad
            failing = true
            val beforeFailure = calls
            nestedStringEnumReject(rule) { api.send(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            nestedStringEnumReject(rule) { withTimeout(10000) { api.watch { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            check(channels.last().isClosedForRead)
            payload = good
            failing = false
            nestedStringEnumSameDefaults(api.send(value) { callbacks++ }, value)
            nestedStringEnumSameDefaults((withTimeout(10000) { api.watch { callbacks++ }.toList() }.single() as WatchEventDto.Holder).data, value)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

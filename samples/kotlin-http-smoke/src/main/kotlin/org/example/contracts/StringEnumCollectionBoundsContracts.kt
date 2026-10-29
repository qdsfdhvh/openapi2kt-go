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
import kotlin.time.Instant
import kotlin.uuid.Uuid
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.datetime.LocalDate
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.serializer
import org.example.stringenumbounds.StringEnumBounds
import org.example.stringenumbounds.api.models.PatchInputDto
import org.example.stringenumbounds.api.models.WatchEventDto
import org.example.stringenumbounds.api.models.WatchTreeEventDto
import org.example.stringenumbounds.api.models.WatchBorrowedEventDto
import org.example.stringenumbounds.api.models.WatchRootWordEventDto
import org.example.stringenumbounds.models.RootWordDto
import org.example.stringenumbounds.models.RootWordAliasDto
import org.example.stringenumbounds.models.RootWordTreeDto
import org.example.stringenumbounds.models.RequiredRootWordDto
import org.example.stringenumbounds.models.`RootWordDto default`
import org.example.stringenumbounds.models.`RootWordAliasDto default`
import org.example.stringenumbounds.models.`MaybeRootWordDto default`
import org.example.stringenumbounds.models.`NullRootWordDto default`
import org.example.stringenumbounds.models.`RootWordTreeDto default`
import org.example.stringenumbounds.models.`RootWordTreeAliasDto default`
import org.example.stringenumbounds.models.`MaybeRootWordTreeDto default`
import org.example.stringenumbounds.models.RootWordTreeHolderDto
import org.example.stringenumbounds.models.HolderDto
import org.example.stringenumbounds.models.Patch
import org.example.stringenumbounds.models.RequiredMapsDto
import org.example.stringenumbounds.models.RequiredStatesDto
import org.example.stringenumbounds.models.RequiredTreesDto
import org.example.stringenumbounds.models.RequiredRootTreesDto
import org.example.stringenumbounds.models.BoundedTreeAliasDto
import org.example.stringenumbounds.models.BoundedTreeDtoInnerValue
import org.example.stringenumbounds.models.`BoundedTreeDto default`
import org.example.stringenumbounds.models.`BoundedTreeAliasDto default`
import org.example.stringenumbounds.models.`MaybeBoundedTreeDto default`
import org.example.stringenumbounds.models.BorrowedTreeDto
import org.example.stringenumbounds.models.BorrowedWordDto
import org.example.stringenumbounds.models.RequiredBorrowedDto
import org.example.stringenumbounds.models.`ScalarRefCountDto default`
import org.example.stringenumbounds.models.`ScalarRefLabelDto default`
import org.example.stringenumbounds.models.ScalarFactoryHolderDto
import org.example.stringenumbounds.models.`ScalarFactoryRowsDto default`
import org.example.stringenumbounds.models.`ScalarFactoryRowsAliasDto default`
import org.example.stringenumbounds.models.`ScalarFactoryMapDto default`
import org.example.stringenumbounds.models.`ScalarNestedMapDto default`
import org.example.stringenumbounds.models.`ScalarNestedMapAliasDto default`
import org.example.stringenumbounds.models.ScalarNestedRowsDto
import org.example.stringenumbounds.models.`ScalarNestedRowsDto default`
import org.example.stringenumbounds.models.`ScalarNestedRowsAliasDto default`
import org.example.stringenumbounds.models.`MaybeScalarNestedRowsDto default`
import org.example.stringenumbounds.models.InlineScalarRowsDto
import org.example.stringenumbounds.models.`InlineScalarValuesDto default`
import org.example.stringenumbounds.models.`InlineScalarValuesAliasDto default`
import org.example.stringenumbounds.models.`InlineDateValuesDto default`
import org.example.stringenumbounds.models.`InlineDateValuesAliasDto default`
import org.example.stringenumbounds.models.`InlineInstantValuesDto default`
import org.example.stringenumbounds.models.`InlineInstantValuesAliasDto default`
import org.example.stringenumbounds.models.`InlineUuidValuesDto default`
import org.example.stringenumbounds.models.`InlineUuidValuesAliasDto default`
import org.example.stringenumbounds.models.`InlineByteValuesDto default`
import org.example.stringenumbounds.models.`InlineByteValuesAliasDto default`
import org.example.stringenumbounds.models.InlineWordValuesDtoInner
import org.example.stringenumbounds.models.`InlineWordValuesDto default`
import org.example.stringenumbounds.models.`InlineWordValuesAliasDto default`
import org.example.stringenumbounds.models.`InlineNativeValuesDto default`
import org.example.stringenumbounds.models.`InlineNativeValuesAliasDto default`
import org.example.stringenumbounds.models.`InlineBooleanValuesDto default`
import org.example.stringenumbounds.models.`InlineBooleanValuesAliasDto default`
import org.example.stringenumbounds.models.`ScalarBytesDto default`
import org.example.stringenumbounds.models.`ScalarBytesRowsDto default`
import org.example.stringenumbounds.models.`BorrowedWordDto default`
import org.example.stringenumbounds.models.`MaybeBorrowedWordDto default`
import org.example.stringenumbounds.models.`BorrowedTreeDto default`
import org.example.stringenumbounds.models.`BorrowedTreeAliasDto default`
import org.example.stringenumbounds.models.`MaybeBorrowedTreeDto default`
import org.example.stringenumbounds.models.BoundedValueDto
import org.example.stringenumbounds.models.`BoundedAliasDto default`
import org.example.stringenumbounds.models.`MaybeBoundedAliasDto default`

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class StringEnumBoundsEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun stringEnumBoundsReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected $rule failure" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun stringEnumBoundsBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> stringEnumBoundsBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected request body $body")
}

suspend fun stringEnumCollectionBoundsContracts(checked: Boolean) {
    val value = HolderDto(required = listOf(HolderDto.RequiredInner.TWO), kotlin = "shadow", kotlinx = "shadow", labelsInner = "shadow")
    check(value.scalarRefCounts == null && value.scalarRefLabels == null && value.inlineNested == null)
    check(`ScalarRefCountDto default`() == 99 && `ScalarRefLabelDto default`() == "root")
    check(`ScalarFactoryRowsDto default`() == listOf(1, 2))
    check(`ScalarFactoryRowsAliasDto default`() == listOf(3))
    check(`ScalarFactoryMapDto default`() == mapOf("map" to "root-map"))
    check(`ScalarNestedMapDto default`() == mapOf("inner" to 11))
    check(`ScalarNestedMapAliasDto default`() == mapOf("alias" to 12))
    check(`ScalarNestedRowsDto default`() == listOf(mapOf("outer" to 13)))
    check(`ScalarNestedRowsAliasDto default`() == listOf(mapOf("alias" to 14)))
    check(`MaybeScalarNestedRowsDto default`() == null)
    check(`InlineScalarValuesDto default`() == listOf(20))
    check(`InlineScalarValuesAliasDto default`() == listOf(21))
    check(`InlineScalarValuesDto default`() !== `InlineScalarValuesDto default`())
    check(`InlineDateValuesDto default`() == listOf(LocalDate.parse("2024-02-29")))
    check(`InlineDateValuesAliasDto default`() == listOf(LocalDate.parse("2024-03-01")))
    check(`InlineInstantValuesDto default`() == listOf(Instant.parse("2024-02-29T00:00:00Z")))
    check(`InlineInstantValuesAliasDto default`() == listOf(Instant.parse("2024-03-01T00:00:00Z")))
    check(`InlineUuidValuesDto default`() == listOf(Uuid.parse("123e4567-e89b-12d3-a456-426614174000")))
    check(`InlineUuidValuesAliasDto default`() == listOf(Uuid.parse("123e4567-e89b-12d3-a456-426614174001")))
    check(`InlineByteValuesDto default`().single().contentEquals(byteArrayOf(1, 2)))
    check(`InlineByteValuesAliasDto default`().single().contentEquals(byteArrayOf(3)))
    val inlineBytes = `InlineByteValuesDto default`()
    val otherInlineBytes = `InlineByteValuesDto default`()
    check(inlineBytes !== otherInlineBytes && inlineBytes.single() !== otherInlineBytes.single())
    check(`InlineWordValuesDto default`() == listOf(InlineWordValuesDtoInner.ONE))
    check(`InlineWordValuesAliasDto default`() == listOf(InlineWordValuesDtoInner.TWO))
    check(`InlineWordValuesDto default`() !== `InlineWordValuesDto default`())
    check(`InlineNativeValuesDto default`() == listOf(1) && `InlineNativeValuesAliasDto default`() == listOf(2))
    check(`InlineBooleanValuesDto default`() == listOf(true) && `InlineBooleanValuesAliasDto default`() == listOf(false))
    val nestedRows = `ScalarNestedRowsDto default`()
    val otherNestedRows = `ScalarNestedRowsDto default`()
    check(nestedRows !== otherNestedRows && nestedRows.single() !== otherNestedRows.single())
    check(`ScalarFactoryRowsDto default`() !== `ScalarFactoryRowsDto default`())
    check(`ScalarFactoryMapDto default`() !== `ScalarFactoryMapDto default`())
    check(`ScalarBytesDto default`().contentEquals(byteArrayOf(1, 2)))
    val byteRows = `ScalarBytesRowsDto default`()
    val otherByteRows = `ScalarBytesRowsDto default`()
    check(byteRows.single().contentEquals(byteArrayOf(5, 6)))
    check(byteRows !== otherByteRows && byteRows.single() !== otherByteRows.single())
    check(ScalarFactoryHolderDto().rows == listOf(4))
    check(ScalarFactoryHolderDto().labels == mapOf("field" to "field"))
    check(ScalarFactoryHolderDto().unannotated == null)
    check(ScalarFactoryHolderDto().inline == listOf(6))
    check(ScalarFactoryHolderDto().bytes?.single()?.contentEquals(byteArrayOf(7)) == true)
    check(ScalarFactoryHolderDto().nested == listOf(mapOf("field" to 15)))
    check(ScalarFactoryHolderDto.default().rows == listOf(5))
    check(ScalarFactoryHolderDto.default().labels == mapOf("owner" to "owner"))
    check(ScalarFactoryHolderDto.default().unannotated == null)
    check(ScalarFactoryHolderDto.default().inline == listOf(7))
    check(ScalarFactoryHolderDto.default().bytes?.single()?.contentEquals(byteArrayOf(8, 9)) == true)
    check(ScalarFactoryHolderDto.default().nested == listOf(mapOf("owner" to 16)))
    check(value.codes == listOf(HolderDto.CodesInner.ONE, null, HolderDto.CodesInner.TWO))
    check(value.labels == mapOf("a/~" to HolderDto.LabelsValue.YES, "\$ref" to HolderDto.LabelsValue.NO))
    check(value.bareCodes == null && value.inlineNative == listOf(1) && value.inlineNativeOnly == null)
    check(value.special?.single()?.serialName == "quote\"\$😀")
    check(value.state?.serialName == "one" && value.metadataState?.serialName == "two" && value.bareState == null)
    val factory = BoundedValueDto.default()
    check(factory.state.serialName == "one" && factory.labels?.get("root")?.serialName == "one")
    check(factory.words !== BoundedValueDto.default().words)
    check(factory.trees?.single()?.get("root")?.serialName == "one")
    check(`BoundedAliasDto default`().trees?.single()?.get("alias")?.serialName == "two")
    check(value.bounded?.trees?.single()?.get("outer")?.serialName == "two")
    check(value.boundedRows?.single()?.trees?.single()?.get("outer")?.serialName == "two")
    check(value.boundedInline?.trees?.single()?.get("inline")?.serialName == "one")
    check(`BoundedTreeDto default`()?.single()?.get("root") == BoundedTreeDtoInnerValue.ONE)
    check(`BoundedTreeAliasDto default`().single()?.get("alias") == BoundedTreeDtoInnerValue.TWO)
    check(`MaybeBoundedTreeDto default`() == null)
    check(`BoundedTreeDto default`() !== `BoundedTreeDto default`())
    check(value.rootTree?.single()?.get("own") == BoundedTreeDtoInnerValue.ONE && value.rootTreeUnannotated == null)
    check(factory.rootTree?.single()?.get("factory") == BoundedTreeDtoInnerValue.TWO)
    check(value.boundedInline?.rootTree?.single()?.get("factory") == BoundedTreeDtoInnerValue.TWO)
    check(`BoundedAliasDto default`().state.serialName == "two" && `MaybeBoundedAliasDto default`() == null)
    check(value.bounded?.state?.serialName == "one" && value.bounded?.labels?.get("outer")?.serialName == "two")
    check(value.boundedRows?.single()?.state?.serialName == "one" && value.boundedInline?.state?.serialName == "two")
    check(value.boundedRows !== HolderDto(required = value.required).boundedRows)
    check(value.metadataRows?.single()?.serialName == "one" && value.metadataLabels?.get("first")?.serialName == "two")
    check(HolderDto.State.X.serialName == "x" && HolderDto.MetadataState.fromSerialName("future").serialName == "future")
    check(value.empty == emptyList<HolderDto.EmptyInner>() && value.emptyMap == emptyMap<String, HolderDto.EmptyMapValue>() && value.nil == null)
    check(HolderDto.Nested().labels == mapOf("one" to HolderDto.Nested.LabelsValue.NESTED, "nil" to null))
    check(HolderDto.CodesInner.entries.map { it.serialName } == listOf("one", "two", "x", "toolong", "BAD"))
    check(HolderDto.CodesInner.fromSerialName("future") == HolderDto.CodesInner.Unrecognized("future"))
    check(HolderDto.CodesInner.serializer().descriptor.serialName == "org.example.stringenumbounds.models.HolderDto.CodesInner")
    val second = HolderDto(required = value.required)
    check(value.codes !== second.codes && value.labels !== second.labels && value.empty !== second.empty && value.emptyMap !== second.emptyMap)
    check(value.trees?.first() == null && value.trees?.get(1)?.get("a/~") == HolderDto.TreesInnerValue.ONE)
    check(value.trees?.get(1)?.get("\$ref") == null)
    check(value.matrix == listOf(listOf(HolderDto.MatrixInnerInner.ONE, null)))
    check(value.forest == mapOf("\$ref" to listOf(HolderDto.ForestValueInner.TWO, null), "nil" to null))
    check(value.maps == null && value.trees !== second.trees && value.trees?.get(1) !== second.trees?.get(1))
    check(HolderDto.TreesInnerValue.serializer().descriptor.serialName == "org.example.stringenumbounds.models.HolderDto.TreesInnerValue")
    val future = HolderDto(required = listOf(HolderDto.RequiredInner.Unrecognized("future")), codes = listOf(null, HolderDto.CodesInner.Unrecognized("future")), labels = mapOf("future" to HolderDto.LabelsValue.Unrecognized("future")), bareCodes = listOf(HolderDto.BareCodesInner.Unrecognized("future")), bareLabels = mapOf("substring" to HolderDto.BareLabelsValue.Unrecognized("futurecat"), "nil" to null))
    val invalid = listOf("x" to "minLength", "toolong" to "maxLength", "BAD" to "pattern", "!" to "minLength", "futurelong" to "maxLength", "FUTURE" to "pattern")
    if (checked) {
        for ((bad, rule) in invalid) {
            stringEnumBoundsReject(rule) { BoundedValueDto(state = BoundedValueDto.State.fromSerialName(bad)) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, state = HolderDto.State.fromSerialName(bad)) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, metadataState = HolderDto.MetadataState.fromSerialName(bad)) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, bareState = HolderDto.BareState.fromSerialName(bad)) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, metadataRows = listOf(HolderDto.MetadataRowsInner.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { PatchInputDto(required = null, state = Patch.Value(PatchInputDto.State.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { HolderDto(required = listOf(HolderDto.RequiredInner.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, codes = listOf(HolderDto.CodesInner.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, labels = mapOf("a" to HolderDto.LabelsValue.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, trees = listOf(mapOf("a" to HolderDto.TreesInnerValue.fromSerialName(bad)))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, matrix = listOf(listOf(HolderDto.MatrixInnerInner.fromSerialName(bad)))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, forest = mapOf("a" to listOf(HolderDto.ForestValueInner.fromSerialName(bad)))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, maps = mapOf("a" to mapOf("b" to HolderDto.MapsValueValue.fromSerialName(bad)))) }
            stringEnumBoundsReject(rule) { PatchInputDto(required = null, trees = Patch.Value(listOf(mapOf("a" to PatchInputDto.TreesInnerValue.fromSerialName(bad))))) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, rootTree = listOf(mapOf("a" to BoundedTreeDtoInnerValue.fromSerialName(bad)))) }
            stringEnumBoundsReject(rule) { PatchInputDto(required = null, rootTree = Patch.Value(listOf(mapOf("a" to BoundedTreeDtoInnerValue.fromSerialName(bad))))) }
        }
        stringEnumBoundsReject("uniqueItems") { HolderDto(required = value.required, trees = listOf(mapOf("a" to HolderDto.TreesInnerValue.ONE), mapOf("a" to HolderDto.TreesInnerValue.Unrecognized("one")))) }
        stringEnumBoundsReject("uniqueItems") { HolderDto(required = value.required, trees = listOf(null, null)) }
        stringEnumBoundsReject("maxItems") { HolderDto(required = value.required, trees = List(4) { emptyMap() }) }
        stringEnumBoundsReject("maxProperties") { HolderDto(required = value.required, trees = listOf(mapOf("a" to null, "b" to null, "c" to null))) }
        stringEnumBoundsReject("minItems") { HolderDto(required = emptyList()) }
        stringEnumBoundsReject("maxItems") { HolderDto(required = value.required, codes = List(4) { HolderDto.CodesInner.ONE }) }
        stringEnumBoundsReject("maxProperties") { HolderDto(required = value.required, labels = mapOf("a" to HolderDto.LabelsValue.YES, "b" to HolderDto.LabelsValue.YES, "c" to HolderDto.LabelsValue.YES)) }
        stringEnumBoundsReject("minLength") { HolderDto(required = value.required, bareCodes = listOf(HolderDto.BareCodesInner.X)) }
        stringEnumBoundsReject("pattern") { HolderDto(required = value.required, bareLabels = mapOf("a" to HolderDto.BareLabelsValue.BAD)) }
        stringEnumBoundsReject("minLength") { PatchInputDto(required = null, codes = Patch.Value(listOf(PatchInputDto.CodesInner.X))) }
        stringEnumBoundsReject("minLength") { PatchInputDto(required = null, labels = Patch.Value(mapOf("a" to PatchInputDto.LabelsValue.Unrecognized("x")))) }
    } else {
        check(HolderDto(required = listOf(HolderDto.RequiredInner.X), codes = listOf(HolderDto.CodesInner.BAD)).codes?.single() == HolderDto.CodesInner.BAD)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // The vocabulary codec remains open and independent of field bounds.
        check(codec.decodeFromString<HolderDto.CodesInner>("\"x\"") == HolderDto.CodesInner.X)
        check(codec.decodeFromString<HolderDto.CodesInner>("\"FUTURE\"") == HolderDto.CodesInner.Unrecognized("FUTURE"))
        check(codec.encodeToString(HolderDto.CodesInner.serializer(), HolderDto.CodesInner.X) == "\"x\"")
        check(codec.decodeFromString<HolderDto.State>("\"x\"") == HolderDto.State.X)
        check(codec.decodeFromString<HolderDto.MetadataState>("\"future\"").serialName == "future")
        val missing = codec.decodeFromString<HolderDto>("""{"required":["two"]}""")
        check(missing.codes == value.codes && missing.labels == value.labels)
        check(missing.bareCodes == null && missing.inlineNative == listOf(1) && missing.inlineNativeOnly == null)
        val supplied = codec.decodeFromString<HolderDto>("""{"required":["two"],"bareCodes":["one"],"inlineNative":[2],"inlineNativeOnly":{"chosen":true}}""")
        check(supplied.bareCodes?.single() == HolderDto.BareCodesInner.ONE && supplied.inlineNative == listOf(2) && supplied.inlineNativeOnly == mapOf("chosen" to true))
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), supplied)) == supplied)
        check(missing.state == value.state && missing.metadataState == value.metadataState)
        check(missing.bounded == value.bounded && missing.boundedRows == value.boundedRows && missing.boundedInline == value.boundedInline)
        check(missing.trees == value.trees && missing.matrix == value.matrix && missing.forest == value.forest)
        check(missing.rootTree == value.rootTree && missing.rootTreeUnannotated == null)
        check(codec.decodeFromString<BoundedTreeDtoInnerValue>("\"x\"") == BoundedTreeDtoInnerValue.X)
        check(codec.decodeFromString<BoundedTreeAliasDto>("""[{"raw":"x"}]""").single()?.get("raw") == BoundedTreeDtoInnerValue.X)
        stringEnumBoundsReject { codec.decodeFromString<RequiredRootTreesDto>("{}") }
        check(codec.decodeFromString<RequiredRootTreesDto>("""{"value":[{"a":"one"}]}""").value.single()?.get("a") == BoundedTreeDtoInnerValue.ONE)
        check(codec.decodeFromString<HolderDto.TreesInnerValue>("\"x\"") == HolderDto.TreesInnerValue.X)
        val nestedFuture = codec.decodeFromString<HolderDto>("""{"required":["two"],"trees":[null,{"a":"future","nil":null}],"matrix":[["future",null]],"forest":{"a":["future",null]},"maps":{"a":{"b":"future"}}}""")
        check(nestedFuture.trees?.get(1)?.get("a") == HolderDto.TreesInnerValue.Unrecognized("future"))
        check(nestedFuture.maps?.get("a")?.get("b") == HolderDto.MapsValueValue.Unrecognized("future"))
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), nestedFuture)) == nestedFuture)
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"trees":null}""").trees == null)
        stringEnumBoundsReject { codec.decodeFromString<RequiredTreesDto>("{}") }
        check(codec.decodeFromString<RequiredTreesDto>("""{"value":[{"a":"one"}]}""").value.single()?.get("a") == RequiredTreesDto.ValueInnerValue.ONE)
        check(codec.decodeFromString<BoundedValueDto>(codec.encodeToString(BoundedValueDto.serializer(), factory)) == factory)
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"state":null,"metadataState":null}""").let { it.state == null && it.metadataState == null })
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"state":"future","metadataState":"future"}""").let { it.state?.serialName == "future" && it.metadataState?.serialName == "future" })
        check(codec.decodeFromString<RequiredStatesDto>("""{"state":null}""").state == null)
        check(runCatching { codec.decodeFromString<RequiredStatesDto>("{}").state }.isSuccess == !nulls)
        stringEnumBoundsReject { codec.decodeFromString<HolderDto>("{}") }
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"codes":null,"labels":null}""").let { it.codes == null && it.labels == null })
        check(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), future)) == future)
        check(codec.decodeFromString<HolderDto>("""{"required":["two"],"bareLabels":{"a":"hotdog","nil":null}}""").bareLabels?.get("a") == HolderDto.BareLabelsValue.HOTDOG)
        val encoded = codec.encodeToJsonElement(HolderDto.serializer(), missing) as JsonObject
        check(encoded.containsKey("codes") == defaults && encoded.containsKey("labels") == defaults)
        check(codec.decodeFromJsonElement(HolderDto.serializer(), encoded) == missing)
        val requiredMaps = RequiredMapsDto(labels = mapOf("a" to RequiredMapsDto.LabelsValue.YES), nullableLabels = null)
        check(codec.decodeFromJsonElement(RequiredMapsDto.serializer(), codec.encodeToJsonElement(RequiredMapsDto.serializer(), requiredMaps)) == requiredMaps)
        stringEnumBoundsReject { codec.decodeFromString<RequiredMapsDto>("{}") }
        val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
        check(absent.codes == Patch.Absent && absent.labels == Patch.Absent && absent.required == null)
        check(absent.state == Patch.Absent)
        check(absent.bounded == Patch.Absent)
        check(absent.trees == Patch.Absent)
        check(absent.rootTree == Patch.Absent)
        check(codec.decodeFromString<PatchInputDto>("""{"required":null,"rootTree":null}""").rootTree == Patch.Null)
        val rootPatch = codec.decodeFromString<PatchInputDto>("""{"required":null,"rootTree":[{"a":"future"}]}""")
        check((rootPatch.rootTree as Patch.Value).value.single()?.get("a") == BoundedTreeDtoInnerValue.Unrecognized("future"))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), rootPatch)) == rootPatch)
        check(codec.decodeFromString<PatchInputDto>("""{"required":null,"trees":null}""").trees == Patch.Null)
        val treePatch = codec.decodeFromString<PatchInputDto>("""{"required":null,"trees":[null,{"a":"future","nil":null}]}""")
        check((treePatch.trees as Patch.Value).value[1]?.get("a") == PatchInputDto.TreesInnerValue.Unrecognized("future"))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), treePatch)) == treePatch)
        check(codec.decodeFromString<PatchInputDto>("""{"required":null,"bounded":null}""").bounded == Patch.Null)
        val boundedPatch = codec.decodeFromString<PatchInputDto>("""{"required":null,"bounded":{"state":"two"}}""").bounded
        check(boundedPatch is Patch.Value && boundedPatch.value.state.serialName == "two")
        check(codec.decodeFromString<PatchInputDto>("""{"required":null,"state":null}""").state == Patch.Null)
        check(codec.decodeFromString<PatchInputDto>("""{"required":null,"state":"future"}""").state == Patch.Value(PatchInputDto.State.fromSerialName("future")))
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), absent) as JsonObject).keys.none { it == "codes" || it == "labels" || it == "trees" || it == "rootTree" })
        val update = PatchInputDto(required = listOf(PatchInputDto.RequiredInner.TWO), codes = Patch.Value(listOf(PatchInputDto.CodesInner.Unrecognized("future"))), labels = Patch.Value(mapOf("yes" to PatchInputDto.LabelsValue.YES, "nil" to null)))
        check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), update)) == update)
        val explicitNull = codec.decodeFromString<PatchInputDto>("""{"required":["two"],"codes":null,"labels":null}""")
        check(explicitNull.codes == Patch.Null && explicitNull.labels == Patch.Null)
        check((codec.encodeToJsonElement(PatchInputDto.serializer(), explicitNull) as JsonObject)["codes"] === JsonNull)
        check(runCatching { codec.decodeFromString<PatchInputDto>("{}").also { check(!nulls && it.required == null) } }.isSuccess == !nulls)
        stringEnumBoundsReject { codec.decodeFromString<HolderDto>("""{"required":[null]}""") }
        stringEnumBoundsReject { codec.decodeFromString<HolderDto>("""{"required":["two"],"labels":{"bad":1}}""") }
        if (checked) {
            for ((bad, rule) in invalid) {
                stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>("""{"required":["two"],"bounded":{"state":"$bad"}}""") }
                stringEnumBoundsReject(rule) { codec.decodeFromString<PatchInputDto>("""{"required":null,"bounded":{"state":"$bad"}}""") }
            }
            for ((bad, rule) in invalid) for (field in listOf("state", "metadataState", "bareState")) {
                stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>("""{"required":["two"],"$field":"$bad"}""") }
            }
            for ((bad, rule) in invalid) {
                val payload = """{"required":["two"],"trees":[null,{"a":"$bad"}]}"""
                stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>(payload) }
                stringEnumBoundsReject(rule) { codec.decodeFromJsonElement(HolderDto.serializer(), codec.parseToJsonElement(payload)) }
                stringEnumBoundsReject(rule) { codec.decodeFromString<PatchInputDto>("""{"required":null,"trees":[{"a":"$bad"}]}""") }
                stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>("""{"required":["two"],"rootTree":[{"a":"$bad"}]}""") }
                stringEnumBoundsReject(rule) { codec.decodeFromString<PatchInputDto>("""{"required":null,"rootTree":[{"a":"$bad"}]}""") }
            }
            stringEnumBoundsReject("uniqueItems") { codec.decodeFromString<HolderDto>("""{"required":["two"],"trees":[{"a":"one","b":"two"},{"b":"two","a":"one"}]}""") }
            for ((bad, rule) in invalid) for (field in listOf("required", "codes", "labels")) {
                val payload = if (field == "labels") """{"required":["two"],"labels":{"a":"$bad"}}""" else if (field == "required") """{"required":["$bad"]}""" else """{"required":["two"],"codes":["$bad"]}"""
                stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>(payload) }
                stringEnumBoundsReject(rule) { codec.decodeFromJsonElement(HolderDto.serializer(), codec.parseToJsonElement(payload)) }
            }
            stringEnumBoundsReject("minItems") { PatchInputDto(required = null, codes = Patch.Value(emptyList())) }
        }
        stringEnumBoundsWire(codec, missing, checked, invalid)
        borrowedCheckedEnumContracts(codec, missing, checked, invalid)
        checkedRootEnumContracts(codec, missing, checked, invalid)
        println("String enum collection bounds defaults=$defaults nulls=$nulls checked=$checked ctor-serde-Patch-HTTP-fragmented-SSE passed")
    }
}

private suspend fun checkedRootEnumContracts(codec: Json, value: HolderDto, checked: Boolean, invalid: List<Pair<String, String>>) {
    check(codec.decodeFromString<ScalarFactoryHolderDto>("{}").rows == listOf(4))
    check(codec.decodeFromString<ScalarFactoryHolderDto>("{}").nested == listOf(mapOf("field" to 15)))
    val rows = codec.decodeFromString<ScalarNestedRowsDto>("""[{"wire":17}]""")
    check(rows == listOf(mapOf("wire" to 17)))
    check(codec.encodeToString(serializer<ScalarNestedRowsDto>(), rows) == """[{"wire":17}]""")
    val inlineRows = codec.decodeFromString<InlineScalarRowsDto>("""[[23]]""")
    check(inlineRows == listOf(listOf(23)))
    check(codec.encodeToString(serializer<InlineScalarRowsDto>(), inlineRows) == """[[23]]""")
    val nested = codec.decodeFromString<ScalarFactoryHolderDto>("""{"nested":[{"wire":17}]}""")
    check(nested.nested == listOf(mapOf("wire" to 17)))
    check(codec.decodeFromString<ScalarFactoryHolderDto>(codec.encodeToString(ScalarFactoryHolderDto.serializer(), nested)).nested == nested.nested)
    check(codec.decodeFromString<ScalarFactoryHolderDto>("""{"rows":[8],"labels":{"wire":"value"}}""").let {
        it.rows == listOf(8) && it.labels == mapOf("wire" to "value") && it.unannotated == null && it.inline == listOf(6) && it.bytes?.single()?.contentEquals(byteArrayOf(7)) == true
    })
    check(`RootWordDto default`() == RootWordDto.ONE && `RootWordAliasDto default`() == RootWordDto.TWO)
    check(`MaybeRootWordDto default`() == null && `NullRootWordDto default`() == null)
    check(`RootWordTreeDto default`()["factory"] == listOf(RootWordDto.ONE))
    check(`RootWordTreeAliasDto default`()["alias"] == listOf(RootWordDto.TWO))
    check(org.example.stringenumbounds.models.`ReferencedNestedValuesDto default`().single()["root"] == listOf(RootWordDto.ONE))
    check(org.example.stringenumbounds.models.`ReferencedNestedValuesAliasDto default`().single()["alias"] == listOf(RootWordDto.TWO))
    check(value.referencedNestedRows == null)
    check(`MaybeRootWordTreeDto default`() == null)
    val freshTree = `RootWordTreeDto default`()
    val anotherTree = `RootWordTreeDto default`()
    check(freshTree !== anotherTree && freshTree["factory"] !== anotherTree["factory"])
    check(RootWordTreeHolderDto.default().tree?.get("outer") == listOf(RootWordDto.TWO))
    check(RootWordTreeHolderDto().tree == null)
    check(value.rootWordTreeAlias?.get("fieldAlias") == listOf(RootWordDto.ONE))
    check(value.rootWordTreeUnannotated == null && value.nativeFactoryUnannotated == null)
    check(org.example.stringenumbounds.models.`FactoryIntDto default`() == 1 && org.example.stringenumbounds.models.`FactoryIntAliasDto default`() == 2)
    check(org.example.stringenumbounds.models.`FactoryIntRowsDto default`() == listOf(1) && org.example.stringenumbounds.models.`FactoryIntRowsAliasDto default`() == listOf(2))
    check(org.example.stringenumbounds.models.`MaybeFactoryIntRowsDto default`() == null && org.example.stringenumbounds.models.`FactoryIntRowsDto default`() !== org.example.stringenumbounds.models.`FactoryIntRowsDto default`())
    check(org.example.stringenumbounds.models.`FactoryLongDto default`() == 1L && org.example.stringenumbounds.models.`FactoryLongAliasDto default`() == 2L)
    check(org.example.stringenumbounds.models.`FactoryLongRowsDto default`() == listOf(1L) && org.example.stringenumbounds.models.`FactoryLongRowsAliasDto default`() == listOf(2L))
    check(org.example.stringenumbounds.models.`MaybeFactoryLongRowsDto default`() == null && org.example.stringenumbounds.models.`FactoryLongRowsDto default`() !== org.example.stringenumbounds.models.`FactoryLongRowsDto default`())
    check(org.example.stringenumbounds.models.`FactoryFloatDto default`() == 1.0f && org.example.stringenumbounds.models.`FactoryFloatAliasDto default`() == 2.0f)
    check(org.example.stringenumbounds.models.`FactoryFloatRowsDto default`() == listOf(1.0f) && org.example.stringenumbounds.models.`FactoryFloatRowsAliasDto default`() == listOf(2.0f))
    check(org.example.stringenumbounds.models.`MaybeFactoryFloatRowsDto default`() == null && org.example.stringenumbounds.models.`FactoryFloatRowsDto default`() !== org.example.stringenumbounds.models.`FactoryFloatRowsDto default`())
    check(org.example.stringenumbounds.models.`FactoryDoubleDto default`() == 1.0 && org.example.stringenumbounds.models.`FactoryDoubleAliasDto default`() == 2.0)
    check(org.example.stringenumbounds.models.`FactoryDoubleRowsDto default`() == listOf(1.0) && org.example.stringenumbounds.models.`FactoryDoubleRowsAliasDto default`() == listOf(2.0))
    check(org.example.stringenumbounds.models.`MaybeFactoryDoubleRowsDto default`() == null && org.example.stringenumbounds.models.`FactoryDoubleRowsDto default`() !== org.example.stringenumbounds.models.`FactoryDoubleRowsDto default`())
    check(org.example.stringenumbounds.models.`FactoryBooleanDto default`() == true && org.example.stringenumbounds.models.`FactoryBooleanAliasDto default`() == false)
    check(org.example.stringenumbounds.models.`FactoryBooleanRowsDto default`() == listOf(true) && org.example.stringenumbounds.models.`FactoryBooleanRowsAliasDto default`() == listOf(false))
    check(org.example.stringenumbounds.models.`MaybeFactoryBooleanRowsDto default`() == null && org.example.stringenumbounds.models.`FactoryBooleanRowsDto default`() !== org.example.stringenumbounds.models.`FactoryBooleanRowsDto default`())
    check(value.rootWord == RootWordDto.ONE && value.rootWords == listOf(RootWordDto.TWO))
    check(value.childOnly == null && value.nullableChildOnly == null && value.mapChildOnly == null)
    check(codec.decodeFromString<HolderDto>("""{"required":["two"],"childOnly":[],"nullableChildOnly":[null],"mapChildOnly":{}}""").let {
        it.childOnly == emptyList<RootWordDto>() && it.nullableChildOnly == listOf(null) && it.mapChildOnly == emptyMap<String, RootWordDto>()
    })
    check(codec.decodeFromString<HolderDto>("""{"required":["two"],"childOnly":["future"]}""").childOnly?.single() == RootWordDto.Unrecognized("future"))
    check(value.rootWordTree?.get("field") == listOf(RootWordDto.ONE) && value.rootWordUnannotated == null)
    check(value.rootWordTree !== HolderDto(required = value.required).rootWordTree)
    check(codec.decodeFromString<RootWordAliasDto>("\"x\"") == RootWordDto.X)
    check(codec.decodeFromString<HolderDto>("""{"required":["two"],"rootWord":"future","rootWordTree":{"a":["future"]}}""").rootWord == RootWordDto.Unrecognized("future"))
    check(codec.decodeFromString<RequiredRootWordDto>("""{"word":"one"}""").word == RootWordDto.ONE)
    stringEnumBoundsReject { codec.decodeFromString<RequiredRootWordDto>("{}") }
    check(codec.decodeFromString<PatchInputDto>("""{"required":null}""").rootWord == Patch.Absent)
    check(codec.decodeFromString<PatchInputDto>("""{"required":null,"rootWord":null}""").rootWord == Patch.Null)
    check(codec.decodeFromString<PatchInputDto>("""{"required":null,"rootWord":"future"}""").rootWord == Patch.Value(RootWordDto.Unrecognized("future")))
    if (checked) {
        for ((bad, rule) in invalid) {
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, rootWord = RootWordDto.fromSerialName(bad)) }
            stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>("""{"required":["two"],"rootWordTree":{"a":["$bad"]}}""") }
            stringEnumBoundsReject(rule) { PatchInputDto(required = null, rootWord = Patch.Value(RootWordDto.fromSerialName(bad))) }
        }
        stringEnumBoundsReject("uniqueItems") { HolderDto(required = value.required, rootWords = listOf(RootWordDto.ONE, RootWordDto.Unrecognized("one"))) }
    }
    var payload = "\"one\""
    var treePayload = """{"wire":["one"]}"""
    var failing = false
    var calls = 0
    val expected: RootWordTreeDto = mapOf("wire" to listOf(RootWordDto.ONE))
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val engine = StringEnumBoundsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/root-word" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(serializer<RootWordAliasDto>(), RootWordDto.ONE))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/root-word-tree" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(serializer<RootWordTreeDto>(), expected))
                respond(treePayload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/root-word-events" -> {
                val channel = ByteChannel(autoFlush = true); channels += channel
                val frame = "event: word\r\ndata: $payload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try { for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte)); if (holdOpen) awaitCancellation() } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected root enum path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StringEnumBounds(http, "http://example.test/", codec).defaults
        check(api.sendRootWord(RootWordDto.ONE) == RootWordDto.ONE && api.sendRootWordTree(expected) == expected)
        val before = calls; val flow = api.watchRootWord(); check(calls == before)
        repeat(2) { check((withTimeout(10000) { flow.toList() }.single() as WatchRootWordEventDto.Word).data == RootWordDto.ONE) }
        payload = "\"future\""; treePayload = """{"a":["future"]}"""
        check(api.sendRootWord(RootWordDto.ONE) == RootWordDto.Unrecognized("future"))
        check(api.sendRootWordTree(expected)["a"]?.single() == RootWordDto.Unrecognized("future"))
        if (checked) {
            val beforeInvalid = calls
            for ((bad, rule) in invalid) {
                stringEnumBoundsReject(rule) { api.sendRootWord(RootWordDto.fromSerialName(bad)) }
                stringEnumBoundsReject(rule) { api.sendRootWordTree(mapOf("a" to listOf(RootWordDto.fromSerialName(bad)))) }
            }
            check(calls == beforeInvalid)
            for ((bad, rule) in invalid) {
                payload = "\"$bad\""; treePayload = """{"a":["$bad"]}"""; failing = true
                stringEnumBoundsReject(rule) { api.sendRootWord(RootWordDto.ONE) }
                stringEnumBoundsReject(rule) { api.sendRootWordTree(expected) }
                stringEnumBoundsReject(rule) { withTimeout(10000) { api.watchRootWord().toList() } }
                check(channels.last().isClosedForRead)
                payload = "\"one\""; treePayload = """{"wire":["one"]}"""; failing = false
                check(api.sendRootWord(RootWordDto.ONE) == RootWordDto.ONE)
                check((withTimeout(10000) { api.watchRootWord().toList() }.single() as WatchRootWordEventDto.Word).data == RootWordDto.ONE)
            }
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

private suspend fun borrowedCheckedEnumContracts(codec: Json, value: HolderDto, checked: Boolean, invalid: List<Pair<String, String>>) {
    check(value.borrowableTree?.get("source")?.get("a") == HolderDto.BorrowableTreeValueValue.ONE)
    check(`BorrowedTreeDto default`()["factory"]?.get("a") == HolderDto.BorrowableTreeValueValue.TWO)
    check(`BorrowedTreeAliasDto default`()["alias"]?.get("a") == HolderDto.BorrowableTreeValueValue.ONE)
    check(`MaybeBorrowedTreeDto default`() == null)
    check(`BorrowedTreeDto default`() !== `BorrowedTreeDto default`())
    check(value.borrowableTree !== HolderDto(required = value.required).borrowableTree)
    check(`BorrowedWordDto default`() == HolderDto.State.TWO && `MaybeBorrowedWordDto default`() == null)
    check(value.borrowedWord == HolderDto.State.ONE && value.borrowedRows == listOf(HolderDto.State.TWO, null))
    check(value.borrowedTree?.get("field")?.get("a") == HolderDto.BorrowableTreeValueValue.ONE && value.borrowedTreeUnannotated == null)
    check(value.borrowedTree !== HolderDto(required = value.required).borrowedTree)
    check(BoundedValueDto.default().borrowedWord == HolderDto.State.TWO)
    check(BoundedValueDto.default().borrowedTree?.get("outer")?.get("a") == HolderDto.BorrowableTreeValueValue.TWO)
    check(codec.decodeFromString<BorrowedWordDto>("\"x\"") == HolderDto.State.X)
    check(codec.decodeFromString<BorrowedTreeDto>("""{"raw":{"a":"x"}}""")["raw"]?.get("a") == HolderDto.BorrowableTreeValueValue.X)
    check(codec.decodeFromString<HolderDto>("""{"required":["two"],"borrowedWord":"future","borrowedTree":{"new":{"a":"future","nil":null}}}""").let { it.borrowedWord == HolderDto.State.Unrecognized("future") && it.borrowedTree?.get("new")?.get("a") == HolderDto.BorrowableTreeValueValue.Unrecognized("future") })
    check(codec.decodeFromString<RequiredBorrowedDto>("""{"word":"one"}""").word == HolderDto.State.ONE)
    stringEnumBoundsReject { codec.decodeFromString<RequiredBorrowedDto>("{}") }
    val absent = codec.decodeFromString<PatchInputDto>("""{"required":null}""")
    check(absent.borrowedWord == Patch.Absent && absent.borrowedTree == Patch.Absent)
    val nil = codec.decodeFromString<PatchInputDto>("""{"required":null,"borrowedWord":null,"borrowedTree":null}""")
    check(nil.borrowedWord == Patch.Null && nil.borrowedTree == Patch.Null)
    val patch = codec.decodeFromString<PatchInputDto>("""{"required":null,"borrowedWord":"future","borrowedTree":{"new":{"a":"future"}}}""")
    check(patch.borrowedWord == Patch.Value(HolderDto.State.Unrecognized("future")))
    check(codec.decodeFromJsonElement(PatchInputDto.serializer(), codec.encodeToJsonElement(PatchInputDto.serializer(), patch)) == patch)
    if (checked) {
        for ((bad, rule) in invalid) {
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, borrowedWord = HolderDto.State.fromSerialName(bad)) }
            stringEnumBoundsReject(rule) { HolderDto(required = value.required, borrowedTree = mapOf("a" to mapOf("b" to HolderDto.BorrowableTreeValueValue.fromSerialName(bad)))) }
            stringEnumBoundsReject(rule) { PatchInputDto(required = null, borrowedWord = Patch.Value(HolderDto.State.fromSerialName(bad))) }
            stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>("""{"required":["two"],"borrowedWord":"$bad"}""") }
            stringEnumBoundsReject(rule) { codec.decodeFromString<HolderDto>("""{"required":["two"],"borrowedTree":{"a":{"b":"$bad"}}}""") }
        }
        stringEnumBoundsReject("uniqueItems") { HolderDto(required = value.required, borrowedRows = listOf(HolderDto.State.ONE, HolderDto.State.Unrecognized("one"))) }
    }
    var wordPayload = "\"one\""
    var treePayload = """{"wire":{"a":"one"}}"""
    var failing = false
    var calls = 0
    val expectedTree: BorrowedTreeDto = mapOf("wire" to mapOf("a" to HolderDto.BorrowableTreeValueValue.ONE))
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val engine = StringEnumBoundsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/borrowed-word" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(serializer<BorrowedWordDto>(), HolderDto.State.ONE))
                respond(wordPayload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/borrowed-tree" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(serializer<BorrowedTreeDto>(), expectedTree))
                respond(treePayload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/borrowed-events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val frame = "event: borrowed\r\ndata: $treePayload\r\n\r\n"
                val holdOpen = failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected borrowed enum path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StringEnumBounds(http, "http://example.test/", codec).defaults
        check(api.sendBorrowedWord(HolderDto.State.ONE) == HolderDto.State.ONE)
        check(api.sendBorrowedTree(expectedTree) == expectedTree)
        val beforeStream = calls
        val stream = api.watchBorrowed()
        check(calls == beforeStream)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchBorrowedEventDto.Borrowed).data == expectedTree) }
        wordPayload = "\"future\""; treePayload = """{"new":{"a":"future"}}"""
        check(api.sendBorrowedWord(HolderDto.State.ONE) == HolderDto.State.Unrecognized("future"))
        check(api.sendBorrowedTree(expectedTree)["new"]?.get("a") == HolderDto.BorrowableTreeValueValue.Unrecognized("future"))
        if (checked) {
            val beforeInvalid = calls
            for ((bad, rule) in invalid) {
                stringEnumBoundsReject(rule) { api.sendBorrowedWord(HolderDto.State.fromSerialName(bad)) }
                stringEnumBoundsReject(rule) { api.sendBorrowedTree(mapOf("a" to mapOf("b" to HolderDto.BorrowableTreeValueValue.fromSerialName(bad)))) }
            }
            check(calls == beforeInvalid)
            for ((bad, rule) in invalid) {
                wordPayload = "\"$bad\""; treePayload = """{"bad":{"a":"$bad"}}"""; failing = true
                stringEnumBoundsReject(rule) { api.sendBorrowedWord(HolderDto.State.ONE) }
                stringEnumBoundsReject(rule) { api.sendBorrowedTree(expectedTree) }
                stringEnumBoundsReject(rule) { withTimeout(10000) { api.watchBorrowed().toList() } }
                check(channels.last().isClosedForRead)
                wordPayload = "\"one\""; treePayload = """{"wire":{"a":"one"}}"""; failing = false
                check(api.sendBorrowedWord(HolderDto.State.ONE) == HolderDto.State.ONE)
                check(api.sendBorrowedTree(expectedTree) == expectedTree)
                check((withTimeout(10000) { api.watchBorrowed().toList() }.single() as WatchBorrowedEventDto.Borrowed).data == expectedTree)
            }
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

private suspend fun stringEnumBoundsWire(codec: Json, value: HolderDto, checked: Boolean, invalid: List<Pair<String, String>>) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    var payload = """{"required":["two"]}"""
    var failing = false
    var calls = 0
    var expectedHolder = value
    var expectedPatch = JsonObject(emptyMap())
    var treePayload = """[{"wire":"one"}]"""
    var treeFailing = false
    val expectedTree: BoundedTreeAliasDto = listOf(mapOf("wire" to BoundedTreeDtoInnerValue.ONE))
    val engine = StringEnumBoundsEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), expectedHolder))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/root-tree" -> {
                check(codec.parseToJsonElement(stringEnumBoundsBody(request.body)) == codec.encodeToJsonElement(serializer<BoundedTreeAliasDto>(), expectedTree))
                respond(treePayload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/events", "/tree-events" -> {
                val channel = ByteChannel(autoFlush = true)
                channels += channel
                val tree = request.url.encodedPath == "/tree-events"
                val frame = if (tree) "event: tree\r\ndata: $treePayload\r\n\r\n" else "event: holder\r\ndata: $payload\r\n\r\n"
                val holdOpen = if (tree) treeFailing else failing
                producers.launch { try {
                    for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                    if (holdOpen) awaitCancellation()
                } finally { channel.close() } }
                respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
            }
            else -> error("Unexpected String enum bounds path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = StringEnumBounds(http, "http://example.test/", codec).defaults
        check(api.sendTree(expectedTree) == expectedTree)
        check((withTimeout(10000) { api.watchTree().toList() }.single() as WatchTreeEventDto.Tree).data == expectedTree)
        treePayload = """[null,{"new":"future"}]"""
        check(api.sendTree(expectedTree)[1]?.get("new") == BoundedTreeDtoInnerValue.Unrecognized("future"))
        check((withTimeout(10000) { api.watchTree().toList() }.single() as WatchTreeEventDto.Tree).data[1]?.get("new") == BoundedTreeDtoInnerValue.Unrecognized("future"))
        if (checked) {
            val beforeInvalidTree = calls
            for ((bad, rule) in invalid) {
                stringEnumBoundsReject(rule) { api.sendTree(listOf(mapOf("bad" to BoundedTreeDtoInnerValue.fromSerialName(bad)))) }
            }
            check(calls == beforeInvalidTree)
            for ((bad, rule) in invalid) {
                treePayload = """[{"bad":"$bad"}]"""; treeFailing = true
                stringEnumBoundsReject(rule) { api.sendTree(expectedTree) }
                stringEnumBoundsReject(rule) { withTimeout(10000) { api.watchTree().toList() } }
                check(channels.last().isClosedForRead)
                treePayload = """[{"wire":"one"}]"""; treeFailing = false
                check(api.sendTree(expectedTree) == expectedTree)
                check((withTimeout(10000) { api.watchTree().toList() }.single() as WatchTreeEventDto.Tree).data == expectedTree)
            }
        }
        check(api.send(value).codes == value.codes)
        val scalarChildren = HolderDto(
            required = value.required, scalarRefCounts = listOf(3), scalarRefLabels = mapOf("key" to "sent"),
            inlineNested = listOf(listOf(23)),
            inlineDates = listOf(listOf(LocalDate.parse("2024-05-01"))),
            inlineInstants = listOf(listOf(Instant.parse("2024-05-01T00:00:00Z"))),
            inlineUuids = listOf(listOf(Uuid.parse("123e4567-e89b-12d3-a456-426614174003"))),
            inlineBytes = listOf(listOf(byteArrayOf(6, 7))),
            inlineWords = listOf(listOf(InlineWordValuesDtoInner.Unrecognized("future"))),
            inlineNativeEnums = listOf(listOf(2)),
            inlineBooleans = listOf(listOf(false)),
            referencedWordRows = listOf(mapOf("wire" to listOf(RootWordDto.Unrecognized("future")))),
            referencedNativeRows = mapOf("wire" to listOf(2)),
            referencedBooleanRows = listOf(listOf(false)),
            referencedNestedRows = listOf(listOf(mapOf("wire" to listOf(RootWordDto.Unrecognized("future"))))),
            scalarFactory = ScalarFactoryHolderDto(rows = listOf(12), labels = mapOf("wire" to "sent"), nested = listOf(mapOf("wire" to 18))),
        )
        expectedHolder = scalarChildren
        payload = codec.encodeToString(HolderDto.serializer(), scalarChildren)
        val scalarResponse = api.send(scalarChildren)
        check(scalarResponse.scalarRefCounts == listOf(3) && scalarResponse.scalarRefLabels == mapOf("key" to "sent"))
        check(scalarResponse.scalarFactory?.rows == listOf(12) && scalarResponse.scalarFactory?.labels == mapOf("wire" to "sent") && scalarResponse.scalarFactory?.inline == listOf(6))
        check(scalarResponse.scalarFactory?.bytes?.single()?.contentEquals(byteArrayOf(7)) == true)
        check(scalarResponse.scalarFactory?.nested == listOf(mapOf("wire" to 18)))
        check(scalarResponse.inlineNested == listOf(listOf(23)))
        check(scalarResponse.inlineDates == scalarChildren.inlineDates && scalarResponse.inlineInstants == scalarChildren.inlineInstants)
        check(scalarResponse.inlineUuids == scalarChildren.inlineUuids)
        check(scalarResponse.inlineBytes?.single()?.single()?.contentEquals(byteArrayOf(6, 7)) == true)
        check(scalarResponse.inlineWords?.single()?.single() == InlineWordValuesDtoInner.Unrecognized("future"))
        check(scalarResponse.inlineNativeEnums == listOf(listOf(2)) && scalarResponse.inlineBooleans == listOf(listOf(false)))
        check(scalarResponse.referencedWordRows?.single()?.get("wire")?.single() == RootWordDto.Unrecognized("future"))
        check(scalarResponse.referencedNativeRows == mapOf("wire" to listOf(2)))
        check(scalarResponse.referencedBooleanRows == listOf(listOf(false)))
        check(scalarResponse.referencedNestedRows?.single()?.single()?.get("wire")?.single() == RootWordDto.Unrecognized("future"))
        val scalarEvent = (withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data
        check(scalarEvent.scalarRefCounts == listOf(3) && scalarEvent.scalarRefLabels == mapOf("key" to "sent"))
        check(scalarEvent.scalarFactory?.rows == listOf(12) && scalarEvent.scalarFactory?.labels == mapOf("wire" to "sent") && scalarEvent.scalarFactory?.inline == listOf(6))
        check(scalarEvent.scalarFactory?.bytes?.single()?.contentEquals(byteArrayOf(7)) == true)
        check(scalarEvent.scalarFactory?.nested == listOf(mapOf("wire" to 18)))
        check(scalarEvent.inlineNested == listOf(listOf(23)))
        check(scalarEvent.inlineDates == scalarChildren.inlineDates && scalarEvent.inlineInstants == scalarChildren.inlineInstants)
        check(scalarEvent.inlineUuids == scalarChildren.inlineUuids)
        check(scalarEvent.inlineBytes?.single()?.single()?.contentEquals(byteArrayOf(6, 7)) == true)
        check(scalarEvent.inlineWords?.single()?.single() == InlineWordValuesDtoInner.Unrecognized("future"))
        check(scalarEvent.inlineNativeEnums == listOf(listOf(2)) && scalarEvent.inlineBooleans == listOf(listOf(false)))
        check(scalarEvent.referencedWordRows?.single()?.get("wire")?.single() == RootWordDto.Unrecognized("future"))
        check(scalarEvent.referencedNativeRows == mapOf("wire" to listOf(2)))
        check(scalarEvent.referencedBooleanRows == listOf(listOf(false)))
        check(scalarEvent.referencedNestedRows?.single()?.single()?.get("wire")?.single() == RootWordDto.Unrecognized("future"))
        expectedHolder = value
        payload = """{"required":["two"]}"""
        if (checked) {
            val beforeInvalid = calls
            for ((bad, rule) in invalid) {
                stringEnumBoundsReject(rule) { api.send(HolderDto(required = listOf(HolderDto.RequiredInner.fromSerialName(bad)))) }
            }
            check(calls == beforeInvalid)
        }
        for (patch in listOf(PatchInputDto(required = null), PatchInputDto(required = null, codes = Patch.Null, labels = Patch.Null, bounded = Patch.Null), PatchInputDto(required = null, codes = Patch.Value(listOf(PatchInputDto.CodesInner.Unrecognized("future"))), labels = Patch.Value(mapOf("nil" to null)), bounded = Patch.Value(BoundedValueDto(state = BoundedValueDto.State.TWO))))) {
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        for (tree in listOf(Patch.Null, Patch.Value(listOf(null, mapOf("a" to PatchInputDto.TreesInnerValue.Unrecognized("future")))))) {
            val patch = PatchInputDto(required = null, trees = tree)
            expectedPatch = codec.encodeToJsonElement(PatchInputDto.serializer(), patch) as JsonObject
            api.patchPatch(patch)
        }
        val before = calls
        val stream = api.watch()
        check(calls == before)
        repeat(2) { check((withTimeout(10000) { stream.toList() }.single() as WatchEventDto.Holder).data.codes == value.codes) }
        payload = """{"required":["future"],"codes":[null,"future"],"labels":{"new":"future"}}"""
        check(api.send(value).required.single() == HolderDto.RequiredInner.Unrecognized("future"))
        expectedHolder = codec.decodeFromString<HolderDto>(payload)
        check(api.send(expectedHolder) == expectedHolder)
        expectedHolder = value
        check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.labels?.get("new") == HolderDto.LabelsValue.Unrecognized("future"))
        val failures = mutableListOf<Pair<String, String?>>("""{"required":[null]}""" to null, """{"required":["two"],"labels":{"bad":1}}""" to null)
        if (checked) {
            for ((bad, rule) in invalid) {
                failures += """{"required":["$bad"]}""" to rule
                failures += """{"required":["two"],"labels":{"a":"$bad"}}""" to rule
                failures += """{"required":["two"],"state":"$bad"}""" to rule
                failures += """{"required":["two"],"metadataState":"$bad"}""" to rule
                failures += """{"required":["two"],"bounded":{"state":"$bad"}}""" to rule
                failures += """{"required":["two"],"trees":[null,{"a":"$bad"}]}""" to rule
            }
            failures += """{"required":[]}""" to "minItems"
            failures += """{"required":["two"],"codes":["one","one","one","one"]}""" to "maxItems"
            failures += """{"required":["two"],"trees":[{"a":"one"},{"a":"one"}]}""" to "uniqueItems"
        }
        for ((bad, rule) in failures) {
            payload = bad; failing = true
            stringEnumBoundsReject(rule) { api.send(value) }
            stringEnumBoundsReject(rule) { withTimeout(10000) { api.watch().toList() } }
            check(channels.last().isClosedForRead)
            payload = """{"required":["two"]}"""; failing = false
            check(api.send(value).codes == value.codes)
            check((withTimeout(10000) { api.watch().toList() }.single() as WatchEventDto.Holder).data.codes == value.codes)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
}

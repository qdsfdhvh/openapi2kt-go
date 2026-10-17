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
import org.example.nestedobject.NestedObject
import org.example.nestedobject.api.models.PatchHolderInputDto
import org.example.nestedobject.api.models.WatchHoldersEventDto
import org.example.nestedobject.models.HolderDto
import org.example.nestedobject.models.Patch
import org.example.nestedobject.models.PatchInputDto
import org.example.nestedobject.models.RequiredNullableCfgDto

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class NestedObjectEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private suspend fun nestedObjectReject(rule: String? = null, block: suspend () -> Unit) {
    val failure = try { block(); null } catch (cause: CancellationException) { throw cause } catch (cause: Throwable) { cause }
    check(failure != null) { "Expected failure: $rule" }
    if (rule != null) check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Wrong $rule failure: $failure" }
}

private fun nestedObjectBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> nestedObjectBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected object default request body $body")
}

private fun nestedObjectCfg(value: HolderDto.Cfg?): Map<String, Any?> = linkedMapOf(
    "name" to value?.name, "age" to value?.age, "status" to value?.status,
    "code" to value?.code, "when" to value?.`when`, "tags" to value?.tags,
    "counts" to value?.counts, "child" to value?.child, "childFlag" to value?.child?.flag,
    "leaf" to value?.leaf?.z, "leafAlias" to value?.leafAlias?.z, "unionProp" to value?.unionProp, "timeRows" to value?.timeRows, "note" to value?.note, "composed" to value?.composed, "linked" to value?.linked,
    "viaBorrowedRows" to value?.viaBorrowedRows, "viaBorrowedNull" to value?.viaBorrowedNull,
    "viaBorrowedMap" to value?.viaBorrowedMap, "viaBorrowedGrid" to value?.viaBorrowedGrid,
    "viaInternalRows" to value?.viaInternalRows, "viaInternalNull" to value?.viaInternalNull,
    "viaInternalMap" to value?.viaInternalMap, "viaInternalGrid" to value?.viaInternalGrid,
    "viaCollectionRows" to value?.viaCollectionRows, "viaCollectionRowsNull" to value?.viaCollectionRowsNull,
    "viaCollectionMap" to value?.viaCollectionMap, "viaCollectionGrid" to value?.viaCollectionGrid,
    "namedRows" to value?.namedRows, "namedMap" to value?.namedMap, "namedMapValues" to value?.namedMapValues, "namedGrid" to value?.namedGrid,
    "plainDtoRows" to value?.plainDtoRows, "plainDtoAliasRows" to value?.plainDtoAliasRows, "plainDtoMap" to value?.plainDtoMap,
    "plainDtoInternalRows" to value?.plainDtoInternalRows, "plainDtoInlineRows" to value?.plainDtoInlineRows,
    "plainDtoRowsAlias" to value?.plainDtoRowsAlias, "plainDtoBridge" to value?.plainDtoBridge, "plainDtoEntityAlias" to value?.plainDtoEntityAlias,
    "plainDtoNestedRows" to value?.plainDtoNestedRows, "plainDtoNestedMixed" to value?.plainDtoNestedMixed, "plainDtoNestedMapList" to value?.plainDtoNestedMapList,
    "memberBareRows" to value?.memberBareRows, "memberBareRowsNull" to value?.memberBareRowsNull,
    "memberBareMap" to value?.memberBareMap, "memberBareGrid" to value?.memberBareGrid,
    "memberInternalRows" to value?.memberInternalRows, "memberInternalNull" to value?.memberInternalNull,
    "memberInternalMap" to value?.memberInternalMap,
    "refRows" to value?.refRows, "refMap" to value?.refMap, "refGrid" to value?.refGrid, "refVariants" to value?.refVariants,
    "memberRows" to value?.memberRows, "memberMap" to value?.memberMap, "memberGrid" to value?.memberGrid,
    "viaAlias" to value?.viaAlias, "aliasNull" to value?.aliasNull,
    "viaChain" to value?.viaChain, "viaChainNull" to value?.viaChainNull, "viaNamedChain" to value?.viaNamedChain,
    "owned" to value?.owned, "ownedAlias" to value?.ownedAlias, "ownedRow" to value?.ownedRow, "ownedVariant" to value?.ownedVariant,
)

private fun nestedObjectHolderFields(value: HolderDto): Map<String, Any?> = linkedMapOf(
    "cfg" to nestedObjectCfg(value.cfg),
    "requiredCfg" to value.requiredCfg.id,
    "allOfCfg" to value.allOfCfg,
    "allOfNull" to value.allOfNull,
    "rows" to value.rows,
)

private fun nestedObjectFresh(left: Any?, right: Any?) {
    check(left == right)
    when (left) {
        is List<*> -> { check(left !== right); right as List<*>; left.indices.forEach { nestedObjectFresh(left[it], right[it]) } }
        is Map<*, *> -> { check(left !== right); right as Map<*, *>; left.keys.forEach { nestedObjectFresh(left[it], right[it]) } }
    }
}

suspend fun nestedObjectDefaultsContracts(checked: Boolean) {
    val value = HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x"))
    check(nestedObjectCfg(value.cfg) == nestedObjectCfg(HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x")).cfg))
    check(value.cfg?.name == "outer" && value.cfg?.age == 9)
    check(value.cfg?.status == HolderDto.Cfg.Status.ON && value.cfg?.code == 2)
    check(value.cfg?.`when`?.toString() == "2024-01-02")
    check(value.cfg?.tags == listOf("a", "b") && value.cfg?.counts == mapOf("k" to 1))
    check(value.cfg?.child?.flag == false && value.cfg?.note == null)
    check(value.cfg?.leaf?.z == "deep" && value.cfg?.leafAlias?.z == "aliased")
    check(value.cfg?.unionProp != null)
    check(value.cfg?.timeRows?.single()?.toString() == "2024-01-02T03:04:05Z")
    // A component-level default is an opt-in factory, not a constructor initializer.
    check(org.example.nestedobject.models.ComponentDefaultDto.default().label == "component")
    // Scalar and enum components own their value, so their factory returns the
    // alias type directly; the scalar form is a top-level function.
    check(org.example.nestedobject.models.`ComponentDefaultNameDto default`() == "component-name")
    check(org.example.nestedobject.models.`ComponentDefaultModeDto default`() == org.example.nestedobject.models.ComponentDefaultModeDto.ON)
    check(org.example.nestedobject.models.ComponentDefaultAllOfDto.default().n == 9)
    check(org.example.nestedobject.models.`ComponentDefaultNamesDto default`() == listOf("component-name"))
    check(org.example.nestedobject.models.`ComponentDefaultDatesDto default`().single().toString() == "2024-02-29")
    check(org.example.nestedobject.models.`ComponentDefaultWhenDto default`().toString() == "2024-01-02T03:04:05Z")
    check(value.cfg?.composed?.label == "outer member" && value.cfg?.composed?.count == 3 && value.cfg?.composed?.child?.flag == false)
    check(value.cfg?.linked?.label == "outer linked" && value.cfg?.linked?.count == 2 && value.cfg?.linked?.child?.flag == true)
    check(value.cfg?.viaAlias?.label == "outer alias" && value.cfg?.viaAlias?.count == 2 && value.cfg?.viaAlias?.child?.flag == true)
    check(value.cfg?.aliasNull == null)
    check(value.cfg?.owned?.label == "outer owned" && value.cfg?.owned?.count == 2 && value.cfg?.owned?.child?.flag == true)
    check(value.cfg?.ownedAlias == null)
    check(value.cfg?.ownedRow?.label == "outer row" && value.cfg?.ownedRow?.count == 2)
    check(value.cfg?.ownedVariant?.label == "outer variant" && value.cfg?.ownedVariant?.count == 2)
    check(value.cfg?.viaChain?.label == "outer chain" && value.cfg?.viaChain?.count == 2 && value.cfg?.viaChain?.child?.flag == true)
    check(value.cfg?.viaChainNull == null && value.cfg?.viaNamedChain?.label == "outer named chain")
    check(value.cfg?.memberRows?.single()?.label == "outer rows" && value.cfg?.memberRows?.single()?.count == 2 && value.cfg?.memberRows?.single()?.child?.flag == true)
    check(value.cfg?.memberMap?.get("value")?.label == "outer map" && value.cfg?.memberMap?.get("none") == null)
    check(value.cfg?.memberGrid?.single()?.get("k")?.label == "outer grid" && value.cfg?.memberGrid?.single()?.get("k")?.count == 2)
    check(value.cfg?.refRows?.single()?.label == "outer ref rows" && value.cfg?.refRows?.single()?.child?.flag == true)
    check(value.cfg?.refMap?.get("value")?.label == "outer ref map" && value.cfg?.refMap?.get("none") == null)
    check(value.cfg?.refGrid?.single()?.get("k")?.label == "outer ref grid" && value.cfg?.refGrid?.single()?.get("k")?.count == 2)
    check(value.cfg?.refVariants?.single()?.label == "outer ref variant" && value.cfg?.refVariants?.single()?.count == 2)
    check(value.cfg?.namedRows?.single()?.label == "outer named rows" && value.cfg?.namedRows?.single()?.child?.flag == true)
    check(value.cfg?.namedMap == null && value.cfg?.namedMapValues?.get("value")?.label == "outer named map" && value.cfg?.namedMapValues?.get("none") == null)
    check(value.cfg?.namedGrid?.single()?.get("k")?.label == "outer named grid")
    check(value.cfg?.plainDtoRows?.single()?.label == "outer plain rows" && value.cfg?.plainDtoRows?.single()?.count == 2)
    check(value.cfg?.plainDtoAliasRows?.single()?.label == "outer plain alias rows")
    check(value.cfg?.plainDtoMap?.get("value")?.label == "outer plain map" && value.cfg?.plainDtoMap?.get("value")?.count == 2)
    check(value.cfg?.plainDtoInternalRows?.single()?.label == "outer plain internal rows")
    check(value.cfg?.plainDtoInlineRows?.single()?.label == "outer plain inline rows")
    check(value.cfg?.plainDtoRowsAlias?.single()?.label == "outer plain alias-chain rows" && value.cfg?.plainDtoRowsAlias?.single()?.count == 2)
    check(value.cfg?.plainDtoBridge?.single()?.label == "outer plain bridge rows" && value.cfg?.plainDtoBridge?.single()?.count == 2)
    check(value.cfg?.plainDtoEntityAlias?.single()?.label == "outer plain entity alias rows")
    check(value.cfg?.plainDtoNestedRows?.single()?.single()?.label == "outer plain nested rows" && value.cfg?.plainDtoNestedRows?.single()?.single()?.count == 2)
    check(value.cfg?.plainDtoNestedMixed?.single()?.get("k")?.label == "outer plain nested mixed")
    check(value.cfg?.plainDtoNestedMapList?.get("k")?.single()?.label == "outer plain nested map list")
    check(value.cfg?.memberBareRows?.single()?.label == "outer bare rows" && value.cfg?.memberBareRows?.single()?.child?.flag == true)
    check(value.cfg?.memberBareRowsNull == null && value.cfg?.memberBareMap?.get("value")?.label == "outer bare map" && value.cfg?.memberBareMap?.get("none") == null)
    check(value.cfg?.memberBareGrid?.single()?.get("k")?.label == "outer bare grid")
    check(value.cfg?.memberInternalRows?.single()?.label == "outer internal rows" && value.cfg?.memberInternalRows?.single()?.child?.flag == true)
    check(value.cfg?.memberInternalNull == null && value.cfg?.memberInternalMap?.get("value")?.label == "outer internal map" && value.cfg?.memberInternalMap?.get("none") == null)
    check(value.cfg?.viaCollectionRows?.single()?.label == "outer bridge rows" && value.cfg?.viaCollectionRows?.single()?.child?.flag == true)
    check(value.cfg?.viaCollectionRowsNull == null && value.cfg?.viaCollectionMap?.get("value")?.label == "outer bridge map" && value.cfg?.viaCollectionMap?.get("none") == null)
    check(value.cfg?.viaCollectionGrid?.single()?.get("k")?.label == "outer bridge grid")
    check(value.cfg?.viaInternalRows?.single()?.label == "outer internal rows" && value.cfg?.viaInternalRows?.single()?.child?.flag == true)
    check(value.cfg?.viaInternalNull == null && value.cfg?.viaInternalMap?.get("value")?.label == "outer internal map" && value.cfg?.viaInternalMap?.get("none") == null)
    check(value.cfg?.viaInternalGrid?.single()?.get("k")?.label == "outer internal grid")
    check(value.cfg?.viaBorrowedRows?.single()?.label == "outer borrowed rows" && value.cfg?.viaBorrowedRows?.single()?.child?.flag == true)
    check(value.cfg?.viaBorrowedNull == null && value.cfg?.viaBorrowedMap?.get("value")?.label == "outer borrowed map" && value.cfg?.viaBorrowedMap?.get("none") == null)
    check(value.cfg?.viaBorrowedGrid?.single()?.get("k")?.label == "outer borrowed grid")
    check(value.requiredCfg.id == "x")
    check(value.allOfCfg?.z == "merged" && value.allOfCfg?.count == 5 && value.allOfCfg?.meta?.flag == true)
    check(value.allOfNull == null)
    val other = HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x"))
    check(value.allOfCfg == other.allOfCfg && value.allOfCfg !== other.allOfCfg)
    check(value.allOfCfg?.meta !== other.allOfCfg?.meta)
    check(value.cfg?.composed == other.cfg?.composed && value.cfg?.composed !== other.cfg?.composed)
    check(value.cfg?.composed?.child !== other.cfg?.composed?.child)
    check(value.cfg?.linked == other.cfg?.linked && value.cfg?.linked !== other.cfg?.linked)
    check(value.cfg?.linked?.child !== other.cfg?.linked?.child)
    check(value.cfg?.viaAlias == other.cfg?.viaAlias && value.cfg?.viaAlias !== other.cfg?.viaAlias)
    check(value.cfg?.viaAlias?.child !== other.cfg?.viaAlias?.child)
    check(value.cfg?.viaChain == other.cfg?.viaChain && value.cfg?.viaChain !== other.cfg?.viaChain)
    check(value.cfg?.viaChain?.child !== other.cfg?.viaChain?.child)
    check(value.cfg?.viaNamedChain == other.cfg?.viaNamedChain && value.cfg?.viaNamedChain !== other.cfg?.viaNamedChain)
    check(value.cfg?.owned == other.cfg?.owned && value.cfg?.owned !== other.cfg?.owned)
    check(value.cfg?.owned?.child !== other.cfg?.owned?.child)
    check(value.cfg?.ownedRow == other.cfg?.ownedRow && value.cfg?.ownedRow !== other.cfg?.ownedRow)
    check(value.cfg?.ownedVariant == other.cfg?.ownedVariant && value.cfg?.ownedVariant !== other.cfg?.ownedVariant)
    check(value.cfg?.memberRows?.single() !== other.cfg?.memberRows?.single())
    check(value.cfg?.memberRows?.single()?.child !== other.cfg?.memberRows?.single()?.child)
    check(value.cfg?.memberMap?.get("value") !== other.cfg?.memberMap?.get("value"))
    check(value.cfg?.memberGrid?.single()?.get("k") !== other.cfg?.memberGrid?.single()?.get("k"))
    check(value.cfg?.refRows?.single() !== other.cfg?.refRows?.single())
    check(value.cfg?.refRows?.single()?.child !== other.cfg?.refRows?.single()?.child)
    check(value.cfg?.refMap?.get("value") !== other.cfg?.refMap?.get("value"))
    check(value.cfg?.refGrid?.single()?.get("k") !== other.cfg?.refGrid?.single()?.get("k"))
    check(value.cfg?.refVariants?.single() !== other.cfg?.refVariants?.single())
    check(value.cfg?.namedRows?.single() !== other.cfg?.namedRows?.single())
    check(value.cfg?.namedRows?.single()?.child !== other.cfg?.namedRows?.single()?.child)
    check(value.cfg?.namedMapValues?.get("value") !== other.cfg?.namedMapValues?.get("value"))
    check(value.cfg?.namedGrid?.single()?.get("k") !== other.cfg?.namedGrid?.single()?.get("k"))
    check(value.cfg?.plainDtoRows?.single() !== other.cfg?.plainDtoRows?.single())
    check(value.cfg?.plainDtoAliasRows?.single() !== other.cfg?.plainDtoAliasRows?.single())
    check(value.cfg?.plainDtoMap?.get("value") !== other.cfg?.plainDtoMap?.get("value"))
    check(value.cfg?.plainDtoInternalRows?.single() !== other.cfg?.plainDtoInternalRows?.single())
    check(value.cfg?.plainDtoInlineRows?.single() !== other.cfg?.plainDtoInlineRows?.single())
    check(value.cfg?.plainDtoRowsAlias?.single() !== other.cfg?.plainDtoRowsAlias?.single())
    check(value.cfg?.plainDtoBridge?.single() !== other.cfg?.plainDtoBridge?.single())
    check(value.cfg?.plainDtoEntityAlias?.single() !== other.cfg?.plainDtoEntityAlias?.single())
    check(value.cfg?.plainDtoNestedRows?.single()?.single() !== other.cfg?.plainDtoNestedRows?.single()?.single())
    check(value.cfg?.plainDtoNestedMixed?.single()?.get("k") !== other.cfg?.plainDtoNestedMixed?.single()?.get("k"))
    check(value.cfg?.plainDtoNestedMapList?.get("k")?.single() !== other.cfg?.plainDtoNestedMapList?.get("k")?.single())
    check(value.cfg?.memberBareRows?.single() !== other.cfg?.memberBareRows?.single())
    check(value.cfg?.memberBareRows?.single()?.child !== other.cfg?.memberBareRows?.single()?.child)
    check(value.cfg?.memberBareMap?.get("value") !== other.cfg?.memberBareMap?.get("value"))
    check(value.cfg?.memberBareGrid?.single()?.get("k") !== other.cfg?.memberBareGrid?.single()?.get("k"))
    check(value.cfg?.memberInternalRows?.single() !== other.cfg?.memberInternalRows?.single())
    check(value.cfg?.memberInternalRows?.single()?.child !== other.cfg?.memberInternalRows?.single()?.child)
    check(value.cfg?.memberInternalMap?.get("value") !== other.cfg?.memberInternalMap?.get("value"))
    check(value.cfg?.viaCollectionRows?.single() !== other.cfg?.viaCollectionRows?.single())
    check(value.cfg?.viaCollectionRows?.single()?.child !== other.cfg?.viaCollectionRows?.single()?.child)
    check(value.cfg?.viaCollectionMap?.get("value") !== other.cfg?.viaCollectionMap?.get("value"))
    check(value.cfg?.viaCollectionGrid?.single()?.get("k") !== other.cfg?.viaCollectionGrid?.single()?.get("k"))
    check(value.cfg?.viaInternalRows !== other.cfg?.viaInternalRows && value.cfg?.viaInternalMap !== other.cfg?.viaInternalMap)
    check(value.cfg?.viaInternalRows?.single() !== other.cfg?.viaInternalRows?.single())
    check(value.cfg?.viaInternalRows?.single()?.child !== other.cfg?.viaInternalRows?.single()?.child)
    check(value.cfg?.viaInternalMap?.get("value") !== other.cfg?.viaInternalMap?.get("value"))
    check(value.cfg?.viaInternalGrid?.single()?.get("k") !== other.cfg?.viaInternalGrid?.single()?.get("k"))
    check(value.cfg?.viaBorrowedRows !== other.cfg?.viaBorrowedRows && value.cfg?.viaBorrowedMap !== other.cfg?.viaBorrowedMap)
    check(value.cfg?.viaBorrowedRows?.single() !== other.cfg?.viaBorrowedRows?.single())
    check(value.cfg?.viaBorrowedRows?.single()?.child !== other.cfg?.viaBorrowedRows?.single()?.child)
    check(value.cfg?.viaBorrowedMap?.get("value") !== other.cfg?.viaBorrowedMap?.get("value"))
    check(value.cfg?.viaBorrowedGrid?.single()?.get("k") !== other.cfg?.viaBorrowedGrid?.single()?.get("k"))
    // A required object default keeps its constructor argument and no initializer.
    check(!HolderDto.serializer().descriptor.isElementOptional(1))
    // Fresh nested trees per instance, not one shared mutable default.
    nestedObjectFresh(nestedObjectCfg(HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x")).cfg), nestedObjectCfg(value.cfg))
    // Nested property own defaults apply when the outer default omits them.
    val fresh = HolderDto.Cfg(name = "n")
    check(fresh.age == 7 && fresh.child?.flag == true && fresh.status == null && fresh.code == null)
    check(fresh.composed?.label == "member" && fresh.composed?.count == 2 && fresh.composed?.child?.flag == true)
    check(fresh.linked?.label == "linked member" && fresh.linked?.count == 2 && fresh.linked?.child?.flag == true)
    check(fresh.viaAlias?.label == "alias member" && fresh.viaAlias?.count == 2 && fresh.viaAlias?.child?.flag == true)
    check(fresh.aliasNull == null)
    check(fresh.viaChain?.label == "chain member" && fresh.viaChain?.count == 2 && fresh.viaChain?.child?.flag == true)
    check(fresh.viaChainNull == null && fresh.viaNamedChain?.label == "named chain member")
    check(fresh.owned?.label == "owned member" && fresh.owned?.count == 2 && fresh.owned?.child?.flag == true)
    check(fresh.ownedAlias == null && fresh.ownedRow?.label == "row member" && fresh.ownedVariant?.label == "variant member")
    check(fresh.memberRows?.single()?.label == "rows member" && fresh.memberRows?.single()?.child?.flag == true)
    check(fresh.memberMap?.get("value")?.label == "map member" && fresh.memberMap?.containsKey("none") == true && fresh.memberMap?.get("none") == null)
    check(fresh.memberGrid?.single()?.get("k")?.label == "grid member")
    check(fresh.refRows?.single()?.label == "ref rows member" && fresh.refRows?.single()?.child?.flag == true)
    check(fresh.refMap?.get("value")?.label == "ref map member" && fresh.refMap?.get("none") == null)
    check(fresh.refGrid?.single()?.get("k")?.label == "ref grid member" && fresh.refVariants?.single()?.label == "ref variant member")
    check(fresh.namedRows?.single()?.label == "named rows member" && fresh.namedMap == null)
    check(fresh.namedMapValues?.get("value")?.label == "named map member" && fresh.namedGrid?.single()?.get("k")?.label == "named grid member")
    check(fresh.plainDtoRows?.single()?.label == "plain rows member" && fresh.plainDtoRows?.single()?.count == 2)
    check(fresh.plainDtoAliasRows?.single()?.label == "plain alias rows member")
    check(fresh.plainDtoMap?.get("value")?.label == "plain map member" && fresh.plainDtoInternalRows?.single()?.label == "plain internal rows member")
    check(fresh.plainDtoInlineRows?.single()?.label == "plain inline rows member")
    check(fresh.plainDtoRowsAlias?.single()?.label == "plain alias-chain rows member" && fresh.plainDtoRowsAlias?.single()?.count == 2)
    check(fresh.plainDtoBridge?.single()?.label == "plain bridge rows member")
    check(fresh.plainDtoEntityAlias?.single()?.label == "plain entity alias rows member")
    check(fresh.plainDtoNestedRows?.single()?.single()?.label == "plain nested rows member" && fresh.plainDtoNestedRows?.single()?.single()?.count == 2)
    check(fresh.plainDtoNestedMixed?.single()?.get("k")?.label == "plain nested mixed member")
    check(fresh.plainDtoNestedMapList?.get("k")?.single()?.label == "plain nested map list member")
    check(fresh.memberBareRows?.single()?.label == "bare rows member" && fresh.memberBareRows?.single()?.child?.flag == true && fresh.memberBareRowsNull == null)
    check(fresh.memberBareMap?.get("value")?.label == "bare map member" && fresh.memberBareGrid?.single()?.get("k")?.label == "bare grid member")
    check(fresh.memberInternalRows?.single()?.label == "internal rows member" && fresh.memberInternalRows?.single()?.child?.flag == true && fresh.memberInternalNull == null)
    check(fresh.memberInternalMap?.get("value")?.label == "internal map member")
    check(fresh.viaCollectionRows?.single()?.label == "bridge rows member" && fresh.viaCollectionRowsNull == null)
    check(fresh.viaCollectionMap?.get("value")?.label == "bridge map member" && fresh.viaCollectionGrid?.single()?.get("k")?.label == "bridge grid member")
    check(fresh.viaInternalRows?.single()?.label == "internal rows member" && fresh.viaInternalNull == null)
    check(fresh.viaInternalMap?.get("value")?.label == "internal map member" && fresh.viaInternalGrid?.single()?.get("k")?.label == "internal grid member")
    check(fresh.viaBorrowedRows?.single()?.label == "borrowed rows member" && fresh.viaBorrowedNull == null)
    check(fresh.viaBorrowedMap?.get("value")?.label == "borrowed map member" && fresh.viaBorrowedGrid?.single()?.get("k")?.label == "borrowed grid member")
    if (checked) {
        nestedObjectReject("enum") { HolderDto(requiredCfg = HolderDto.RequiredCfg(id = "x"), cfg = HolderDto.Cfg(name = "n", code = 9)) }
    } else {
        check(HolderDto.Cfg(name = "n", code = 9).code == 9)
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        // requiredCfg is required: a missing key fails regardless of Json settings.
        check(runCatching { codec.decodeFromString<HolderDto>("{}") }.isFailure)
        val missing = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"}}""")
        check(nestedObjectHolderFields(missing) == nestedObjectHolderFields(value))
        val overridden = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"y"},"cfg":{"name":"z"}}""")
        check(overridden.requiredCfg.id == "y" && overridden.cfg?.name == "z" && overridden.cfg?.age == 7 && overridden.cfg?.child?.flag == true)
        check(overridden.cfg?.composed?.label == "member" && overridden.cfg?.composed?.count == 2)
        check(overridden.cfg?.linked?.label == "linked member" && overridden.cfg?.linked?.count == 2)
        check(overridden.cfg?.viaAlias?.label == "alias member" && overridden.cfg?.viaAlias?.count == 2 && overridden.cfg?.aliasNull == null)
        check(overridden.cfg?.viaChain?.label == "chain member" && overridden.cfg?.viaNamedChain?.label == "named chain member")
        check(overridden.cfg?.memberRows?.single()?.label == "rows member" && overridden.cfg?.memberMap?.get("value")?.label == "map member")
        val collections = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberRows":[{"label":"replacement"}],"memberMap":{"nil":null,"new":{"label":"map"}},"memberGrid":[{"new":{"label":"grid"}}]}}""")
        check(collections.cfg?.memberRows?.single()?.label == "replacement" && collections.cfg?.memberRows?.single()?.child?.flag == true)
        check(collections.cfg?.memberMap?.keys == setOf("nil", "new") && collections.cfg?.memberMap?.get("nil") == null && collections.cfg?.memberMap?.get("new")?.count == 2)
        check(collections.cfg?.memberGrid?.single()?.keys == setOf("new") && collections.cfg?.memberGrid?.single()?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), collections))) == nestedObjectHolderFields(collections))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberRows":null}}""").cfg?.memberRows == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberRows":[]}}""").cfg?.memberRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberRows":[null]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberGrid":[{"k":{}}]}}""") }.isFailure)
        check(overridden.cfg?.refRows?.single()?.label == "ref rows member" && overridden.cfg?.refMap?.get("value")?.label == "ref map member")
        val references = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","refRows":[{"label":"replacement"}],"refMap":{"nil":null,"new":{"label":"map"}},"refGrid":[{"new":{"label":"grid"}}],"refVariants":[{"kind":"all","label":"variant"}]}}""")
        check(references.cfg?.refRows?.single()?.label == "replacement" && references.cfg?.refRows?.single()?.child?.flag == true)
        check(references.cfg?.refMap?.keys == setOf("nil", "new") && references.cfg?.refMap?.get("nil") == null && references.cfg?.refMap?.get("new")?.count == 2)
        check(references.cfg?.refGrid?.single()?.keys == setOf("new") && references.cfg?.refGrid?.single()?.get("new")?.count == 2)
        check(references.cfg?.refVariants?.single()?.label == "variant")
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), references))) == nestedObjectHolderFields(references))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","refRows":null}}""").cfg?.refRows == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","refRows":[]}}""").cfg?.refRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","refRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","refRows":[null]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","refGrid":[{"k":{}}]}}""") }.isFailure)
        check(overridden.cfg?.namedRows?.single()?.label == "named rows member" && overridden.cfg?.namedMapValues?.get("value")?.label == "named map member")
        val named = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","namedRows":[{"label":"replacement"}],"namedMap":{"nil":null,"new":{"label":"map"}},"namedGrid":[{"new":{"label":"grid"}}]}}""")
        check(named.cfg?.namedRows?.single()?.label == "replacement" && named.cfg?.namedRows?.single()?.child?.flag == true)
        check(named.cfg?.namedMap?.keys == setOf("nil", "new") && named.cfg?.namedMap?.get("nil") == null && named.cfg?.namedMap?.get("new")?.count == 2)
        check(named.cfg?.namedGrid?.single()?.keys == setOf("new") && named.cfg?.namedGrid?.single()?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), named))) == nestedObjectHolderFields(named))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","namedRows":[]}}""").cfg?.namedRows?.isEmpty() == true)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","namedMap":null}}""").cfg?.namedMap == null)
        check(overridden.cfg?.memberBareRows?.single()?.label == "bare rows member" && overridden.cfg?.memberBareMap?.get("value")?.label == "bare map member")
        val bare = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberBareRows":[{"label":"replacement"}],"memberBareRowsNull":[{"label":"set"}],"memberBareMap":{"nil":null,"new":{"label":"map"}},"memberBareGrid":[{"new":{"label":"grid"}}]}}""")
        check(bare.cfg?.memberBareRows?.single()?.label == "replacement" && bare.cfg?.memberBareRows?.single()?.child?.flag == true)
        check(bare.cfg?.memberBareRowsNull?.single()?.label == "set")
        check(bare.cfg?.memberBareMap?.keys == setOf("nil", "new") && bare.cfg?.memberBareMap?.get("nil") == null && bare.cfg?.memberBareMap?.get("new")?.count == 2)
        check(bare.cfg?.memberBareGrid?.single()?.keys == setOf("new") && bare.cfg?.memberBareGrid?.single()?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), bare))) == nestedObjectHolderFields(bare))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberBareRowsNull":null}}""").cfg?.memberBareRowsNull == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberBareRows":[]}}""").cfg?.memberBareRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberBareRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberBareRows":[null]}}""") }.isFailure)
        check(overridden.cfg?.memberInternalRows?.single()?.label == "internal rows member" && overridden.cfg?.memberInternalMap?.get("value")?.label == "internal map member")
        val internalMember = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberInternalRows":[{"label":"replacement"}],"memberInternalNull":[{"label":"set"}],"memberInternalMap":{"nil":null,"new":{"label":"map"}}}}""")
        check(internalMember.cfg?.memberInternalRows?.single()?.label == "replacement" && internalMember.cfg?.memberInternalRows?.single()?.child?.flag == true)
        check(internalMember.cfg?.memberInternalNull?.single()?.label == "set")
        check(internalMember.cfg?.memberInternalMap?.keys == setOf("nil", "new") && internalMember.cfg?.memberInternalMap?.get("nil") == null && internalMember.cfg?.memberInternalMap?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), internalMember))) == nestedObjectHolderFields(internalMember))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberInternalNull":null}}""").cfg?.memberInternalNull == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberInternalRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","memberInternalRows":[null]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","namedRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","namedRows":[null]}}""") }.isFailure)
        check(overridden.cfg?.plainDtoRows?.single()?.label == "plain rows member" && overridden.cfg?.plainDtoMap?.get("value")?.label == "plain map member")
        val plain = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRows":[{"label":"replacement"}],"plainDtoAliasRows":[{"label":"alias replacement"}],"plainDtoMap":{"new":{"label":"map"}},"plainDtoInternalRows":[{"label":"internal replacement"}],"plainDtoInlineRows":[{"label":"inline replacement"}]}}""")
        check(plain.cfg?.plainDtoRows?.single()?.label == "replacement" && plain.cfg?.plainDtoRows?.single()?.count == 2)
        check(plain.cfg?.plainDtoAliasRows?.single()?.label == "alias replacement" && plain.cfg?.plainDtoMap?.keys == setOf("new"))
        check(plain.cfg?.plainDtoMap?.get("new")?.count == 2 && plain.cfg?.plainDtoInternalRows?.single()?.label == "internal replacement")
        check(plain.cfg?.plainDtoInlineRows?.single()?.label == "inline replacement")
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), plain))) == nestedObjectHolderFields(plain))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRows":null}}""").cfg?.plainDtoRows == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRows":[]}}""").cfg?.plainDtoRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRows":[null]}}""") }.isFailure)
        val plainAlias = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRowsAlias":[{"label":"alias-chain replacement"}],"plainDtoBridge":[{"label":"bridge replacement"}],"plainDtoEntityAlias":[{"label":"entity alias replacement"}]}}""")
        check(plainAlias.cfg?.plainDtoRowsAlias?.single()?.label == "alias-chain replacement" && plainAlias.cfg?.plainDtoRowsAlias?.single()?.count == 2)
        check(plainAlias.cfg?.plainDtoBridge?.single()?.label == "bridge replacement" && plainAlias.cfg?.plainDtoEntityAlias?.single()?.label == "entity alias replacement")
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), plainAlias))) == nestedObjectHolderFields(plainAlias))
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoRowsAlias":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoBridge":[null]}}""") }.isFailure)
        val plainNested = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoNestedRows":[[{"label":"nested replacement"}]],"plainDtoNestedMixed":[{"k":{"label":"mixed replacement"}}],"plainDtoNestedMapList":{"k":[{"label":"map list replacement"}]}}}""")
        check(plainNested.cfg?.plainDtoNestedRows?.single()?.single()?.label == "nested replacement" && plainNested.cfg?.plainDtoNestedRows?.single()?.single()?.count == 2)
        check(plainNested.cfg?.plainDtoNestedMixed?.single()?.get("k")?.label == "mixed replacement" && plainNested.cfg?.plainDtoNestedMapList?.get("k")?.single()?.label == "map list replacement")
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), plainNested))) == nestedObjectHolderFields(plainNested))
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoNestedRows":[[{}]]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","plainDtoNestedMixed":[{"k":null}]}}""") }.isFailure)
        check(overridden.cfg?.viaCollectionRows?.single()?.label == "bridge rows member" && overridden.cfg?.viaCollectionMap?.get("value")?.label == "bridge map member")
        val bridged = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaCollectionRows":[{"label":"replacement"}],"viaCollectionRowsNull":[{"label":"set"}],"viaCollectionMap":{"nil":null,"new":{"label":"map"}},"viaCollectionGrid":[{"new":{"label":"grid"}}]}}""")
        check(bridged.cfg?.viaCollectionRows?.single()?.label == "replacement" && bridged.cfg?.viaCollectionRows?.single()?.child?.flag == true)
        check(bridged.cfg?.viaCollectionRowsNull?.single()?.label == "set")
        check(bridged.cfg?.viaCollectionMap?.keys == setOf("nil", "new") && bridged.cfg?.viaCollectionMap?.get("nil") == null && bridged.cfg?.viaCollectionMap?.get("new")?.count == 2)
        check(bridged.cfg?.viaCollectionGrid?.single()?.keys == setOf("new") && bridged.cfg?.viaCollectionGrid?.single()?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), bridged))) == nestedObjectHolderFields(bridged))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaCollectionRowsNull":null}}""").cfg?.viaCollectionRowsNull == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaCollectionRows":[]}}""").cfg?.viaCollectionRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaCollectionRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaCollectionRows":[null]}}""") }.isFailure)
        check(overridden.cfg?.viaInternalRows?.single()?.label == "internal rows member" && overridden.cfg?.viaInternalMap?.get("value")?.label == "internal map member")
        val internal = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaInternalRows":[{"label":"replacement"}],"viaInternalNull":[{"label":"set"}],"viaInternalMap":{"nil":null,"new":{"label":"map"}},"viaInternalGrid":[{"new":{"label":"grid"}}]}}""")
        check(internal.cfg?.viaInternalRows?.single()?.label == "replacement" && internal.cfg?.viaInternalRows?.single()?.child?.flag == true)
        check(internal.cfg?.viaInternalNull?.single()?.label == "set")
        check(internal.cfg?.viaInternalMap?.keys == setOf("nil", "new") && internal.cfg?.viaInternalMap?.get("nil") == null && internal.cfg?.viaInternalMap?.get("new")?.count == 2)
        check(internal.cfg?.viaInternalGrid?.single()?.keys == setOf("new") && internal.cfg?.viaInternalGrid?.single()?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), internal))) == nestedObjectHolderFields(internal))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaInternalNull":null}}""").cfg?.viaInternalNull == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaInternalRows":[]}}""").cfg?.viaInternalRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaInternalRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaInternalRows":[null]}}""") }.isFailure)
        check(overridden.cfg?.viaBorrowedRows?.single()?.label == "borrowed rows member" && overridden.cfg?.viaBorrowedMap?.get("value")?.label == "borrowed map member")
        val borrowed = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaBorrowedRows":[{"label":"replacement"}],"viaBorrowedNull":[{"label":"set"}],"viaBorrowedMap":{"nil":null,"new":{"label":"map"}},"viaBorrowedGrid":[{"new":{"label":"grid"}}]}}""")
        check(borrowed.cfg?.viaBorrowedRows?.single()?.label == "replacement" && borrowed.cfg?.viaBorrowedRows?.single()?.child?.flag == true)
        check(borrowed.cfg?.viaBorrowedNull?.single()?.label == "set")
        check(borrowed.cfg?.viaBorrowedMap?.keys == setOf("nil", "new") && borrowed.cfg?.viaBorrowedMap?.get("nil") == null && borrowed.cfg?.viaBorrowedMap?.get("new")?.count == 2)
        check(borrowed.cfg?.viaBorrowedGrid?.single()?.keys == setOf("new") && borrowed.cfg?.viaBorrowedGrid?.single()?.get("new")?.count == 2)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), borrowed))) == nestedObjectHolderFields(borrowed))
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaBorrowedNull":null}}""").cfg?.viaBorrowedNull == null)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaBorrowedRows":[]}}""").cfg?.viaBorrowedRows?.isEmpty() == true)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaBorrowedRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaBorrowedRows":[null]}}""") }.isFailure)
        val chainOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaChain":{"label":"replacement"},"viaChainNull":{"label":"set"},"viaNamedChain":{"label":"named"}}}""")
        check(chainOverride.cfg?.viaChain?.label == "replacement" && chainOverride.cfg?.viaChain?.child?.flag == true)
        check(chainOverride.cfg?.viaChainNull?.label == "set" && chainOverride.cfg?.viaNamedChain?.label == "named")
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaChainNull":null}}""").cfg?.viaChainNull == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaChain":{}}}""") }.isFailure)
        check(overridden.cfg?.owned?.label == "owned member" && overridden.cfg?.ownedRow?.label == "row member" && overridden.cfg?.ownedVariant?.label == "variant member")
        val ownedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","owned":{"label":"replacement"},"ownedAlias":{"label":"set"},"ownedRow":{"label":"row"},"ownedVariant":{"label":"variant"}}}""")
        check(ownedOverride.cfg?.owned?.label == "replacement" && ownedOverride.cfg?.owned?.count == 2 && ownedOverride.cfg?.owned?.child?.flag == true)
        check(ownedOverride.cfg?.ownedAlias?.label == "set" && ownedOverride.cfg?.ownedAlias?.child?.flag == true)
        check(ownedOverride.cfg?.ownedRow?.label == "row" && ownedOverride.cfg?.ownedVariant?.label == "variant")
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","ownedAlias":null}}""").cfg?.ownedAlias == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","owned":{}}}""") }.isFailure)
        val aliasOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaAlias":{"label":"replacement"},"aliasNull":{"label":"set"}}}""")
        check(aliasOverride.cfg?.viaAlias?.label == "replacement" && aliasOverride.cfg?.viaAlias?.child?.flag == true)
        check(aliasOverride.cfg?.aliasNull?.label == "set" && aliasOverride.cfg?.aliasNull?.count == 2)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","viaAlias":{}}}""") }.isFailure)
        val linkedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","linked":{"label":"replacement"}}}""")
        check(linkedOverride.cfg?.linked?.label == "replacement" && linkedOverride.cfg?.linked?.count == 2 && linkedOverride.cfg?.linked?.child?.flag == true)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","linked":null}}""").cfg?.linked == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","linked":{}}}""") }.isFailure)
        val memberOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","composed":{"label":"replacement"}}}""")
        check(memberOverride.cfg?.composed?.label == "replacement" && memberOverride.cfg?.composed?.count == 2 && memberOverride.cfg?.composed?.child?.flag == true)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","composed":null}}""").cfg?.composed == null)
        check(runCatching { codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":{"name":"n","composed":{}}}""") }.isFailure)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"cfg":null}""").cfg == null)
        val mergedOverride = codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"allOfCfg":{"z":"replacement","count":8},"allOfNull":{"a":"set","b":1}}""")
        check(mergedOverride.allOfCfg?.z == "replacement" && mergedOverride.allOfCfg?.count == 8 && mergedOverride.allOfCfg?.meta?.flag == true)
        check(mergedOverride.allOfNull?.a == "set" && mergedOverride.allOfNull?.b == 1)
        check(codec.decodeFromString<HolderDto>("""{"requiredCfg":{"id":"x"},"allOfCfg":null}""").allOfCfg == null)
        check(nestedObjectHolderFields(codec.decodeFromJsonElement(HolderDto.serializer(), codec.encodeToJsonElement(HolderDto.serializer(), missing))) == nestedObjectHolderFields(missing))
        // Own defaults on the referenced component remain ordinary nullable fields.
        val referenced = codec.decodeFromString<PatchInputDto>("{}")
        check(referenced.cfg?.a == "x" && referenced.cfgNullable == null)
        // Inline request object defaults keep complete validation but Patch.Absent.
        val absent = codec.decodeFromString<PatchHolderInputDto>("""{"required":null}""")
        check(absent.cfg == Patch.Absent && absent.allOfCfg == Patch.Absent && absent.aliasCfg == Patch.Absent && absent.required == null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), absent) as JsonObject).keys.none { it == "cfg" || it == "allOfCfg" || it == "aliasCfg" })
        val cleared = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"cfg":null,"allOfCfg":null,"aliasCfg":null}""")
        check(cleared.cfg == Patch.Null && cleared.allOfCfg == Patch.Null && cleared.aliasCfg == Patch.Null)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["cfg"] === JsonNull)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["allOfCfg"] === JsonNull)
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), cleared) as JsonObject)["aliasCfg"] === JsonNull)
        val aliasUpdate = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"member":{"label":"update"},"owned":{"label":"owned update"},"chained":{"label":"chain update"},"rows":[{"label":"rows update"}],"refs":[{"label":"refs update"}],"namedRows":[{"label":"named update"}],"collectionBridgeRows":[{"label":"bridge update"}],"collectionInternalRows":[{"label":"internal update"}],"collectionEntityRows":[{"label":"entity update"}]}}""").aliasCfg
        check(aliasUpdate is Patch.Value && aliasUpdate.value.member?.label == "update" && aliasUpdate.value.member?.count == 2)
        check(aliasUpdate.value.owned?.label == "owned update" && aliasUpdate.value.owned?.child?.flag == true)
        check(aliasUpdate.value.chained?.label == "chain update" && aliasUpdate.value.chained?.child?.flag == true)
        check(aliasUpdate.value.rows?.single()?.label == "rows update" && aliasUpdate.value.rows?.single()?.count == 2)
        check(aliasUpdate.value.refs?.single()?.label == "refs update" && aliasUpdate.value.refs?.single()?.child?.flag == true)
        check(aliasUpdate.value.namedRows?.single()?.label == "named update" && aliasUpdate.value.namedRows?.single()?.child?.flag == true)
        check(aliasUpdate.value.collectionBridgeRows?.single()?.label == "bridge update" && aliasUpdate.value.collectionBridgeRows?.single()?.child?.flag == true)
        check(aliasUpdate.value.collectionInternalRows?.single()?.label == "internal update" && aliasUpdate.value.collectionInternalRows?.single()?.child?.flag == true)
        check(aliasUpdate.value.collectionEntityRows?.single()?.label == "entity update" && aliasUpdate.value.collectionEntityRows?.single()?.child?.flag == true)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"collectionEntityRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"collectionInternalRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"collectionBridgeRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"namedRows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"refs":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"rows":[{}]}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"chained":{}}}""") }.isFailure)
        check(runCatching { codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"aliasCfg":{"member":{}}}""") }.isFailure)
        val mergedUpdate = codec.decodeFromString<PatchHolderInputDto>("""{"required":null,"allOfCfg":{"a":"u","b":3}}""")
        check(mergedUpdate.allOfCfg == Patch.Value(PatchHolderInputDto.AllOfCfg(a = "u", b = 3)))
        val update = PatchHolderInputDto(required = null, cfg = Patch.Value(PatchHolderInputDto.Cfg(a = "u")))
        check((codec.encodeToJsonElement(PatchHolderInputDto.serializer(), update) as JsonObject)["cfg"] == codec.parseToJsonElement("""{"a":"u"}"""))
        // Required nullable object keeps a required argument; explicit null is null.
        check(codec.decodeFromString<RequiredNullableCfgDto>("""{"value":null}""").value == null)
        check(codec.decodeFromString<RequiredNullableCfgDto>("""{"value":{"n":3}}""").value?.n == 3)
        val nullWire = codec.encodeToJsonElement(RequiredNullableCfgDto.serializer(), RequiredNullableCfgDto(value = null)) as JsonObject
        check(nullWire.containsKey("value") == nulls)
        check(codec.decodeFromJsonElement(RequiredNullableCfgDto.serializer(), nullWire).value == null)
        nestedObjectWire(codec, value, checked)
    }
}

private suspend fun nestedObjectWire(codec: Json, value: HolderDto, checked: Boolean) {
    val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val channels = CopyOnWriteArrayList<ByteChannel>()
    val framesWritten = CopyOnWriteArrayList<CompletableDeferred<Unit>>()
    var payload = codec.encodeToString(HolderDto.serializer(), value)
    var failing = false
    var calls = 0
    var callbacks = 0
    var expectedPatch = JsonObject(emptyMap())
    val engine = NestedObjectEngine(MockEngineConfig().apply { addHandler { request ->
        calls++
        check(callbacks == calls) { "Callback must run exactly once before dispatch" }
        when (request.url.encodedPath) {
            "/holder" -> {
                check(codec.parseToJsonElement(nestedObjectBody(request.body)) == codec.encodeToJsonElement(HolderDto.serializer(), value))
                respond(payload, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            "/patch" -> {
                check(codec.parseToJsonElement(nestedObjectBody(request.body)) == expectedPatch)
                respond("", HttpStatusCode.NoContent)
            }
            "/required" -> {
                check(codec.parseToJsonElement(nestedObjectBody(request.body)) == codec.parseToJsonElement(codec.encodeToString(RequiredNullableCfgDto.serializer(), RequiredNullableCfgDto(value = null))))
                respond(codec.encodeToString(RequiredNullableCfgDto.serializer(), RequiredNullableCfgDto(value = null)), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
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
            else -> error("Unexpected nested object default path")
        }
    } })
    val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
    try {
        val api = NestedObject(http, "http://example.test/", codec).defaults
        check(nestedObjectHolderFields(api.sendHolder(value) { callbacks++ }) == nestedObjectHolderFields(value))
        expectedPatch = codec.encodeToJsonElement(PatchHolderInputDto.serializer(), PatchHolderInputDto(required = null)) as JsonObject
        api.patchHolder(PatchHolderInputDto(required = null)) { callbacks++ }
        check(api.sendRequired(RequiredNullableCfgDto(value = null)) { callbacks++ }.value == null)
        val before = calls
        val beforeCallbacks = callbacks
        val stream = api.watchHolders { callbacks++ }
        check(calls == before && callbacks == beforeCallbacks)
        val event = (withTimeout(10000) { stream.toList() }.single() as WatchHoldersEventDto.Holder).data
        check(nestedObjectHolderFields(event) == nestedObjectHolderFields(value))
        if (checked) {
            payload = """{"requiredCfg":{"id":"x"},"cfg":{"name":"n","code":9}}"""
            failing = true
            val beforeFailure = calls
            nestedObjectReject("enum") { api.sendHolder(value) { callbacks++ } }
            check(calls == beforeFailure + 1 && callbacks == calls)
            nestedObjectReject("enum") { withTimeout(10000) { api.watchHolders { callbacks++ }.toList() } }
            check(calls == beforeFailure + 2 && callbacks == calls)
            withTimeout(10000) { framesWritten.last().await() }
            check(channels.last().isClosedForRead) { "Failed full-frame still-open stream was not closed" }
            payload = codec.encodeToString(HolderDto.serializer(), value)
            failing = false
            check(nestedObjectHolderFields(api.sendHolder(value) { callbacks++ }) == nestedObjectHolderFields(value))
            val recovered = (withTimeout(10000) { api.watchHolders { callbacks++ }.toList() }.single() as WatchHoldersEventDto.Holder).data
            check(nestedObjectHolderFields(recovered) == nestedObjectHolderFields(value))
        } else {
            payload = """{"requiredCfg":{"id":"x"},"cfg":{"name":"n","code":9}}"""
            val unknown = api.sendHolder(value) { callbacks++ }
            check(unknown.cfg?.code == 9)
            payload = codec.encodeToString(HolderDto.serializer(), value)
        }
    } finally { http.close(); producers.cancel(); channels.forEach { it.close() } }
    println("Nested object defaults checked=$checked constructor-serde-Patch-HTTP-liveSSE passed")
}

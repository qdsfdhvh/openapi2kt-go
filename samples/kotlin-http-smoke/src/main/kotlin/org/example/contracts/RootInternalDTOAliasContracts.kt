@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.rootinternaldtos.models.ChainAliasDto
import org.example.rootinternaldtos.models.HopAliasDto
import org.example.rootinternaldtos.models.MapAliasDto
import org.example.rootinternaldtos.models.MaybeAliasDto
import org.example.rootinternaldtos.models.NarrowAliasDto
import org.example.rootinternaldtos.models.NodeAliasDto
import org.example.rootinternaldtos.models.OptionalAliasDto
import org.example.rootinternaldtos.models.RowAliasDto
import org.example.rootinternaldtos.models.TreeKeyDto

fun rootInternalDTOAliasContracts(checked: Boolean) {
    // Nested classifiers use their real owner, not an unsupported alias.Child.
    val node: NodeAliasDto = NodeAliasDto("alias", child = TreeKeyDto.Node.Child("canonical"))
    val chained: ChainAliasDto = node
    val hopped: HopAliasDto = chained
    val narrowed: NarrowAliasDto = NarrowAliasDto("narrowed", back = hopped)
    val maybe: MaybeAliasDto = narrowed
    val row: RowAliasDto = RowAliasDto("row")
    val mapped: MapAliasDto = MapAliasDto("map")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<NodeAliasDto>(codec.encodeToString(node)) == node)
        check(codec.decodeFromString<ChainAliasDto>(codec.encodeToString(chained)) == node)
        check(codec.decodeFromString<HopAliasDto>(codec.encodeToString(hopped)) == node)
        check(codec.decodeFromString<MaybeAliasDto>(codec.encodeToString(maybe)) == narrowed)
        check(codec.decodeFromString<OptionalAliasDto>("null") == null)
        check(codec.decodeFromString<MaybeAliasDto>("null") == null)
        check(codec.encodeToString<MaybeAliasDto>(null) == "null")
        check(runCatching { codec.decodeFromString<NarrowAliasDto>("null") }.isFailure)
        check(codec.decodeFromString<RowAliasDto>(codec.encodeToString(row)) == row)
        check(codec.decodeFromString<MapAliasDto>(codec.encodeToString(mapped)) == mapped)
        check(runCatching { codec.decodeFromString<RowAliasDto>("null") }.isFailure)
        check(runCatching { codec.decodeFromString<MapAliasDto>("null") }.isFailure)
        check(NodeAliasDto.serializer().descriptor.serialName == TreeKeyDto.Node.serializer().descriptor.serialName)
        check(codec.decodeFromString(NodeAliasDto.serializer(), codec.encodeToString(node)) == node)
        if (checked) {
            check(runCatching { NodeAliasDto("") }.isFailure)
            check(runCatching { codec.decodeFromString<NodeAliasDto>("{\"id\":\"\"}") }.isFailure)
        }
        println("Root/internal DTO aliases PASS: defaults=$defaults/nulls=$nulls; canonical ctor/serializer/nullable/collection leaf")
    }
}

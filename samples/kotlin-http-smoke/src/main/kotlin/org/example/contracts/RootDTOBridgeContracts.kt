@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.rootdtobridges.models.CollectionAliasUseDto
import org.example.rootdtobridges.models.MapAliasChainDto
import org.example.rootdtobridges.models.MaybeAliasChainDto
import org.example.rootdtobridges.models.NodeAliasChainDto
import org.example.rootdtobridges.models.RowAliasChainDto

fun rootDTOBridgeContracts() {
    val node = NodeAliasChainDto("valid")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<NodeAliasChainDto>(codec.encodeToString(node)) == node)
        check(codec.decodeFromString<MaybeAliasChainDto>("null") == null)
        val collections = CollectionAliasUseDto(RowAliasChainDto("row"), MapAliasChainDto("mapping"))
        check(codec.decodeFromString<CollectionAliasUseDto>(codec.encodeToString(collections)) == collections)
        check(NodeAliasChainDto.serializer().descriptor.serialName.endsWith("NodeDto") ||
            NodeAliasChainDto.serializer().descriptor.serialName.endsWith("TreeKeyDto.Node"))
        println("Root DTO bridge codec PASS: defaults=$defaults/nulls=$nulls; canonical constructor/reified nullable codec")
    }
}

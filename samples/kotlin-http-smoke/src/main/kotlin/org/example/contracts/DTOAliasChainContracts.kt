@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.dtoaliaschains.models.CollectionAliasUseDto
import org.example.dtoaliaschains.models.MapAliasChainDto
import org.example.dtoaliaschains.models.MaybeAliasChainDto
import org.example.dtoaliaschains.models.NodeAliasChainDto
import org.example.dtoaliaschains.models.RowAliasChainDto

// Compiled against either named or internal physical canonical owners.
fun dtoAliasChainContracts() {
    val node = NodeAliasChainDto("valid")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<NodeAliasChainDto>(codec.encodeToString(node)) == node)
        check(codec.decodeFromString<MaybeAliasChainDto>("null") == null)
        val collections = CollectionAliasUseDto(RowAliasChainDto("row"), MapAliasChainDto("mapping"))
        check(codec.decodeFromString<CollectionAliasUseDto>(codec.encodeToString(collections)) == collections)
        check(NodeAliasChainDto.serializer().descriptor.serialName.endsWith("NodeDto") ||
            NodeAliasChainDto.serializer().descriptor.serialName.endsWith("TreeKeyDto.Node"))
        println("DTO alias chain codec PASS: defaults=$defaults/nulls=$nulls; canonical constructor/reified nullable codec")
    }
}

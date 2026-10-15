@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.aliasdepth.models.ConsumerDto
import org.example.aliasdepth.models.ZAliasDto
import org.example.aliasuniondepth.models.ConsumerDto as UnionConsumerDto
import org.example.aliasuniondepth.models.ZAliasDto as UnionAliasDto

fun aliasConsumerDepthContracts() {
    val payload = "[".repeat(46) + "\"ok\"" + "]".repeat(46)
    val wire = "{\"v\":$payload}"
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val alias = codec.decodeFromString<ZAliasDto>(payload)
        val model = codec.decodeFromString<ConsumerDto>(wire)
        check(model.v == alias && model.copy(v = alias) == model)
        check(codec.encodeToString(model) == wire)
        check(codec.decodeFromJsonElement(ConsumerDto.serializer(), codec.parseToJsonElement(wire)) == model)
        for (leaf in listOf("\"ok\"", "null")) {
            val unionPayload = "[".repeat(45) + leaf + "]".repeat(45)
            val unionWire = "{\"v\":$unionPayload}"
            val unionAlias = codec.decodeFromString<UnionAliasDto>(unionPayload)
            val unionModel = codec.decodeFromString<UnionConsumerDto>(unionWire)
            check(unionModel.v == unionAlias && unionModel.copy(v = unionAlias) == unionModel)
            check(codec.encodeToString(unionModel) == unionWire)
        }
        println("Cumulative alias consumer48-depth compilation/ctor-copy-serde passed: defaults=$defaults/nulls=$nulls; original46-generic and union45+branch1")
    }
}

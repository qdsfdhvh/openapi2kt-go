package org.example.platform

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.platformincrements.models.Bounds
import org.example.platformincrements.models.ByteTrees
import org.example.platformincrements.models.NodeA
import org.example.platformincrements.models.NodeZ
import org.example.platformincrements.models.`FactoryRows default`

@OptIn(ExperimentalSerializationApi::class)
internal fun modelIncrementContracts(json: Json) {
    check(Bounds().value == "abc")
    check(json.decodeFromString<Bounds>("{}").value == "abc")
    check(runCatching { Bounds(value = "a") }.isFailure)
    check(runCatching { Bounds(value = "abcd") }.isFailure)
    check(runCatching { json.decodeFromString<Bounds>("{\"value\":\"a\"}") }.isFailure)
    val bytes = ByteTrees()
    val fresh = ByteTrees()
    check(bytes.rows!![0].contentEquals(byteArrayOf(-128, 0, 127)))
    check(bytes.rows!![1].isEmpty() && bytes.bag!!["a"]!!.isEmpty())
    check(bytes.payload!!.blob.contentEquals(byteArrayOf(-128, 127)))
    check(bytes.rows!![0] !== fresh.rows!![0] && bytes.payload!!.blob !== fresh.payload!!.blob)
    fresh.rows!![0][0] = 7
    check(bytes.rows!![0][0] == (-128).toByte())
    val encoded = json.encodeToString(bytes)
    check(json.parseToJsonElement(encoded).jsonObject["rows"] == json.parseToJsonElement("[[-128,0,127],[]]"))
    check(json.encodeToString(json.decodeFromString<ByteTrees>(encoded)) == encoded)
    check(runCatching { json.decodeFromString<ByteTrees>("{\"rows\":[[128]]}") }.isFailure)
    val factory = `FactoryRows default`()
    check(factory.single().contentEquals(byteArrayOf(9)))
    check(factory.single() !== `FactoryRows default`().single())
    val leaf: NodeZ = NodeA()
    val tree: NodeZ = NodeA(next = leaf, children = listOf(leaf))
    check(serializer<NodeZ>().descriptor.serialName == serializer<NodeA>().descriptor.serialName)
    check(json.decodeFromString<NodeZ>(json.encodeToString(tree)) == tree)
    check(json.decodeFromString<NodeZ>("{}").value == "ab")
    check(runCatching { NodeZ(value = "a") }.isFailure)
    check(runCatching { json.decodeFromString<NodeZ>("{\"next\":{\"value\":\"a\"}}") }.isFailure)
}

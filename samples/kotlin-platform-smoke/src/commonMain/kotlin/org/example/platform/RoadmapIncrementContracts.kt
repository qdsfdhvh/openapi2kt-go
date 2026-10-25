package org.example.platform

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.platformroadmap.models.EnumRows
import org.example.platformroadmap.models.Holder
import org.example.platformroadmap.models.MaybeRows
import org.example.platformroadmap.models.Node
import org.example.platformroadmap.models.Rows
import org.example.platformroadmap.models.Seed
import org.example.platformroadmap.models.SeedRows
import org.example.platformroadmap.models.State
import org.example.platformroadmap.models.StringUnionFirst
import org.example.platformroadmap.models.StringUnionSecond
import org.example.platformroadmap.models.NullUnionFirst
import org.example.platformroadmap.models.NullUnionSecond
import org.example.platformroadmap.models.UnionFirst
import org.example.platformroadmap.models.UnionSecond
import org.example.platformroadmap.models.`SeedAlias default`
import org.example.platformroadmap.models.`State default`

internal fun roadmapIncrementContracts(json: Json) {
    val node=Node(value="valid",next=Node(value="leaf"))
    check(node.count==7)
    val rows: Rows=listOf(node)
    check(json.decodeFromString<Rows>(json.encodeToString(rows))==rows)
    check(json.decodeFromString<Rows>("[{\"value\":\"valid\"}]").single().count==7)
    val maybe: MaybeRows=listOf(null,node)
    check(json.decodeFromString<MaybeRows>(json.encodeToString(maybe))==maybe)
    check(runCatching {Node(value="x")}.isFailure)
    check(runCatching {json.decodeFromString<Rows>("[{\"value\":\"x\"}]")}.isFailure)
    val seed=Seed.default()
    check(seed.value=="seed" && seed.tags==listOf("one"))
    check(seed.tags !== Seed.default().tags)
    check(`SeedAlias default`().value=="alias" && `SeedAlias default`().tags==listOf("other"))
    check(Seed(value="runtime").tags==null)
    check(json.decodeFromString<SeedRows>("[{\"value\":\"runtime\"}]").single().tags==null)
    check(runCatching {json.decodeFromString<SeedRows>("[{}]")}.isFailure)
    check(`State default`().serialName=="ready")
    check(json.decodeFromString<EnumRows>("[\"future\"]").single() is State.Unrecognized)
    val left: UnionFirst=StringUnionFirst("text")
    val right: UnionSecond=StringUnionSecond("text")
    check(left==right)
    check(UnionFirst.serializer().descriptor.serialName==UnionSecond.serializer().descriptor.serialName)
    check(json.encodeToString<UnionSecond>(right)=="\"text\"")
    check(json.decodeFromString<UnionSecond>("null")===NullUnionFirst)
    check(NullUnionSecond===NullUnionFirst)
    val tree=listOf(mapOf("a" to listOf(1,2),"b" to listOf(3)))
    val holder=Holder(rows=rows,tree=tree)
    check(holder.values==listOf("fallback"))
    check(holder.values !== Holder(rows=rows,tree=tree).values)
    val decoded=json.decodeFromString<Holder>("{\"rows\":[{\"value\":\"valid\"}],\"tree\":[{\"a\":[1,2],\"b\":[3]}]}")
    check(decoded.values==listOf("fallback"))
    check(json.decodeFromString<Holder>(json.encodeToString(holder))==holder)
    check(runCatching {Holder(rows=rows,tree=tree,values=emptyList())}.isFailure)
    check(runCatching {Holder(rows=rows,tree=tree+mapOf("b" to listOf(3),"a" to listOf(1,2)))}.isFailure)
    check(runCatching {json.decodeFromString<Holder>("{\"rows\":[{\"value\":\"valid\"}],\"tree\":[{\"a\":[1]},{\"a\":[1]}]}")}.isFailure)
}

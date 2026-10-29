@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.serializer
import org.example.nulldefaults.models.BuiltinChildFactoryDto
import org.example.nulldefaults.models.EmptyBuiltinChildFactoryDto
import org.example.nulldefaults.models.LeafBytesDto
import org.example.nulldefaults.models.LeafCountsDto
import org.example.nulldefaults.models.LeafDaysDto
import org.example.nulldefaults.models.LeafModesDto
import org.example.nulldefaults.models.LeafModesDtoInner
import org.example.nulldefaults.models.LeafNamesDto
import org.example.nulldefaults.models.LeafRequiredDto
import org.example.nulldefaults.models.LeafStampsDto
import org.example.nulldefaults.models.NilLeafCountsDto
import org.example.nulldefaults.models.NilLeafNamesDto
import org.example.nulldefaults.models.`LeafBytesDto default`
import org.example.nulldefaults.models.`LeafCodesDto default`
import org.example.nulldefaults.models.`LeafCountsDto default`
import org.example.nulldefaults.models.`LeafDaysDto default`
import org.example.nulldefaults.models.`LeafModesDto default`
import org.example.nulldefaults.models.`LeafNamesDto default`
import org.example.nulldefaults.models.`LeafStampsDto default`
import org.example.nulldefaults.models.`LeafWrappedDto default`
import org.example.nulldefaults.models.`NilLeafCountsDto default`
import org.example.nulldefaults.models.`NilLeafNamesDto default`
import org.example.nulldefaults.models.TreeBytesDto
import org.example.nulldefaults.models.TreeDaysDto
import org.example.nulldefaults.models.TreeEmptyDto
import org.example.nulldefaults.models.TreeListsDto
import org.example.nulldefaults.models.TreeListMapsDto
import org.example.nulldefaults.models.TreeMapListsDto
import org.example.nulldefaults.models.TreeMapsDto
import org.example.nulldefaults.models.TreeModesDto
import org.example.nulldefaults.models.TreeModesDtoInnerValue
import org.example.nulldefaults.models.TreeNilDto
import org.example.nulldefaults.models.TreeRequiredDto
import org.example.nulldefaults.models.TreeStampsDto
import org.example.nulldefaults.models.TreeWrappedDto
import org.example.nulldefaults.models.`TreeBytesDto default`
import org.example.nulldefaults.models.`TreeCheckedDto default`
import org.example.nulldefaults.models.`TreeCodesDto default`
import org.example.nulldefaults.models.`TreeDaysDto default`
import org.example.nulldefaults.models.`TreeEmptyDto default`
import org.example.nulldefaults.models.`TreeListsDto default`
import org.example.nulldefaults.models.`TreeListMapsDto default`
import org.example.nulldefaults.models.`TreeMapListsDto default`
import org.example.nulldefaults.models.`TreeMapsDto default`
import org.example.nulldefaults.models.`TreeModesDto default`
import org.example.nulldefaults.models.`TreeNilDto default`
import org.example.nulldefaults.models.`TreeStampsDto default`
import org.example.nulldefaults.models.`TreeWrappedDto default`
import org.example.nulldefaults.models.CountsDto
import org.example.nulldefaults.models.DayDto
import org.example.nulldefaults.models.HolderDto
import org.example.nulldefaults.models.LabelDto
import org.example.nulldefaults.models.ModeDto
import org.example.nulldefaults.models.NameDto
import org.example.nulldefaults.models.NamesDto
import org.example.nulldefaults.models.`CountsDto default`
import org.example.nulldefaults.models.`DayDto default`
import org.example.nulldefaults.models.`ModeDto default`
import org.example.nulldefaults.models.`NameDto default`
import org.example.nulldefaults.models.`NamesDto default`

// An explicitly nullable component root may carry a **non-null** default. The
// declaration keeps its nullable public type, so the factory returns a non-null
// instance of it: absent -> value, present null -> still null.
suspend fun nullableComponentDefaultsContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        componentNullDefaultsContracts(codec)
        nullableCollectionLeafDefaults(codec)
        nestedComponentDefaultFactories(codec, checked)
        val childFactory = BuiltinChildFactoryDto.default()
        check(childFactory.counts == listOf(2) && childFactory.labels == mapOf("first" to "field"))
        check(childFactory.days == listOf(kotlinx.datetime.LocalDate.parse("2024-03-01")))
        check(childFactory.counts !== BuiltinChildFactoryDto.default().counts)
        check(EmptyBuiltinChildFactoryDto.default().let { it.counts == null && it.instants == null })
        val childRoundTrip = codec.decodeFromString<BuiltinChildFactoryDto>(codec.encodeToString(childFactory))
        check(childRoundTrip.counts == listOf(2) && childRoundTrip.labels == mapOf("first" to "field") && childRoundTrip.days == childFactory.days)

        // Every admitted shape decodes its own annotation through the alias.
        check(`NameDto default`() == "fallback")
        check(`NamesDto default`() == listOf("a", "b"))
        check(`CountsDto default`() == mapOf("k" to 1))
        check(`DayDto default`() == kotlinx.datetime.LocalDate.parse("2024-02-29"))
        check(`ModeDto default`() == ModeDto.A)
        check(LabelDto.default()?.z == "label") { "nullable object factory lost its value" }

        // The nullable public type still accepts null, which no factory forces.
        val explicitNull: NameDto = null
        check(explicitNull == null)
        val nullNames: NamesDto = null
        check(nullNames == null)

        // Each call evaluates the factory again, so no caller shares one list.
        check(`NamesDto default`() !== `NamesDto default`())

        // A field referencing a defaulted component keeps its neutral presence.
        val holder = HolderDto()
        check(holder.name == null) { "Component default leaked into a referencing field" }
        check(holder.names == null) { "Component default leaked into a referencing field" }
        check(holder.inline == "inline")

        // The factory output round-trips through the same wire the fields use.
        val decoded = codec.decodeFromString<NameDto>(codec.encodeToString(`NameDto default`()))
        check(decoded == "fallback")
        check(codec.encodeToJsonElement(`NamesDto default`()) == codec.parseToJsonElement("""["a","b"]"""))
        check(codec.encodeToJsonElement(`CountsDto default`()) == codec.parseToJsonElement("""{"k":1}"""))
        check(codec.encodeToJsonElement(`DayDto default`()) == codec.parseToJsonElement("\"2024-02-29\""))
        check(codec.encodeToJsonElement(LabelDto.default()!!) == codec.parseToJsonElement("""{"z":"label"}"""))

        // Supplying null stays a distinct value from omitting the field.
        val explicit = codec.decodeFromString<HolderDto>("""{"name":null,"names":["c"]}""")
        check(explicit.name == null && explicit.names == listOf("c"))
        val supplied = codec.decodeFromString<HolderDto>("""{"name":"given"}""")
        check(supplied.name == "given")

        println("NullableComponentDefaults Json defaults=$defaults nulls=$nulls passed")
    }
}

private fun nullableCollectionLeafDefaults(codec: Json) {
    val names: LeafNamesDto = `LeafNamesDto default`()
    val counts: LeafCountsDto = `LeafCountsDto default`()
    val days: LeafDaysDto = `LeafDaysDto default`()
    val stamps: LeafStampsDto = `LeafStampsDto default`()
    val modes: LeafModesDto = `LeafModesDto default`()
    val bytes: LeafBytesDto = `LeafBytesDto default`()
    check(names == listOf(null, "a") && counts == mapOf("a" to null, "z" to 7))
    check(days == listOf(null, kotlinx.datetime.LocalDate.parse("2024-02-29")))
    check(stamps == mapOf("a" to null, "z" to kotlin.time.Instant.parse("2024-02-29T00:00:00Z")))
    check(modes == listOf(null, LeafModesDtoInner.A))
    check(`LeafCodesDto default`() == mapOf("a" to null, "z" to 2))
    check(`LeafWrappedDto default`() == listOf(null, "wrapped"))
    check(bytes[0] == null && bytes[1]!!.contentEquals(byteArrayOf(-128, 127)))
    bytes[1]!![0] = 0
    check(`LeafBytesDto default`()[1]!!.contentEquals(byteArrayOf(-128, 127)))
    check(names !== `LeafNamesDto default`() && counts !== `LeafCountsDto default`())

    // Element nullability and the nullable outer alias use separate serializers.
    check(!serializer<LeafNamesDto>().descriptor.isNullable)
    check(serializer<LeafNamesDto>().descriptor.getElementDescriptor(0).isNullable)
    check(serializer<LeafCountsDto>().descriptor.isNullable)
    check(`NilLeafNamesDto default`() == null && `NilLeafCountsDto default`() == null)
    check(codec.decodeFromJsonElement<NilLeafNamesDto>(JsonNull) == null)
    check(codec.decodeFromJsonElement<NilLeafCountsDto>(JsonNull) == null)
    check(runCatching { codec.decodeFromString<LeafNamesDto>("null") }.isFailure)
    check(runCatching { codec.decodeFromString<LeafNamesDto>("[null,1]") }.isFailure)
    check(codec.decodeFromString<LeafNamesDto>(codec.encodeToString(names)) == names)
    check(codec.decodeFromString<LeafCountsDto>(codec.encodeToString(counts)) == counts)
    check(codec.decodeFromString<LeafDaysDto>(codec.encodeToString(days)) == days)
    check(codec.decodeFromString<LeafStampsDto>(codec.encodeToString(stamps)) == stamps)
    check(codec.decodeFromString<LeafModesDto>(codec.encodeToString(modes)) == modes)
    check(codec.decodeFromString<LeafModesDto>("[null,\"future\"]")[1] is LeafModesDtoInner.Unrecognized)
    check(codec.encodeToJsonElement(names) == codec.parseToJsonElement("[null,\"a\"]"))
    check(codec.encodeToJsonElement(counts) == codec.parseToJsonElement("""{"a":null,"z":7}"""))
    check(codec.encodeToJsonElement(`LeafBytesDto default`()) == codec.parseToJsonElement("[null,[-128,127]]"))

    val holder = codec.decodeFromString<HolderDto>("{}")
    check(holder.leafNames == null && holder.leafCounts == null && holder.leafWrapped == null)
    val explicit = codec.decodeFromString<HolderDto>("""{"leafNames":null,"leafCounts":null,"leafWrapped":null}""")
    check(explicit.leafNames == null && explicit.leafCounts == null && explicit.leafWrapped == null)
    val supplied = codec.decodeFromString<HolderDto>("""{"leafNames":[null,"given"],"leafCounts":{"a":null}}""")
    check(supplied.leafNames == listOf(null, "given") && supplied.leafCounts == mapOf("a" to null))
    check(runCatching { codec.decodeFromString<LeafRequiredDto>("{}") }.isFailure)
    check(runCatching { codec.decodeFromString<LeafRequiredDto>("""{"names":null}""") }.isFailure)
    check(codec.decodeFromString<LeafRequiredDto>("""{"names":[null]}""").names == listOf<String?>(null))
}

private fun nestedComponentDefaultFactories(codec: Json, checked: Boolean) {
    val lists: TreeListsDto = `TreeListsDto default`()
    val listMaps: TreeListMapsDto = `TreeListMapsDto default`()
    val mapLists: TreeMapListsDto = `TreeMapListsDto default`()
    val maps: TreeMapsDto = `TreeMapsDto default`()
    val days: TreeDaysDto = `TreeDaysDto default`()
    val stamps: TreeStampsDto = `TreeStampsDto default`()
    val modes: TreeModesDto = `TreeModesDto default`()
    val bytes: TreeBytesDto = `TreeBytesDto default`()
    val wrapped: TreeWrappedDto = `TreeWrappedDto default`()
    val nil: TreeNilDto = `TreeNilDto default`()
    val empty: TreeEmptyDto = `TreeEmptyDto default`()
    check(lists == listOf(listOf(1), emptyList<Int>()))
    check(listMaps == listOf(mapOf("a" to "first", "z" to "last"), emptyMap<String, String>()))
    check(mapLists == mapOf("z" to listOf(1, null), "nil" to null, "empty" to emptyList<Int?>()))
    check(maps == mapOf("z" to mapOf("a" to 2.5f, "z" to 1.5f), "a" to emptyMap<String, Float>()))
    check(days == listOf(mapOf("a" to null, "z" to kotlinx.datetime.LocalDate.parse("2024-02-29"))))
    check(stamps == mapOf("rows" to listOf(kotlin.time.Instant.parse("2024-02-29T00:00:00Z"))))
    check(modes == listOf(mapOf("a" to null, "z" to TreeModesDtoInnerValue.A)))
    check(`TreeCodesDto default`() == mapOf("rows" to listOf(null, 2)))
    check(empty.isEmpty() && nil == null)
    check(codec.decodeFromString<TreeNilDto>("null") == null)
    check(runCatching { codec.decodeFromString<TreeListsDto>("null") }.isFailure)
    check(runCatching { codec.decodeFromString<TreeListsDto>("[null]") }.isFailure)
    check(runCatching { codec.decodeFromString<TreeListsDto>("[[null]]") }.isFailure)
    check(bytes[0][0] == null && bytes[0][1]!!.contentEquals(byteArrayOf(-128, 127)))
    bytes[0][1]!![0] = 0
    check(`TreeBytesDto default`()[0][1]!!.contentEquals(byteArrayOf(-128, 127)))
    check(lists !== `TreeListsDto default`() && lists[0] !== `TreeListsDto default`()[0])
    check(mapLists !== `TreeMapListsDto default`() && mapLists!!["z"] !== `TreeMapListsDto default`()!!["z"])
    check(codec.decodeFromString<TreeListsDto>(codec.encodeToString(lists)) == lists)
    check(codec.decodeFromString<TreeListMapsDto>(codec.encodeToString(listMaps)) == listMaps)
    check(codec.decodeFromString<TreeMapListsDto>(codec.encodeToString(mapLists)) == mapLists)
    check(codec.decodeFromString<TreeMapsDto>(codec.encodeToString(maps)) == maps)
    check(codec.decodeFromString<TreeDaysDto>(codec.encodeToString(days)) == days)
    check(codec.decodeFromString<TreeStampsDto>(codec.encodeToString(stamps)) == stamps)
    check(codec.decodeFromString<TreeModesDto>(codec.encodeToString(modes)) == modes)
    check(codec.decodeFromString<TreeWrappedDto>(codec.encodeToString(wrapped)) == wrapped)
    check(codec.encodeToJsonElement(mapLists) == codec.parseToJsonElement("""{"empty":[],"nil":null,"z":[1,null]}"""))
    check(codec.encodeToJsonElement(wrapped).toString() == """{"a":{"${'$'}ref":"data","/~":"key"},"z":{"a":"first","z":"last"}}""")
    check(codec.decodeFromString<TreeModesDto>("""[{"a":null,"z":"future"}]""")[0]["z"] is TreeModesDtoInnerValue.Unrecognized)

    // Referencing fields never inherit a component or wrapper annotation.
    val missing = codec.decodeFromString<HolderDto>("{}")
    check(missing.treeLists == null && missing.treeMapLists == null && missing.treeWrapped == null)
    val explicit = codec.decodeFromString<HolderDto>("""{"treeLists":null,"treeMapLists":null,"treeWrapped":null}""")
    check(explicit.treeLists == null && explicit.treeMapLists == null && explicit.treeWrapped == null)
    val supplied = codec.decodeFromString<HolderDto>("""{"treeLists":[[9]],"treeMapLists":{"new":null}}""")
    check(supplied.treeLists == listOf(listOf(9)) && supplied.treeMapLists == mapOf("new" to null))
    check(runCatching { codec.decodeFromString<TreeRequiredDto>("{}") }.isFailure)
    check(codec.decodeFromString<TreeRequiredDto>("""{"rows":[[]]}""").rows == listOf(emptyList<Int>()))
    val valid = TreeRequiredDto(rows = lists, checked = `TreeCheckedDto default`())
    check(codec.decodeFromString<TreeRequiredDto>(codec.encodeToString(valid)).checked == listOf(listOf(1, null)))
    for (invalid in listOf(emptyList(), listOf(emptyList()), listOf(listOf(0)), listOf(listOf(1, 1)))) {
        check(runCatching { TreeRequiredDto(rows = lists, checked = invalid) }.isFailure == checked)
    }
    check(runCatching { codec.decodeFromString<TreeRequiredDto>("""{"rows":[[1]],"checked":[[0]]}""") }.isFailure == checked)
}

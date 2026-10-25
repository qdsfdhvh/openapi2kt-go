package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import org.example.falsecomponents.models.Blocked as BlockedRoot
import org.example.falsecomponents.models.Peer as PeerRoot
import org.example.falsecomponents.models.Record as RecordRoot
import org.example.falsecomponents.models.Serializer as SerializerRoot
import org.example.falsecomponents.models.Serializer2 as Serializer2Root
import org.example.falsecomponents.models.Companion as CompanionRoot
import org.example.falsecomponents.models.Alias as AliasRoot
import org.example.falsecomponents.models.Nullable as NullableRoot
import org.example.falsecomponents.models.Inherited as InheritedRoot
import org.example.falsecomponents.models.Narrow as NarrowRoot
import org.example.falsecomponents.models.Again as AgainRoot

// Reified encoder call sites compile even though no non-null root can be built.
private fun encodeBlocked(value: BlockedRoot): String = Json.encodeToString(value)
private fun encodeAlias(value: AliasRoot): String = Json.encodeToString(value)

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun falseComponentRootsContracts() {
    check(serializer<BlockedRoot>().descriptor.serialName.endsWith(".models.Blocked") || serializer<BlockedRoot>().descriptor.serialName.endsWith(".models.BlockedDto"))
    check(serializer<AliasRoot>().descriptor == serializer<BlockedRoot>().descriptor)
    check(serializer<NarrowRoot>().descriptor == serializer<BlockedRoot>().descriptor)
    check(!serializer<BlockedRoot>().descriptor.isNullable)
    check(serializer<NullableRoot>().descriptor.isNullable)
    check(serializer<InheritedRoot>().descriptor == serializer<NullableRoot>().descriptor)
    check(serializer<AgainRoot>().descriptor == serializer<NullableRoot>().descriptor)
    check(serializer<PeerRoot>().descriptor.serialName != serializer<BlockedRoot>().descriptor.serialName)
    check(java.lang.reflect.Modifier.isAbstract(BlockedRoot::class.java.modifiers))
    val sources = listOf("null","true","false","1","1.5","\"text\"","[]","{}","{\"v\":[null,true]}")
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        for (source in sources) {
            for (failure in listOf(
                runCatching { codec.decodeFromString<BlockedRoot>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<AliasRoot>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<NarrowRoot>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<PeerRoot>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<RecordRoot>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<SerializerRoot>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<Serializer2Root>(source) }.exceptionOrNull(),
                runCatching { codec.decodeFromString<CompanionRoot>(source) }.exceptionOrNull()
            )) check(failure is SerializationException && failure.message == "Schema false admits no JSON value.")
            if (source != "null") {
                check(runCatching { codec.decodeFromString<NullableRoot>(source) }.exceptionOrNull() is SerializationException)
                check(runCatching { codec.decodeFromString<InheritedRoot>(source) }.exceptionOrNull() is SerializationException)
                check(runCatching { codec.decodeFromString<AgainRoot>(source) }.exceptionOrNull() is SerializationException)
            }
        }
        val value:NullableRoot=codec.decodeFromString("null")
        check(value == null && codec.encodeToString(value) == "null")
        check(codec.decodeFromString<InheritedRoot>("null") == null)
        check(codec.decodeFromString<AgainRoot>("null") == null)
        println("FalseComponentRoots defaults=$defaults nulls=$nulls uninhabited-private-actual-codec-alias-nullable-allkind-refusal passed")
    }
}

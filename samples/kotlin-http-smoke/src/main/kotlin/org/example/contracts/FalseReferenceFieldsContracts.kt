package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.falsefields.models.Optional as OptionalHolder
import org.example.falsefields.models.Required as RequiredHolder
import org.example.falsefields.models.NullableHolder as NullableFields
import org.example.falsefields.models.Reserved as ReservedHolder
import org.example.falsefields.models.Twin as TwinHolder
import org.example.falsefields.models.Blocked as BlockedRoot
import org.example.falsefields.models.Peer as PeerRoot

// Required uninhabited arguments remain callable by typed consumers.
private fun buildRequired(value: BlockedRoot): RequiredHolder = RequiredHolder(value)

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun falseReferenceFieldsContracts() {
    val descriptor = serializer<OptionalHolder>().descriptor
    check(descriptor.elementsCount == 3)
    for (index in 0 until descriptor.elementsCount) {
        check(descriptor.isElementOptional(index))
        check(descriptor.getElementDescriptor(index).isNullable)
    }
    check(!serializer<RequiredHolder>().descriptor.isElementOptional(0))
    check(!serializer<RequiredHolder>().descriptor.getElementDescriptor(0).isNullable)
    check(serializer<BlockedRoot>().descriptor.serialName != serializer<PeerRoot>().descriptor.serialName)
    val values = listOf("null", "true", "false", "0", "1.5", "\"text\"", "[]", "{}", "{\"nested\":[null]}")
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val optional = json.decodeFromString<OptionalHolder>("{}")
        check(optional.blocked == null && optional.alias == null && optional.narrow == null)
        check(json.encodeToString(optional) == "{}")
        check(json.encodeToString(OptionalHolder(blocked = null, alias = null, narrow = null)) == "{}")
        check(json.encodeToString(TwinHolder()) == "{}")
        check(json.encodeToString(json.decodeFromString<ReservedHolder>("{}")) == "{}")
        check(runCatching { json.decodeFromString<RequiredHolder>("{}") }.exceptionOrNull() is SerializationException)
        for (value in values) {
            for (field in listOf("blocked", "alias", "narrow")) {
                val failure = runCatching { json.decodeFromString<OptionalHolder>("{\"$field\":$value}") }.exceptionOrNull()
                check(failure is SerializationException && failure.message == "A false field must be absent.")
            }
            check(runCatching { json.decodeFromString<RequiredHolder>("{\"blocked\":$value}") }.exceptionOrNull() is SerializationException)
            for (field in listOf("absentSerializer", "companion", "serializer")) {
                check(runCatching { json.decodeFromString<ReservedHolder>("{\"$field\":$value}") }.exceptionOrNull() is SerializationException)
            }
            if (value != "null") check(runCatching { json.decodeFromString<NullableFields>("{\"required\":$value}") }.exceptionOrNull() is SerializationException)
        }
        // Shared required nullable fields retain the established default-null policy.
        val nullable = json.decodeFromString<NullableFields>("{\"required\":null,\"optional\":null}")
        check(nullable.required == null && nullable.optional == null)
        check(json.decodeFromString<NullableFields>("{}").required == null)
        val fields = json.parseToJsonElement(json.encodeToString(nullable)).jsonObject
        check(fields.size == if (defaults && nulls) 2 else 0)
        println("FalseReferenceFields defaults=$defaults nulls=$nulls actual-required/strict-optional/nullable-alias/nominal-codec passed")
    }
}

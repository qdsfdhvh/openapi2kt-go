@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.KSerializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import org.example.nestednativeenum_models.models.HolderDto as UncheckedHolder
import org.example.nestednativeenum_models.models.RequiredNullableDto as UncheckedNullable
import org.example.nestednativeenum_models.models.RequiredTreesDto as UncheckedRequired
import org.example.nestednativeenum_models_checked.models.HolderDto as CheckedHolder
import org.example.nestednativeenum_models_checked.models.RequiredNullableDto as CheckedNullable
import org.example.nestednativeenum_models_checked.models.RequiredTreesDto as CheckedRequired

private fun <T> nativeModelTrees(
    scalar: T, rows: List<List<T?>?>?, listMap: List<Map<String, T?>?>?,
    mapList: Map<String, List<T?>?>?, mapMap: Map<String, Map<String, T?>?>?,
    mixed: List<Map<String, List<Map<String, T>>>>?
) {
    check(rows == listOf(listOf(scalar, null), null, emptyList()))
    check(listMap == listOf(mapOf("\$ref" to scalar, "a/~" to null), null, emptyMap()))
    check(mapList == mapOf("a" to listOf(scalar, null), "nil" to null, "empty" to emptyList()))
    check(mapMap == mapOf("a" to mapOf("\$ref" to scalar, "a/~" to null), "nil" to null, "empty" to emptyMap()))
    check(mixed == listOf(mapOf("\$ref" to listOf(mapOf("a/~" to scalar)))))
}

private fun nativeModelFresh(left: Any?, right: Any?) {
    check(left == right)
    when (left) {
        is List<*> -> { check(left !== right); right as List<*>; left.indices.forEach { nativeModelFresh(left[it], right[it]) } }
        is Map<*, *> -> { check(left !== right); right as Map<*, *>; left.keys.forEach { nativeModelFresh(left[it], right[it]) } }
    }
}

private fun nativeModelReject(rule: String, action: () -> Unit) {
    val failure = runCatching(action).exceptionOrNull()
    check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains(rule) }) { "Expected $rule, got $failure" }
}

private fun <T> nativeModelProfile(
    checked: Boolean, serializer: KSerializer<T>, create: () -> T,
    fields: (T) -> List<Any?>, abi: (T) -> Unit,
    constructors: List<Pair<String, () -> T>>, copies: List<Pair<String, (T) -> T>>
) {
    val value = create()
    abi(value)
    nativeModelFresh(fields(value), fields(create()))
    check(!serializer.descriptor.isElementOptional(serializer.descriptor.getElementIndex("required")))
    for ((rule, construct) in constructors) {
        if (checked) nativeModelReject(rule) { construct() } else construct()
    }
    for ((rule, copy) in copies) {
        if (checked) nativeModelReject(rule) { copy(value) } else copy(value)
    }
    val unknown = listOf(
        """{"required":[[2]],"intRows":[[9]]}""" to "enum",
        """{"required":[[2]],"longListMap":[{"a":9}]}""" to "enum",
        """{"required":[[2]],"floatMapList":{"a":[9.0]}}""" to "enum",
        """{"required":[[2]],"doubleMapMap":{"a":{"b":9.0}}}""" to "enum",
        """{"required":[[2]],"boolMixed":[{"a":[{"b":false}]}]}""" to "enum",
        """{"required":[[2]],"intRows":[[1]]}""" to "minimum",
        """{"required":[[2]],"longListMap":[{"a":1}]}""" to "minimum",
        """{"required":[[2]],"floatMapList":{"a":[1.5]}}""" to "maximum",
        """{"required":[[2]],"doubleMapMap":{"a":{"b":1.5}}}""" to "maximum",
        """{"required":[[2]],"intRows":[null,null,null,null]}""" to "maxItems",
        """{"required":[[2]],"longListMap":[{"a":null,"b":null,"c":null}]}""" to "maxProperties"
    )
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val missing = json.decodeFromString(serializer, """{"required":[[2]]}""")
        nativeModelFresh(fields(value), fields(missing))
        val encoded = json.encodeToJsonElement(serializer, missing) as JsonObject
        for (native in listOf("int", "long", "float", "double", "bool")) for (shape in listOf("Rows", "ListMap", "MapList", "MapMap", "Mixed")) check(encoded.containsKey(native + shape) == defaults)
        if (defaults) {
            check(encoded["intRows"] == json.parseToJsonElement("[[2,null],null,[]]"))
            check(encoded["longListMap"] == json.parseToJsonElement("""[{"${'$'}ref":2,"a/~":null},null,{}]"""))
            check(encoded["floatMapList"] == json.parseToJsonElement("""{"a":[1.0,null],"empty":[],"nil":null}"""))
            check(encoded["doubleMapMap"] == json.parseToJsonElement("""{"a":{"${'$'}ref":1.0,"a/~":null},"empty":{},"nil":null}"""))
            check(encoded["boolMixed"] == json.parseToJsonElement("""[{"${'$'}ref":[{"a/~":true}]}]"""))
        }
        check(fields(json.decodeFromJsonElement(serializer, encoded)) == fields(missing))
        val clear = json.decodeFromString(serializer, """{"required":[[2]],"intRows":null,"longListMap":null,"floatMapList":null,"doubleMapMap":null,"boolMixed":null}""")
        val clearWire = json.encodeToJsonElement(serializer, clear) as JsonObject
        for (field in listOf("intRows", "longListMap", "floatMapList", "doubleMapMap", "boolMixed")) check(clearWire[field] == if (nulls) kotlinx.serialization.json.JsonNull else null)
        val replacement = json.decodeFromString(serializer, """{"required":[[2]],"intRows":[[3]],"longListMap":[{"new":3}],"floatMapList":{"new":[0.5]},"doubleMapMap":{"new":{"new":0.5}},"boolMixed":[{"new":[{"new":true}]}]}""")
        val replaced = json.encodeToJsonElement(serializer, replacement) as JsonObject
        check(replaced["intRows"] == json.parseToJsonElement("[[3]]"))
        check(replaced["longListMap"] == json.parseToJsonElement("""[{"new":3}]"""))
        check(replaced["floatMapList"] == json.parseToJsonElement("""{"new":[0.5]}"""))
        check(replaced["doubleMapMap"] == json.parseToJsonElement("""{"new":{"new":0.5}}"""))
        check(replaced["boolMixed"] == json.parseToJsonElement("""[{"new":[{"new":true}]}]"""))
        check(fields(json.decodeFromJsonElement(serializer, replaced)) == fields(replacement))
        for ((wire, rule) in unknown) {
            if (checked) nativeModelReject(rule) { json.decodeFromString(serializer, wire) }
            else {
                val future = json.decodeFromString(serializer, wire)
                val futureWire = json.encodeToJsonElement(serializer, future) as JsonObject
                for ((field, literal) in (json.parseToJsonElement(wire) as JsonObject)) check(futureWire[field] == literal)
                check(fields(json.decodeFromJsonElement(serializer, futureWire)) == fields(future))
            }
        }
        check(runCatching { json.decodeFromString(serializer, "{}") }.isFailure)
        check(runCatching { json.decodeFromString(serializer, """{"required":[[null]]}""") }.isFailure)
    }
}

internal fun nestedNativeEnumOwnDefaultsModelContracts() {
    val required = listOf(listOf(2))
    nativeModelProfile(false, UncheckedHolder.serializer(),
        { UncheckedHolder(required = required, kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow") },
        { listOf(it.intRows, it.longRows, it.floatRows, it.doubleRows, it.boolRows, it.intListMap, it.longListMap, it.floatListMap, it.doubleListMap, it.boolListMap, it.intMapList, it.longMapList, it.floatMapList, it.doubleMapList, it.boolMapList, it.intMapMap, it.longMapMap, it.floatMapMap, it.doubleMapMap, it.boolMapMap, it.intMixed, it.longMixed, it.floatMixed, it.doubleMixed, it.boolMixed, it.longEnds, it.rounded, it.empty, it.emptyMap, it.nil) },
        {
            nativeModelTrees<Int>(2, it.intRows, it.intListMap, it.intMapList, it.intMapMap, it.intMixed)
            nativeModelTrees<Long>(2L, it.longRows, it.longListMap, it.longMapList, it.longMapMap, it.longMixed)
            nativeModelTrees<Float>(1.0f, it.floatRows, it.floatListMap, it.floatMapList, it.floatMapMap, it.floatMixed)
            nativeModelTrees<Double>(1.0, it.doubleRows, it.doubleListMap, it.doubleMapList, it.doubleMapMap, it.doubleMixed)
            nativeModelTrees<Boolean>(true, it.boolRows, it.boolListMap, it.boolMapList, it.boolMapMap, it.boolMixed)
            check(it.longEnds == mapOf("type" to listOf(Long.MIN_VALUE, Long.MAX_VALUE)) && it.rounded == listOf(mapOf("type" to 1.0f)))
        },
        listOf(
            "enum" to { UncheckedHolder(required = required, intRows = listOf(listOf(9))) },
            "enum" to { UncheckedHolder(required = required, longListMap = listOf(mapOf("a" to 9L))) },
            "enum" to { UncheckedHolder(required = required, floatMapList = mapOf("a" to listOf(9.0f))) },
            "enum" to { UncheckedHolder(required = required, doubleMapMap = mapOf("a" to mapOf("b" to 9.0))) },
            "enum" to { UncheckedHolder(required = required, boolMixed = listOf(mapOf("a" to listOf(mapOf("b" to false))))) }
        ),
        listOf(
            "enum" to { v: UncheckedHolder -> v.copy(intRows = listOf(listOf(9))) },
            "enum" to { v: UncheckedHolder -> v.copy(longListMap = listOf(mapOf("a" to 9L))) },
            "enum" to { v: UncheckedHolder -> v.copy(floatMapList = mapOf("a" to listOf(9.0f))) },
            "enum" to { v: UncheckedHolder -> v.copy(doubleMapMap = mapOf("a" to mapOf("b" to 9.0))) },
            "enum" to { v: UncheckedHolder -> v.copy(boolMixed = listOf(mapOf("a" to listOf(mapOf("b" to false))))) }
        ))
    nativeModelProfile(true, CheckedHolder.serializer(),
        { CheckedHolder(required = required, kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow") },
        { listOf(it.intRows, it.longRows, it.floatRows, it.doubleRows, it.boolRows, it.intListMap, it.longListMap, it.floatListMap, it.doubleListMap, it.boolListMap, it.intMapList, it.longMapList, it.floatMapList, it.doubleMapList, it.boolMapList, it.intMapMap, it.longMapMap, it.floatMapMap, it.doubleMapMap, it.boolMapMap, it.intMixed, it.longMixed, it.floatMixed, it.doubleMixed, it.boolMixed, it.longEnds, it.rounded, it.empty, it.emptyMap, it.nil) },
        {
            nativeModelTrees<Int>(2, it.intRows, it.intListMap, it.intMapList, it.intMapMap, it.intMixed)
            nativeModelTrees<Long>(2L, it.longRows, it.longListMap, it.longMapList, it.longMapMap, it.longMixed)
            nativeModelTrees<Float>(1.0f, it.floatRows, it.floatListMap, it.floatMapList, it.floatMapMap, it.floatMixed)
            nativeModelTrees<Double>(1.0, it.doubleRows, it.doubleListMap, it.doubleMapList, it.doubleMapMap, it.doubleMixed)
            nativeModelTrees<Boolean>(true, it.boolRows, it.boolListMap, it.boolMapList, it.boolMapMap, it.boolMixed)
            check(it.longEnds == mapOf("type" to listOf(Long.MIN_VALUE, Long.MAX_VALUE)) && it.rounded == listOf(mapOf("type" to 1.0f)))
        },
        listOf(
            "enum" to { CheckedHolder(required = required, intRows = listOf(listOf(9))) },
            "enum" to { CheckedHolder(required = required, longListMap = listOf(mapOf("a" to 9L))) },
            "enum" to { CheckedHolder(required = required, floatMapList = mapOf("a" to listOf(9.0f))) },
            "enum" to { CheckedHolder(required = required, doubleMapMap = mapOf("a" to mapOf("b" to 9.0))) },
            "enum" to { CheckedHolder(required = required, boolMixed = listOf(mapOf("a" to listOf(mapOf("b" to false))))) },
            "minimum" to { CheckedHolder(required = required, longRows = listOf(listOf(1L))) },
            "maximum" to { CheckedHolder(required = required, floatListMap = listOf(mapOf("a" to 1.5f))) },
            "minItems" to { CheckedHolder(required = listOf(emptyList())) },
            "maxItems" to { CheckedHolder(required = required, intRows = List(4) { null }) },
            "maxProperties" to { CheckedHolder(required = required, doubleMapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null))) }
        ),
        listOf(
            "enum" to { v: CheckedHolder -> v.copy(intRows = listOf(listOf(9))) },
            "enum" to { v: CheckedHolder -> v.copy(longListMap = listOf(mapOf("a" to 9L))) },
            "enum" to { v: CheckedHolder -> v.copy(floatMapList = mapOf("a" to listOf(9.0f))) },
            "enum" to { v: CheckedHolder -> v.copy(doubleMapMap = mapOf("a" to mapOf("b" to 9.0))) },
            "enum" to { v: CheckedHolder -> v.copy(boolMixed = listOf(mapOf("a" to listOf(mapOf("b" to false))))) },
            "minimum" to { v: CheckedHolder -> v.copy(longRows = listOf(listOf(1L))) },
            "maximum" to { v: CheckedHolder -> v.copy(floatListMap = listOf(mapOf("a" to 1.5f))) },
            "minItems" to { v: CheckedHolder -> v.copy(required = listOf(emptyList())) },
            "maxItems" to { v: CheckedHolder -> v.copy(intRows = List(4) { null }) },
            "maxProperties" to { v: CheckedHolder -> v.copy(doubleMapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null))) }
        ))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(runCatching { json.decodeFromString(UncheckedNullable.serializer(), "{}") }.isSuccess == !nulls)
        check(runCatching { json.decodeFromString(CheckedNullable.serializer(), "{}") }.isSuccess == !nulls)
        check(runCatching { json.decodeFromString(UncheckedRequired.serializer(), "{}") }.isFailure)
        check(runCatching { json.decodeFromString(CheckedRequired.serializer(), "{}") }.isFailure)
    }
    println("Nested native enum models: native5 x LL/LM/ML/MM/LMLM, unchecked/checked, fourJson/defaults/fresh/presence/constructor/copy/serde passed in EnumContextContracts")
}

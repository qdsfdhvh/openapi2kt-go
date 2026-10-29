package org.example.platform

import kotlin.time.Instant
import kotlin.uuid.Uuid
import kotlinx.datetime.LocalDate
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.platformenumbounds.models.Record
import org.example.platformenumbounds.models.Consumer as EnumDefaultConsumer
import org.example.platformenumbounds.models.`RecordAlias default`
import org.example.platformenumbounds.models.`MaybeAlias default`
import org.example.platformenumbounds.models.EnumTreeAlias
import org.example.platformenumbounds.models.EnumTreeInnerValue
import org.example.platformenumbounds.models.`EnumTree default`
import org.example.platformenumbounds.models.`EnumTreeAlias default`
import org.example.platformenumbounds.models.`MaybeEnumTree default`
import org.example.platformenumbounds.models.BorrowedWord
import org.example.platformenumbounds.models.BorrowedTree
import org.example.platformenumbounds.models.`BorrowedWord default`
import org.example.platformenumbounds.models.`MaybeBorrowedWord default`
import org.example.platformenumbounds.models.FactoryConsumer
import org.example.platformenumbounds.models.ChildAnnotationHolder
import org.example.platformenumbounds.models.ChildAnnotationFactory
import org.example.platformenumbounds.models.ChildAnnotationEmptyFactory
import org.example.platformenumbounds.models.RootWord
import org.example.platformenumbounds.models.RootWordTreeAlias
import org.example.platformenumbounds.models.RootWordTreeHolder
import org.example.platformenumbounds.models.`RootWord default`
import org.example.platformenumbounds.models.`RootWordAlias default`
import org.example.platformenumbounds.models.`MaybeRootWord default`
import org.example.platformenumbounds.models.`RootWordTree default`
import org.example.platformenumbounds.models.`RootWordTreeAlias default`
import org.example.platformenumbounds.models.`MaybeRootWordTree default`
import org.example.platformenumbounds.models.`FactoryInt default`
import org.example.platformenumbounds.models.`FactoryIntAlias default`
import org.example.platformenumbounds.models.`FactoryIntRows default`
import org.example.platformenumbounds.models.`FactoryIntRowsAlias default`
import org.example.platformenumbounds.models.`MaybeFactoryIntRows default`
import org.example.platformenumbounds.models.`FactoryLong default`
import org.example.platformenumbounds.models.`FactoryLongAlias default`
import org.example.platformenumbounds.models.`FactoryLongRows default`
import org.example.platformenumbounds.models.`FactoryLongRowsAlias default`
import org.example.platformenumbounds.models.`MaybeFactoryLongRows default`
import org.example.platformenumbounds.models.`FactoryFloat default`
import org.example.platformenumbounds.models.`FactoryFloatAlias default`
import org.example.platformenumbounds.models.`FactoryFloatRows default`
import org.example.platformenumbounds.models.`FactoryFloatRowsAlias default`
import org.example.platformenumbounds.models.`MaybeFactoryFloatRows default`
import org.example.platformenumbounds.models.`FactoryDouble default`
import org.example.platformenumbounds.models.`FactoryDoubleAlias default`
import org.example.platformenumbounds.models.`FactoryDoubleRows default`
import org.example.platformenumbounds.models.`FactoryDoubleRowsAlias default`
import org.example.platformenumbounds.models.`MaybeFactoryDoubleRows default`
import org.example.platformenumbounds.models.`FactoryBoolean default`
import org.example.platformenumbounds.models.`FactoryBooleanAlias default`
import org.example.platformenumbounds.models.`FactoryBooleanRows default`
import org.example.platformenumbounds.models.`FactoryBooleanRowsAlias default`
import org.example.platformenumbounds.models.`MaybeFactoryBooleanRows default`

internal fun enumFieldBoundsContracts(json: Json) {
    val value = Record(required = Record.Required.ONE)
    check(value.state?.serialName == "one" && value.nullableWord == null)
    check(value.words?.single()?.serialName == "one" && value.labels?.get("first")?.serialName == "one")
    check(value.words !== Record(required = value.required).words)
    check(value.trees?.first() == null && value.trees?.get(1)?.get("a/~") == Record.TreesInnerValue.ONE)
    check(value.trees !== Record(required = value.required).trees)
    check(value.rootTrees?.single()?.get("field") == EnumTreeInnerValue.ONE)
    check(value.borrowedWord == Record.State.ONE && value.borrowedWords?.single() == Record.State.ONE)
    check(value.borrowedTree?.get("field")?.get("a") == Record.BorrowableTreeValueValue.ONE)
    check(value.unannotatedWord == null && value.borrowableTree == null)
    check(value.borrowedWords !== Record(required = value.required).borrowedWords)
    check(value.borrowedTree !== Record(required = value.required).borrowedTree)
    check(`BorrowedWord default`() == Record.State.TWO && `MaybeBorrowedWord default`() == null)
    check(json.decodeFromString<BorrowedWord>("\"x\"") == Record.State.X)
    check(json.decodeFromString<BorrowedTree>("""{"raw":{"a":"x"}}""")["raw"]?.get("a") == Record.BorrowableTreeValueValue.X)
    val borrowed = json.decodeFromString<Record>("""{"required":"one","borrowedWord":"future","borrowedWords":["future"],"borrowedTree":{"a":{"b":"future"}},"unannotatedWord":"future"}""")
    check(borrowed.borrowedWord == Record.State.Unrecognized("future"))
    check(borrowed.borrowedTree?.get("a")?.get("b") == Record.BorrowableTreeValueValue.Unrecognized("future"))
    check(json.decodeFromString<Record>(json.encodeToString(borrowed)) == borrowed)
    check(json.decodeFromString<Record>("""{"required":"one","borrowedWord":null,"borrowedWords":null,"borrowedTree":null}""").let { it.borrowedWord == null && it.borrowedWords == null && it.borrowedTree == null })
    check(runCatching { Record(required = value.required, borrowedWords = listOf(Record.State.ONE, Record.State.Unrecognized("one"))) }.isFailure)
    check(runCatching { json.decodeFromString<Record>("""{"required":"one","borrowedWords":["one","one"]}""") }.isFailure)
    check(`EnumTree default`()?.single()?.get("root") == EnumTreeInnerValue.ONE)
    check(`EnumTreeAlias default`().single()?.get("alias") == EnumTreeInnerValue.TWO)
    check(`MaybeEnumTree default`() == null && `EnumTree default`() !== `EnumTree default`())
    check(json.decodeFromString<EnumTreeAlias>("""[{"raw":"x"}]""").single()?.get("raw") == EnumTreeInnerValue.X)
    check(Record.default() == value && Record.default().words !== Record.default().words)
    check(`RecordAlias default`().required.serialName == "two" && `RecordAlias default`().state?.serialName == "two")
    check(`RecordAlias default`().trees?.single()?.get("alias")?.serialName == "two")
    check(`RecordAlias default`().rootTrees?.single()?.get("alias") == EnumTreeInnerValue.TWO)
    check(`MaybeAlias default`() == null)
    val consumer = EnumDefaultConsumer()
    check(consumer.value?.required?.serialName == "one" && consumer.value?.state?.serialName == "one")
    check(consumer.rows?.single()?.required?.serialName == "one" && consumer.rows?.single()?.state?.serialName == "two")
    check(consumer.rows?.single()?.rootTrees?.single()?.get("outer") == EnumTreeInnerValue.TWO)
    check(consumer.unannotated == null && consumer.rows !== EnumDefaultConsumer().rows)
    check(json.decodeFromString<EnumDefaultConsumer>(json.encodeToString(consumer)) == consumer)
    check(`RootWord default`() == RootWord.ONE && `RootWordAlias default`() == RootWord.TWO)
    check(`MaybeRootWord default`() == null)
    check(`RootWordTree default`()["factory"] == listOf(RootWord.ONE))
    check(`RootWordTreeAlias default`()["alias"] == listOf(RootWord.TWO))
    check(`MaybeRootWordTree default`() == null)
    check(`RootWordTree default`() !== `RootWordTree default`())
    check(`RootWordTree default`()["factory"] !== `RootWordTree default`()["factory"])
    check(RootWordTreeHolder.default().tree?.get("outer") == listOf(RootWord.TWO))
    val factoryConsumer = FactoryConsumer()
    check(factoryConsumer.wordTree?.get("field") == listOf(RootWord.ONE))
    check(factoryConsumer.intRows == listOf(1))
    check(factoryConsumer.unannotatedWordTree == null && factoryConsumer.unannotatedIntRows == null)
    check(factoryConsumer.wordTree !== FactoryConsumer().wordTree && factoryConsumer.intRows !== FactoryConsumer().intRows)
    check(json.decodeFromString<FactoryConsumer>(json.encodeToString(factoryConsumer)) == factoryConsumer)
    check(json.decodeFromString<RootWordTreeAlias>("""{"raw":["future"]}""")["raw"] == listOf(RootWord.Unrecognized("future")))
    check(`FactoryInt default`() == 1 && `FactoryIntAlias default`() == 2)
    check(`FactoryIntRows default`() == listOf(1) && `FactoryIntRowsAlias default`() == listOf(2))
    check(`MaybeFactoryIntRows default`() == null && `FactoryIntRows default`() !== `FactoryIntRows default`())
    check(`FactoryLong default`() == 1L && `FactoryLongAlias default`() == 2L)
    check(`FactoryLongRows default`() == listOf(1L) && `FactoryLongRowsAlias default`() == listOf(2L))
    check(`MaybeFactoryLongRows default`() == null && `FactoryLongRows default`() !== `FactoryLongRows default`())
    check(`FactoryFloat default`() == 1.0f && `FactoryFloatAlias default`() == 2.0f)
    check(`FactoryFloatRows default`() == listOf(1.0f) && `FactoryFloatRowsAlias default`() == listOf(2.0f))
    check(`MaybeFactoryFloatRows default`() == null && `FactoryFloatRows default`() !== `FactoryFloatRows default`())
    check(`FactoryDouble default`() == 1.0 && `FactoryDoubleAlias default`() == 2.0)
    check(`FactoryDoubleRows default`() == listOf(1.0) && `FactoryDoubleRowsAlias default`() == listOf(2.0))
    check(`MaybeFactoryDoubleRows default`() == null && `FactoryDoubleRows default`() !== `FactoryDoubleRows default`())
    check(`FactoryBoolean default`() && !`FactoryBooleanAlias default`())
    check(`FactoryBooleanRows default`() == listOf(true) && `FactoryBooleanRowsAlias default`() == listOf(false))
    check(`MaybeFactoryBooleanRows default`() == null && `FactoryBooleanRows default`() !== `FactoryBooleanRows default`())
    val childAnnotations = ChildAnnotationHolder()
    check(childAnnotations.inlineWords == null && childAnnotations.refWords == null)
    check(childAnnotations.counts == null && childAnnotations.labels == null)
    check(childAnnotations.ids == null && childAnnotations.dates == null && childAnnotations.instants == null)
    check(json.decodeFromString<ChildAnnotationHolder>("{}") == childAnnotations)
    val suppliedChildren = json.decodeFromString<ChildAnnotationHolder>(
        """{"inlineWords":["one"],"refWords":["two"],"counts":[1],"labels":{"a":"value"},"ids":["550E8400E29B41D4A716446655440000"],"dates":{"day":"2024-02-29"},"instants":["2024-02-29T12:34:56+02:30",null]}"""
    )
    check(suppliedChildren.inlineWords?.single()?.serialName == "one")
    check(suppliedChildren.refWords?.single() == RootWord.TWO)
    check(suppliedChildren.counts == listOf(1) && suppliedChildren.labels == mapOf("a" to "value"))
    val id = Uuid.parse("550e8400-e29b-41d4-a716-446655440000")
    val instant = Instant.parse("2024-02-29T10:04:56Z")
    check(suppliedChildren.ids == listOf(id))
    check(suppliedChildren.dates == mapOf("day" to LocalDate.parse("2024-02-29")))
    check(suppliedChildren.instants == listOf(instant, null))
    check(json.decodeFromString<ChildAnnotationHolder>(json.encodeToString(suppliedChildren)) == suppliedChildren)
    check(runCatching { json.decodeFromString<ChildAnnotationHolder>("""{"ids":["invalid"]}""") }.isFailure)
    check(runCatching { json.decodeFromString<ChildAnnotationHolder>("""{"dates":{"day":"2023-02-29"}}""") }.isFailure)
    check(runCatching { json.decodeFromString<ChildAnnotationHolder>("""{"instants":["invalid"]}""") }.isFailure)
    val childFactory = ChildAnnotationFactory.default()
    check(childFactory.counts == listOf(2) && childFactory.labels == mapOf("first" to "field"))
    check(childFactory.days == listOf(LocalDate.parse("2024-03-01")))
    check(childFactory.ids == mapOf("chosen" to id) && childFactory.instants == listOf(instant, null))
    check(childFactory.counts !== ChildAnnotationFactory.default().counts)
    check(childFactory.days !== ChildAnnotationFactory.default().days)
    val emptyChildFactory = ChildAnnotationEmptyFactory.default()
    check(emptyChildFactory.counts == null && emptyChildFactory.days == null)
    check(emptyChildFactory.ids == null && emptyChildFactory.instants == null)
    check(json.decodeFromString<ChildAnnotationFactory>(json.encodeToString(childFactory)) == childFactory)
    check(json.decodeFromString<Record>(json.encodeToString(value)) == value)
    check(json.decodeFromString<Record>("""{"required":"one"}""") == value)
    check(json.decodeFromString<Record>("""{"required":"one","trees":null}""").trees == null)
    val nested = json.decodeFromString<Record>("""{"required":"one","trees":[null,{"a":"future","nil":null}]}""")
    check(nested.trees?.get(1)?.get("a") == Record.TreesInnerValue.Unrecognized("future"))
    check(json.decodeFromString<Record>(json.encodeToString(nested)) == nested)
    check(json.decodeFromString<Record.TreesInnerValue>("\"x\"") == Record.TreesInnerValue.X)
    check(runCatching { Record(required = value.required, trees = listOf(null, null)) }.isFailure)
    check(runCatching { Record(required = value.required, trees = listOf(mapOf("a" to Record.TreesInnerValue.ONE), mapOf("a" to Record.TreesInnerValue.Unrecognized("one")))) }.isFailure)
    check(runCatching { json.decodeFromString<Record>("""{"required":"one","trees":[{"a":"one","b":"two"},{"b":"two","a":"one"}]}""") }.isFailure)
    check(runCatching { json.decodeFromString<Record>("{}") }.isFailure)
    // Vocabulary codecs stay open; the DTO use imposes the bounds.
    check(json.decodeFromString<Record.State>("\"x\"") == Record.State.X)
    check(Record.State.fromSerialName("future").serialName == "future")
    val future = json.decodeFromString<Record>("""{"required":"future","state":"future","nullableWord":"future","words":["future"],"labels":{"first":"future"}}""")
    check(future.required.serialName == "future" && future.state?.serialName == "future")
    check(json.decodeFromString<Record>(json.encodeToString(future)) == future)
    check(json.decodeFromString<Record>("""{"required":"one","state":null,"nullableWord":null}""").let { it.state == null && it.nullableWord == null })
    for (bad in listOf("x", "UPPER", "toolong")) {
        check(runCatching { Record(required = Record.Required.fromSerialName(bad)) }.isFailure)
        check(runCatching { Record(required = value.required, state = Record.State.fromSerialName(bad)) }.isFailure)
        check(runCatching { Record(required = value.required, borrowedWord = Record.State.fromSerialName(bad)) }.isFailure)
        check(runCatching { Record(required = value.required, borrowedTree = mapOf("a" to mapOf("b" to Record.BorrowableTreeValueValue.fromSerialName(bad)))) }.isFailure)
        for (field in listOf("borrowedWord", "unannotatedWord")) {
            check(runCatching { json.decodeFromString<Record>("""{"required":"one","$field":"$bad"}""") }.isFailure)
        }
        check(runCatching { json.decodeFromString<Record>("""{"required":"one","borrowedWords":["$bad"]}""") }.isFailure)
        for (field in listOf("borrowableTree", "borrowedTree")) {
            check(runCatching { json.decodeFromString<Record>("""{"required":"one","$field":{"a":{"b":"$bad"}}}""") }.isFailure)
        }
        check(runCatching { Record(required = value.required, trees = listOf(mapOf("a" to Record.TreesInnerValue.fromSerialName(bad)))) }.isFailure)
        check(runCatching { Record(required = value.required, rootTrees = listOf(mapOf("a" to EnumTreeInnerValue.fromSerialName(bad)))) }.isFailure)
        check(runCatching { json.decodeFromString<Record>("""{"required":"one","rootTrees":[{"a":"$bad"}]}""") }.isFailure)
        check(runCatching { json.decodeFromString<Record>("""{"required":"one","trees":[null,{"a":"$bad"}]}""") }.isFailure)
        for (field in listOf("required", "state")) {
            val payload = if (field == "required") """{"required":"$bad"}""" else """{"required":"one","state":"$bad"}"""
            check(runCatching { json.decodeFromString<Record>(payload) }.isFailure)
        }
        check(runCatching { json.decodeFromString<Record>("""{"required":"one","words":["$bad"]}""") }.isFailure)
        check(runCatching { json.decodeFromString<Record>("""{"required":"one","labels":{"first":"$bad"}}""") }.isFailure)
    }
    check(runCatching { json.decodeFromString<Record>("""{"required":"one","state":7}""") }.isFailure)
}

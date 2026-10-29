package org.example.platform

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

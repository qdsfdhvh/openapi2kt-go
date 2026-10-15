package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.*
import org.example.object_unions.models.Envelope
import org.example.object_unions.models.Shared
import org.example.object_unions_suffix.models.EnvelopeDto
import org.example.object_union_limits.models.WideEnvelope

fun main() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val fallbackRaw = json.parseToJsonElement("{\"kind\":\"future\",\"extra\":true}")
        val fallback = json.decodeFromJsonElement(Envelope.Fallbacks.serializer(), fallbackRaw) as Envelope.UnknownFallbacks
        check(fallback.values.isEmpty() && fallback.f32 == 0.0f && !fallback.flag && fallback.long == 0L && fallback.label == null && fallback.maybe == null)
        check(runCatching { fallback.raw }.exceptionOrNull() is IllegalStateException)
        check(json.encodeToJsonElement(Envelope.Fallbacks.serializer(), fallback) == fallbackRaw)
        val knownFallback = Envelope.Fallbacks.A(values = listOf(1.25f), raw = JsonNull, f32 = 2.5f, flag = true, long = Long.MIN_VALUE, label = null, maybe = Long.MAX_VALUE)
        check(json.decodeFromString(Envelope.Fallbacks.serializer(), json.encodeToString(Envelope.Fallbacks.serializer(), knownFallback)) == knownFallback)
        check(json.decodeFromString(Envelope.RefTags.serializer(), "{\"kind\":\"alpha\"}") === Envelope.RefTags.Alpha)
        check(json.decodeFromString(Envelope.RefTags.serializer(), "{\"kind\":\"beta\"}") === Envelope.RefTags.Beta)
        val note = Envelope.Tagged.Note(id = "a", count = 3, payload = Envelope.Tagged.Note.Payload(active = true))
        val tagged = json.encodeToJsonElement(Envelope.Tagged.serializer(), note).jsonObject
        check(tagged.getValue("kind").jsonPrimitive.content == "note")
        check("type" !in tagged)
        check(json.decodeFromJsonElement(Envelope.Tagged.serializer(), tagged) == note)
        val missing = JsonObject(tagged - "count")
        if (nulls) check(runCatching { json.decodeFromJsonElement(Envelope.Tagged.serializer(), missing) }.exceptionOrNull() is SerializationException)
        else check((json.decodeFromJsonElement(Envelope.Tagged.serializer(), missing) as Envelope.Tagged.Note).count == null)
        for (tag in listOf("batch", "batch-more")) {
            val batch = json.decodeFromString(Envelope.Tagged.serializer(), "{\"kind\":\"$tag\",\"id\":\"b\",\"values\":[\"x\"]}") as Envelope.Tagged.Batch
            check(batch.kind == tag && batch.count == null)
            check(json.decodeFromString(Envelope.Tagged.serializer(), json.encodeToString(Envelope.Tagged.serializer(), batch)) == batch)
        }
        val raw = json.parseToJsonElement("{\"kind\":\"future\",\"extra\":[1,{\"a/b~c\":null}],\"object\":{\"v\":true}}")
        val unknown = json.decodeFromJsonElement(Envelope.Tagged.serializer(), raw) as Envelope.UnknownTagged
        check(unknown.rawElement == raw && unknown.id == "" && unknown.kind == "" && unknown.count == null)
        check(json.encodeToJsonElement(Envelope.Tagged.serializer(), unknown) == raw)
        val left = json.decodeFromString(Envelope.Presence.serializer(), "{\"id\":\"left\",\"left\":9223372036854775807}") as Envelope.Presence.Variant1
        val right = json.decodeFromString(Envelope.Presence.serializer(), "{\"id\":\"right\",\"right\":false}") as Envelope.Presence.Variant2
        check(left.left == Long.MAX_VALUE && !right.right)
        check(json.decodeFromString(Envelope.Presence.serializer(), json.encodeToString(Envelope.Presence.serializer(), left)) == left)
        check(json.decodeFromString(Envelope.Presence.serializer(), "{\"id\":\"unknown\"}") is Envelope.UnknownPresence)
        check(runCatching { json.decodeFromString(Envelope.Presence.serializer(), "{\"id\":\"bad\",\"left\":null}") }.exceptionOrNull() is SerializationException)
        val fixed = json.decodeFromString(Envelope.Mixed.serializer(), "{\"kind\":\"fixed\",\"shared\":\"s\"}") as Envelope.Mixed.Fixed
        val body = json.decodeFromString(Envelope.Mixed.serializer(), "{\"body\":{\"id\":\"body\"}}") as Envelope.Mixed.Variant2
        check(fixed.shared == "s" && body.body == Shared(id = "body"))
        val orphan = json.parseToJsonElement("{\"extra\":\"not-required\",\"additional\":7}")
        check((json.decodeFromJsonElement(Envelope.Mixed.serializer(), orphan) as Envelope.UnknownMixed).rawElement == orphan)
        check(json.decodeFromString(Envelope.Singleton.serializer(), "{\"kind\":\"idle\",\"revision\":99,\"extra\":true}") === Envelope.Singleton.Idle)
        val idle = json.encodeToJsonElement(Envelope.Singleton.serializer(), Envelope.Singleton.Idle).jsonObject
        check(idle == buildJsonObject { put("kind", "idle"); put("revision", 2) })
        check(json.decodeFromString(Envelope.Singleton.serializer(), "{\"kind\":\"ready\"}") === Envelope.Singleton.Ready)
        check(json.decodeFromString(Envelope.Singleton.Ready.serializer(), "[1,2]") === Envelope.Singleton.Ready)
        check(json.encodeToJsonElement(Envelope.Singleton.serializer(), Envelope.Singleton.Ready).jsonObject.getValue("kind").jsonPrimitive.content == "ready")
        check(json.decodeFromString(Envelope.Duplicates.serializer(), "{\"kind\":\"same\",\"value\":\"first\"}") is Envelope.Duplicates.Same)
        val noKeyRaw = json.parseToJsonElement("[1,true,{\"future\":null}]")
        val noKey = json.decodeFromJsonElement(Envelope.NoKey.serializer(), noKeyRaw) as Envelope.UnknownNoKey
        check(json.encodeToJsonElement(Envelope.NoKey.serializer(), noKey) == noKeyRaw)
        val envelope = Envelope(tagged = note, presence = left, singleton = Envelope.Singleton.Ready, items = listOf(Envelope.ItemsInner.Dicecheck, Envelope.ItemsInner.V1, null), nested = Envelope.Nested(event = Envelope.Nested.Event.Companion1(message = "nested")))
        check(json.decodeFromString(Envelope.serializer(), json.encodeToString(Envelope.serializer(), envelope)) == envelope)
        val suffix = json.decodeFromJsonElement(EnvelopeDto.Tagged.serializer(), tagged) as EnvelopeDto.Tagged.Note
        check(suffix.payload.active && suffix.id == "a")
        val wideWire = buildJsonObject {
            put("kind", "wide"); put("flag", true)
            for (i in 0 until 119) put("field$i", if (i == 118) Long.MAX_VALUE else i.toLong())
        }
        val wide = json.decodeFromJsonElement(WideEnvelope.Event.serializer(), wideWire) as WideEnvelope.Event.Wide
        check(wide.field118 == Long.MAX_VALUE && wide.flag)
        check(json.encodeToJsonElement(WideEnvelope.Event.serializer(), wide) == wideWire)
    }
    println("Inline object union generated-code contracts passed")
}

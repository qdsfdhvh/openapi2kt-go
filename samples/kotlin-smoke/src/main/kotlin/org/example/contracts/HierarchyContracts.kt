package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.add
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import org.example.hierarchy_unions.models.AlphaNode
import org.example.hierarchy_unions.models.BetaNode
import org.example.hierarchy_unions.models.Channel
import org.example.hierarchy_unions.models.ChannelPhase
import org.example.hierarchy_unions.models.DeepFirst
import org.example.hierarchy_unions.models.DeepInner
import org.example.hierarchy_unions.models.DeepInnerPhase
import org.example.hierarchy_unions.models.DeepMiddleStep
import org.example.hierarchy_unions.models.DeepRoot
import org.example.hierarchy_unions.models.DeepRootKind
import org.example.hierarchy_unions.models.Flat
import org.example.hierarchy_unions.models.FlatTag
import org.example.hierarchy_unions.models.Flux
import org.example.hierarchy_unions.models.Holder
import org.example.hierarchy_unions.models.InferredGroupZone
import org.example.hierarchy_unions.models.InferredRoot
import org.example.hierarchy_unions.models.InferredRootFamily
import org.example.hierarchy_unions.models.PinLeaf
import org.example.hierarchy_unions.models.PlainGroup
import org.example.hierarchy_unions.models.PlainLeft
import org.example.hierarchy_unions.models.PlainRoot
import org.example.hierarchy_unions.models.Quiet
import org.example.hierarchy_unions.models.RangeLeaf
import org.example.hierarchy_unions.models.Signal
import org.example.hierarchy_unions.models.SignalFamily
import org.example.hierarchy_unions.models.Status
import org.example.hierarchy_unions.models.UnknownChannel
import org.example.hierarchy_unions.models.UnknownDeepInner
import org.example.hierarchy_unions.models.UnknownFlat
import org.example.hierarchy_unions.models.UnknownInferredGroup
import org.example.hierarchy_unions.models.UnknownSignal
import org.example.hierarchy_unions_suffix.models.SignalDto
import org.example.hierarchy_unions_suffix.models.FlatDto

fun main() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val flux = Flux(phase = ChannelPhase.DAWN, samples = listOf(Long.MIN_VALUE, Long.MAX_VALUE))
        check(flux.family === SignalFamily.STREAM)
        val wire = json.encodeToJsonElement(Signal.serializer(), flux).jsonObject
        check(wire.getValue("family").jsonPrimitive.content == "stream")
        check(wire.getValue("phase").jsonPrimitive.content == "dawn" && "type" !in wire)
        check(json.decodeFromJsonElement(Signal.serializer(), wire) == flux)
        check(json.decodeFromJsonElement(Channel.serializer(), wire) == flux)
        check(json.decodeFromJsonElement(SignalDto.serializer(), wire) is org.example.hierarchy_unions_suffix.models.FluxDto)
        val noon = flux.copy(phase = ChannelPhase.NOON)
        check(json.decodeFromJsonElement(Signal.serializer(), json.encodeToJsonElement(Signal.serializer(), noon)) == noon)
        for (raw in listOf("{\"family\":\"future\",\"extra\":[1,null,{\"a/b~c\":false}]}", "{}")) {
            val element = json.parseToJsonElement(raw)
            val unknown = json.decodeFromJsonElement(Signal.serializer(), element) as UnknownSignal
            check(unknown.family == SignalFamily.Unrecognized(""))
            check(json.encodeToJsonElement(Signal.serializer(), unknown) == element)
        }
        for (phase in listOf("future", "alias-only")) {
            val raw = buildJsonObject { put("family", "stream"); put("phase", phase); put("payload", buildJsonArray { add(9) }) }
            val unknown = json.decodeFromJsonElement(Signal.serializer(), raw) as UnknownChannel
            check(unknown.family == SignalFamily.Unrecognized(""))
            check(unknown.phase == ChannelPhase.Unrecognized(""))
            check(json.encodeToJsonElement(Signal.serializer(), unknown) == raw)
            check(json.encodeToJsonElement(Channel.serializer(), unknown) == raw)
        }
        check(json.decodeFromString(Signal.serializer(), "{\"family\":\"stream\",\"phase\":null}") is UnknownChannel)
        for (bad in listOf("[]", "{\"family\":[]}", "{\"family\":\"stream\",\"phase\":\"dawn\"}")) {
            check(runCatching { json.decodeFromString(Signal.serializer(), bad) }.isFailure)
        }
        val consumed = json.decodeFromString(Signal.serializer(), "{\"family\":\"stream\",\"phase\":\"night\",\"extra\":1}")
        check(consumed === Quiet)
        check(json.encodeToJsonElement(Signal.serializer(), consumed) == buildJsonObject { put("family", "stream"); put("phase", "night") })
        val status = Status(online = false)
        check(json.decodeFromJsonElement(Signal.serializer(), json.encodeToJsonElement(Signal.serializer(), status)) == status)
        val holder = Holder(signal = flux, signals = listOf(status, Quiet))
        check(json.decodeFromString(Holder.serializer(), json.encodeToString(Holder.serializer(), holder)) == holder)
        check(json.decodeFromString(InferredRoot.serializer(), "{\"family\":\"inferred\",\"zone\":\"near\",\"distance\":3}") === PinLeaf)
        val far = json.decodeFromString(InferredRoot.serializer(), "{\"family\":\"inferred\",\"zone\":\"far\",\"distance\":3}") as RangeLeaf
        check(far.zone === InferredGroupZone.FAR && far.distance == 3.0)
        check(json.decodeFromJsonElement(InferredRoot.serializer(), json.encodeToJsonElement(InferredRoot.serializer(), far)) == far)
        val unknownInferred = json.decodeFromString(InferredRoot.serializer(), "{\"family\":\"inferred\",\"zone\":\"remote\"}") as UnknownInferredGroup
        check(unknownInferred.family == InferredRootFamily.Unrecognized(""))
        check(unknownInferred.zone == InferredGroupZone.Unrecognized(""))
        val plain: PlainGroup = PlainLeft(left = "ordinary")
        val plainWire = json.encodeToJsonElement(PlainGroup.serializer(), plain).jsonObject
        check(plainWire.getValue("type").jsonPrimitive.content == "plainleft")
        check(json.decodeFromJsonElement(PlainRoot.serializer(), plainWire) == plain)
        check(runCatching { json.decodeFromString(PlainRoot.serializer(), "{\"family\":\"plain\",\"type\":\"future\"}") }.exceptionOrNull() is SerializationException)
        // The root must preserve the closed intermediate's subtype discriminator.
        val rootPlainWire = json.encodeToJsonElement(PlainRoot.serializer(), plain).jsonObject
        check(rootPlainWire == plainWire)
        check(json.decodeFromJsonElement(PlainRoot.serializer(), rootPlainWire) == plain)
        val customJson = Json(json) { classDiscriminator = "_class" }
        val customWire = customJson.encodeToJsonElement(PlainRoot.serializer(), plain).jsonObject
        check(customWire.getValue("_class").jsonPrimitive.content == "plainleft" && "type" !in customWire)
        check(customJson.decodeFromJsonElement(PlainRoot.serializer(), customWire) == plain)
        check(customJson.decodeFromJsonElement(PlainGroup.serializer(), customWire) == plain)
        val deep = json.decodeFromString(DeepRoot.serializer(), "{\"kind\":\"deep\",\"step\":\"nested\",\"phase\":\"first\",\"extra\":true}")
        check(deep === DeepFirst)
        check(json.encodeToJsonElement(DeepRoot.serializer(), deep) == buildJsonObject { put("kind", "deep"); put("step", "nested"); put("phase", "first") })
        val deepRaw = buildJsonObject { put("kind", "deep"); put("step", "nested"); put("phase", "future"); put("unused", buildJsonArray { add(JsonNull) }) }
        val deepUnknown = json.decodeFromJsonElement(DeepRoot.serializer(), deepRaw) as UnknownDeepInner
        check(deepUnknown.kind == DeepRootKind.Unrecognized("") && deepUnknown.step == DeepMiddleStep.Unrecognized("") && deepUnknown.phase == DeepInnerPhase.Unrecognized(""))
        check(json.encodeToJsonElement(DeepRoot.serializer(), deepUnknown) == deepRaw)
        check(json.encodeToJsonElement(DeepInner.serializer(), deepUnknown) == deepRaw)
        val wideRaw = buildJsonObject { put("tag", "wide"); put("flag", true); for (i in 0 until 119) put("field$i", if (i % 2 == 0) Long.MIN_VALUE else Long.MAX_VALUE) }
        val wide = json.decodeFromJsonElement(org.example.hierarchy_limits.models.WideRoot.serializer(), wideRaw) as org.example.hierarchy_limits.models.WideLeaf
        check(wide.field0 == Long.MIN_VALUE && wide.field118 == Long.MIN_VALUE && wide.flag)
        check(json.encodeToJsonElement(org.example.hierarchy_limits.models.WideRoot.serializer(), wide) == wideRaw)
        val alpha = json.decodeFromString(Flat.serializer(), "{\"tag\":\"a/b~wire\",\"label\":\"local\"}") as AlphaNode
        check(alpha.tag == FlatTag.Serializer.fromSerialName("a/b~wire"))
        check(json.decodeFromJsonElement(Flat.serializer(), json.encodeToJsonElement(Flat.serializer(), alpha)) == alpha)
        check(json.decodeFromString(Flat.serializer(), "{\"tag\":\"betanode\",\"count\":4}") is BetaNode)
        check(json.decodeFromString(Flat.serializer(), "{\"tag\":\"ignored-alias\"}") is UnknownFlat)
        check(json.decodeFromString(FlatDto.serializer(), "{\"tag\":\"betanode\",\"count\":4}") is org.example.hierarchy_unions_suffix.models.BetaNodeDto)
    }
    println("Generated discriminator hierarchy contracts passed")
}

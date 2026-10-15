package org.example.contracts

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import org.example.hierarchybindingbackends.models.AliasInferredRootDto
import org.example.hierarchybindingbackends.models.AliasInferredGroupDto
import org.example.hierarchybindingbackends.models.AliasRangeLeafDto
import org.example.hierarchybindingbackends.models.AliasPinLeafDto
import org.example.hierarchybindingbackends.models.AliasPlainRootDto
import org.example.hierarchybindingbackends.models.AliasPlainGroupDto
import org.example.hierarchybindingbackends.models.AliasPlainLeftDto
import org.example.hierarchybindingbackends.models.AliasDeepRootDto
import org.example.hierarchybindingbackends.models.AliasDeepInnerDto
import org.example.hierarchybindingbackends.models.AliasDeepFirstDto
import org.example.hierarchybindingbackends.models.UnknownInferredGroupDto
import org.example.hierarchybindingbackends.models.UnknownDeepInnerDto
import org.example.hierarchybindingbackends.models.InferredGroupDtoZone
import org.example.hierarchybindingbackends.models.DeepRootDtoKind
import org.example.hierarchybindingbackends.models.DeepMiddleDtoStep
import org.example.hierarchybindingbackends.models.DeepInnerDtoPhase

fun rootHierarchyAliasBackendContracts(codec: Json) {
    val raw=codec.parseToJsonElement("{\"family\":\"inferred\",\"zone\":\"far\",\"distance\":3.0}")
    val far=codec.decodeFromJsonElement(AliasInferredRootDto.serializer(),raw) as AliasRangeLeafDto
    check(far.zone===InferredGroupDtoZone.FAR && far.distance==3.0)
    check(codec.decodeFromJsonElement(AliasInferredRootDto.serializer(),codec.encodeToJsonElement(AliasInferredRootDto.serializer(),far))==far)
    check(codec.decodeFromString(AliasInferredRootDto.serializer(),"{\"family\":\"inferred\",\"zone\":\"near\",\"distance\":3.0}")===AliasPinLeafDto)
    val inferredRaw=codec.parseToJsonElement("{\"family\":\"inferred\",\"zone\":\"future\",\"extra\":9}")
    val unknown=codec.decodeFromJsonElement(AliasInferredRootDto.serializer(),inferredRaw) as UnknownInferredGroupDto
    check(codec.encodeToJsonElement(AliasInferredRootDto.serializer(),unknown)==inferredRaw)
    check(codec.encodeToJsonElement(AliasInferredGroupDto.serializer(),unknown)==inferredRaw)
    val plain: AliasPlainGroupDto=AliasPlainLeftDto("value")
    for(config in listOf(codec,Json(codec){classDiscriminator="category"},Json(codec){useArrayPolymorphism=true})){
        val wire=config.encodeToJsonElement(AliasPlainGroupDto.serializer(),plain)
        check(config.decodeFromJsonElement(AliasPlainGroupDto.serializer(),wire)==plain)
    }
    val groupWire=codec.encodeToJsonElement(AliasPlainGroupDto.serializer(),plain)
    check(codec.decodeFromJsonElement(AliasPlainRootDto.serializer(),groupWire)==plain)
    val ancestorWire=codec.encodeToJsonElement(AliasPlainRootDto.serializer(),plain) as JsonObject
    check("type" !in ancestorWire)
    try {codec.decodeFromJsonElement(AliasPlainRootDto.serializer(),ancestorWire);error("Ancestor invented intermediate class discriminator")}
    catch (_: SerializationException) { }
    val deep=codec.decodeFromString(AliasDeepRootDto.serializer(),"{\"kind\":\"deep\",\"step\":\"nested\",\"phase\":\"first\",\"extra\":true}")
    check(deep===AliasDeepFirstDto)
    val deepRaw=codec.parseToJsonElement("{\"kind\":\"deep\",\"step\":\"nested\",\"phase\":\"future\",\"extra\":9}")
    val deepUnknown=codec.decodeFromJsonElement(AliasDeepRootDto.serializer(),deepRaw) as UnknownDeepInnerDto
    check(deepUnknown.kind==DeepRootDtoKind.Unrecognized("") && deepUnknown.step==DeepMiddleDtoStep.Unrecognized("") && deepUnknown.phase==DeepInnerDtoPhase.Unrecognized(""))
    check(codec.encodeToJsonElement(AliasDeepRootDto.serializer(),deepUnknown)==deepRaw)
    check(codec.encodeToJsonElement(AliasDeepInnerDto.serializer(),deepUnknown)==deepRaw)
}

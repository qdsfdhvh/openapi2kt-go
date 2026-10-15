@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement
import org.example.dependencies.models.CallerDto
import org.example.dependencies.api.models.BetaResponseDto
fun operationDependencyContracts(){
for(defaults in listOf(false,true))for(nulls in listOf(false,true)){
 val json=Json{encodeDefaults=defaults;explicitNulls=nulls}
 val caller=CallerDto(BetaResponseDto.Child.Status.READY)
 check(caller==BetaResponseDto.Child(BetaResponseDto.Child.Status.READY))
 check(json.decodeFromJsonElement(CallerDto.serializer(),json.encodeToJsonElement(caller))==caller)
 val outer=BetaResponseDto("ok",caller)
 check(json.decodeFromJsonElement(BetaResponseDto.serializer(),json.encodeToJsonElement(outer))==outer)
 println("OpaqueIncomingProvider defaults=$defaults nulls=$nulls preserved Beta.Child ctor-root-alias-codec-wire checks passed")
}
}

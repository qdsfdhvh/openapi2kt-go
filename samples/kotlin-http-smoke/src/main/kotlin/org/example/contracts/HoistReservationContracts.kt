@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.encodeToJsonElement
import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpHeaders
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import org.example.hoistreservations.api.models.AlphaInput as DAlphaInput
import org.example.hoistreservations.api.models.BetaInput as DBetaInput
import org.example.hoistreservations.api.models.AlphaResponse as DAlphaResponse
import org.example.hoistreservations.api.models.BetaResponse as DBetaResponse
import org.example.hoistreservations.api.models.AlphaPatch as DAlphaPatch
import org.example.hoistreservations.api.models.BetaPatch as DBetaPatch
import org.example.hoistreservations.api.models.AlphaPatchSerializer as DAlphaPatchSerializer
import org.example.hoistreservations.api.models.BetaPatchSerializer as DBetaPatchSerializer
import org.example.hoistreservations.api.models.AlphaLongAsStringSerializer as DAlphaLongAsStringSerializer
import org.example.hoistreservations.api.models.BetaLongAsStringSerializer as DBetaLongAsStringSerializer
import org.example.hoistreservations.models.Patch as DHelperPatch
import org.example.hoistreservations.HoistReservationsApiClient as DClient
import org.example.hoistreservationsmoved.api.models.AlphaInput as MAlphaInput
import org.example.hoistreservationsmoved.api.models.BetaInput as MBetaInput
import org.example.hoistreservationsmoved.api.models.AlphaResponse as MAlphaResponse
import org.example.hoistreservationsmoved.api.models.BetaResponse as MBetaResponse
import org.example.hoistreservationsmoved.api.models.AlphaPatch as MAlphaPatch
import org.example.hoistreservationsmoved.api.models.BetaPatch as MBetaPatch
import org.example.hoistreservationsmoved.api.models.AlphaPatchSerializer as MAlphaPatchSerializer
import org.example.hoistreservationsmoved.api.models.BetaPatchSerializer as MBetaPatchSerializer
import org.example.hoistreservationsmoved.api.models.AlphaLongAsStringSerializer as MAlphaLongAsStringSerializer
import org.example.hoistreservationsmoved.api.models.BetaLongAsStringSerializer as MBetaLongAsStringSerializer
import org.example.hoistreservationssupport.Patch as MHelperPatch
import org.example.hoistreservationsmoved.HoistReservationsApiClient as MClient
import org.example.hoistreservationsqualified.api.models.AlphaInput as QAlphaInput
import org.example.hoistreservationsqualified.api.models.BetaInput as QBetaInput
import org.example.hoistreservationsqualified.api.models.AlphaResponse as QAlphaResponse
import org.example.hoistreservationsqualified.api.models.BetaResponse as QBetaResponse
import org.example.hoistreservationsqualified.api.models.AlphaPatch as QAlphaPatch
import org.example.hoistreservationsqualified.api.models.BetaPatch as QBetaPatch
import org.example.hoistreservationsqualified.api.models.AlphaPatchSerializer as QAlphaPatchSerializer
import org.example.hoistreservationsqualified.api.models.BetaPatchSerializer as QBetaPatchSerializer
import org.example.hoistreservationsqualified.api.models.AlphaLongAsStringSerializer as QAlphaLongAsStringSerializer
import org.example.hoistreservationsqualified.api.models.BetaLongAsStringSerializer as QBetaLongAsStringSerializer
import org.example.hoistreservationsqualified.models.Patch as QHelperPatch
import org.example.hoistreservationsqualified.HoistReservationsApiClient as QClient
suspend fun hoistReservationContracts(){
for(defaults in listOf(false,true))for(nulls in listOf(false,true)){
val json=Json{encodeDefaults=defaults;explicitNulls=nulls}
check(DAlphaInput::class.java===DBetaInput::class.java)
check(DAlphaResponse::class.java===DBetaResponse::class.java)
check(DAlphaPatch::class.java===DBetaPatch::class.java)
check(DAlphaPatchSerializer::class.java===DBetaPatchSerializer::class.java)
check(DAlphaLongAsStringSerializer::class.java===DBetaLongAsStringSerializer::class.java)
val Dvalue=DBetaResponse("ok",12L)
check(json.encodeToJsonElement(Dvalue).let{it as JsonObject}["amount"]==JsonPrimitive("12"))
check(json.decodeFromJsonElement(DBetaResponse.serializer(),json.encodeToJsonElement(Dvalue))==Dvalue)
check(DAlphaPatch.fromSerialName("future").serialName=="future")
check(DBetaPatch.fromSerialName("fast")==org.example.hoistreservations.models.Patch2.FAST)
val DAbsent=DBetaInput(note=DHelperPatch.Absent)
check(json.decodeFromJsonElement(DBetaInput.serializer(),json.encodeToJsonElement(DAbsent))==DAbsent)
val DNull=DBetaInput(note=DHelperPatch.Null)
check(json.decodeFromJsonElement(DBetaInput.serializer(),json.encodeToJsonElement(DNull))==DNull)
val DValue=DBetaInput(note=DHelperPatch.Value("x"))
check(json.decodeFromJsonElement(DBetaInput.serializer(),json.encodeToJsonElement(DValue))==DValue)
var Dcalls=0
val Dengine=MockEngine{r->
Dcalls++;check(r.url.parameters["patch"]=="fast"&&r.url.parameters["patchSerializer"]=="safe"&&r.url.parameters["longAsStringSerializer"]=="fast")
respond("""{"id":"ok","amount":"12"}""",headers=headersOf(HttpHeaders.ContentType,"application/json"))}
val Dhttp=HttpClient(Dengine){install(ContentNegotiation){json(json)}}
try{val api=DClient(Dhttp,"https://example.invalid").default
check(api.beta(DBetaPatch.fromSerialName("fast"),DBetaPatchSerializer.fromSerialName("safe"),DBetaLongAsStringSerializer.fromSerialName("fast"),DValue)==Dvalue)
check(Dcalls==1)}finally{Dhttp.close()}
check(MAlphaInput::class.java===MBetaInput::class.java)
check(MAlphaResponse::class.java===MBetaResponse::class.java)
check(MAlphaPatch::class.java===MBetaPatch::class.java)
check(MAlphaPatchSerializer::class.java===MBetaPatchSerializer::class.java)
check(MAlphaLongAsStringSerializer::class.java===MBetaLongAsStringSerializer::class.java)
val Mvalue=MBetaResponse("ok",12L)
check(json.encodeToJsonElement(Mvalue).let{it as JsonObject}["amount"]==JsonPrimitive("12"))
check(json.decodeFromJsonElement(MBetaResponse.serializer(),json.encodeToJsonElement(Mvalue))==Mvalue)
check(MAlphaPatch.fromSerialName("future").serialName=="future")
check(MBetaPatch.fromSerialName("fast")==org.example.hoistreservationsmoved.models.Patch.FAST)
val MAbsent=MBetaInput(note=MHelperPatch.Absent)
check(json.decodeFromJsonElement(MBetaInput.serializer(),json.encodeToJsonElement(MAbsent))==MAbsent)
val MNull=MBetaInput(note=MHelperPatch.Null)
check(json.decodeFromJsonElement(MBetaInput.serializer(),json.encodeToJsonElement(MNull))==MNull)
val MValue=MBetaInput(note=MHelperPatch.Value("x"))
check(json.decodeFromJsonElement(MBetaInput.serializer(),json.encodeToJsonElement(MValue))==MValue)
var Mcalls=0
val Mengine=MockEngine{r->
Mcalls++;check(r.url.parameters["patch"]=="fast"&&r.url.parameters["patchSerializer"]=="safe"&&r.url.parameters["longAsStringSerializer"]=="fast")
respond("""{"id":"ok","amount":"12"}""",headers=headersOf(HttpHeaders.ContentType,"application/json"))}
val Mhttp=HttpClient(Mengine){install(ContentNegotiation){json(json)}}
try{val api=MClient(Mhttp,"https://example.invalid").default
check(api.beta(MBetaPatch.fromSerialName("fast"),MBetaPatchSerializer.fromSerialName("safe"),MBetaLongAsStringSerializer.fromSerialName("fast"),MValue)==Mvalue)
check(Mcalls==1)}finally{Mhttp.close()}
check(QAlphaInput::class.java===QBetaInput::class.java)
check(QAlphaResponse::class.java===QBetaResponse::class.java)
check(QAlphaPatch::class.java===QBetaPatch::class.java)
check(QAlphaPatchSerializer::class.java===QBetaPatchSerializer::class.java)
check(QAlphaLongAsStringSerializer::class.java===QBetaLongAsStringSerializer::class.java)
val Qvalue=QBetaResponse("ok",12L)
check(json.encodeToJsonElement(Qvalue).let{it as JsonObject}["amount"]==JsonPrimitive("12"))
check(json.decodeFromJsonElement(QBetaResponse.serializer(),json.encodeToJsonElement(Qvalue))==Qvalue)
check(QAlphaPatch.fromSerialName("future").serialName=="future")
check(QBetaPatch.fromSerialName("fast")==org.example.hoistreservationsqualified.models.AlphaPatch.FAST)
val QAbsent=QBetaInput(note=QHelperPatch.Absent)
check(json.decodeFromJsonElement(QBetaInput.serializer(),json.encodeToJsonElement(QAbsent))==QAbsent)
val QNull=QBetaInput(note=QHelperPatch.Null)
check(json.decodeFromJsonElement(QBetaInput.serializer(),json.encodeToJsonElement(QNull))==QNull)
val QValue=QBetaInput(note=QHelperPatch.Value("x"))
check(json.decodeFromJsonElement(QBetaInput.serializer(),json.encodeToJsonElement(QValue))==QValue)
var Qcalls=0
val Qengine=MockEngine{r->
Qcalls++;check(r.url.parameters["patch"]=="fast"&&r.url.parameters["patchSerializer"]=="safe"&&r.url.parameters["longAsStringSerializer"]=="fast")
respond("""{"id":"ok","amount":"12"}""",headers=headersOf(HttpHeaders.ContentType,"application/json"))}
val Qhttp=HttpClient(Qengine){install(ContentNegotiation){json(json)}}
try{val api=QClient(Qhttp,"https://example.invalid").default
check(api.beta(QBetaPatch.fromSerialName("fast"),QBetaPatchSerializer.fromSerialName("safe"),QBetaLongAsStringSerializer.fromSerialName("fast"),QValue)==Qvalue)
check(Qcalls==1)}finally{Qhttp.close()}
println("HoistReservations defaults=$defaults nulls=$nulls samepkg-helper-numbering/otherpkg-plain/qualified components-Patch3state-bigint-string-typedHTTP checks passed")
}
}

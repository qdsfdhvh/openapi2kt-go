@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import org.example.split.api.DefaultApi
import org.example.split.api.models.AlphaResponseDto
import org.example.split.api.models.BetaResponseDto
import org.example.split.models.ADto
import org.example.split.models.BDto
import org.example.split.models.ResponseDto
import org.example.singleprojection.api.DefaultApi as ProjectionApi
import org.example.singleprojection.api.models.AlphaResponseDto as ProjectionAlpha
import org.example.singleprojection.models.ResponseDto as ProjectionResponse

suspend fun splitLockedAdoptionContracts(checked: Boolean) {
 for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  check(serializer<AlphaResponseDto>().descriptor.serialName == "org.example.split.models.ResponseDto")
  check(serializer<BetaResponseDto>().descriptor.serialName == serializer<AlphaResponseDto>().descriptor.serialName)
  check(serializer<ProjectionAlpha>().descriptor.serialName == "org.example.singleprojection.models.ResponseDto")
  val original:AlphaResponseDto=ResponseDto(value="valid",nested=ResponseDto.Nested(label="nested"))
  check(codec.decodeFromString<BetaResponseDto>(codec.encodeToString(original)) == original)
  check(codec.decodeFromString<BDto>("{}").count == 7)
  check(codec.decodeFromString<ADto>("""{"name":"valid"}""").name == "valid")
  check(codec.decodeFromString<ProjectionAlpha>(codec.encodeToString(original)).value == "valid")
  check(runCatching {BDto(count=-1)}.isFailure == checked)
  check(runCatching {ResponseDto(value="x")}.isFailure == checked)
  check(runCatching {ProjectionResponse(value="x")}.isFailure == checked)
  val unknown=codec.decodeFromString<ProjectionAlpha>("""{"value":"valid","mode":"future"}""")
  check(unknown.mode!!.serialName == "future")
  var payload=codec.encodeToString(original)
  var calls=0
  val engine=MockEngine {request ->
   calls++
   check(request.url.encodedPath == "/a" || request.url.encodedPath == "/b")
   respond(payload,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
  }
  val client=HttpClient(engine) {install(ContentNegotiation) {json(codec)}}
  try {
   val api=DefaultApi(client,"https://example.test/")
   val projected=ProjectionApi(client,"https://example.test/")
   check(api.alpha()==original && api.beta()==original)
   check(projected.alpha().value=="valid")
   payload="{}"
   check(runCatching {api.alpha()}.isFailure && runCatching {projected.alpha()}.isFailure)
   if (checked) {payload="""{"value":"x"}""";check(runCatching {api.beta()}.isFailure && runCatching {projected.alpha()}.isFailure)}
   payload=codec.encodeToString(original)
   check(api.alpha()==original && projected.alpha().value=="valid")
   check(calls >= 7)
  } finally {client.close();engine.close()}
  println("SplitLockedAdoption checked=$checked defaults=$defaults nulls=$nulls genuine-subsets-original-descriptors-singleton-lock-defaults-open-enum-HTTP-reuse passed")
 }
}

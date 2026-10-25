@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.ResponseException
import io.ktor.client.plugins.api.createClientPlugin
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlin.coroutines.cancellation.CancellationException
import kotlin.time.Instant
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.errorroots.api.DefaultApi
import org.example.errorroots.api.errorBody
import org.example.errorroots.api.models.FetchError400Dto
import org.example.errorroots.api.models.FetchError401Dto
import org.example.errorroots.api.models.FetchError402Dto
import org.example.errorroots.api.models.FetchError403DtoInner
import org.example.errorroots.api.models.FetchError404DtoValue
import org.example.errorroots.api.models.FetchError406Dto
import org.example.errorroots.api.models.IntFetchError406Dto
import org.example.errorroots.api.models.FetchError407Dto
import org.example.errorroots.api.models.UnknownFetchError401Dto

private class ErrorRootsCancelDecode

suspend fun inlineErrorRootsContracts(checked: Boolean) {
 for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
  val codec = Json { encodeDefaults=defaults; explicitNulls=nulls; ignoreUnknownKeys=true }
  var status=400
  var payload="""{"message":"given"}"""
  var calls=0
  val cancellation=CancellationException("synthetic error roots cancellation")
  val cancelPlugin=createClientPlugin("SyntheticErrorRootsCancellation") {
   transformResponseBody { _, _, type ->
    if (type.type == ErrorRootsCancelDecode::class) throw cancellation
    null
   }
  }
  val engine=MockEngine {
   calls++
   check(it.url.encodedPath == "/fetch")
   respond(payload, HttpStatusCode.fromValue(status), headersOf("Content-Type","application/json"))
  }
  val client=HttpClient(engine) { expectSuccess=true; install(cancelPlugin); install(ContentNegotiation) { json(codec) } }
  try {
   val api=DefaultApi(client,"https://example.test/")
   suspend fun failure():ResponseException {
    try {api.fetch(); error("Missing caller-owned HTTP error validation")}
    catch (error:ResponseException) {return error}
   }
   val first=failure()
   check(first.errorBody<FetchError400Dto>()!!.message == "given")
   check(first.errorBody<FetchError400Dto>()!!.code == 7)
   try {first.errorBody<ErrorRootsCancelDecode>(); error("Missing cancellation")}
   catch (error:CancellationException) {check(error === cancellation)}
   status=401; payload="""{"kind":"a","message":"given"}"""
   check((failure().errorBody<FetchError401Dto>() as FetchError401Dto.A).message == "given")
   payload="""{"kind":"future","extra":[true,null]}"""
   val unknown=failure().errorBody<FetchError401Dto>() as UnknownFetchError401Dto
   check(codec.encodeToJsonElement<FetchError401Dto>(unknown) == codec.parseToJsonElement(payload))
   status=402; payload="\"future\""
   check(failure().errorBody<FetchError402Dto>()!!.serialName == "future")
   status=403; payload="""[{"message":"given"}]"""
   check(failure().errorBody<List<FetchError403DtoInner>>()!!.single().message == "given")
   status=404; payload="""{"a/~":{"message":"given"}}"""
   check(failure().errorBody<Map<String,FetchError404DtoValue>>()!!.getValue("a/~").message == "given")
   status=405; payload="""{"anything":false}"""
   check(failure().errorBody<JsonObject>() == codec.parseToJsonElement(payload))
   status=406; payload="42"
   check((failure().errorBody<FetchError406Dto>() as IntFetchError406Dto).value == 42L)
   status=407; payload="null"
   check(failure().errorBody<FetchError407Dto?>() == null)
   payload="""{"message":"given"}"""
   check(failure().errorBody<FetchError407Dto?>()!!.code == 7)
   status=408; payload="null"
   check(failure().errorBody<JsonElement>() === JsonNull)
   status=409; payload="\"2026-01-02T05:04:05+02:00\""
   check(failure().errorBody<Instant>() == Instant.parse("2026-01-02T03:04:05Z"))
   payload="\"invalid\""
   check(failure().errorBody<Instant>() == null)
   status=410; payload="null"
   check(failure().errorBody<JsonNull>() === JsonNull)
   status=400
   for (bad in listOf("{", "{}")) {payload=bad; check(failure().errorBody<FetchError400Dto>() == null)}
   payload="""{"message":""}"""
   check((failure().errorBody<FetchError400Dto>() == null) == checked)
   status=403; payload="[{}]"
   check(failure().errorBody<List<FetchError403DtoInner>>() == null)
   status=404; payload="""{"one":{}}"""
   check(failure().errorBody<Map<String,FetchError404DtoValue>>() == null)
   status=204; payload=""
   val before=calls
   api.fetch();check(calls == before+1)
  } finally {client.close(); engine.close()}
  println("InlineErrorRoots checked=$checked defaults=$defaults nulls=$nulls allOf-union-enum-list-map-const-primitive-null-raw-temporal-repeated-cancellation-reuse passed")
 }
}

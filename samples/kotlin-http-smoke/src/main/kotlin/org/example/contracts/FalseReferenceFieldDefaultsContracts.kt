package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import org.example.falsefielddefaults.FalseFieldDefaultsClient
import org.example.falsefielddefaults.models.Optional as OptionalRoot
import org.example.falsefielddefaults.models.Required as RequiredRoot
import org.example.falsefielddefaults.models.DirectRequired as DirectRequiredRoot
import org.example.falsefielddefaults.models.LegacyRequired as LegacyRequiredRoot
import org.example.falsefielddefaults.models.NullableOwner as NullableOwnerRoot

private fun falseFieldDefaultBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseFieldDefaultBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic field default body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseReferenceFieldDefaultsContracts() {
    check(!serializer<RequiredRoot>().descriptor.isElementOptional(0))
    check(serializer<RequiredRoot>().descriptor.getElementDescriptor(0).isNullable)
    check(!serializer<DirectRequiredRoot>().descriptor.isElementOptional(0))
    check(serializer<LegacyRequiredRoot>().descriptor.isElementOptional(0))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val optional=OptionalRoot()
        check(optional.value == null && optional.inherited == null && optional.restored == null && optional.targetOnly == null && optional.narrow == null)
        val fields=codec.parseToJsonElement(codec.encodeToString(optional)).jsonObject
        check(fields.size == if(defaults && nulls) 4 else 0)
        check("narrow" !in fields)
        check(codec.decodeFromString<OptionalRoot>("{}").value == null)
        check(codec.decodeFromString<OptionalRoot>("{\"value\":null,\"inherited\":null,\"restored\":null}").restored == null)
        check(runCatching { codec.decodeFromString<OptionalRoot>("{\"narrow\":null}") }.exceptionOrNull() is SerializationException)
        for(value in listOf("true","0","1.5","\"text\"","[]","{}")) {
            for(field in listOf("value","inherited","restored")) {
                check(runCatching { codec.decodeFromString<OptionalRoot>("{\"$field\":$value}") }.exceptionOrNull() is SerializationException)
            }
            check(runCatching { codec.decodeFromString<RequiredRoot>("{\"value\":$value}") }.exceptionOrNull() is SerializationException)
        }
        val required=RequiredRoot(value=null)
        val direct=DirectRequiredRoot(value=null)
        check(codec.decodeFromString<DirectRequiredRoot>(codec.encodeToString(direct)).value == null)
        val missing=runCatching { codec.decodeFromString<RequiredRoot>("{}") }
        if(nulls) check(missing.exceptionOrNull() is SerializationException) else check(missing.getOrThrow().value == null)
        check(LegacyRequiredRoot().value == null)
        check(codec.decodeFromString<LegacyRequiredRoot>("{}").value == null)
        val nullableOwner=NullableOwnerRoot()
        check(codec.decodeFromString<NullableOwnerRoot>(codec.encodeToString(nullableOwner)).value == null)
        check(codec.decodeFromString<NullableOwnerRoot?>("null") == null)
        val body=codec.encodeToString(required)
        check(body == if(nulls) "{\"value\":null}" else "{}")
        var badResponse=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/required")
            check(falseFieldDefaultBody(request.body) == body)
            respond(if(badResponse) "{\"value\":true}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseFieldDefaultsClient(http,"http://example.test/").fields
            check(api.send(required).value == null)
            badResponse=true
            check(runCatching { api.send(required) }.isFailure)
        } finally { http.close() }
        println("FalseReferenceFieldDefaults defaults=$defaults nulls=$nulls own-null/required-argument/optional-initializer/target-noinherit/strictnarrow/shared-json/http passed")
    }
}

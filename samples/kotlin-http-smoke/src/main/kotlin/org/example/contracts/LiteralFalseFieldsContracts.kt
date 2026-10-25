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
import org.example.literalfalse.LiteralFalseClient
import org.example.literalfalse.models.Optional as OptionalRoot
import org.example.literalfalse.models.Mixed as MixedRoot
import org.example.literalfalse.models.Required as RequiredRoot
import org.example.literalfalse.models.Collision as CollisionRoot
import org.example.literalfalse.models.FalseValue as FalseValueRoot
import org.example.literalfalse.models.NullableOwner as NullableRoot
import org.example.literalfalse.models.Reserved as ReservedRoot

private fun constructRequired(value: RequiredRoot.FalseValue): RequiredRoot = RequiredRoot(value)
private fun constructMixed(value: MixedRoot.FalseValue): MixedRoot = MixedRoot(required=value)

private fun literalFalseBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> literalFalseBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic literal false body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun literalFalseFieldsContracts() {
    val optional=OptionalRoot()
    check(optional.blocked == null && optional.other == null)
    val mixed=serializer<MixedRoot>().descriptor
    check(!mixed.isElementOptional(mixed.getElementIndex("required")))
    check(!mixed.getElementDescriptor(mixed.getElementIndex("required")).isNullable)
    check(mixed.isElementOptional(mixed.getElementIndex("optional")))
    check(mixed.getElementDescriptor(mixed.getElementIndex("optional")).isNullable)
    check(serializer<CollisionRoot.FalseValue2>().descriptor.serialName.endsWith("Collision.FalseValue2") || serializer<CollisionRoot.FalseValue2>().descriptor.serialName.endsWith("CollisionDto.FalseValue2"))
    val collision=CollisionRoot(falseValue=CollisionRoot.FalseValue(label="synthetic"))
    check(collision.falseValue?.label == "synthetic" && collision.blocked == null)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.encodeToString(optional) == "{}")
        check(codec.encodeToString(OptionalRoot(blocked=null,other=null)) == "{}")
        check(codec.decodeFromString<OptionalRoot>("{}").blocked == null)
        val reserved=codec.decodeFromString<ReservedRoot>("{}")
        check(reserved.payloadText == "@kotlinx.serialization.EncodeDefault(kotlinx.serialization.EncodeDefault.Mode.NEVER) @kotlinx.serialization.Serializable(with = org.example.Serializer::class)")
        check(codec.parseToJsonElement(codec.encodeToString(reserved)).jsonObject.keys == if(defaults) setOf("payloadText") else emptySet<String>())
        check(codec.encodeToString(codec.decodeFromString<FalseValueRoot>("{}")) == "{}")
        check(codec.encodeToString(codec.decodeFromString<NullableRoot>("{}")) == "{}")
        check(codec.decodeFromString<NullableRoot?>("null") == null)
        check(runCatching { codec.decodeFromString<RequiredRoot>("{}") }.isFailure)
        check(runCatching { codec.decodeFromString<MixedRoot>("{}") }.isFailure)
        for(value in listOf("null","true","false","0","1.5","\"text\"","[]","{}")) {
            for(field in listOf("blocked","other")) {
                val failure=runCatching { codec.decodeFromString<OptionalRoot>("{\"$field\":$value}") }.exceptionOrNull()
                check(failure is SerializationException && failure.message == "A false field must be absent.")
            }
            check(runCatching { codec.decodeFromString<RequiredRoot>("{\"blocked\":$value}") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<OptionalRoot.FalseValue>(value) }.exceptionOrNull() is SerializationException)
            for(field in listOf("kotlin","kotlinx","org","serializer","absentSerializer","companion","OpenapiKtEncodeDefaultMode","OpenapiKtTypeUseSerializer")) {
                check(runCatching { codec.decodeFromString<ReservedRoot>("{\"$field\":$value}") }.exceptionOrNull() is SerializationException)
            }
        }
        val body=codec.encodeToString(optional)
        var bad=false
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/values")
            check(literalFalseBody(request.body) == body)
            respond(if(bad) "{\"blocked\":null}" else body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=LiteralFalseClient(http,"http://example.test/").values
            check(api.send(optional).blocked == null)
            bad=true
            check(runCatching { api.send(optional) }.isFailure)
        } finally {http.close()}
        println("LiteralFalseFields defaults=$defaults nulls=$nulls actualnestedcodec/required/strictabsence/names/http passed")
    }
}

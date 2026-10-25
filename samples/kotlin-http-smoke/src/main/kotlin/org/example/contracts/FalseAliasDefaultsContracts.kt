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
import org.example.falsealiasdefaults.FalseAliasDefaultsClient
import org.example.falsealiasdefaults.models.Blocked as BlockedRoot
import org.example.falsealiasdefaults.models.Peer as PeerRoot
import org.example.falsealiasdefaults.models.Maybe as MaybeRoot
import org.example.falsealiasdefaults.models.OwnInherited as OwnInheritedRoot
import org.example.falsealiasdefaults.models.Inherited as InheritedRoot
import org.example.falsealiasdefaults.models.Narrow as NarrowRoot
import org.example.falsealiasdefaults.models.Again as AgainRoot
import org.example.falsealiasdefaults.models.Other as OtherRoot
import org.example.falsealiasdefaults.models.Holder as HolderRoot
import org.example.falsealiasdefaults.models.Required as RequiredRoot
import org.example.falsealiasdefaults.models.`Maybe default` as maybeDefaultFactory
import org.example.falsealiasdefaults.models.`OwnInherited default` as ownInheritedDefaultFactory
import org.example.falsealiasdefaults.models.`Again default` as againDefaultFactory
import org.example.falsealiasdefaults.models.`Other default` as otherDefaultFactory
import org.example.falsealiasdefaults.models.`Rows default` as rowsDefaultFactory
import org.example.falsealiasdefaults.models.`Mapping default` as mapDefaultFactory

private fun falseAliasBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> falseAliasBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic nullable false body")
}
private fun falseAliasRequiredCall(value: NarrowRoot): RequiredRoot = RequiredRoot(value)

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseAliasDefaultsContracts() {
    val maybe: MaybeRoot = maybeDefaultFactory()
    val own: OwnInheritedRoot = ownInheritedDefaultFactory()
    val again: AgainRoot = againDefaultFactory()
    val other: OtherRoot = otherDefaultFactory()
    check(maybe == null && own == null && again == null && other == null)
    check(rowsDefaultFactory() == listOf(null))
    check(mapDefaultFactory() == mapOf("description" to null))
    check(serializer<MaybeRoot>().descriptor.isNullable)
    check(serializer<InheritedRoot>().descriptor == serializer<MaybeRoot>().descriptor)
    check(!serializer<NarrowRoot>().descriptor.isNullable)
    check(serializer<BlockedRoot>().descriptor.serialName != serializer<PeerRoot>().descriptor.serialName)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.encodeToString(maybe) == "null")
        check(codec.encodeToString(own) == "null")
        check(codec.encodeToString(again) == "null")
        check(codec.encodeToString(other) == "null")
        check(codec.decodeFromString<MaybeRoot>("null") == null)
        check(codec.decodeFromString<InheritedRoot>("null") == null)
        for (value in listOf("null","true","0","1.5","\"text\"","[]","{}")) {
            check(runCatching { codec.decodeFromString<BlockedRoot>(value) }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<NarrowRoot>(value) }.exceptionOrNull() is SerializationException)
            if(value != "null") {
                check(runCatching { codec.decodeFromString<MaybeRoot>(value) }.exceptionOrNull() is SerializationException)
                check(runCatching { codec.decodeFromString<OwnInheritedRoot>(value) }.exceptionOrNull() is SerializationException)
                check(runCatching { codec.decodeFromString<AgainRoot>(value) }.exceptionOrNull() is SerializationException)
            }
        }
        val holder=HolderRoot()
        check(holder.first == null && holder.inherited == null && holder.again == null)
        val body=codec.encodeToString(holder)
        check(codec.parseToJsonElement(body).jsonObject.size == if(defaults && nulls) 3 else 0)
        check(codec.decodeFromString<HolderRoot>(body).first == null)
        check(runCatching { codec.decodeFromString<RequiredRoot>("{}") }.exceptionOrNull() is SerializationException)
        val engine=MockEngine { request ->
            check(request.url.encodedPath == "/holder")
            check(falseAliasBody(request.body) == body)
            respond(body,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
        }
        val http=HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=FalseAliasDefaultsClient(http,"http://example.test/").aliases
            check(api.send(holder).first == null)
        } finally { http.close() }
        println("FalseAliasDefaults defaults=$defaults nulls=$nulls own-null-factory/nearest-intent/no-inherit/strict-nonnull-codec/shared-presence/http passed")
    }
}

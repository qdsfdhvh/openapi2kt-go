package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import org.example.falsedtorootdefaults.FalseDTORootDefaultsClient
import org.example.falsedtorootdefaults.models.Outer as OwnerRoot
import org.example.falsedtorootdefaults.models.Target as TargetRoot
import org.example.falsedtorootdefaults.models.Omitted as OmittedRoot
import org.example.falsedtorootdefaults.models.NullableMember as NullableMemberRoot
import org.example.falsedtorootdefaults.models.NullableOwner as NullableOwnerRoot
import org.example.falsedtorootdefaults.models.OwnerWithoutFactory as OwnerWithoutFactoryRoot
import org.example.falsedtorootdefaults.models.EarlierTarget as EarlierTargetRoot
import org.example.falsedtorootdefaults.models.LaterOwner as LaterOwnerRoot
import org.example.falsedtorootdefaults.models.CountedOwner as CountedOwnerRoot

private fun falseDTOOwnerBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> falseDTOOwnerBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic root-default body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseDTOOwnerRootDefaultsContracts() {
    val first = OwnerRoot.default()
    val second = OwnerRoot.default()
    check(first !== second && first.target !== second.target)
    check(first.target.labels == listOf("owner") && first.target.labels !== second.target.labels)
    check(first.target.mapping !== second.target.mapping && first.target.mapping!!["description"] !== second.target.mapping!!["description"])
    check(first.target.mapping!!["description"]!!.containsKey("items") && first.target.mapping!!["description"]!!["items"] == null)
    check(first.target.spares!!.isEmpty() && first.target.falseValue == "owner-name")
    check(first.missing == null && first.org == null && first.kotlinx == null)
    check(TargetRoot.default().labels == listOf("target"))
    check(TargetRoot.default().mapping!!["description"]!!["items"]!!.isEmpty())
    check(OmittedRoot.default().target == null)
    check(NullableMemberRoot.default().target == null && NullableOwnerRoot.default() == null)
    check(OwnerWithoutFactoryRoot.default().target!!.rows.isEmpty())
    check(EarlierTargetRoot.default().labels == listOf("earlier"))
    check(LaterOwnerRoot.default().target!!.labels == listOf("later"))
    check(CountedOwnerRoot.default().target?.value == null)
    check(!OwnerRoot.serializer().descriptor.isElementOptional(OwnerRoot.serializer().descriptor.getElementIndex("target")))
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val body = codec.encodeToString(first)
        val decoded = codec.decodeFromString<OwnerRoot>(body)
        check(decoded.target.labels == listOf("owner") && decoded.target.rows.isEmpty())
        val targetFields = codec.parseToJsonElement(body).jsonObject.getValue("target").jsonObject
        check("blocked" !in targetFields && "reference" !in targetFields)
        val counted = CountedOwnerRoot.default()
        check(codec.decodeFromString<CountedOwnerRoot>(codec.encodeToString(counted)).target?.value == null)
        for (invalid in listOf("{\"target\":{\"rows\":[null]}}", "{\"target\":{\"rows\":[],\"blocked\":null}}", "{\"target\":{}}")) {
            check(runCatching { codec.decodeFromString<OwnerRoot>(invalid) }.isFailure)
        }
        var bad = false
        val engine = MockEngine { request ->
            check(request.url.encodedPath == "/root" && falseDTOOwnerBody(request.body) == body)
            respond(if (bad) "{\"target\":{\"rows\":[null]}}" else body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = FalseDTORootDefaultsClient(http, "http://example.test/").objects
            check(api.send(first).target.rows.isEmpty())
            bad = true
            check(runCatching { api.send(first) }.isFailure)
        } finally { http.close() }
        println("FalseDTOOwnerRootDefaults defaults=$defaults nulls=$nulls root-edge/independent-factories/fresh-data/counts/required/http passed")
    }
}

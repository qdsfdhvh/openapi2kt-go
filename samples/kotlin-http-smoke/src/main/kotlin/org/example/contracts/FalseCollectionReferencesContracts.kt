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
import kotlinx.serialization.serializer
import org.example.falsecollections.FalseCollectionsClient
import org.example.falsecollections.models.Rows as RowsRoot
import org.example.falsecollections.models.Mapping as MapRoot
import org.example.falsecollections.models.Nested as NestedRoot
import org.example.falsecollections.models.NullableRows as NullableRowsRoot
import org.example.falsecollections.models.NullElements as NullElementsRoot
import org.example.falsecollections.models.NarrowElements as NarrowElementsRoot
import org.example.falsecollections.models.NullMapping as NullMapRoot
import org.example.falsecollections.models.RowsAlias as AliasRowsRoot
import org.example.falsecollections.models.Holder as HolderRoot

private fun falseCollectionBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> falseCollectionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic collection body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun falseCollectionReferencesContracts() {
    val rows: RowsRoot = emptyList()
    val mapping: MapRoot = emptyMap()
    val nested: NestedRoot = listOf(mapOf("key" to emptyList()))
    val nullableElements: NullElementsRoot = listOf(null, null)
    val nullableMapping: NullMapRoot = mapOf("key" to null)
    val holder = HolderRoot(rows, mapping, nested, nullableElements, nullableMapping)
    check(serializer<RowsRoot>().descriptor.getElementDescriptor(0).serialName.endsWith(".Blocked") || serializer<RowsRoot>().descriptor.getElementDescriptor(0).serialName.endsWith(".BlockedDto"))
    check(!serializer<RowsRoot>().descriptor.getElementDescriptor(0).isNullable)
    check(serializer<NullElementsRoot>().descriptor.getElementDescriptor(0).isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.decodeFromString<RowsRoot>("[]") == rows)
        check(codec.encodeToString(rows) == "[]")
        check(codec.decodeFromString<AliasRowsRoot>("[]") == rows)
        check(codec.decodeFromString<MapRoot>("{}") == mapping)
        check(codec.encodeToString(mapping) == "{}")
        check(codec.decodeFromString<NestedRoot>(codec.encodeToString(nested)) == nested)
        check(codec.decodeFromString<NullElementsRoot>("[null,null]") == nullableElements)
        check(codec.decodeFromString<NullMapRoot>("{\"key\":null}") == nullableMapping)
        check(codec.encodeToString(nullableMapping) == "{\"key\":null}")
        check(codec.decodeFromString<NullableRowsRoot>("null") == null)
        check(codec.decodeFromString<NullableRowsRoot>("[]") == rows)
        val absentRoot: NullableRowsRoot = null
        check(codec.encodeToString(absentRoot) == "null")
        check(codec.decodeFromString<HolderRoot>(codec.encodeToString(holder)).rows == rows)
        check(runCatching { codec.decodeFromString<HolderRoot>("{}") }.exceptionOrNull() is SerializationException)
        for (value in listOf("null", "true", "0", "1.5", "\"text\"", "[]", "{}")) {
            check(runCatching { codec.decodeFromString<RowsRoot>("[$value]") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<AliasRowsRoot>("[$value]") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<NarrowElementsRoot>("[$value]") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<MapRoot>("{\"key\":$value}") }.exceptionOrNull() is SerializationException)
            check(runCatching { codec.decodeFromString<NestedRoot>("[{\"key\":[$value]}]") }.exceptionOrNull() is SerializationException)
            if (value != "null") check(runCatching { codec.decodeFromString<NullElementsRoot>("[$value]") }.exceptionOrNull() is SerializationException)
        }
        var requests = 0
        val engine = MockEngine { request ->
            requests++
            check(request.url.encodedPath == "/rows")
            check(falseCollectionBody(request.body) == "[]")
            respond("{\"key\":null}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = FalseCollectionsClient(http, "http://example.test/").collections
            check(api.sendRows(rows) == nullableMapping)
            check(requests == 1)
        } finally { http.close() }
        println("FalseCollectionReferences defaults=$defaults nulls=$nulls actual-empty-list-map/nested/nullable-alias/nonempty-refusal/http passed")
    }
}

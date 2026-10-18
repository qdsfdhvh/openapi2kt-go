@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.example.inlinehier.models.CircleDto
import org.example.inlinehier.models.ShapeDto
import org.example.inlinehier.models.ShapeDtoKind
import org.example.inlinehier.models.SquareDto
import org.example.inlinehier.models.OnlyDto
import org.example.inlinehier.models.TaggedDto
import org.example.inlinehier.models.TaggedDtoKind
import org.example.inlinehier.models.UnknownShapeDto

// An inline branch has no component name, so its discriminator constant supplies
// both the class name and the dispatch value. The runtime shape must be the same
// as the `$ref` spelling, including the Unknown fallback and a required tag for
// vocabulary leaves.
suspend fun hierarchyInlineMembersContracts() {
    val circle = CircleDto(r = 3)
    check(circle.kind == ShapeDtoKind.CIRCLE) { "Const branch lost its fixed tag: ${circle.kind}" }
    val square = SquareDto(side = 4)
    check(square.kind == ShapeDtoKind.SQUARE) { "Const branch lost its fixed tag: ${square.kind}" }

    // A vocabulary leaf keeps the required typed constructor argument.
    // A vocabulary branch keeps its required typed tag argument and takes the
    // class name from the single vocabulary value.
    val tagged = OnlyDto(kind = TaggedDtoKind.ONLY, note = "n")
    check(tagged.kind == TaggedDtoKind.ONLY)

    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }

        // Dispatch on the wire tag reaches the inline leaf.
        val decoded = codec.decodeFromString<ShapeDto>("""{"kind":"circle","r":7}""")
        check(decoded is CircleDto && decoded.r == 7) { "Inline branch dispatch failed: $decoded" }
        check(codec.decodeFromString<ShapeDto>("""{"kind":"square","side":9}""") is SquareDto)
        check(codec.decodeFromString<TaggedDto>("""{"kind":"only","note":"x"}""") is OnlyDto)

        // The fixed tag is always written, so the value round-trips.
        val tree = codec.encodeToString<ShapeDto>(circle)
        check(codec.parseToJsonElement(tree).toString() == """{"r":3,"kind":"circle"}""") { "Wire changed: $tree" }
        check(codec.decodeFromString<ShapeDto>(tree) is CircleDto)

        // An unknown tag still lands in the raw-preserving Unknown stand-in.
        val unknown = codec.decodeFromString<ShapeDto>("""{"kind":"hexagon","x":1}""")
        check(unknown is UnknownShapeDto) { "Unknown fallback lost: $unknown" }
        check(codec.parseToJsonElement(codec.encodeToString<ShapeDto>(unknown)).toString() == """{"kind":"hexagon","x":1}""")

        // A recognized tag with an invalid body still fails rather than becoming Unknown.
        check(runCatching { codec.decodeFromString<ShapeDto>("""{"kind":"circle"}""") }.isFailure) {
            "A known payload without its required member became Unknown"
        }

        println("HierarchyInlineMembers Json defaults=$defaults nulls=$nulls const-named-dispatch-unknown passed")
    }
}

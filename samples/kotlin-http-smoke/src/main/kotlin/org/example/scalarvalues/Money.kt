package org.example.scalarvalues

// A real caller-owned classifier/serializer. No generated DTO stands in for
// this type, and the generator cannot infer its wire representation.
@kotlinx.serialization.Serializable
@JvmInline
value class Money(val decimal: String)

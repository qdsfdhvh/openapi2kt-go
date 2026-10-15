import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    kotlin("jvm") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}
repositories { mavenCentral() }
dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-datetime:0.7.1")
    implementation("io.ktor:ktor-client-mock-jvm:3.5.2")
    implementation("io.ktor:ktor-client-content-negotiation-jvm:3.5.2")
    implementation("io.ktor:ktor-serialization-kotlinx-json-jvm:3.5.2")
}
java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}
kotlin {
    compilerOptions.jvmTarget.set(JvmTarget.JVM_17)
    // Flat metadata type tables avoid Kotlin's protobuf recursion limit for
    // independently tested deep generic aliases; schema caps stay unchanged.
    compilerOptions.freeCompilerArgs.add("-Xuse-type-table")
    sourceSets.main {
        kotlin.srcDir(layout.buildDirectory.dir("generated"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-limits"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-root-internal"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-dto-alias-chains"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-root-dto-bridges"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-collections"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-bodies"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-references"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-options"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-operation-models"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-operation-collisions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-operation-excluded"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-request-presence"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-request-unions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-deep-object"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-deep-bindings"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-errors"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-projections"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-response-kinds"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-structural-values"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-nullable-unions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-request-kinds"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-request-kinds-excluded"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-operation-scalar-unions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-empty-response-unions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-request-field-unions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-parameter-values"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-special-scalars"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-special-scalar-values"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-native-enum-values"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-sse"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-value-classes"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-constraints"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-dto-constraints"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-patterns"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-alias-constraints"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-element-constraints"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-numeric-constraints"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-numeric-legacy-constraints"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-collection-counts"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-collection-unique"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-presence-collisions"))
        kotlin.srcDir(layout.buildDirectory.dir("generated-presence-excluded"))
    }
}
tasks.register<JavaExec>("runHttpContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.HttpContractsKt")
}

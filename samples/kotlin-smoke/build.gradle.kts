import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    kotlin("jvm") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}

repositories {
    mavenCentral()
}

dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

kotlin {
    compilerOptions.jvmTarget.set(JvmTarget.JVM_17)
    sourceSets.main {
        kotlin.srcDir(layout.buildDirectory.dir("generated"))
        kotlin.srcDir(layout.buildDirectory.dir("dictionaries-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("dictionaries-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("dictionary-limits-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("enum-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("enum-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("enum-limit-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("enum-naming-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("enum-contexts-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("enum-contexts-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("primitive-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("primitive-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("primitive-limit-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("primitive-unions-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("primitive-unions-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("primitive-union-combinations-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("object-unions-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("object-unions-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("object-union-limits-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("union-normalization-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("union-normalization-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("union-normalization-limits-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("root-kind-limits-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("root-kind-unions-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("root-kind-unions-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("root-unions-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("root-unions-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("hierarchy-unions-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("hierarchy-unions-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("hierarchy-limits-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("scalar-projections-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("scalar-projections-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("scalar-projection-limits-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("nullable-composition-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("const-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("const-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("const-limit-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("composition-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("composition-suffix-generated"))
        kotlin.srcDir(layout.buildDirectory.dir("composition-limit-generated"))
        providers.gradleProperty("extraModelSources").orNull?.let { kotlin.srcDir(it) }
        val fixtureRoots = layout.buildDirectory.file("fixture-source-roots.txt").get().asFile
        if (fixtureRoots.exists()) {
            fixtureRoots.readLines().filter { it.isNotBlank() }.forEach { id ->
                require(id.matches(Regex("[A-Za-z0-9-]+"))) { "Invalid fixture source root" }
                kotlin.srcDir(layout.buildDirectory.dir("fixture-generated/$id"))
            }
        }
    }
}

tasks.register<JavaExec>("runDictionaryContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.DictionaryContractsKt")
}

tasks.register<JavaExec>("runUnionNormalizationContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.UnionNormalizationContractsKt")
}

tasks.register<JavaExec>("runRootKindContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.RootKindContractsKt")
}

tasks.register<JavaExec>("runScalarProjectionContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.ScalarProjectionContractsKt")
}

tasks.register<JavaExec>("runHierarchyContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.HierarchyContractsKt")
}

tasks.register<JavaExec>("runRootUnionContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.RootUnionContractsKt")
}

tasks.register<JavaExec>("runObjectUnionContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.ObjectUnionContractsKt")
}

tasks.register<JavaExec>("runPrimitiveUnionContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.PrimitiveUnionContractsKt")
}

tasks.register<JavaExec>("runEnumContextContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.EnumContextContractsKt")
}

tasks.register<JavaExec>("runEnumContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.EnumContractsKt")
}

tasks.register<JavaExec>("runPrimitiveContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.PrimitiveContractsKt")
}

tasks.register<JavaExec>("runObjectCompositionContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.ObjectCompositionContractsKt")
}

tasks.register<JavaExec>("runConstContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.ConstContractsKt")
}

tasks.register<JavaExec>("runNullableCompositionContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.NullableCompositionContractsKt")
}

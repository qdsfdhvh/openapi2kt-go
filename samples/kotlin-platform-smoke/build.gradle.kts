import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.targets.js.nodejs.NodeJsEnvSpec

plugins {
    kotlin("multiplatform") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}

repositories { mavenCentral() }

val platformTarget = providers.gradleProperty("platformTarget").get()
kotlin {
    when (platformTarget) {
        "jvm" -> jvm { compilerOptions.jvmTarget.set(JvmTarget.JVM_17) }
        "js" -> js { nodejs(); binaries.executable() }
        "native" -> macosArm64 { binaries.executable { entryPoint = "org.example.platform.main" } }
        "metadata" -> { jvm(); macosArm64() }
        else -> error("Unknown platformTarget")
    }
    sourceSets.commonMain {
        kotlin.srcDir("generated")
        kotlin.srcDir("generated-increments")
        dependencies { implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0") }
    }
}

if (platformTarget == "js") {
    rootProject.plugins.withType<org.jetbrains.kotlin.gradle.targets.js.nodejs.NodeJsRootPlugin> {
        rootProject.extensions.configure<NodeJsEnvSpec> { download.set(false) }
    }
    rootProject.plugins.withType<org.jetbrains.kotlin.gradle.targets.js.yarn.YarnPlugin> {
        rootProject.extensions.configure<org.jetbrains.kotlin.gradle.targets.js.yarn.YarnRootEnvSpec> {
            download.set(false)
        }
    }
    tasks.withType<org.jetbrains.kotlin.gradle.targets.js.npm.tasks.KotlinNpmInstallTask>().configureEach {
        args.add("--offline")
        args.add("--ignore-scripts")
    }
}

if (platformTarget == "jvm") {
    tasks.register<JavaExec>("runPlatformContracts") {
        dependsOn("jvmMainClasses")
        val target = kotlin.targets.getByName("jvm")
        val main = target.compilations.getByName("main")
        classpath = files(main.output.allOutputs, configurations.getByName("jvmRuntimeClasspath"))
        mainClass.set("org.example.platform.PlatformContractsKt")
    }
}

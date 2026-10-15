package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

// exportModelSmoke is test-only IO. Every invocation requires a new directory
// under the repository's ignored verification area; stale Rust/generated files
// cannot accidentally satisfy Go Kotlin compilation.
func exportModelSmoke(t testing.TB, destination string) {
	t.Helper()
	repo, err := filepath.Abs(testutil.RepoRoot())
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(destination)
	if err != nil {
		t.Fatal(err)
	}
	allowed := filepath.Join(repo, ".pi", "go-migration") + string(filepath.Separator)
	if !strings.HasPrefix(root, allowed) {
		t.Fatal("export must stay in the repository verification directory")
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("export requires a fresh directory", err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(relative string, data []byte) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			t.Fatal(writeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	generateRequest := func(rq core.GenerationRequest, directory string) {
		t.Helper()
		if exportThroughCLI(t, root, directory, rq) {
			return
		}
		r, err := core.Generate(rq)
		if err != nil {
			t.Fatal(directory, err)
		}
		for _, f := range r.Files {
			write("generated/"+directory+"/"+f.Path, []byte(f.Content))
		}
	}
	generate := func(input []byte, pkg, suffix, directory string) {
		t.Helper()
		rq := request(input, suffix)
		rq.Options.PackageName = pkg
		generateRequest(rq, directory)
	}
	withHierarchy := os.Getenv("OPENAPI_KT_TEST_EXPORT_HIERARCHY") == "1"
	withObject := withHierarchy || os.Getenv("OPENAPI_KT_TEST_EXPORT_OBJECT") == "1"
	withPrimitive := withObject || os.Getenv("OPENAPI_KT_TEST_EXPORT_PRIMITIVE") == "1"
	fixtures := []struct{ fixture, pkg string }{
		{"open-enums", "enums"}, {"enum-naming", "enum_naming"},
		{"primitive-components", "primitives"}, {"nullable-composition", "nullable_composition"},
		{"const-fields", "const_fields"}, {"object-composition", "composition"}, {"enum-contexts", "enum_contexts"},
		{"dictionaries", "dictionaries"},
	}
	if withPrimitive {
		fixtures = append(fixtures, struct{ fixture, pkg string }{"primitive-unions", "primitive_unions"}, struct{ fixture, pkg string }{"union-normalization", "union_normalization"})
	}
	if withObject {
		fixtures = append(fixtures,
			struct{ fixture, pkg string }{"object-unions", "object_unions"}, struct{ fixture, pkg string }{"root-unions", "root_unions"},
			struct{ fixture, pkg string }{"root-kind-unions", "root_kind_unions"}, struct{ fixture, pkg string }{"scalar-projections", "scalar_projections"})
	}
	if withHierarchy {
		fixtures = append(fixtures, struct{ fixture, pkg string }{"hierarchy-unions", "hierarchy_unions"})
	}
	for _, tc := range fixtures {
		input, err := os.ReadFile(filepath.Join(repo, "tests", "fixtures", "model-semantics", tc.fixture+".json"))
		if err != nil {
			t.Fatal(err)
		}
		generate(input, "org.example."+tc.pkg, "", tc.fixture)
		suffixPackage := tc.pkg + "_suffix"
		if tc.fixture == "root-kind-unions" {
			suffixPackage = "root_kind_union_suffix"
		}
		generate(input, "org.example."+suffixPackage, "Dto", tc.fixture+"-suffix")
	}
	positive := 0
	for _, c := range testutil.Corpus(t) {
		if c.Expect.Kind != "success" {
			continue
		}
		rq := request(testutil.Read(t, c.Input), c.ModelNameSuffix)
		rq.Options.PackageName = c.PackageName
		rq.Format = core.InputFormat(c.InputFormat)
		rq.SourceName = c.Input
		if c.SourceName != nil {
			rq.SourceName = *c.SourceName
		}
		generateRequest(rq, "corpus-"+c.ID)
		positive++
	}
	marshalSchemas := func(s map[string]any) []byte {
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": s}})
		if err != nil {
			t.Fatal(err)
		}
		return input
	}
	vocabulary := make([]string, 240)
	for i := range vocabulary {
		vocabulary[i] = fmt.Sprintf("v%d", i)
	}
	generate(marshalSchemas(map[string]any{"BoundaryStatus": map[string]any{"type": "string", "enum": vocabulary}, "TypelessBoundary": map[string]any{"enum": vocabulary}}), "org.example.enum_limits", "", "enum-limits")
	primitive := map[string]any{"LongScalar": map[string]any{"type": "integer", "format": "int64"}}
	for _, tc := range []struct {
		name                 string
		count                int
		nullable, nativeEnum bool
	}{{"Wide", 120, false, false}, {"Boxed", 240, true, false}, {"NativeEnumWide", 120, false, true}} {
		fields := map[string]any{}
		required := make([]string, 0, tc.count)
		for i := range tc.count {
			name := fmt.Sprintf("field%d", i)
			required = append(required, name)
			if tc.nativeEnum {
				fields[name] = map[string]any{"type": "integer", "format": "int64", "enum": []int{1, 2}}
			} else {
				fields[name] = map[string]any{"$ref": "#/components/schemas/LongScalar", "nullable": tc.nullable}
			}
		}
		primitive[tc.name] = map[string]any{"type": "object", "properties": fields, "required": required}
	}
	generate(marshalSchemas(primitive), "org.example.primitive_limits", "", "primitive-limits")
	constants := map[string]any{}
	for i := range 120 {
		constants[fmt.Sprintf("field%d", i)] = map[string]any{"type": "integer", "format": "int64", "const": i}
	}
	generate(marshalSchemas(map[string]any{"ConstWide": map[string]any{"type": "object", "properties": constants}}), "org.example.const_limits", "", "const-limits")
	base := map[string]any{}
	required := []string{}
	for i := range 119 {
		name := fmt.Sprintf("field%d", i)
		base[name] = map[string]any{"type": "integer", "format": "int64"}
		required = append(required, name)
	}
	generate(marshalSchemas(map[string]any{
		"BaseWide":   map[string]any{"type": "object", "properties": base, "required": required},
		"MergedWide": map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/BaseWide"}, map[string]any{"properties": map[string]any{"field119": map[string]any{"type": "integer", "format": "int64"}}, "required": []string{"field119"}}}},
	}), "org.example.composition_limits", "", "composition-limits")
	generate(schemas(`{"BoolEnum":{"type":"boolean","enum":[true]},"IntEnum":{"type":"integer","enum":[0]},"LongEnum":{"type":"integer","format":"int64","enum":[0]},"FloatEnum":{"type":"number","format":"float","enum":[1.0]},"DoubleEnum":{"type":"number","enum":[1.0]},"Pinned":{"type":"integer","format":"int64","const":-9223372036854775808},"Record":{"type":"object","required":["native","items"],"properties":{"native":{"$ref":"#/components/schemas/IntEnum"},"items":{"type":"array","items":{"type":"boolean","const":true}},"marker":{"$ref":"#/components/schemas/Pinned","nullable":true},"wrapped":{"allOf":[{"$ref":"#/components/schemas/Pinned"}],"nullable":true},"code":{"type":"integer","const":-2147483648},"fraction":{"type":"number","format":"float","const":1.25}}}}`), "org.example.scalar_port", "", "scalar-port")
	dictionaryFields := map[string]any{}
	dictionaryRequired := []string{}
	for i := range 240 {
		key := fmt.Sprintf("field%d", i)
		dictionaryFields[key] = map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "integer", "format": "int64"}}
		dictionaryRequired = append(dictionaryRequired, key)
	}
	generate(marshalSchemas(map[string]any{"Wide": map[string]any{"type": "object", "properties": dictionaryFields, "required": dictionaryRequired}}), "org.example.dictionary_limits", "", "dictionary-limits")
	// Reuse the HTTP fixture's ordinary models without bringing HTTP dependencies
	// into this project. Both policies use the same guarded Core/CLI export seam.
	nativeInput, err := os.ReadFile(filepath.Join(repo, "tests", "fixtures", "http", "nested-native-enum-collection-defaults.json"))
	if err != nil {
		t.Fatal(err)
	}
	var nativeDocument map[string]any
	nativeDecoder := json.NewDecoder(bytes.NewReader(nativeInput))
	nativeDecoder.UseNumber() // Preserve exact Long extrema and original enum tokens.
	if err := nativeDecoder.Decode(&nativeDocument); err != nil {
		t.Fatal(err)
	}
	delete(nativeDocument, "paths")
	for _, checked := range []bool{false, true} {
		directory := "nested-native-enum-models"
		pkg := "org.example.nestednativeenum_models"
		if checked {
			directory += "-checked"
			pkg += "_checked"
			var visit func(map[string]any)
			visit = func(schema map[string]any) {
				if values, ok := schema["enum"].([]any); ok && len(values) == 3 {
					if values[0] == json.Number("1") && values[1] == json.Number("2") && values[2] == json.Number("3") {
						schema["minimum"], schema["maximum"], schema["multipleOf"] = 2, 3, 1
					}
					if values[0] == json.Number("0.5") && values[1] == json.Number("1") && values[2] == json.Number("1.5") {
						schema["minimum"], schema["maximum"] = 0.5, 1
					}
				}
				for _, key := range []string{"items", "additionalProperties"} {
					if child, ok := schema[key].(map[string]any); ok {
						visit(child)
					}
				}
				if properties, ok := schema["properties"].(map[string]any); ok {
					for _, value := range properties {
						visit(value.(map[string]any))
					}
				}
			}
			models := nativeDocument["components"].(map[string]any)["schemas"].(map[string]any)
			for _, value := range models {
				visit(value.(map[string]any))
			}
			fields := models["Holder"].(map[string]any)["properties"].(map[string]any)
			required := fields["required"].(map[string]any)
			required["minItems"] = 1
			required["items"].(map[string]any)["minItems"] = 1
			for _, native := range []string{"int", "long", "float", "double", "bool"} {
				rows := fields[native+"Rows"].(map[string]any)
				rows["maxItems"] = 3
				rows["items"].(map[string]any)["maxItems"] = 2
				listMap := fields[native+"ListMap"].(map[string]any)
				listMap["items"].(map[string]any)["maxProperties"] = 2
				mapList := fields[native+"MapList"].(map[string]any)
				mapList["additionalProperties"].(map[string]any)["maxItems"] = 2
				mapMap := fields[native+"MapMap"].(map[string]any)
				mapMap["additionalProperties"].(map[string]any)["maxProperties"] = 2
			}
		}
		input, err := json.Marshal(nativeDocument)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.SourceName = "nested-native-enum-collection-defaults.json"
		rq.Options.PackageName = pkg
		rq.Options.Validation = checked
		generateRequest(rq, directory)
	}
	contracts := []string{"EnumContracts", "PrimitiveContracts", "NullableCompositionContracts", "ConstContracts", "ObjectCompositionContracts", "EnumContextContracts", "DictionaryContracts"}
	if withPrimitive {
		contracts = append(contracts, "PrimitiveUnionContracts", "UnionNormalizationContracts")
		kinds := []string{"string", "integer", "number", "boolean"}
		combos := map[string]any{}
		for mask := 1; mask < 16; mask++ {
			var branches []any
			for bit, kind := range kinds {
				if mask&(1<<bit) != 0 {
					branches = append(branches, map[string]any{"type": kind})
				}
			}
			if len(branches) >= 2 {
				combos[fmt.Sprintf("case%d", mask)] = map[string]any{"oneOf": branches}
			}
		}
		generate(marshalSchemas(map[string]any{"CombinationEnvelope": map[string]any{"type": "object", "properties": combos}}), "org.example.primitive_union_combinations", "", "primitive-kind-combinations")
		wide := map[string]any{"anyOf": []any{map[string]any{"type": "integer", "minimum": int64(-2147483649)}, map[string]any{"type": "integer", "maximum": int64(2147483648)}}}
		fields := map[string]any{}
		names := []string{}
		for i := range 120 {
			name := fmt.Sprintf("field%d", i)
			fields[name] = wide
			names = append(names, name)
		}
		vocab := make([]string, 240)
		for i := range vocab {
			vocab[i] = fmt.Sprintf("token_%d", i)
		}
		generate(marshalSchemas(map[string]any{
			"WideFold":   map[string]any{"type": "object", "properties": fields, "required": names},
			"EnumHolder": map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"anyOf": []any{map[string]any{"enum": vocab[:120]}, map[string]any{"enum": vocab[120:]}, map[string]any{"type": "string"}}}}, "required": []string{"value"}},
		}), "org.example.union_normalization_limits", "", "normalization-limits")
		generate(schemas(`{"Choice":{"oneOf":[{"type":"integer","format":"int32","const":12},{"type":"string","enum":["known"]},{"type":"boolean"},{"type":"null"}]},"Mixed":{"anyOf":[{"type":"object","required":["id"],"properties":{"id":{"type":"string"},"tag":{"type":"string","const":"object"}}},{"type":"string"},{"type":"null"}]},"Carrier":{"type":"object","required":["value","items"],"properties":{"value":{"$ref":"#/components/schemas/Choice"},"items":{"type":"array","items":{"$ref":"#/components/schemas/Choice"}}}}}`), "org.example.root_primitive_port", "", "root-primitive-port")
		rootFields := map[string]any{}
		rootNames := []string{}
		for i := range 120 {
			name := fmt.Sprintf("field%d", i)
			rootFields[name] = map[string]any{"type": "integer", "format": "int64"}
			rootNames = append(rootNames, name)
		}
		generate(marshalSchemas(map[string]any{"WideRoot": map[string]any{"oneOf": []any{map[string]any{"type": "object", "properties": rootFields, "required": rootNames}, map[string]any{"type": "boolean"}}}}), "org.example.root_primitive_limits", "", "root-primitive-limits")
		write("src/main/kotlin/org/example/contracts/RootPrimitivePortContracts.kt", []byte(rootPrimitivePortRuntime))
	}
	if withObject {
		contracts = append(contracts, "ObjectUnionContracts", "RootUnionContracts", "RootKindContracts", "ScalarProjectionContracts")
		exportObjectResources(generate, marshalSchemas)
	}
	if withHierarchy {
		contracts = append(contracts, "HierarchyContracts")
		exportHierarchyResources(generate, marshalSchemas)
		write("src/main/kotlin/org/example/contracts/HierarchyPortContracts.kt", []byte(hierarchyPortRuntime))
	}
	consumers := append([]string{}, contracts...)
	consumers = append(consumers, "NestedNativeEnumOwnDefaultsModelContracts")
	for _, name := range consumers {
		input, err := os.ReadFile(filepath.Join(repo, "samples", "kotlin-smoke", "src", "main", "kotlin", "org", "example", "contracts", name+".kt"))
		if err != nil {
			t.Fatal(err)
		}
		write("src/main/kotlin/org/example/contracts/"+name+".kt", input)
	}
	write("src/main/kotlin/org/example/contracts/ScalarPortContracts.kt", []byte(scalarPortRuntime))
	write("settings.gradle.kts", []byte("pluginManagement { repositories { gradlePluginPortal(); mavenCentral() } }\nrootProject.name = \"openapi-kt-go-model-smoke\"\n"))
	build := `import org.jetbrains.kotlin.gradle.dsl.JvmTarget
plugins {
    kotlin("jvm") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}
repositories { mavenCentral() }
dependencies { implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0") }
java { sourceCompatibility = JavaVersion.VERSION_17; targetCompatibility = JavaVersion.VERSION_17 }
kotlin {
    compilerOptions.jvmTarget.set(JvmTarget.JVM_17)
    sourceSets.main { kotlin.srcDir(layout.projectDirectory.dir("generated")) }
}
`
	tasks := append(contracts, "ScalarPortContracts")
	if withPrimitive {
		tasks = append(tasks, "RootPrimitivePortContracts")
	}
	if withHierarchy {
		tasks = append(tasks, "HierarchyPortContracts")
	}
	for _, name := range tasks {
		build += fmt.Sprintf("\ntasks.register<JavaExec>(\"run%s\") {\n    dependsOn(tasks.named(\"classes\"))\n    classpath = sourceSets[\"main\"].runtimeClasspath\n    mainClass.set(\"org.example.contracts.%sKt\")\n}\n", name, name)
	}
	write("build.gradle.kts", []byte(build))
	t.Logf("exported %d original positive corpus cases, %d semantic fixtures baseline/suffix and enum/slot limits; %d generated-source runtime suites", positive, len(fixtures), len(tasks))
}

const scalarPortRuntime = `@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.*
import org.example.scalar_port.models.*

fun main() {
    check(Json.encodeToString<BoolEnum>(false) == "false")
    check(Json.encodeToString<IntEnum>(2147483647) == "2147483647")
    check(Json.encodeToString<LongEnum>(Long.MAX_VALUE) == "9223372036854775807")
    check(Json.encodeToString<FloatEnum>(3.125f) == "3.125")
    check(Json.encodeToString<DoubleEnum>(-0.5) == "-0.5")
    check(Json.encodeToString<Pinned>(5L) == "5")
    check(Json.decodeFromString<Pinned>("5") == 5L)
    check(Record.serializer().descriptor.elementsCount == 6)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = Record(native = 7, items = listOf(false, true))
        val wire = json.encodeToJsonElement(Record.serializer(), value).jsonObject
        check(wire.getValue("native").jsonPrimitive.int == 7 && !wire.getValue("native").jsonPrimitive.isString)
        check(wire.getValue("items").jsonArray.map { it.jsonPrimitive.boolean } == listOf(false, true))
        check(wire.getValue("marker").jsonPrimitive.long == Long.MIN_VALUE && !wire.getValue("marker").jsonPrimitive.isString)
        check(wire.getValue("wrapped").jsonPrimitive.long == Long.MIN_VALUE)
        check(wire.getValue("code").jsonPrimitive.int == Int.MIN_VALUE)
        check(wire.getValue("fraction").jsonPrimitive.float == 1.25f)
        check(json.decodeFromJsonElement(Record.serializer(), wire) == value)
        val decoded = json.decodeFromString(Record.serializer(), """{"native":1,"items":[false]}""")
        check(decoded.marker == Long.MIN_VALUE && decoded.wrapped == Long.MIN_VALUE)
        check(decoded.code == Int.MIN_VALUE && decoded.fraction == 1.25f)
        check(runCatching { json.decodeFromJsonElement(Record.serializer(), JsonObject(wire - "native")) }.isFailure)
        check(runCatching { json.decodeFromJsonElement(Record.serializer(), JsonObject(wire + ("marker" to JsonNull))) }.isFailure)
    }
    println("Go-generated native aliases/const-reference/scalar-item contracts passed")
}
`

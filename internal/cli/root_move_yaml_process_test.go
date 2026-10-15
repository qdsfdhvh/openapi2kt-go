package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveNativeExistingYAMLFileAndStdinByteControl(t *testing.T) {
	base := t.TempDir()
	yaml := []byte("openapi: 3.0.3\ncomponents:\n  schemas:\n    Record:\n      type: object\n      required: [id]\n      properties:\n        id: {type: string}\n")
	input := filepath.Join(base, "openapi.yaml")
	put(t, input, yaml)
	src, dst := filepath.Join(base, "old"), filepath.Join(base, "new")
	args := []string{"generate", "--input", input, "--output", src, "--package-name=org.example.moved", "--diagnostic-format=json"}
	result(t, invoke(t, base, args...), 0)
	before := snapshot(t, src)
	moved := []string{"generate", "--input", "-", "--input-format=yaml", "--move-output-from", src, "--output", dst, "--package-name=org.example.moved", "--diagnostic-format=json"}
	result(t, invokeInput(t, base, bytes.NewReader(yaml), append(moved, "--check")...), 3)
	result(t, invokeInput(t, base, bytes.NewReader(yaml), lifecycleValidate(moved)...), 0)
	if !reflect.DeepEqual(before, snapshot(t, src)) {
		t.Fatal("YAML readonly move changed source")
	}
	result(t, invokeInput(t, base, bytes.NewReader(yaml), moved...), 0)
	after := snapshot(t, dst)
	for path, old := range before {
		if path != ".openapi-kt.lock" && !reflect.DeepEqual(old, after[path]) {
			t.Fatal("YAML file/stdin changed moved bytes", path)
		}
	}
	args[4] = dst
	result(t, invoke(t, base, append(args, "--check")...), 0)
	result(t, invoke(t, base, args...), 0)
	if !reflect.DeepEqual(after, snapshot(t, dst)) {
		t.Fatal("YAML target ordinary generation rewrote bytes/mtime")
	}
}

func TestRootMoveNativeReferencedCollectionDefaultsJSONYAMLByteControl(t *testing.T) {
	jsonInput := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows","default":[{"b":2,"a":1}]}}},"Rows":{"type":"array","items":{"additionalProperties":{"type":"integer","format":"int64"}}}}}}`)
	yamlInput := []byte("openapi: 3.1.0\ncomponents:\n  schemas:\n    Holder:\n      properties:\n        rows:\n          $ref: '#/components/schemas/Rows'\n          default: [{b: 2, a: 1}]\n    Rows:\n      type: array\n      items:\n        additionalProperties: {type: integer, format: int64}\n")
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			t.Run(style+"/"+map[bool]string{false: "off", true: "on"}[checked], func(t *testing.T) {
				var jsonKotlin map[string]string
				for _, format := range []string{"json", "yaml"} {
					base := t.TempDir()
					input := filepath.Join(base, "input."+format)
					src, dst, control := filepath.Join(base, "old"), filepath.Join(base, "new"), filepath.Join(base, "control")
					data := jsonInput
					if format == "yaml" {
						data = yamlInput
					}
					put(t, input, data)
					args := []string{"generate", "--input", input, "--output", src, "--package-name=org.example.referenceddefaults", "--model-style=" + style, "--diagnostic-format=json"}
					if checked {
						args = append(args, "--validation")
					}
					result(t, invoke(t, base, args...), 0)
					put(t, filepath.Join(src, "Notes.kt"), []byte("// handwritten\n"))
					before := snapshot(t, src)
					move := append([]string(nil), args...)
					move[4] = dst
					move = append(move, "--move-output-from", src)
					result(t, invoke(t, base, append(move, "--check")...), 3)
					result(t, invoke(t, base, lifecycleValidate(move)...), 0)
					if !reflect.DeepEqual(before, snapshot(t, src)) {
						t.Fatal("referenced defaults readonly move changed source", format)
					}
					if _, err := os.Stat(dst); !os.IsNotExist(err) {
						t.Fatal("readonly move created target", format, err)
					}
					result(t, invoke(t, base, move...), 0)
					after := snapshot(t, dst)
					for path, old := range before {
						if path == ".openapi-kt.lock" || path == "Notes.kt" {
							continue
						}
						if !reflect.DeepEqual(old, after[path]) {
							t.Fatal("referenced default move changed bytes/attributes", format, path)
						}
					}
					if snapshot(t, src)["Notes.kt"] != before["Notes.kt"] {
						t.Fatal("handwritten source changed")
					}
					fresh := append([]string(nil), args...)
					fresh[4] = control
					result(t, invoke(t, base, fresh...), 0)
					kotlin := map[string]string{}
					for path, file := range snapshot(t, control) {
						if filepath.Ext(path) == ".kt" {
							if file.Bytes != after[path].Bytes {
								t.Fatal("fresh referenced-default control differs", format, path)
							}
							kotlin[path] = file.Bytes
						}
					}
					if format == "json" {
						jsonKotlin = kotlin
					} else if !reflect.DeepEqual(jsonKotlin, kotlin) {
						t.Fatal("JSON/YAML referenced-default Kotlin differs")
					}
					ordinary := append([]string(nil), args...)
					ordinary[4] = dst
					result(t, invoke(t, base, append(ordinary, "--check")...), 0)
					result(t, invoke(t, base, ordinary...), 0)
					if !reflect.DeepEqual(after, snapshot(t, dst)) {
						t.Fatal("referenced-default target no-op changed bytes/attributes")
					}
				}
			})
		}
	}
}

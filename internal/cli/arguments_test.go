package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestArgumentContracts(t *testing.T) {
	for _, args := range [][]string{
		{"generate", "--input=x.json", "--output=out", "--package-name=org.example", "--model-name-suffix=", "--"},
		{"validate", "--input=x", "--api-client-name=ChosenClient", "--exclude-api=WidgetsApi", "--exclude-api=DefaultApi"},
		{"validate", "--input=x", "--validation"},
		{"validate", "--input", "-"}, {"generate", "--input=-file", "--output=o", "--package-name=p", "--check"},
	} {
		parsed, display, valid := parse(args)
		if !valid || display != nil || parsed.input == "" {
			t.Fatal(args, parsed, display, valid)
		}
	}
	for _, args := range [][]string{{"-hV"}, {"-Vh"}, {"generate", "-h"}} {
		_, d, valid := parse(args)
		if !valid || d == nil {
			t.Fatal(args)
		}
	}
	for _, args := range [][]string{
		{"generate", "--input", "--output"}, {"validate", "--input=-file", "--check"},
		{"validate", "--input=x", "--api-client-name=A", "--api-client-name=B"},
		{"validate", "--input=x", "--exclude-api"},
		{"validate", "--input=x", "--validation=true"},
		{"validate", "--input=x", "--validation", "--validation"},
		{"validate", "--input=x", "--validation", "false"},
		{"validate", "--input=x", "--input=x"}, {"validate", "--input=x", "--input-format="},
		{"generate", "--check=false"}, {"validate", "--input=x", "--", "positional"},
	} {
		_, _, valid := parse(args)
		if valid {
			t.Fatal("accepted uncontracted arguments", args)
		}
	}
	parsed, _, valid := parse([]string{"validate", "--input=x"})
	if !valid || parsed.packageName != "org.example.generated" || parsed.suffix != "" || parsed.validation {
		t.Fatal(parsed)
	}
	parsed, _, valid = parse([]string{"validate", "--input=x", "--validation"})
	if !valid || !parsed.validation {
		t.Fatal(parsed)
	}
}

func FuzzArguments(f *testing.F) {
	for _, seed := range []string{"generate\x00--input=x\x00--output=y\x00--package-name=p\x00--hoist-shared-shapes\x00--lock-hoisted-names", "validate\x00--input=x\x00--output=y\x00--model-name-suffix=Dto\x00--hoist-shared-shapes\x00--lock-hoisted-names=true", "generate\x00--config=x\x00--spec=a\x00--lock-hoisted-names=false", "download\x00--lock-hoisted-names", "generate\x00--config=x\x00--spec=a\x00--hoist-shared-shapes=false\x00--qualify-hoisted-names=false", "validate\x00--input=x\x00--hoist-shared-shapes\x00--qualify-hoisted-names", "generate\x00--config=x\x00--spec=a\x00--deduplicate-models=false", "validate\x00--input=x\x00--deduplicate-models", "download\x00--config=x\x00--spec=a\x00--header-env=X-Trace=ENV_A", "generate\x00--config=x\x00--metrics\x00--check", "validate\x00--config=x\x00--spec=a\x00--metrics", "generate\x00--config=x\x00--spec=a\x00--runtime-package=org.example.shared.runtime", "generate\x00--input=x\x00--output=y\x00--package-name=org.example.a\x00--runtime-package=", "generate\x00--config=x\x00--spec=a\x00--adopt-legacy\x00--check", "generate\x00--config=x\x00--adopt-legacy", "generate\x00--config=x\x00--spec=a\x00--adopt-legacy=true", "generate\x00--config=x\x00--spec=a\x00--check", "generate\x00--config=x\x00--reconcile", "validate\x00--config=x\x00--input=x", "download\x00--url=https://example.test/spec\x00--output=spec.json", "", "--diagnostic-format=json", "generate\x00--input=x\x00--output=o\x00--package-name=p", "validate\x00--help", "help\x00generate", "validate\x00--input=x\x00--validation", "validate\x00--input=x\x00--api-client-name=A\x00--exclude-api=DefaultApi\x00--exclude-api=OtherApi", "generate\x00--input=x\x00--output=o\x00--package-name=p\x00--deny-warnings\x00--allow-warning=W_JSON_ELEMENT_APPROXIMATION", "validate\x00--input=x\x00--deny-warnings\x00--allow-warning=W_BAD,*", "generate\x00--input=x\x00--output=o\x00--package-name=p\x00--metrics\x00--check", "validate\x00--input=x\x00--metrics=true"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 65536 {
			t.Skip()
		}
		args := strings.Split(input, "\x00")
		parsed, display, valid := parse(args)
		if display != nil && !valid {
			t.Fatal("invalid display")
		}
		if !valid || display != nil {
			return
		}
		if parsed.command != "generate" && parsed.command != "validate" {
			t.Fatal("unknown command")
		}
		if parsed.diagnosticFormat != "text" && parsed.diagnosticFormat != "json" {
			t.Fatal("format")
		}
	})
}
func TestUsageJSONNeverEchoesUntrustedArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"synthetic_secret_bad_command", "--diagnostic-format=json"}, &stdout, &stderr)
	if code != 2 || stderr.Len() != 0 || bytes.Contains(stdout.Bytes(), []byte("synthetic_secret")) {
		t.Fatal(code, stdout.String(), stderr.String())
	}
	var r report
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil || r.Diagnostics[0].Code != "E_USAGE" {
		t.Fatal(err, r)
	}
}

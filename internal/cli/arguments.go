// Package cli is the thin command/IO boundary around the pure generator.
package cli

import (
	"io"
	"strings"

	"openapi-kt/core"
	"openapi-kt/internal/naming"
	"openapi-kt/internal/output"
)

const version = "0.2.0-alpha.1"

type arguments struct {
	command, input, output, packageName, suffix, diagnosticFormat               string
	format                                                                      core.InputFormat
	check, validation, denyWarnings, metrics                                    bool
	allowWarnings                                                               []string
	clientName, runtimePackage                                                  string
	modelStyle                                                                  core.ModelStyle
	deduplicateModels, hoistSharedShapes, qualifyHoistedNames, lockHoistedNames bool
	excludeAPIs, excludeMergeModels                                             []string
	excludeMergeModelPatterns, excludeMergeModelSuffixes                        []string
	mergeIdenticalModels                                                        bool
	inputReader                                                                 io.Reader // Caller-owned stdin, selected only by --input -.
	config, spec                                                                string
	reconcile, adoptLegacy                                                      bool
	adoptLockedOwner                                                            bool
	nameLockAction                                                              output.NameLockAction
	moveOutputFrom                                                              string            // invocation-only complete root move
	ownerRenames                                                                map[string]string // new -> old; invocation-only
	explicit                                                                    map[string]bool
}
type display struct{ text string }

func wantsJSON(args []string) bool {
	for index, arg := range args {
		if arg == "--diagnostic-format=json" || arg == "--diagnostic-format" && index+1 < len(args) && args[index+1] == "json" {
			return true
		}
	}
	return false
}

func parse(args []string) (arguments, *display, bool) {
	var result arguments
	if len(args) == 0 {
		return result, nil, false
	}
	if strings.HasPrefix(args[0], "-h") && !strings.HasPrefix(args[0], "--") {
		return result, &display{help("")}, true
	}
	if strings.HasPrefix(args[0], "-V") {
		return result, &display{"openapi-kt " + version + "\n"}, true
	}
	switch args[0] {
	case "--help", "-h":
		return result, &display{help("")}, true
	case "--version", "-V":
		return result, &display{"openapi-kt " + version + "\n"}, true
	case "help":
		if len(args) == 1 {
			return result, &display{help("")}, true
		}
		if len(args) == 2 && (args[1] == "generate" || args[1] == "validate") {
			return result, &display{help(args[1])}, true
		}
		return result, nil, false
	case "generate", "validate":
		result.command = args[0]
	default:
		return result, nil, false
	}
	result.format = core.Auto
	result.diagnosticFormat = "text"
	if result.command == "validate" {
		result.packageName = "org.example.generated"
	}
	seen := map[string]bool{}
	for index := 1; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			if index == len(args)-1 {
				break
			}
			return result, nil, false
		}
		if arg == "--help" || strings.HasPrefix(arg, "-h") && !strings.HasPrefix(arg, "--") {
			return result, &display{help(result.command)}, true
		}
		key, value, inline := strings.Cut(arg, "=")
		allowed := key == "--config" || key == "--spec" || key == "--input" || key == "--input-format" || key == "--diagnostic-format" || key == "--package-name" || key == "--runtime-package" || key == "--model-style" || key == "--deduplicate-models" || key == "--merge-identical-models" || key == "--exclude-merge-model" || key == "--exclude-merge-model-pattern" || key == "--exclude-merge-model-suffix" || key == "--hoist-shared-shapes" || key == "--qualify-hoisted-names" || key == "--lock-hoisted-names" || key == "--name-lock-action" || key == "--move-output-from" || key == "--rename-owner" || key == "--adopt-locked-owner" || key == "--output" || key == "--model-name-suffix" || key == "--api-client-name" || key == "--exclude-api" || key == "--validation" || key == "--deny-warnings" || key == "--allow-warning" || key == "--metrics"
		allowed = allowed || result.command == "generate" && (key == "--output" || key == "--model-name-suffix" || key == "--check" || key == "--reconcile" || key == "--adopt-legacy")
		if !allowed || seen[key] && key != "--exclude-api" && key != "--exclude-merge-model" && key != "--exclude-merge-model-pattern" && key != "--exclude-merge-model-suffix" && key != "--allow-warning" && key != "--rename-owner" {
			return result, nil, false
		}
		seen[key] = true
		if key == "--deduplicate-models" || key == "--merge-identical-models" || key == "--hoist-shared-shapes" || key == "--qualify-hoisted-names" || key == "--lock-hoisted-names" {
			if inline && value != "true" && value != "false" {
				return result, nil, false
			}
			setting := !inline || value == "true"
			switch key {
			case "--merge-identical-models":
				result.mergeIdenticalModels = setting
			case "--deduplicate-models":
				result.deduplicateModels = setting
			case "--hoist-shared-shapes":
				result.hoistSharedShapes = setting
			case "--qualify-hoisted-names":
				result.qualifyHoistedNames = setting
			case "--lock-hoisted-names":
				result.lockHoistedNames = setting
			}
			continue
		}
		if key == "--check" || key == "--validation" || key == "--deny-warnings" || key == "--metrics" || key == "--reconcile" || key == "--adopt-legacy" || key == "--adopt-locked-owner" {
			if inline {
				return result, nil, false
			}
			if key == "--check" {
				result.check = true
			} else if key == "--validation" {
				result.validation = true
			} else if key == "--deny-warnings" {
				result.denyWarnings = true
			} else if key == "--adopt-locked-owner" {
				result.adoptLockedOwner = true
			} else if key == "--adopt-legacy" {
				result.adoptLegacy = true
			} else if key == "--reconcile" {
				result.reconcile = true
			} else {
				result.metrics = true
			}
			continue
		}
		if !inline {
			index++
			if index >= len(args) {
				return result, nil, false
			}
			value = args[index]
			if strings.HasPrefix(value, "-") && value != "-" {
				return result, nil, false
			}
		}
		switch key {
		case "--move-output-from":
			result.moveOutputFrom = value
		case "--rename-owner":
			old, next, ok := strings.Cut(value, ":")
			if !ok || !ownerRenameIdentifier(old) || !ownerRenameIdentifier(next) {
				return result, nil, false
			}
			if result.ownerRenames == nil {
				result.ownerRenames = map[string]string{}
			}
			if _, repeated := result.ownerRenames[next]; repeated {
				return result, nil, false
			}
			for _, source := range result.ownerRenames {
				if source == old {
					return result, nil, false
				}
			}
			result.ownerRenames[next] = old
		case "--name-lock-action":
			if value != string(output.NameLockReset) && value != string(output.NameLockDisable) {
				return result, nil, false
			}
			result.nameLockAction = output.NameLockAction(value)
		case "--config":
			result.config = value
		case "--spec":
			result.spec = value
		case "--input":
			result.input = value
		case "--output":
			result.output = value
		case "--package-name":
			result.packageName = value
		case "--model-style":
			if value != "" && value != string(core.DataClass) && value != string(core.PlainClass) {
				return result, nil, false
			}
			result.modelStyle = core.ModelStyle(value)
		case "--runtime-package":
			result.runtimePackage = value
		case "--model-name-suffix":
			result.suffix = value
		case "--api-client-name":
			result.clientName = value
		case "--exclude-merge-model":
			result.excludeMergeModels = append(result.excludeMergeModels, value)
		case "--exclude-merge-model-pattern":
			result.excludeMergeModelPatterns = append(result.excludeMergeModelPatterns, value)
		case "--exclude-merge-model-suffix":
			result.excludeMergeModelSuffixes = append(result.excludeMergeModelSuffixes, value)
		case "--exclude-api":
			result.excludeAPIs = append(result.excludeAPIs, value)
		case "--allow-warning":
			if !warningCode(value) {
				return result, nil, false
			}
			result.allowWarnings = append(result.allowWarnings, value)
		case "--input-format":
			if value != "auto" && value != "json" && value != "yaml" {
				return result, nil, false
			}
			result.format = core.InputFormat(value)
		case "--diagnostic-format":
			if value != "text" && value != "json" {
				return result, nil, false
			}
			result.diagnosticFormat = value
		}
	}
	if seen["--allow-warning"] && !result.denyWarnings {
		return result, nil, false
	}
	if result.adoptLockedOwner && (!seen["--config"] || !seen["--spec"] || result.spec == "" || result.adoptLegacy || result.reconcile || result.nameLockAction != "" || len(result.ownerRenames) != 0 || seen["--move-output-from"]) {
		return result, nil, false
	}
	if seen["--move-output-from"] && (result.moveOutputFrom == "" || !seen["--output"] || result.output == "" || seen["--spec"] || result.adoptLegacy || result.reconcile || result.nameLockAction != "" || len(result.ownerRenames) != 0 || result.adoptLockedOwner) {
		return result, nil, false
	}
	if len(result.ownerRenames) != 0 && (!seen["--config"] || result.adoptLegacy || result.reconcile || result.nameLockAction != "") {
		return result, nil, false
	}
	if result.nameLockAction != "" && (result.adoptLegacy || result.reconcile) {
		return result, nil, false
	}
	if seen["--config"] {
		if result.config == "" || seen["--input"] || seen["--spec"] && result.spec == "" || result.reconcile && seen["--spec"] || result.adoptLegacy && (!seen["--spec"] || result.reconcile) {
			return result, nil, false
		}
		result.explicit = seen
		return result, nil, true
	}
	if naming.ValidateMergeExclusions(result.mergeIdenticalModels, result.excludeMergeModels, result.excludeMergeModelPatterns, result.excludeMergeModelSuffixes) != nil {
		return result, nil, false
	}
	if (result.qualifyHoistedNames || result.lockHoistedNames) && !result.hoistSharedShapes {
		return result, nil, false
	}
	if !validNameLockAction(result) {
		return result, nil, false
	}
	if result.command == "validate" && ((seen["--output"] || seen["--model-name-suffix"]) && !result.lockHoistedNames && result.nameLockAction == "" && result.moveOutputFrom == "" || (result.lockHoistedNames || result.nameLockAction != "") && (!seen["--output"] || result.output == "")) {
		return result, nil, false
	}
	if seen["--spec"] || seen["--reconcile"] || seen["--adopt-legacy"] {
		return result, nil, false
	}
	if !seen["--input"] || result.command == "generate" && (!seen["--output"] || !seen["--package-name"]) {
		return result, nil, false
	}
	return result, nil, true
}

func help(command string) string {
	if command == "" {
		return `Generate Kotlin models and supported HTTP clients from a local OpenAPI JSON document

Usage: openapi-kt <COMMAND>

Commands:
  generate  Generate Kotlin sources, or compare them without writing using --check
  validate  Check this generator's supported subset; not full OpenAPI validation
  help      Print this message or the help of the given subcommand(s)

Options:
  -h, --help     Print help
  -V, --version  Print version
`
	}
	if command != "generate" && command != "validate" {
		return ""
	}
	usage := "openapi-kt validate [OPTIONS] --input <INPUT>"
	description := "Check this generator's supported subset; not full OpenAPI validation."
	if command == "generate" {
		usage = "openapi-kt generate [OPTIONS] --input <INPUT> --output <OUTPUT> --package-name <PACKAGE_NAME>"
		description = "Generate Kotlin sources, or compare them without writing using --check."
	}
	text := description + "\n\nUsage: " + usage + "\n\nOptions:\n  --input <INPUT>                         Local JSON/YAML file; - reads stdin until EOF (8 MiB limit)\n  --input-format <INPUT_FORMAT>            [default: auto] [possible values: auto, json, yaml]\n  --diagnostic-format <DIAGNOSTIC_FORMAT>  [default: text] [possible values: text, json]\n"
	if command == "generate" {
		text += "  --output <OUTPUT>\n  --package-name <PACKAGE_NAME>\n  --model-name-suffix <MODEL_NAME_SUFFIX>  [default: \"\"]\n  --check\n"
	} else {
		text += "  --package-name <PACKAGE_NAME>            [default: org.example.generated]\n"
		text += "  --output <OUTPUT>                    Read-only state root; requires locking, lifecycle action, config rename or root move\n  --model-name-suffix <SUFFIX>          Profile suffix; requires locking, lifecycle action, config rename or root move\n"
	}
	text += "  --config <FILE>                      Local TOML configuration (disjoint output roots) instead of --input\n  --spec <NAME>                        Select one configured spec without dropping other owners\n"
	if command == "generate" {
		text += "  --reconcile                          Allow an explicitly empty configured inventory\n  --adopt-legacy                       Explicit exact-match v1 ownership migration; requires --spec\n"
	}
	text += "  --adopt-locked-owner                 Exact same-root v3 to v4 adoption; one configured spec and --spec\n"
	text += "  --move-output-from <SOURCE>        Complete v1-v4 root move to explicit --output; config requires full single-root inventory\n"
	text += "  --rename-owner <OLD:NEW>             Exact v2/v4 config owner rename; repeat for every selected spec\n"
	text += "  --merge-identical-models[=true|false] Merge eligible resolved DTO/enum roots; default false\n"
	text += "  --exclude-merge-model <FQ_NAME>       Exact root classifier, neither absorbs nor is absorbed; repeatable\n"
	text += "  --exclude-merge-model-pattern <REGEX> Whole FQ root match; bounded portable ASCII regex; repeatable\n"
	text += "  --exclude-merge-model-suffix <SUFFIX>  Literal case-sensitive generated short-name suffix; repeatable\n"
	text += "  --deduplicate-models[=true|false]    Fold equivalent operation DTO/enum roots; default false\n"
	text += "  --hoist-shared-shapes[=true|false]   Move repeated operation DTO/enum roots to models\n"
	text += "  --qualify-hoisted-names[=true|false] Keep owner-qualified hoisted names; requires hoist\n"
	text += "  --lock-hoisted-names[=true|false]    Persist/reuse hoisted names; requires hoist, default false\n"
	text += "  --name-lock-action <reset|disable>  Explicit existing-state lifecycle; review --check migration report\n"
	text += "  --model-style <STYLE>                data_class (default) or plain_class; empty resets\n"
	text += "  --runtime-package <PACKAGE>          Shared Patch/bigint helper package; empty uses models\n"
	text += "  --validation                         Enforce supported DTO/ref/element/container bounds and DTO JSON counts\n"
	text += "  --deny-warnings                      Reject unallowed generation warnings before output mutation\n  --allow-warning <CODE>                Repeat to allow W_* codes; requires --deny-warnings\n"
	text += "  --metrics                            Report CLI/core timings, planned sizes and available process peak RSS\n"
	text += "  --api-client-name <NAME>              [default: derived from info.title]\n  --exclude-api <CLASS_NAME>             Repeat to exclude exact API class names\n"
	return text + "  -h, --help                              Print help\n"
}

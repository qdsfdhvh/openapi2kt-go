package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"openapi-kt/core"
	"openapi-kt/internal/output"
)

type report struct {
	RetainedLockedAdoption  *retainedLockedAdoptionReport  `json:"retained_locked_adoption,omitempty"`
	LockedOwnerReassignment *lockedOwnerReassignmentReport `json:"locked_owner_reassignment,omitempty"`
	OutputRootMigration     *rootMoveReport                `json:"output_root_migration,omitempty"`
	LockedOwnerAdoptions    []lockedOwnerAdoptionReport    `json:"locked_owner_adoptions,omitempty"`
	NameLockMigrations      []nameLockMigration            `json:"name_lock_migrations,omitempty"`
	OwnerRenames            []ownerRenameReport            `json:"owner_renames,omitempty"`
	Status                  string                         `json:"status"`
	Files                   []string                       `json:"files"`
	Diagnostics             core.Diagnostics               `json:"diagnostics"`
	Warnings                core.Diagnostics               `json:"warnings,omitempty"`
	Metrics                 *executionMetrics              `json:"metrics,omitempty"`
	Spec                    string                         `json:"spec,omitempty"`
	Specs                   []specReport                   `json:"specs,omitempty"`
}

func failed(code, message string) report {
	return report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{{Code: code, Message: message}}}
}

func execute(args arguments) (int, report) {
	return executeWithClock(args, time.Now)
}

func executeWithClock(args arguments, now func() time.Time) (code int, result report) {
	timer := newMetricCollector(args.metrics, now)
	defer func() { result.Metrics = timer.finish() }()
	if args.config != "" {
		return executeConfig(args, timer)
	}
	if args.moveOutputFrom != "" {
		return executeDirectRootMove(args, timer)
	}
	previous, failureCode, failure := readPreviousNameLock(args, "", timer)
	if failureCode != 0 {
		return failureCode, report{Status: "error", Files: []string{}, Diagnostics: failure}
	}
	generated, failureCode, failure := readGeneratePrevious(args, timer, previous)
	if failureCode != 0 {
		return failureCode, report{Status: "error", Files: []string{}, Diagnostics: failure}
	}
	var migrations []nameLockMigration
	if args.nameLockAction != "" {
		migration, err := nameLockMigrationReport(args, "", previous, generated.Artifacts)
		if err != nil {
			return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
		}
		migrations = append(migrations, migration)
		if args.nameLockAction == output.NameLockDisable {
			generated.Artifacts.NameLock = nil
		}
	}
	timer.planned(generated.Artifacts)
	if args.denyWarnings {
		finishPolicy := timer.phase("warning_policy")
		denied := deniedWarnings(generated.Warnings, args.allowWarnings)
		finishPolicy()
		if len(denied) != 0 {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: denied, Warnings: generated.Warnings}
		}
	}
	changed := false
	if args.command == "generate" || args.nameLockAction != "" {
		var err *core.Diagnostic
		phase := "commit"
		readonly := args.check || args.command == "validate"
		if readonly {
			phase = "check"
		}
		finishOutput := timer.phase(phase)
		if args.nameLockAction != "" {
			changed, err = output.WriteNameLockLifecycle(args.output, generated.Artifacts, previous, args.nameLockAction, readonly)
		} else if args.lockHoistedNames {
			changed, err = output.WriteNameLocked(args.output, generated.Artifacts, previous, args.check)
		} else {
			changed, err = output.Write(args.output, generated.Artifacts, args.check)
		}
		finishOutput()
		if err != nil {
			return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
		}
	}
	status, code := "ok", 0
	if args.check && changed {
		status, code = "different", 3
	}
	files := make([]string, 0, len(generated.Artifacts.Files))
	for _, file := range generated.Artifacts.Files {
		files = append(files, file.Path)
	}
	return code, report{Status: status, Files: files, Diagnostics: core.Diagnostics{}, Warnings: generated.Warnings, NameLockMigrations: migrations}
}
func printReport(jsonMode bool, result report, stdout, stderr io.Writer) {
	if jsonMode {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(result)
		return
	}
	if len(result.Diagnostics) == 0 {
		_, _ = fmt.Fprintf(stdout, "%s: %d model file(s)\n", result.Status, len(result.Files))
	}
	for _, migration := range result.NameLockMigrations {
		_, _ = fmt.Fprintf(stdout, "name-lock %s", migration.Action)
		if migration.Spec != "" {
			_, _ = fmt.Fprintf(stdout, " spec=%s", migration.Spec)
		}
		_, _ = fmt.Fprintf(stdout, ": %d occurrence change(s), %d Kotlin file change(s)\n%s\n", len(migration.Changes), len(migration.Files), migration.Impact)
		for _, change := range migration.Changes {
			_, _ = fmt.Fprintf(stdout, "  %s %s: %s -> %s; descriptor %s -> %s; shape_changed=%t\n", change.Role, strconv.Quote(change.Occurrence), change.Before, change.After, change.DescriptorBefore, change.DescriptorAfter, change.ShapeChanged)
		}
		for _, file := range migration.Files {
			_, _ = fmt.Fprintf(stdout, "  %s: %s -> %s\n", file.Path, file.Before, file.After)
		}
	}
	if migration := result.OutputRootMigration; migration != nil {
		printRootMove(stdout, *migration)
	}
	for _, migration := range result.OwnerRenames {
		_, _ = fmt.Fprintf(stdout, "owner %s -> %s (manifest v%d): %d unchanged Kotlin file(s)\n%s\n", migration.OldOwner, migration.NewOwner, migration.Version, len(migration.Files), migration.Impact)
		for _, file := range migration.Files {
			_, _ = fmt.Fprintf(stdout, "  %s: %s\n", file.Path, file.SHA256)
		}
		if state := migration.NameLock; state != nil {
			_, _ = fmt.Fprintf(stdout, "  name-lock SHA256=%s; profile package=%s suffix=%s qualified=%t\n", state.SHA256, state.Profile.Package, strconv.Quote(state.Profile.Suffix), state.Profile.Qualified)
			for _, symbol := range state.Symbols {
				_, _ = fmt.Fprintf(stdout, "  %s %s; descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
			}
		}
	}
	if migration := result.RetainedLockedAdoption; migration != nil {
		printRetainedLockedAdoption(stdout, *migration)
	}
	if migration := result.LockedOwnerReassignment; migration != nil {
		printLockedOwnerReassignment(stdout, *migration)
	}
	for _, migration := range result.LockedOwnerAdoptions {
		if migration.FromOwner != "" {
			_, _ = fmt.Fprintf(stdout, "allocation donor %s\n", migration.FromOwner)
		}
		for _, symbol := range migration.SourceSymbols {
			_, _ = fmt.Fprintf(stdout, "  source %s %s; descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
		}
		_, _ = fmt.Fprintf(stdout, "locked owner %s (manifest v%d -> v%d): %d unchanged Kotlin file(s)\nmanifest SHA256 %s -> %s\n%s\n", migration.Owner, migration.BeforeVersion, migration.AfterVersion, len(migration.Files), migration.ManifestBeforeSHA256, migration.ManifestAfterSHA256, migration.Impact)
		for _, file := range migration.Files {
			_, _ = fmt.Fprintf(stdout, "  %s: %s\n", file.Path, file.SHA256)
		}
		state := migration.NameLock
		_, _ = fmt.Fprintf(stdout, "  name-lock SHA256 %s -> %s; profile package=%s suffix=%s qualified=%t\n", state.BeforeSHA256, state.AfterSHA256, state.Profile.Package, strconv.Quote(state.Profile.Suffix), state.Profile.Qualified)
		for _, symbol := range state.Symbols {
			_, _ = fmt.Fprintf(stdout, "  %s %s; descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
		}
	}
	for _, list := range []core.Diagnostics{result.Diagnostics, result.Warnings} {
		for _, d := range list {
			location := ""
			if d.Pointer != nil {
				location = " at " + strconv.Quote(*d.Pointer)
			}
			_, _ = fmt.Fprintf(stderr, "%s%s: %s\n", d.Code, location, d.Message)
		}
	}
	printMetrics(result.Metrics, stderr)
}

// Run uses process stdin only when --input - selects it. The pure generation
// core receives bounded bytes, never process streams or filesystem access.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWithInput(args, os.Stdin, stdout, stderr)
}

// RunWithInput accepts a caller-owned stream without closing or replacing it.
// Help and argument errors return before any input read.
func RunWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	parsed, display, valid := parse(args)
	if display != nil {
		_, _ = io.WriteString(stdout, display.text)
		return 0
	}
	if !valid {
		if wantsJSON(args) {
			printReport(true, failed("E_USAGE", "Invalid command arguments. See --help for usage."), stdout, stderr)
		} else {
			_, _ = io.WriteString(stderr, "error: Invalid command arguments. See --help for usage.\n\nUsage: openapi-kt <COMMAND>\n")
		}
		return 2
	}
	parsed.inputReader = stdin
	code, result := execute(parsed)
	printReport(parsed.diagnosticFormat == "json", result, stdout, stderr)
	return code
}

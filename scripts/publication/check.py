#!/usr/bin/env python3
"""Check publication candidates without printing matched values or file contents."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys

LOCAL_ROOTS = {".git", ".pi", ".planning", "local", "private", "scratch", "artifacts", "dist", "generated"}
BUILD_DIRS = {"target", "build", "out", ".gradle", ".kotlin", ".konan", "__pycache__", ".cache"}
LOCAL_NOTES = {"task_plan.md", "findings.md", "progress.md"}
SENSITIVE_NAMES = {".env", ".envrc", "credentials.json", "secrets.json", "secrets.toml", "settings.local.json", "local.properties"}
KEY_EXTENSIONS = {".pem", ".key", ".p12", ".pfx", ".jks", ".keystore"}
SECRET_PATTERNS = {
    "private-key": re.compile(rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    "provider-token": re.compile(rb"\b(?:AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{50,})\b"),
    "machine-user-path": re.compile(rb"/" rb"Users/|[A-Z]:\\Users\\"),
}


def local_path(path):
    return (
        path.parts[0] in LOCAL_ROOTS
        or str(path) in LOCAL_NOTES
        or any(part in BUILD_DIRS for part in path.parts)
        or path.name.endswith((".test", ".test.exe"))
    )


def candidates(root, source_tree):
    if source_tree:
        paths = []
        for directory, dirs, names in os.walk(root, followlinks=False):
            relative = Path(directory).relative_to(root)
            kept = []
            for name in dirs:
                path = relative / name
                if local_path(path):
                    continue
                if (root / path).is_symlink():
                    paths.append(path)
                else:
                    kept.append(name)
            dirs[:] = kept
            paths.extend(relative / name for name in names if not local_path(relative / name))
        return sorted(paths), "source-tree-only"
    # Include tracked files even when now ignored: .gitignore cannot untrack secrets.
    top = subprocess.run(["git", "rev-parse", "--show-toplevel"], cwd=root, capture_output=True)
    if top.returncode or Path(os.fsdecode(top.stdout).strip()).resolve() != root.resolve():
        raise ValueError("No repository index is available; use --source-tree for a file-only check.")
    result = subprocess.run(["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"], cwd=root, capture_output=True, check=True)
    return sorted({Path(os.fsdecode(p)) for p in result.stdout.split(b"\0") if p}), "git-candidates-not-history"


def scan(root, paths, forbidden_terms):
    findings = []
    checked = 0
    # Scan bytes as well as paths; this also detects terms in a binary candidate.
    terms = [term.encode("utf-8").lower() for term in forbidden_terms]
    for candidate_id, path in enumerate(paths, 1):
        kinds = set()
        basename = path.name.lower()
        if basename in SENSITIVE_NAMES or path.suffix.lower() in KEY_EXTENSIONS or (
            basename.startswith(".env.") and basename not in {".env.example", ".env.template"}
        ):
            kinds.add("sensitive-file-name")
        if path.is_absolute() or ".." in path.parts:
            kinds.add("unsafe-path")
        elif local_path(path):
            kinds.add("local-only-candidate")
        else:
            full = root / path
            if not full.resolve().is_relative_to(root.resolve()):
                kinds.add("symlink-escape")
            elif full.is_symlink():
                target = Path(os.path.relpath(full.resolve(), root.resolve()))
                if not full.exists():
                    kinds.add("broken-symlink")
                if local_path(target):
                    kinds.add("local-only-symlink")
                for i, term in enumerate(terms):
                    if term in os.fsencode(os.readlink(full)).lower():
                        kinds.add(f"forbidden-term-{i + 1}-symlink")
            elif full.is_file():
                contents = full.read_bytes()
                checked += 1
                for kind, pattern in SECRET_PATTERNS.items():
                    if pattern.search(contents):
                        kinds.add(kind)
                for i, term in enumerate(terms):
                    if term in contents.lower():
                        kinds.add(f"forbidden-term-{i + 1}")
            else:
                kinds.add("unsupported-entry")
        path_bytes = os.fsencode(path).lower()
        for i, term in enumerate(terms):
            if term in path_bytes:
                kinds.add(f"forbidden-term-{i + 1}-path")
        if kinds:
            # Paths can contain confidential terms, so reports use candidate numbers.
            findings.append({"candidate": candidate_id, "categories": sorted(kinds)})
    return {"candidate_count": len(paths), "files_checked": checked, "findings": findings}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--source-tree", action="store_true", help="File-only scan, excluding local/build state; does not check Git index or history")
    parser.add_argument("--policy", type=Path, help="Local JSON policy with a forbidden_terms string array; never prints its values")
    args = parser.parse_args()
    try:
        terms = []
        if args.policy:
            policy = json.loads(args.policy.read_text())
            if set(policy) != {"forbidden_terms"} or not isinstance(policy["forbidden_terms"], list):
                raise ValueError("Invalid policy shape.")
            terms = policy["forbidden_terms"]
            if not all(isinstance(term, str) and term.strip() for term in terms):
                raise ValueError("Policy terms must be non-empty strings.")
        paths, scope = candidates(args.root, args.source_tree)
        report = scan(args.root, paths, terms)
        report.update(scope=scope, forbidden_term_count=len(terms), history_checked=False)
        print(json.dumps(report, indent=2))
        return 1 if report["findings"] else 0
    except (OSError, ValueError, subprocess.SubprocessError):
        print(json.dumps({"status": "error", "message": "Cannot complete the publication scan. Check the repository, policy and readable files; without Git, explicitly select --source-tree."}))
        return 2


if __name__ == "__main__":
    sys.exit(main())

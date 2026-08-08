#!/usr/bin/env python3
"""PR Review bot: reviews a GitHub PR's diff with Claude, using graphify's
own MCP server as the model's only source of repo context (search the code
graph, don't read whole files) — then posts findings as a PR comment.

Stdlib only, on purpose: this runs both on a developer's machine and inside
CI, and the fewer things that need `pip install` before it works, the fewer
ways it breaks in someone else's environment.

Auth: this shells out to the `claude` CLI, so it uses whatever
authenticates that CLI in the current environment — a personal
`claude login` session locally, or ANTHROPIC_API_KEY in CI (personal OAuth
logins can't work unattended, so CI must use an API key).

Usage:
    graphify bot pr-review <pr-number> [--repo PATH] [--dry-run]

or directly:
    python3 bots/pr_review.py <pr-number> [--repo PATH] [--graphify-bin PATH] [--dry-run]
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

REVIEW_CRITERIA = """\
Review this pull request diff. For each issue you find, judge it against
ALL of these categories — not just "is this code correct":

1. breaking-change: does this change an existing function/route/type's
   contract in a way that could break callers elsewhere in the repo?
   Use get_callers on anything changed to check who's affected.
2. quality: normal code-quality concerns — unclear naming, missing error
   handling at a real boundary, dead code, obviously wrong logic.
3. duplication: does this add a new function/helper that already exists
   elsewhere in the codebase (i.e. it should have reused something
   instead of rewriting it)? Use search_symbol to check before flagging.
4. unnecessary-rewrite: does this reimplement or restructure code that
   didn't need to change for the PR's stated purpose?
5. pattern-mismatch: does the new code diverge from how the rest of the
   codebase does the same kind of thing (error handling style, naming
   convention, project structure)?
6. hallucinated-contract: does one side of a boundary (e.g. a caller, a
   frontend consumer) assume a shape/type/behavior that the other side
   (e.g. the callee, the backend response) doesn't actually provide?
   This is the most important category to check carefully — it's the
   kind of bug that's easy for both a human and an AI to miss because
   each side looks locally correct.

You have graphify MCP tools (search_symbol, get_symbol, get_callers,
get_callees, get_file_slice, get_tree, get_stats) for the CURRENT state of
the repo (before this PR's changes, unless the repo was already indexed
after checkout — check get_stats if unsure). Use them to check claims
instead of guessing — e.g. before flagging duplication, search_symbol to
confirm the existing function is really equivalent.

Respond with ONLY a fenced ```json code block (no other prose) containing
an array of findings, each shaped exactly like:
{"severity": "high"|"medium"|"low", "category": one of the six category
 slugs above, "file": "path/from/repo/root", "line": <int or null>,
 "summary": "<one sentence>", "detail": "<why this matters, 1-3 sentences>"}

If you find nothing worth flagging, respond with an empty array: []
"""


def run(cmd: list[str], **kwargs) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, capture_output=True, text=True, **kwargs)


def gh(args: list[str], cwd: Path) -> str:
    result = run(["gh"] + args, cwd=cwd)
    if result.returncode != 0:
        raise RuntimeError(f"gh {' '.join(args)} failed:\n{result.stderr}")
    return result.stdout


def find_graphify_bin(explicit: str | None) -> str:
    if explicit:
        return explicit
    # Prefer a binary sitting next to this script's repo root (built via
    # `go build -o graphify ./cmd/graphify`), falling back to PATH.
    repo_root = Path(__file__).resolve().parent.parent
    local = repo_root / "graphify"
    if local.exists() and os.access(local, os.X_OK):
        return str(local)
    found = shutil.which("graphify")
    if found:
        return found
    raise RuntimeError(
        "no graphify binary found — build one with "
        "`go build -o graphify ./cmd/graphify` or pass --graphify-bin"
    )


def ensure_indexed(graphify_bin: str, repo: Path) -> None:
    result = run([graphify_bin, "parse", str(repo)], cwd=repo)
    if result.returncode != 0:
        raise RuntimeError(f"graphify parse failed:\n{result.stderr}")


def build_mcp_config(graphify_bin: str, repo: Path, tmpdir: Path) -> Path:
    config = {
        "mcpServers": {
            "graphify": {
                "command": graphify_bin,
                "args": ["mcp", str(repo)],
            }
        }
    }
    path = tmpdir / "mcp-config.json"
    path.write_text(json.dumps(config))
    return path


GRAPHIFY_TOOLS = [
    "mcp__graphify__search_symbol",
    "mcp__graphify__get_symbol",
    "mcp__graphify__get_file_symbols",
    "mcp__graphify__get_callers",
    "mcp__graphify__get_callees",
    "mcp__graphify__get_file_slice",
    "mcp__graphify__get_tree",
    "mcp__graphify__get_stats",
]


def run_claude_review(prompt: str, mcp_config: Path) -> str:
    cmd = [
        "claude", "-p", prompt,
        "--mcp-config", str(mcp_config),
        "--strict-mcp-config",
        "--output-format", "json",
        "--allowedTools", ",".join(GRAPHIFY_TOOLS),
    ]
    result = run(cmd)
    if result.returncode != 0:
        raise RuntimeError(f"claude invocation failed (exit {result.returncode}):\n{result.stdout}\n{result.stderr}")

    try:
        envelope = json.loads(result.stdout)
    except json.JSONDecodeError as e:
        raise RuntimeError(f"claude did not return valid JSON envelope: {e}\nraw: {result.stdout[:2000]}")

    if envelope.get("is_error"):
        raise RuntimeError(f"claude returned an error: {envelope.get('result')}")

    return envelope.get("result", "")


def extract_findings(claude_output: str) -> list[dict]:
    match = re.search(r"```json\s*(\[.*?\])\s*```", claude_output, re.DOTALL)
    text = match.group(1) if match else claude_output.strip()
    try:
        findings = json.loads(text)
    except json.JSONDecodeError as e:
        raise RuntimeError(f"could not parse findings JSON from claude's response: {e}\nraw: {claude_output[:2000]}")
    if not isinstance(findings, list):
        raise RuntimeError(f"expected a JSON array of findings, got: {type(findings)}")
    return findings


SEVERITY_ORDER = {"high": 0, "medium": 1, "low": 2}
SEVERITY_EMOJI = {"high": "🔴", "medium": "🟡", "low": "⚪"}


def format_comment(pr_number: int, findings: list[dict]) -> str:
    if not findings:
        return (
            "## 🤖 graphify PR review\n\n"
            "No issues found across breaking changes, quality, duplication, "
            "unnecessary rewrites, pattern consistency, or cross-boundary "
            "contract mismatches."
        )

    findings = sorted(findings, key=lambda f: SEVERITY_ORDER.get(f.get("severity", "low"), 2))
    lines = [f"## 🤖 graphify PR review", "", f"{len(findings)} finding(s):", ""]
    for f in findings:
        emoji = SEVERITY_EMOJI.get(f.get("severity", "low"), "⚪")
        loc = f.get("file", "?")
        if f.get("line"):
            loc += f":{f['line']}"
        lines.append(f"### {emoji} `{f.get('category', '?')}` — {loc}")
        lines.append(f"**{f.get('summary', '')}**")
        if f.get("detail"):
            lines.append("")
            lines.append(f.get("detail"))
        lines.append("")
    lines.append("---")
    lines.append("_Reviewed using the graphify code graph, not raw file reads._")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("pr_number", type=int)
    parser.add_argument("--repo", default=".", help="path to the local repo checkout (default: cwd)")
    parser.add_argument("--graphify-bin", default=None, help="path to the graphify binary (default: auto-detect)")
    parser.add_argument("--dry-run", action="store_true", help="print the review instead of posting it as a PR comment")
    args = parser.parse_args()

    repo = Path(args.repo).resolve()

    try:
        graphify_bin = find_graphify_bin(args.graphify_bin)
        print(f"[pr-review] using graphify binary: {graphify_bin}", file=sys.stderr)

        print(f"[pr-review] indexing {repo}...", file=sys.stderr)
        ensure_indexed(graphify_bin, repo)

        print(f"[pr-review] fetching PR #{args.pr_number} diff...", file=sys.stderr)
        diff = gh(["pr", "diff", str(args.pr_number)], cwd=repo)
        meta_raw = gh(["pr", "view", str(args.pr_number), "--json", "title,body"], cwd=repo)
        meta = json.loads(meta_raw)

        prompt = (
            f"{REVIEW_CRITERIA}\n\n"
            f"PR title: {meta.get('title', '')}\n"
            f"PR description: {meta.get('body', '') or '(none)'}\n\n"
            f"Diff:\n```diff\n{diff}\n```"
        )

        with tempfile.TemporaryDirectory() as tmp:
            mcp_config = build_mcp_config(graphify_bin, repo, Path(tmp))
            print("[pr-review] running claude review (this calls the model, may take a minute)...", file=sys.stderr)
            claude_output = run_claude_review(prompt, mcp_config)

        findings = extract_findings(claude_output)
        print(f"[pr-review] {len(findings)} finding(s)", file=sys.stderr)

        comment = format_comment(args.pr_number, findings)

        if args.dry_run:
            print(comment)
        else:
            with tempfile.NamedTemporaryFile("w", suffix=".md", delete=False) as f:
                f.write(comment)
                body_file = f.name
            gh(["pr", "comment", str(args.pr_number), "--body-file", body_file], cwd=repo)
            os.unlink(body_file)
            print(f"[pr-review] posted comment to PR #{args.pr_number}", file=sys.stderr)

        return 1 if any(f.get("severity") == "high" for f in findings) else 0

    except RuntimeError as e:
        print(f"[pr-review] error: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

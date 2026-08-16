"""Report worktree branch ownership, setup state, and safe remediation."""

from __future__ import annotations

import argparse
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Worktree:
    path: Path
    branch: str | None


def git(*arguments: str) -> str:
    return subprocess.run(
        ["git", *arguments], check=True, capture_output=True, text=True
    ).stdout


def worktrees() -> list[Worktree]:
    entries: list[Worktree] = []
    path: Path | None = None
    branch: str | None = None
    for line in git("worktree", "list", "--porcelain").splitlines() + [""]:
        if line.startswith("worktree "):
            path = Path(line.removeprefix("worktree "))
            branch = None
        elif line.startswith("branch "):
            branch = line.removeprefix("branch ")
        elif not line and path is not None:
            entries.append(Worktree(path, branch))
            path = None
    return entries


def setup_state(entry: Worktree) -> str:
    if not entry.path.is_dir():
        return "missing"
    stamp = entry.path / ".agents" / "worktree-snapshot"
    try:
        revision = stamp.read_text(encoding="ascii").strip()
    except FileNotFoundError:
        return "not run"
    except (OSError, UnicodeError):
        return "unreadable"
    head = git("--git-dir", entry.path, "rev-parse", "HEAD").strip()
    return "current" if revision == head else "stale"


def is_merged(branch: str, base: str) -> bool:
    return (
        subprocess.run(
            ["git", "merge-base", "--is-ancestor", branch, base],
            check=False,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        ).returncode
        == 0
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--branch", help="Check the worktree that owns this branch")
    parser.add_argument(
        "--base", default="origin/main", help="Merged-state base branch"
    )
    args = parser.parse_args()

    try:
        bare = git("rev-parse", "--is-bare-repository").strip() == "true"
        entries = worktrees()
    except subprocess.CalledProcessError as error:
        print(
            error.stderr.strip() or "error: unable to inspect Git worktrees",
            file=sys.stderr,
        )
        return 2

    if not args.branch:
        for entry in entries:
            branch = entry.branch or "(detached HEAD)"
            print(f"{entry.path}: {branch} (setup {setup_state(entry)})")
        if bare:
            print(
                "warning: this directory is a bare Git repository; run "
                "development commands from a listed worktree."
            )
        return 0

    reference = f"refs/heads/{args.branch}"
    owners = [entry for entry in entries if entry.branch == reference]
    if not owners:
        print(f"Branch {args.branch} is not checked out by a registered worktree.")
        return 0

    owner = owners[0]
    merged = is_merged(args.branch, args.base)
    state = "merged" if merged else "not known to be merged"
    print(
        f"Branch {args.branch} is checked out at {owner.path} "
        f"(setup {setup_state(owner)}, {state} into {args.base})."
    )
    if args.branch == "main":
        print(
            "The primary main worktree must remain in place; do not use "
            "git worktree remove or --force as a remediation."
        )
        print(
            "For an additional read-only main snapshot, use: "
            "git worktree add --detach <path> origin/main"
        )
        print(
            "For changes, create an Issue branch instead: "
            "git worktree add -b issue/<number> <path> origin/main"
        )
        return 0

    print(
        "Do not remove the worktree automatically. Inspect its status, then run "
        f"git worktree remove {str(owner.path)!r} only when it is no longer active."
    )
    if setup_state(owner) != "current":
        print(
            f"Setup for {owner.path} did not finish. Run "
            f"'mise run setup' there before continuing."
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

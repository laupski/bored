#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""
GitHub repository automation script.

Automates common GitHub operations across multiple repositories:
- Committing and pushing changes
- Opening pull requests
- Assigning reviewers to PRs
- Setting repository secrets

Prerequisites:
- GitHub CLI (gh) must be installed and authenticated: https://cli.github.com/
- Run `gh auth login` before using this script

Usage:
    uv run gh_repo_automation.py --repos repos.json --action <action> [options]

Actions:
    sync-files      Clone repos, add/update files, commit, push, and create PR
    create-pr       Create a pull request
    set-secret      Set a repository secret
    add-reviewers   Add reviewers to a pull request

Examples:
    # Sync files across repos (clone, commit, push, create PR)
    # Single file:
    uv run gh_repo_automation.py --repos repos.json --action sync-files \\
        --source .github/PULL_REQUEST_TEMPLATE.md \\
        --branch add-pr-template \\
        --title "Add PR template" \\
        --body "Adds standardized PR template with ADO integration"

    # Multiple files:
    uv run gh_repo_automation.py --repos repos.json --action sync-files \\
        --source .github/PULL_REQUEST_TEMPLATE.md \\
        --source .github/workflows/sync-pr-to-ado.yml \\
        --source .ado/metadata.json \\
        --branch add-ado-integration \\
        --title "Add ADO integration"

    # Entire directory:
    uv run gh_repo_automation.py --repos repos.json --action sync-files \\
        --source .github/ \\
        --source .ado/ \\
        --branch add-ado-integration \\
        --title "Add ADO integration"

    # Create PRs across repos
    uv run gh_repo_automation.py --repos repos.json --action create-pr \\
        --branch feature-branch --title "Add feature" --body "Description"

    # Set a secret across repos
    uv run gh_repo_automation.py --repos repos.json --action set-secret \\
        --secret-name ADO_PAT --secret-value "your-pat-value"

    # Add reviewers to existing PRs
    uv run gh_repo_automation.py --repos repos.json --action add-reviewers \\
        --pr-number 123 --reviewers "user1,user2"

repos.json format:
    {
        "repositories": [
            "owner/repo1",
            "owner/repo2"
        ],
        "default_reviewers": ["reviewer1", "reviewer2"],
        "default_base_branch": "main"
    }
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


def run_gh(
    args: list[str], capture: bool = True, check: bool = True
) -> subprocess.CompletedProcess:
    """Run a GitHub CLI command."""
    cmd = ["gh"] + args
    print(f"  Running: gh {' '.join(args)}")

    try:
        result = subprocess.run(
            cmd,
            capture_output=capture,
            text=True,
            check=check,
        )
        return result
    except subprocess.CalledProcessError as e:
        print(f"  Error: {e.stderr}", file=sys.stderr)
        raise


def check_gh_installed() -> bool:
    """Check if GitHub CLI is installed and authenticated."""
    try:
        result = subprocess.run(
            ["gh", "auth", "status"],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            print("Error: GitHub CLI is not authenticated.", file=sys.stderr)
            print("Run 'gh auth login' to authenticate.", file=sys.stderr)
            return False
        return True
    except FileNotFoundError:
        print("Error: GitHub CLI (gh) is not installed.", file=sys.stderr)
        print("Install from: https://cli.github.com/", file=sys.stderr)
        return False


def load_repos_config(config_path: str) -> dict:
    """Load repository configuration from JSON file."""
    path = Path(config_path)
    if not path.exists():
        print(f"Error: Config file not found: {config_path}", file=sys.stderr)
        sys.exit(1)

    with open(path) as f:
        return json.load(f)


def create_pr(
    repo: str,
    branch: str,
    base: str,
    title: str,
    body: str,
    reviewers: list[str] | None = None,
    draft: bool = False,
) -> str | None:
    """Create a pull request in the specified repository.

    Returns the PR URL if successful, None otherwise.
    """
    args = [
        "pr",
        "create",
        "--repo",
        repo,
        "--head",
        branch,
        "--base",
        base,
        "--title",
        title,
        "--body",
        body,
    ]

    if draft:
        args.append("--draft")

    if reviewers:
        args.extend(["--reviewer", ",".join(reviewers)])

    try:
        result = run_gh(args)
        pr_url = result.stdout.strip()
        print(f"  Created PR: {pr_url}")
        return pr_url
    except subprocess.CalledProcessError:
        return None


def add_reviewers(repo: str, pr_number: int, reviewers: list[str]) -> bool:
    """Add reviewers to an existing pull request."""
    args = [
        "pr",
        "edit",
        str(pr_number),
        "--repo",
        repo,
        "--add-reviewer",
        ",".join(reviewers),
    ]

    try:
        run_gh(args)
        print(f"  Added reviewers: {', '.join(reviewers)}")
        return True
    except subprocess.CalledProcessError:
        return False


def set_secret(repo: str, secret_name: str, secret_value: str) -> bool:
    """Set a repository secret."""
    args = [
        "secret",
        "set",
        secret_name,
        "--repo",
        repo,
        "--body",
        secret_value,
    ]

    try:
        run_gh(args)
        print(f"  Set secret: {secret_name}")
        return True
    except subprocess.CalledProcessError:
        return False


def commit_and_push(
    repo_path: str,
    branch: str,
    commit_message: str,
    files: list[str] | None = None,
) -> bool:
    """Commit changes and push to remote.

    This operates on a local repository clone.
    """
    try:
        # Create branch if it doesn't exist
        subprocess.run(
            ["git", "checkout", "-B", branch],
            cwd=repo_path,
            capture_output=True,
            check=True,
        )

        # Add files
        if files:
            subprocess.run(
                ["git", "add"] + files,
                cwd=repo_path,
                capture_output=True,
                check=True,
            )
        else:
            subprocess.run(
                ["git", "add", "-A"],
                cwd=repo_path,
                capture_output=True,
                check=True,
            )

        # Commit
        subprocess.run(
            ["git", "commit", "-m", commit_message],
            cwd=repo_path,
            capture_output=True,
            check=True,
        )

        # Push
        subprocess.run(
            ["git", "push", "-u", "origin", branch],
            cwd=repo_path,
            capture_output=True,
            check=True,
        )

        print(f"  Committed and pushed to {branch}")
        return True
    except subprocess.CalledProcessError as e:
        print(f"  Error: {e.stderr.decode() if e.stderr else str(e)}", file=sys.stderr)
        return False


def clone_repo(repo: str, dest: str) -> bool:
    """Clone a repository to a destination directory."""
    args = ["repo", "clone", repo, dest]

    try:
        run_gh(args)
        print(f"  Cloned to {dest}")
        return True
    except subprocess.CalledProcessError:
        return False


def copy_source_to_dest(source: Path, dest_dir: Path) -> list[str]:
    """Copy a file or directory to the destination.

    Returns list of relative paths that were copied.
    """
    copied = []

    if source.is_file():
        dest_file = dest_dir / source
        dest_file.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, dest_file)
        copied.append(str(source))
        print(f"    Copied file: {source}")

    elif source.is_dir():
        dest_subdir = dest_dir / source
        if dest_subdir.exists():
            shutil.rmtree(dest_subdir)
        shutil.copytree(source, dest_subdir)
        # Collect all files in the directory
        for root, _, files in os.walk(source):
            for f in files:
                rel_path = Path(root) / f
                copied.append(str(rel_path))
        print(f"    Copied directory: {source}/ ({len(copied)} files)")

    return copied


def sync_files_to_repo(
    repo: str,
    sources: list[Path],
    branch: str,
    base: str,
    commit_message: str,
    pr_title: str,
    pr_body: str,
    reviewers: list[str] | None = None,
    draft: bool = False,
) -> tuple[bool, str | None]:
    """Clone a repo, add/update files, commit, push, and create a PR.

    Returns (success, pr_url).
    """
    with tempfile.TemporaryDirectory() as tmpdir:
        repo_dir = Path(tmpdir) / "repo"

        # Clone the repository
        if not clone_repo(repo, str(repo_dir)):
            return False, None

        # Checkout base branch and create new branch
        try:
            subprocess.run(
                ["git", "checkout", base],
                cwd=repo_dir,
                capture_output=True,
                check=True,
            )
            subprocess.run(
                ["git", "checkout", "-b", branch],
                cwd=repo_dir,
                capture_output=True,
                check=True,
            )
        except subprocess.CalledProcessError as e:
            print(
                f"  Error creating branch: {e.stderr.decode() if e.stderr else str(e)}",
                file=sys.stderr,
            )
            return False, None

        # Copy all sources
        all_copied = []
        for source in sources:
            copied = copy_source_to_dest(source, repo_dir)
            all_copied.extend(copied)

        if not all_copied:
            print("  No files to copy", file=sys.stderr)
            return False, None

        # Commit and push
        if not commit_and_push(str(repo_dir), branch, commit_message, all_copied):
            return False, None

        # Create PR
        pr_url = create_pr(
            repo=repo,
            branch=branch,
            base=base,
            title=pr_title,
            body=pr_body,
            reviewers=reviewers,
            draft=draft,
        )

        return pr_url is not None, pr_url


def action_sync_files(config: dict, args: argparse.Namespace) -> None:
    """Execute sync-files action across repositories."""
    repos = config.get("repositories", [])
    base = args.base or config.get("default_base_branch", "main")
    reviewers = (
        args.reviewers.split(",")
        if args.reviewers
        else config.get("default_reviewers", [])
    )

    # Validate sources
    sources = []
    for src in args.source:
        path = Path(src)
        if not path.exists():
            print(f"Error: Source not found: {src}", file=sys.stderr)
            sys.exit(1)
        sources.append(path)

    # Build commit message
    if args.commit_message:
        commit_message = args.commit_message
    elif len(sources) == 1:
        commit_message = f"Add {sources[0]}"
    else:
        commit_message = f"Add {len(sources)} files/directories"

    source_desc = ", ".join(str(s) for s in sources)
    print(f"Syncing [{source_desc}] across {len(repos)} repositories...")

    success = 0
    failed = 0
    pr_urls = []

    for repo in repos:
        print(f"\n[{repo}]")
        ok, pr_url = sync_files_to_repo(
            repo=repo,
            sources=sources,
            branch=args.branch,
            base=base,
            commit_message=commit_message,
            pr_title=args.title,
            pr_body=args.body or "",
            reviewers=reviewers if reviewers else None,
            draft=args.draft,
        )
        if ok:
            success += 1
            if pr_url:
                pr_urls.append(pr_url)
        else:
            failed += 1

    print(f"\nResults: {success} succeeded, {failed} failed")
    if pr_urls:
        print("\nCreated PRs:")
        for url in pr_urls:
            print(f"  {url}")


def action_create_pr(config: dict, args: argparse.Namespace) -> None:
    """Execute create-pr action across repositories."""
    repos = config.get("repositories", [])
    base = args.base or config.get("default_base_branch", "main")
    reviewers = (
        args.reviewers.split(",")
        if args.reviewers
        else config.get("default_reviewers", [])
    )

    print(f"Creating PRs across {len(repos)} repositories...")

    success = 0
    failed = 0

    for repo in repos:
        print(f"\n[{repo}]")
        result = create_pr(
            repo=repo,
            branch=args.branch,
            base=base,
            title=args.title,
            body=args.body,
            reviewers=reviewers if reviewers else None,
            draft=args.draft,
        )
        if result:
            success += 1
        else:
            failed += 1

    print(f"\nResults: {success} succeeded, {failed} failed")


def action_set_secret(config: dict, args: argparse.Namespace) -> None:
    """Execute set-secret action across repositories."""
    repos = config.get("repositories", [])

    # Get secret value from argument or environment variable
    secret_value = args.secret_value
    if not secret_value and args.secret_env:
        secret_value = os.environ.get(args.secret_env)
        if not secret_value:
            print(
                f"Error: Environment variable {args.secret_env} is not set",
                file=sys.stderr,
            )
            sys.exit(1)

    if not secret_value:
        print("Error: --secret-value or --secret-env is required", file=sys.stderr)
        sys.exit(1)

    print(f"Setting secret '{args.secret_name}' across {len(repos)} repositories...")

    success = 0
    failed = 0

    for repo in repos:
        print(f"\n[{repo}]")
        if set_secret(repo, args.secret_name, secret_value):
            success += 1
        else:
            failed += 1

    print(f"\nResults: {success} succeeded, {failed} failed")


def action_add_reviewers(config: dict, args: argparse.Namespace) -> None:
    """Execute add-reviewers action across repositories."""
    repos = config.get("repositories", [])
    reviewers = (
        args.reviewers.split(",")
        if args.reviewers
        else config.get("default_reviewers", [])
    )

    if not reviewers:
        print("Error: No reviewers specified", file=sys.stderr)
        sys.exit(1)

    print(
        f"Adding reviewers to PR #{args.pr_number} across {len(repos)} repositories..."
    )

    success = 0
    failed = 0

    for repo in repos:
        print(f"\n[{repo}]")
        if add_reviewers(repo, args.pr_number, reviewers):
            success += 1
        else:
            failed += 1

    print(f"\nResults: {success} succeeded, {failed} failed")


def main():
    parser = argparse.ArgumentParser(
        description="Automate GitHub operations across multiple repositories"
    )

    parser.add_argument(
        "--repos",
        required=True,
        help="Path to JSON file containing repository list and config",
    )
    parser.add_argument(
        "--action",
        required=True,
        choices=["sync-files", "create-pr", "set-secret", "add-reviewers"],
        help="Action to perform",
    )

    # File sync options
    parser.add_argument(
        "--source",
        action="append",
        help="Local file or directory to sync (can be specified multiple times)",
    )
    parser.add_argument("--commit-message", help="Commit message")

    # PR creation options
    parser.add_argument("--branch", help="Source branch for PR")
    parser.add_argument(
        "--base", help="Base branch for PR (default from config or 'main')"
    )
    parser.add_argument("--title", help="PR title")
    parser.add_argument("--body", help="PR body/description")
    parser.add_argument("--draft", action="store_true", help="Create as draft PR")
    parser.add_argument("--reviewers", help="Comma-separated list of reviewers")

    # Secret options
    parser.add_argument("--secret-name", help="Name of the secret to set")
    parser.add_argument("--secret-value", help="Value of the secret")
    parser.add_argument(
        "--secret-env", help="Environment variable containing secret value"
    )

    # Reviewer options
    parser.add_argument("--pr-number", type=int, help="PR number to add reviewers to")

    args = parser.parse_args()

    # Check GitHub CLI
    if not check_gh_installed():
        sys.exit(1)

    # Load config
    config = load_repos_config(args.repos)

    # Validate and execute action
    if args.action == "sync-files":
        if not args.source or not args.branch or not args.title:
            parser.error("sync-files requires --source, --branch, and --title")
        action_sync_files(config, args)

    elif args.action == "create-pr":
        if not args.branch or not args.title:
            parser.error("create-pr requires --branch and --title")
        action_create_pr(config, args)

    elif args.action == "set-secret":
        if not args.secret_name:
            parser.error("set-secret requires --secret-name")
        if not args.secret_value and not args.secret_env:
            parser.error("set-secret requires --secret-value or --secret-env")
        action_set_secret(config, args)

    elif args.action == "add-reviewers":
        if not args.pr_number:
            parser.error("add-reviewers requires --pr-number")
        action_add_reviewers(config, args)


if __name__ == "__main__":
    main()

#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "httpx",
# ]
# ///
"""
Sync GitHub PR description to Azure DevOps work item.

This script extracts the ADO work item number from a GitHub PR description
(looking for "ADO Card: <number>") and updates the corresponding work item
with the PR description.

Supports:
- Tags: "ADO Tags: tag1, tag2, tag3" - replaces all tags on the work item
- Child Items: "ADO Children:" followed by a list of items - creates tasks
  as children of the work item (idempotent, won't create duplicates)
- Comment: "ADO Comment:" followed by text - adds a comment to the work item
  (idempotent, updates existing comment from this sync tool)
- Links: "ADO Links:" followed by a list of URLs with optional names -
  adds hyperlinks to the work item (idempotent, won't create duplicates)

Configuration is read from .ado/metadata.json in the repository root.
The ADO PAT is read from the environment variable specified in metadata.json
(defaults to ADO_PAT).

Usage:
    uv run sync_pr_to_ado.py --pr-number <number> --repo <owner/repo>
    uv run sync_pr_to_ado.py --pr-description "<description>"

Example PR description:
    This PR implements feature X.

    ADO Card: 12345
    ADO Tags: frontend, bug-fix, priority-high
    ADO Comment: This work is being tracked in PR #123
    ADO Links:
    - https://github.com/org/repo/pull/123 | PR #123
    - https://docs.example.com/feature-spec
    ADO Children:
    - Implement the API endpoint
    - Add unit tests
    - Update documentation
"""

import argparse
import base64
import os
import re
import sys
from pathlib import Path

import httpx

# Set to True for verbose debugging output
DEBUG = os.environ.get("ADO_SYNC_DEBUG", "").lower() in ("1", "true", "yes")


def debug(msg: str) -> None:
    """Print debug message if DEBUG is enabled."""
    if DEBUG:
        print(f"[DEBUG] {msg}", file=sys.stderr)


class AdoClient:
    """Client for Azure DevOps REST API."""

    def __init__(self, organization: str, project: str, pat: str):
        self.organization = organization
        self.project = project
        self.base_url = f"https://dev.azure.com/{organization}/{project}/_apis"
        auth_string = base64.b64encode(f":{pat}".encode()).decode()
        self.headers = {
            "Authorization": f"Basic {auth_string}",
        }
        debug(f"ADO Client initialized for {organization}/{project}")
        debug(f"Base URL: {self.base_url}")
        debug(f"PAT length: {len(pat)} characters")
        debug(f"PAT prefix: {pat[:4]}..." if len(pat) > 4 else "PAT too short!")

    def _log_response(self, method: str, url: str, response: httpx.Response) -> None:
        """Log response details for debugging."""
        debug(f"{method} {url}")
        debug(f"Response status: {response.status_code}")
        if response.status_code >= 400:
            debug(f"Response headers: {dict(response.headers)}")
            debug(f"Response body: {response.text[:500]}")

    def get_work_item(self, work_item_id: int) -> dict | None:
        """Fetch a work item by ID."""
        url = f"{self.base_url}/wit/workitems/{work_item_id}?api-version=7.0&$expand=relations"
        debug(f"GET {url}")
        response = httpx.get(url, headers=self.headers)
        self._log_response("GET", url, response)
        if response.status_code == 200:
            return response.json()
        if response.status_code == 401:
            print(
                f"Authentication failed (401). Check your PAT token.", file=sys.stderr
            )
            print(f"URL: {url}", file=sys.stderr)
        return None

    def update_work_item(self, work_item_id: int, patch_document: list) -> bool:
        """Update a work item with a JSON patch document."""
        url = f"{self.base_url}/wit/workitems/{work_item_id}?api-version=7.0"
        headers = {**self.headers, "Content-Type": "application/json-patch+json"}
        debug(f"PATCH {url}")
        debug(f"Patch document: {patch_document}")
        response = httpx.patch(url, json=patch_document, headers=headers)
        self._log_response("PATCH", url, response)

        if response.status_code == 200:
            return True
        print(
            f"Error updating work item #{work_item_id}: {response.status_code}",
            file=sys.stderr,
        )
        if response.status_code == 401:
            print(
                "Authentication failed. Check your PAT token has the required scopes:",
                file=sys.stderr,
            )
            print("  - Work Items (Read & Write)", file=sys.stderr)
        print(f"Details: {response.text}", file=sys.stderr)
        return False

    def create_work_item(
        self, work_item_type: str, patch_document: list
    ) -> dict | None:
        """Create a new work item."""
        url = f"{self.base_url}/wit/workitems/${work_item_type}?api-version=7.0"
        headers = {**self.headers, "Content-Type": "application/json-patch+json"}
        debug(f"POST {url}")
        debug(f"Patch document: {patch_document}")
        response = httpx.post(url, json=patch_document, headers=headers)
        self._log_response("POST", url, response)

        if response.status_code == 200:
            return response.json()
        print(f"Error creating work item: {response.status_code}", file=sys.stderr)
        if response.status_code == 401:
            print(
                "Authentication failed. Check your PAT token has the required scopes:",
                file=sys.stderr,
            )
            print("  - Work Items (Read & Write)", file=sys.stderr)
        print(f"Details: {response.text}", file=sys.stderr)
        return None

    def query_work_items(self, wiql: str) -> list[dict]:
        """Execute a WIQL query and return work items."""
        url = f"{self.base_url}/wit/wiql?api-version=7.0"
        headers = {**self.headers, "Content-Type": "application/json"}
        debug(f"POST {url}")
        debug(f"WIQL: {wiql}")
        response = httpx.post(url, json={"query": wiql}, headers=headers)
        self._log_response("POST", url, response)

        if response.status_code != 200:
            print(f"Error executing query: {response.status_code}", file=sys.stderr)
            if response.status_code == 401:
                print("Authentication failed. Check your PAT token.", file=sys.stderr)
            return []

        work_item_refs = response.json().get("workItems", [])
        if not work_item_refs:
            return []

        # Fetch full work item details
        ids = ",".join(str(ref["id"]) for ref in work_item_refs)
        url = f"{self.base_url}/wit/workitems?ids={ids}&api-version=7.0"
        debug(f"GET {url}")
        response = httpx.get(url, headers=self.headers)
        self._log_response("GET", url, response)

        if response.status_code == 200:
            return response.json().get("value", [])
        return []

    def get_comments(self, work_item_id: int) -> list[dict]:
        """Get all comments on a work item."""
        url = f"{self.base_url}/wit/workitems/{work_item_id}/comments?api-version=7.0-preview.3"
        debug(f"GET {url}")
        response = httpx.get(url, headers=self.headers)
        self._log_response("GET", url, response)

        if response.status_code == 200:
            return response.json().get("comments", [])
        if response.status_code == 401:
            print(
                "Authentication failed fetching comments. Check your PAT token.",
                file=sys.stderr,
            )
        return []

    def add_comment(self, work_item_id: int, text: str) -> dict | None:
        """Add a comment to a work item."""
        url = f"{self.base_url}/wit/workitems/{work_item_id}/comments?api-version=7.0-preview.3"
        headers = {**self.headers, "Content-Type": "application/json"}
        debug(f"POST {url}")
        response = httpx.post(url, json={"text": text}, headers=headers)
        self._log_response("POST", url, response)

        if response.status_code == 200:
            return response.json()
        print(f"Error adding comment: {response.status_code}", file=sys.stderr)
        if response.status_code == 401:
            print("Authentication failed. Check your PAT token.", file=sys.stderr)
        print(f"Details: {response.text}", file=sys.stderr)
        return None

    def update_comment(self, work_item_id: int, comment_id: int, text: str) -> bool:
        """Update an existing comment on a work item."""
        url = f"{self.base_url}/wit/workitems/{work_item_id}/comments/{comment_id}?api-version=7.0-preview.3"
        headers = {**self.headers, "Content-Type": "application/json"}
        debug(f"PATCH {url}")
        response = httpx.patch(url, json={"text": text}, headers=headers)
        self._log_response("PATCH", url, response)

        if response.status_code == 200:
            return True
        print(f"Error updating comment: {response.status_code}", file=sys.stderr)
        if response.status_code == 401:
            print("Authentication failed. Check your PAT token.", file=sys.stderr)
        print(f"Details: {response.text}", file=sys.stderr)
        return False


def load_ado_metadata() -> dict:
    """Load ADO configuration from .ado/metadata.json."""
    metadata_path = Path(__file__).parent / ".ado" / "metadata.json"

    if not metadata_path.exists():
        print(f"Error: ADO metadata file not found at {metadata_path}", file=sys.stderr)
        sys.exit(1)

    import json

    with open(metadata_path) as f:
        return json.load(f)


def get_ado_pat(metadata: dict) -> str:
    """Get ADO PAT from environment variable."""
    env_var = metadata.get("pat_env_var", "ADO_PAT")
    pat = os.environ.get(env_var)

    if not pat:
        print(f"Error: Environment variable {env_var} is not set", file=sys.stderr)
        sys.exit(1)

    return pat


def extract_ado_card_number(pr_description: str) -> int | None:
    """Extract ADO work item number from PR description.

    Looks for pattern: "ADO Card: <number>" (case-insensitive)
    """
    pattern = r"ADO\s*Card\s*:\s*(\d+)"
    match = re.search(pattern, pr_description, re.IGNORECASE)

    if match:
        return int(match.group(1))
    return None


def extract_tags(pr_description: str) -> list[str]:
    """Extract tags from PR description.

    Looks for pattern: "ADO Tags: tag1, tag2, tag3" (case-insensitive)
    """
    pattern = r"ADO\s*Tags\s*:\s*(.+?)(?:\n|$)"
    match = re.search(pattern, pr_description, re.IGNORECASE)

    if match:
        tags_str = match.group(1)
        return [tag.strip() for tag in tags_str.split(",") if tag.strip()]
    return []


def extract_children(pr_description: str) -> list[str]:
    """Extract child item titles from PR description.

    Looks for pattern:
    ADO Children:
    - Item 1
    - Item 2
    """
    pattern = r"ADO\s*Children\s*:\s*\n((?:\s*[-*]\s*.+\n?)+)"
    match = re.search(pattern, pr_description, re.IGNORECASE)

    if match:
        items_block = match.group(1)
        items = re.findall(r"[-*]\s*(.+)", items_block)
        return [item.strip() for item in items if item.strip()]
    return []


def extract_comment(pr_description: str) -> str | None:
    """Extract comment text from PR description.

    Looks for pattern: "ADO Comment: <text>" (single line or multiline until next ADO directive)
    """
    # Match single line comment
    pattern = r"ADO\s*Comment\s*:\s*(.+?)(?:\n(?=ADO\s)|$)"
    match = re.search(pattern, pr_description, re.IGNORECASE | re.DOTALL)

    if match:
        return match.group(1).strip()
    return None


def extract_links(pr_description: str) -> list[tuple[str, str]]:
    """Extract hyperlinks from PR description.

    Looks for pattern:
    ADO Links:
    - https://example.com | Link Name
    - https://example.com (name is optional, URL used as name if not provided)

    Returns list of (url, name) tuples.
    """
    pattern = r"ADO\s*Links\s*:\s*\n((?:\s*[-*]\s*.+\n?)+)"
    match = re.search(pattern, pr_description, re.IGNORECASE)

    if not match:
        return []

    links_block = match.group(1)
    links = []

    for line in re.findall(r"[-*]\s*(.+)", links_block):
        line = line.strip()
        if not line:
            continue

        # Check for "url | name" format
        if "|" in line:
            parts = line.split("|", 1)
            url = parts[0].strip()
            name = parts[1].strip()
        else:
            url = line
            name = line

        if url:
            links.append((url, name))

    return links


def get_github_pr_description(
    repo: str, pr_number: int, github_token: str | None = None
) -> str:
    """Fetch PR description from GitHub API."""
    url = f"https://api.github.com/repos/{repo}/pulls/{pr_number}"

    headers = {
        "Accept": "application/vnd.github.v3+json",
        "User-Agent": "sync-pr-to-ado",
    }

    if github_token:
        headers["Authorization"] = f"token {github_token}"

    response = httpx.get(url, headers=headers)

    if response.status_code != 200:
        print(f"Error fetching PR from GitHub: {response.status_code}", file=sys.stderr)
        sys.exit(1)

    return response.json().get("body", "")


def update_work_item_description_and_tags(
    client: AdoClient, work_item_id: int, description: str, tags: list[str]
) -> bool:
    """Update work item description and tags."""
    patch_document = [
        {"op": "replace", "path": "/fields/System.Description", "value": description}
    ]

    # Tags are stored as a semicolon-separated string
    if tags:
        tags_value = "; ".join(tags)
        patch_document.append(
            {"op": "replace", "path": "/fields/System.Tags", "value": tags_value}
        )
    else:
        # Remove all tags if none specified
        patch_document.append({"op": "remove", "path": "/fields/System.Tags"})

    return client.update_work_item(work_item_id, patch_document)


def get_existing_child_titles(client: AdoClient, parent_id: int) -> dict[str, int]:
    """Get existing child work items and their titles.

    Returns a dict mapping lowercase title to work item ID.
    """
    # Query for child items of the parent
    wiql = f"""
    SELECT [System.Id], [System.Title]
    FROM WorkItemLinks
    WHERE [Source].[System.Id] = {parent_id}
      AND [System.Links.LinkType] = 'System.LinkTypes.Hierarchy-Forward'
    MODE (MustContain)
    """

    url = f"{client.base_url}/wit/wiql?api-version=7.0"
    headers = {**client.headers, "Content-Type": "application/json"}
    response = httpx.post(url, json={"query": wiql}, headers=headers)

    if response.status_code != 200:
        return {}

    relations = response.json().get("workItemRelations", [])
    child_ids = [
        rel["target"]["id"]
        for rel in relations
        if rel.get("target") and rel["target"]["id"] != parent_id
    ]

    if not child_ids:
        return {}

    # Fetch details for child items
    ids_str = ",".join(str(id) for id in child_ids)
    url = f"{client.base_url}/wit/workitems?ids={ids_str}&api-version=7.0"
    response = httpx.get(url, headers=client.headers)

    if response.status_code != 200:
        return {}

    children = {}
    for item in response.json().get("value", []):
        title = item.get("fields", {}).get("System.Title", "")
        children[title.lower()] = item["id"]

    return children


def sync_child_items(
    client: AdoClient, parent_id: int, child_titles: list[str]
) -> None:
    """Create or update child work items.

    Only creates new children if they don't already exist (by title).
    """
    if not child_titles:
        return

    # Get existing children
    existing = get_existing_child_titles(client, parent_id)
    print(f"Found {len(existing)} existing child items")

    # Get parent work item to inherit area path
    parent = client.get_work_item(parent_id)
    if not parent:
        print(f"Error: Could not fetch parent work item #{parent_id}", file=sys.stderr)
        return

    area_path = parent.get("fields", {}).get("System.AreaPath", "")
    iteration_path = parent.get("fields", {}).get("System.IterationPath", "")

    for title in child_titles:
        if title.lower() in existing:
            print(f"  Child already exists: '{title}' (#{existing[title.lower()]})")
            continue

        # Create new child task
        patch_document = [
            {"op": "add", "path": "/fields/System.Title", "value": title},
            {
                "op": "add",
                "path": "/relations/-",
                "value": {
                    "rel": "System.LinkTypes.Hierarchy-Reverse",
                    "url": f"{client.base_url}/wit/workitems/{parent_id}",
                },
            },
        ]

        if area_path:
            patch_document.append(
                {"op": "add", "path": "/fields/System.AreaPath", "value": area_path}
            )

        if iteration_path:
            patch_document.append(
                {
                    "op": "add",
                    "path": "/fields/System.IterationPath",
                    "value": iteration_path,
                }
            )

        result = client.create_work_item("Task", patch_document)
        if result:
            print(f"  Created child: '{title}' (#{result['id']})")
        else:
            print(f"  Failed to create child: '{title}'", file=sys.stderr)


# Marker used to identify comments created by this sync tool
COMMENT_MARKER = "<!-- ado-sync-comment -->"


def sync_comment(client: AdoClient, work_item_id: int, comment_text: str) -> None:
    """Add or update a comment on the work item.

    Uses a marker to identify comments created by this tool for idempotency.
    """
    if not comment_text:
        return

    # Add marker to comment
    full_comment = f"{COMMENT_MARKER}\n{comment_text}"

    # Check for existing comment with our marker
    comments = client.get_comments(work_item_id)

    for comment in comments:
        if COMMENT_MARKER in comment.get("text", ""):
            # Update existing comment
            if client.update_comment(work_item_id, comment["id"], full_comment):
                print(f"Updated existing comment (#{comment['id']})")
            return

    # No existing comment found, create new one
    result = client.add_comment(work_item_id, full_comment)
    if result:
        print(f"Added new comment (#{result['id']})")


def get_existing_hyperlinks(client: AdoClient, work_item_id: int) -> set[str]:
    """Get existing hyperlink URLs on a work item."""
    work_item = client.get_work_item(work_item_id)
    if not work_item:
        return set()

    relations = work_item.get("relations", []) or []
    urls = set()

    for rel in relations:
        if rel.get("rel") == "Hyperlink":
            urls.add(rel.get("url", ""))

    return urls


def sync_hyperlinks(
    client: AdoClient, work_item_id: int, links: list[tuple[str, str]]
) -> None:
    """Add hyperlinks to the work item.

    Idempotent - won't add duplicate links (matched by URL).
    """
    if not links:
        return

    existing = get_existing_hyperlinks(client, work_item_id)
    print(f"Found {len(existing)} existing hyperlinks")

    for url, name in links:
        if url in existing:
            print(f"  Link already exists: '{name}' ({url})")
            continue

        patch_document = [
            {
                "op": "add",
                "path": "/relations/-",
                "value": {
                    "rel": "Hyperlink",
                    "url": url,
                    "attributes": {"comment": name},
                },
            }
        ]

        if client.update_work_item(work_item_id, patch_document):
            print(f"  Added link: '{name}' ({url})")
        else:
            print(f"  Failed to add link: '{name}'", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(
        description="Sync GitHub PR description to Azure DevOps work item"
    )

    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument(
        "--pr-description",
        help="PR description text (use this if you already have the description)",
    )
    group.add_argument(
        "--pr-number", type=int, help="GitHub PR number to fetch description from"
    )

    parser.add_argument(
        "--repo",
        help="GitHub repository in owner/repo format (required with --pr-number)",
    )
    parser.add_argument(
        "--github-token",
        help="GitHub token for API access (or set GITHUB_TOKEN env var)",
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="Enable debug output (or set ADO_SYNC_DEBUG=1)",
    )

    args = parser.parse_args()

    # Enable debug mode if --debug flag is set
    global DEBUG
    if args.debug:
        DEBUG = True
        debug("Debug mode enabled via --debug flag")

    # Validate arguments
    if args.pr_number and not args.repo:
        parser.error("--repo is required when using --pr-number")

    # Get PR description
    if args.pr_description:
        pr_description = args.pr_description
    else:
        github_token = args.github_token or os.environ.get("GITHUB_TOKEN")
        pr_description = get_github_pr_description(
            args.repo, args.pr_number, github_token
        )

    if not pr_description:
        print("Error: PR description is empty", file=sys.stderr)
        sys.exit(1)

    # Extract ADO card number
    ado_card_number = extract_ado_card_number(pr_description)

    if not ado_card_number:
        print("No ADO Card number found in PR description", file=sys.stderr)
        print("Expected format: 'ADO Card: <number>'", file=sys.stderr)
        sys.exit(1)

    print(f"Found ADO Card: #{ado_card_number}")

    # Extract tags, children, comment, and links
    tags = extract_tags(pr_description)
    children = extract_children(pr_description)
    comment = extract_comment(pr_description)
    links = extract_links(pr_description)

    if tags:
        print(f"Found tags: {', '.join(tags)}")
    if children:
        print(f"Found {len(children)} child items to sync")
    if comment:
        print("Found comment to sync")
    if links:
        print(f"Found {len(links)} links to sync")

    # Load ADO configuration
    metadata = load_ado_metadata()
    organization = metadata.get("organization")
    project = metadata.get("project")

    if not organization or not project:
        print(
            "Error: 'organization' and 'project' are required in metadata.json",
            file=sys.stderr,
        )
        sys.exit(1)

    # Create client and sync
    pat = get_ado_pat(metadata)
    client = AdoClient(organization, project, pat)

    # Update description and tags
    if update_work_item_description_and_tags(
        client, ado_card_number, pr_description, tags
    ):
        print(f"Successfully updated ADO work item #{ado_card_number}")
    else:
        sys.exit(1)

    # Sync child items
    if children:
        print("Syncing child items...")
        sync_child_items(client, ado_card_number, children)

    # Sync comment
    if comment:
        print("Syncing comment...")
        sync_comment(client, ado_card_number, comment)

    # Sync hyperlinks
    if links:
        print("Syncing hyperlinks...")
        sync_hyperlinks(client, ado_card_number, links)

    print("Done!")


if __name__ == "__main__":
    main()

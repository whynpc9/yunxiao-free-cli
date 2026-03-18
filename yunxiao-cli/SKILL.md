---
name: yunxiao-cli
description: Use when the user wants to query Yunxiao free-edition organizations, members, projects, work items, work item types, or test cases through the local `yx` CLI, or when an agent skill needs a stable CLI wrapper around Yunxiao OpenAPI instead of calling raw HTTP.
---

# Yunxiao CLI

Use this skill when the task is about reading Yunxiao data through the local `yx` command.

## Quick start

1. Ensure the CLI exists.
   - Check with `command -v yx`.
   - If missing, install with:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/whynpc9/yunxiao-free-cli/main/install.sh | bash
   ```
   - Fallback if needed:
   ```bash
   GOBIN="$HOME/.local/bin" go install github.com/whynpc9/yunxiao-free-cli/cmd/yx@latest
   ```
2. Ensure auth exists.
   - Run `yx config show`.
   - If no token is configured, run `yx auth` and follow the PAT prompt.
3. Prefer `--json` whenever the result will be filtered, summarized, or fed into later steps.
4. Prefer explicit IDs (`--org`, `--project`, `--repo`, `--id`) when the user already provided them. Otherwise rely on the configured default organization.

## Command map

- Organizations:
  - `yx org list --json`
  - `yx org members --query <keyword> --json`
- Projects:
  - `yx project search --json`
  - `yx project get --id <projectId> --json`
- Work items:
  - `yx workitem search --category <Req|Bug|Task> --space-id <projectId> --json`
  - `yx workitem get --id <workitemId> --json`
  - `yx workitem types --project <projectId> --category <Req|Bug|Task> --json`
  - `yx workitem all-types --categories Req,Bug,Task --json`
  - `yx workitem fields --project <projectId> --type <workitemTypeId> --json`
- Test cases:
  - `yx testcase search --repo <testRepoId> --json`
  - `yx testcase get --repo <testRepoId> --id <testCaseId> --json`
  - `yx testcase dirs --repo <testRepoId> --json`
  - `yx testcase fields --repo <testRepoId> --json`

## Search conditions

`project search`, `workitem search`, and `testcase search` accept raw JSON object strings through `--conditions`.

Read [references/conditions.md](references/conditions.md) when you need concrete examples for `conditionGroups`, subject/name filtering, or status-based filtering.

Do not invent non-JSON shorthand. Pass the JSON object string exactly as the CLI expects.

## Operating rules

- Stay within the free-edition read-only scope unless the user explicitly asks to extend the CLI itself.
- Use table output for quick human inspection and `--json` for anything programmatic.
- If a query fails, surface the exact command and the API error summary so the user can distinguish auth issues, permission issues, and bad IDs.
- If `yx` is installed but not on `PATH`, call it via its absolute path after locating it in `$HOME/.local/bin/yx`.

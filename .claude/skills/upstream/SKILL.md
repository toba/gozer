---
name: upstream
description: Check upstream repos for updates. Use when user says "/upstream", "check upstream", "sync with upstream", or wants to see what's changed in the source repos this project was derived from.
---

# Upstream Check

Compare this project against its upstream sources to find valuable updates.

## Upstream Repos

| Source | Repo | What we use |
|--------|------|-------------|
| Tree-sitter grammar | `ngalaiko/tree-sitter-go-template` | Go template parser in `grammars/gotmpl/` |
| Zed queries | `hjr265/zed-gotmpl` | Tree-sitter query patterns for syntax highlighting |

## State Tracking

Last checked SHAs are stored in `.last-checked` in this skill's directory (JSON format).

## Workflow

1. Read the last checked SHAs:
```bash
cat .claude/skills/upstream-check/.last-checked 2>/dev/null || echo '{}'
```

2. Get commits from each upstream since last check:
```bash
# Tree-sitter grammar
gh api repos/ngalaiko/tree-sitter-go-template/commits --jq '.[0:20] | .[] | "\(.sha[0:7]) \(.commit.message | split("\n")[0])"'

# Zed queries
gh api repos/hjr265/zed-gotmpl/commits --jq '.[0:20] | .[] | "\(.sha[0:7]) \(.commit.message | split("\n")[0])"'
```
Stop at each repo's last checked SHA. If no state exists, show recent 10.

3. For interesting commits, fetch the diff:
```bash
gh api repos/OWNER/REPO/commits/SHA --jq '.files[] | "\(.filename)\n\(.patch)"'
```

4. After evaluation, update the state file with latest SHAs:
```bash
cat > .claude/skills/upstream-check/.last-checked << 'EOF'
{
  "ngalaiko/tree-sitter-go-template": "SHA",
  "hjr265/zed-gotmpl": "SHA"
}
EOF
```

## What to Look For

**Tree-sitter grammar (`ngalaiko/tree-sitter-go-template`):**
- Grammar fixes for edge cases
- New node types
- Performance improvements

**Zed queries (`hjr265/zed-gotmpl`):**
- New highlighting patterns
- Query fixes
- Zed-specific improvements

## Previously Reviewed & Skipped

**`hjr265/zed-gotmpl` — per-filetype language split (`70efaa4` and related commits `0bc0f37`, `3fb890e`):**
Upstream split `gotmpl` into separate language directories per embedded language (gotmpl-bash, gotmpl-css, gotmpl-go, gotmpl-js, gotmpl-json, gotmpl-sql, gotmpl-toml, gotmpl-xml, gotmpl-yaml, gotmpl-typescript). Not applicable — gozer focuses on htmx/HTML templates, not general-purpose template embeddings for other languages.

## Output

Summarize findings as:
- Commits worth porting (with rationale)
- Commits to skip (already have, not relevant, etc.)

Then update `.last-checked` with the latest upstream SHAs.

---
name: commit
description: Stage changed files and create a git commit
disable-model-invocation: true
allowed-tools:
  - Bash(git status)
  - Bash(git diff*)
  - Bash(git log*)
  - Bash(git add*)
  - Bash(git commit*)
---

# Git Commit

Stage changed files and create a commit following project conventions.

## Steps

1. Run in parallel:
   - `git status` to see all changes
   - `git diff` and `git diff --staged` to understand what changed
   - `git log --oneline -5` to see recent commit style

2. Analyze all changes and draft a commit message:
   - Write in English
   - Concise (1-2 sentences), focus on "why" not "what"
   - Do NOT include `Co-Authored-By` lines

3. Stage the relevant files with `git add` (prefer specific files over `git add -A`)

4. Create the commit using a HEREDOC:
   ```
   git commit -m "$(cat <<'EOF'
   Commit message here
   EOF
   )"
   ```

5. Run `git status` to verify success.

## Rules

- Never commit files that may contain secrets (.env, credentials, tokens)
- Never amend existing commits unless explicitly asked
- Never push unless explicitly asked
- If a pre-commit hook fails, fix the issue and create a NEW commit (do not amend)

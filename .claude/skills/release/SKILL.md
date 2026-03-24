---
name: release
description: Cut a new gcrunner release — bumps version in all files, commits, tags, and pushes. Usage: /release v0.2.0
argument-hint: <version>
allowed-tools: Read, Edit, Bash(git *)
---

# Cut a gcrunner Release

Release version: **$ARGUMENTS**

First, confirm the version looks like a valid semver tag (e.g. `v0.2.0`). If not, stop and tell the user.

## Steps

### 1. Check current version

Read `terraform/variables.tf` and note the current `gcrunner_version` default so you can show what's changing.

### 2. Update version in all four places

Update the version string from the old value to `$ARGUMENTS` in:

- **`terraform/variables.tf`** — the `default` value of `gcrunner_version`
- **`README.md`** — two occurrences:
  - `cloudshell_git_branch=<old>` in the Cloud Shell badge URL
  - `git clone --branch <old>` in the deploy command
- **`tutorial.md`** — `TF_VAR_gcrunner_version=<old>` in the configure variables step
- **`website/content/docs/updating.md`** — `git clone --branch <old>` in the manual update step

### 3. Commit the version bump

Stage and commit only those four files:

```
git add terraform/variables.tf README.md tutorial.md website/content/docs/updating.md
git commit -m "Bump version to $ARGUMENTS"
```

### 4. Create an annotated tag

```
git tag -a $ARGUMENTS -m "Release $ARGUMENTS"
```

### 5. Confirm before pushing

Show the user what you're about to push (commit + tag) and ask them to confirm before running:

```
git push origin HEAD
git push origin $ARGUMENTS
```

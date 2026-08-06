# Lefthook Setup and Usage Guide

This document provides developers with a guide to configuring and initializing the [Lefthook](https://github.com/evilmartians/lefthook) Git Hooks tool in this project.

---

## Table of Contents

1. [Introduction](#introduction)
2. [Prerequisites](#prerequisites)
3. [Installing Lefthook](#installing-lefthook)
4. [Initializing Git Hooks](#initializing-git-hooks)
5. [Common Commands & Operations](#common-commands--operations)
6. [Configuration File Notes](#configuration-file-notes)

---

## Introduction

This project uses **Lefthook** to manage Git Hooks (such as `pre-commit`). Before code is committed, Lefthook automatically triggers code formatting and static linting checks to ensure code committed to the repository meets quality standards.

---

## Prerequisites

Running the preset Hook commands requires the following local dependencies:

- **Frontend Dependencies**: Node.js & `pnpm` (`oxlint` and `oxfmt` are configured in `frontend/`)
- **Backend Dependencies**: Go environment & `golangci-lint`

If `golangci-lint` is not installed, you can install it using the following commands:

```bash
# macOS (Homebrew)
brew install golangci-lint

# Go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

## Installing Lefthook

Please choose an installation method suitable for your development environment:

### Method 1: Homebrew (Recommended for macOS / Linux)

```bash
brew install lefthook
```

### Method 2: pnpm / npm Global or Local Installation

```bash
# Root directory or global installation
pnpm add -D lefthook
```

### Method 3: Go install

```bash
go install github.com/evilmartians/lefthook@latest
```

### Method 4: Scoop / Winget (Windows)

```bash
# Scoop
scoop install lefthook

# Winget
winget install evilmartians.lefthook
```

After installation, verify it by running:

```bash
lefthook version
```

---

## Initializing Git Hooks

After cloning the repository or pulling the latest code, **you must run the following command in the project root directory** to write Hook scripts into the `.git/hooks/` directory:

```bash
lefthook install
```

Upon successful execution, the terminal will output content similar to:

```text
Lefthook v1.x.x
SYNC  hooks installed
```

At this point, Git Hooks are successfully mounted and will be triggered automatically on subsequent `git commit` commands.

---

## Common Commands & Operations

### 1. Manually Test Pre-commit Checks

Run all configured `pre-commit` checks manually without creating an actual commit:

```bash
lefthook run pre-commit
```

To test specific sub-commands only:

```bash
lefthook run pre-commit --commands backend-lint
lefthook run pre-commit --commands frontend-format
```

### 2. Temporarily Skip Hook Checks

In exceptional or urgent situations, if you need to bypass checks when committing:

```bash
# Skip using environment variable
LEFTHOOK=0 git commit -m "chore: temporary commit"

# Or use Git's built-in --no-verify option
git commit -m "chore: temporary commit" --no-verify
```

### 3. Update or Re-sync Hooks

When `lefthook.yml` in the project root changes, run the following command to sync and apply the changes:

```bash
lefthook install
```

---

## Configuration File Notes

For detailed hook definitions and commands, refer to the `lefthook.yml` file located at the project root directory.

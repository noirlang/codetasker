---
name: codetasker
description: >-
  Comprehensive guide and operational runbook for the CodeTasker CLI.
  Use this skill when scanning codebases for TODO, FIXME, BUG, HACK, and NOTE annotations,
  measuring technical debt and refactoring costs, synchronizing GitHub repositories,
  managing task lifecycles, injecting tasks as automated GitHub Pull Requests,
  reading/posting task comments, and configuring CLI authentication and preferences.
---

# CodeTasker CLI — Complete Operational Skill & Reference Manual

**CodeTasker** is an automated technical debt management and code annotation orchestration platform powered by Go and GitHub integration. The `codetasker` CLI enables developers and autonomous AI agents to scan codebases, manage tasks, calculate refactoring debt costs, and open automated Pull Requests directly from the terminal or CI/CD pipelines.

---

## 📑 Table of Contents
1. [Architecture & Design Principles](#-architecture--design-principles)
2. [Installation & Setup](#-installation--setup)
3. [Authentication & Environment Variables](#-authentication--environment-variables)
4. [Command Reference](#-command-reference)
   - [`codetasker scan`](#1-offline-codebase-scanner-scan)
   - [`codetasker debt`](#2-technical-debt-analysis-debt)
   - [`codetasker repo`](#3-repository-management-repo)
   - [`codetasker task`](#4-task-lifecycle--pr-injection-task)
   - [`codetasker notify`](#5-notification-management-notify)
   - [`codetasker config`](#6-configuration-management-config)
   - [`codetasker auth`](#7-authentication-auth)
5. [JSON Output & Scripting Pipelines](#-json-output--scripting-pipelines)
6. [Agentic Best Practices & Autonomous Workflows](#-agentic-best-practices--autonomous-workflows)
7. [Troubleshooting & FAQs](#-troubleshooting--faqs)

---

## 🏛️ Architecture & Design Principles

```
┌──────────────────────────────────────────────────────────────────┐
│                         CodeTasker CLI                           │
├─────────────────────────┬────────────────────────────────────────┤
│   Offline Engine        │          Connected Engine              │
│   • AST/Regex Scanner   │   • Personal Access / App Tokens       │
│   • Git Churn Analyzer  │   • Fiber REST API Client (/api/*)     │
│   • Complexity Scoring  │   • GitHub Webhook & PR Orchestration  │
└─────────────────────────┴────────────────────────────────────────┘
```

1. **Dual Execution Engine**:
   - **Offline (Zero-Network)**: `scan` and `debt analyze` run locally with no servers, credentials, or databases required.
   - **Connected**: `repo`, `task`, `notify`, and remote `debt` communicate securely over HTTPS with the CodeTasker Fiber backend.
2. **Deterministic & Scriptable**: Every query command supports `--json` for lossless piping into `jq`, automated scripts, and LLM context windows.
3. **Immutability & Security**: Tokens are never stored in plaintext in logs. App Tokens use SHA-256 server-side hashing with encrypted OAuth storage.

---

## 📦 Installation & Setup

### 1. Automated One-Line Installers

#### Linux & macOS (bash):
```bash
# Using curl:
curl -fsSL https://raw.githubusercontent.com/noirlang/codetasker/main/scripts/install-linux.sh | bash

# Using wget:
wget -qO- https://raw.githubusercontent.com/noirlang/codetasker/main/scripts/install-linux.sh | bash
```

#### Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/noirlang/codetasker/main/scripts/install-windows.ps1 | iex
```

### 2. Manual Source Compilation
```bash
cd backend
go build -ldflags="-s -w" -o /usr/local/bin/codetasker ./cmd/codetasker
```

---

## 🔑 Authentication & Environment Variables

### Interactive Login
```bash
codetasker auth login --token "<your-app-token-or-jwt>" --server "https://codetasker.noirlang.tr"
```

### Environment Variables
Environment variables take precedence over config files, ideal for CI/CD, Docker, and agent sessions:

| Variable | Description | Example |
|---|---|---|
| `CODETASKER_TOKEN` | Personal Access / App Token (`ct_app_...`) or JWT | `ct_app_afcc3d...` |
| `CODETASKER_SERVER` | CodeTasker backend base URL | `https://codetasker.noirlang.tr` |
| `CODETASKER_REPO` | Default GitHub repository context | `melihemik/codetester-test` |
| `CODETASKER_CONFIG` | Custom JSON configuration filepath | `/etc/codetasker/config.json` |

---

## 🛠️ Command Reference

### 1. Offline Codebase Scanner (`scan`)
Scans directories for `TODO`, `FIXME`, `BUG`, `HACK`, and `NOTE` comments across Go, TypeScript, JavaScript, Python, Rust, Ruby, C/C++, Java, SQL, HTML, CSS, Shell, and Markdown.

```bash
# Scan current directory
codetasker scan .

# Scan specific directory and filter by annotation type
codetasker scan ./src --type FIXME

# Output structured JSON for automation
codetasker scan . --json
```

**Flags:**
- `--type string`: Filter by type (`TODO`, `FIXME`, `BUG`, `HACK`, `NOTE`).
- `--json`: Output raw JSON array without ASCII banner.

---

### 2. Technical Debt Analysis (`debt`)
Calculates refactoring debt scores (0-100), hotspot risk levels (LOW, MEDIUM, HIGH, CRITICAL), and estimated monthly carrying costs based on Git commit churn, line count, and cyclomatic complexity.

```bash
# Analyze debt over last 90 days with $35/hr engineer rate
codetasker debt analyze .

# Custom analysis window and hourly cost
codetasker debt analyze . --days 30 --cost 50.0

# Remote repository debt analysis
codetasker debt analyze owner/repo --days 180
```

**Flags:**
- `-d, --days int`: Number of days of Git commit history to analyze (default `90`).
- `-c, --cost float`: Hourly developer cost in USD (default `35.0`).
- `--json`: Output JSON summary and hotspot list.

---

### 3. Repository Management (`repo`)
Lists and synchronizes GitHub repositories connected to the CodeTasker backend.

```bash
# List all repositories accessible to the user
codetasker repo list

# Trigger full codebase resync and task extraction
codetasker repo sync <owner/repo>

# View recursive file tree for a repository branch
codetasker repo tree <owner/repo> --branch main

# List repository collaborators and assigned roles
codetasker repo collab <owner/repo>
```

**Flags:**
- `--json`: Output results in JSON format.
- `-b, --branch string`: Branch name for tree traversal (default `HEAD` / default branch).

---

### 4. Task Lifecycle & PR Injection (`task`)
Manages task status, comments, assignees, and opens automated GitHub Pull Requests injecting code annotations directly into source files.

#### Listing Tasks:
```bash
# List all tasks for a repository
codetasker task list --repo <owner/repo>

# Filter tasks by status and type
codetasker task list --repo <owner/repo> --status open --type FIXME
```

#### Injecting Tasks & Opening Pull Requests:
CodeTasker supports single location, multi-line, multi-file, new file creation, and interactive wizard modes:

```bash
# 1. Single location in an existing file:
codetasker task inject \
  --repo "favilances/codetester-test" \
  --file "main.rb" \
  --line 5 \
  --type "TODO" \
  --note "Refactor authentication middleware" \
  --branch "main"

# 2. Create a brand new file with a task annotation via PR:
codetasker task inject \
  --repo "owner/repo" \
  --file "pkg/auth/jwt.go" \
  --new-file \
  --type "TODO" \
  --note "Implement token refresh loop"

# 3. Multiple lines in the same file:
codetasker task inject \
  --repo "owner/repo" \
  --file "main.go" \
  --lines "12,25,50" \
  --note "Add bounds check"

# 4. Multi-location across multiple files (with optional new files) in ONE atomic commit + PR:
codetasker task inject \
  --repo "owner/repo" \
  -L "src/main.go:42:Refactor handler" \
  -L "src/auth.go:15:Validate session token" \
  -L "pkg/scaffold.go:new:Initial scaffold module"

# 5. Interactive Builder Wizard:
codetasker task inject -i
```

#### Updating Task Status & Assignee:
```bash
# Update lifecycle status
codetasker task update <task-id> --status in_progress
codetasker task update <task-id> --status resolved
codetasker task update <task-id> --status open

# Assign task to GitHub user
codetasker task update <task-id> --assign "username"
```

#### Task Discussion & Comments:
```bash
# View comments on a task
codetasker task comment list <task-id>

# Post a new comment
codetasker task comment add <task-id> "Root cause identified in session token parsing."
```

---

### 5. Notification Management (`notify`)
Monitors and acknowledges assignment, mention, and comment notifications.

```bash
# List all notifications
codetasker notify list

# Show unread notifications only
codetasker notify list --unread

# Mark specific notification as read
codetasker notify read <notification-id>

# Mark all notifications as read
codetasker notify read-all
```

---

### 6. Configuration Management (`config`)
Manages persistent local configuration keys stored in `$HOME/.config/codetasker/config.json`.

```bash
# Display full configuration
codetasker config list

# Get specific configuration value
codetasker config get server_url
codetasker config get default_repo

# Set default repository context
codetasker config set default_repo "owner/repo"

# Set default hourly engineer cost
codetasker config set default_hourly_cost 45.0
```

---

### 7. Authentication (`auth`)
```bash
# Interactive or token-based login
codetasker auth login --token "ct_app_..." --server "https://codetasker.noirlang.tr"

# Check active session and connected server profile
codetasker auth status

# Log out and clear local credentials
codetasker auth logout
```

---

## 🤖 Agentic Best Practices & Autonomous Workflows

When an AI agent (Antigravity, Claude, Gemini, GPT) is assisting a developer in a repository:

### 1. Codebase Discovery Phase
Before making major refactors, run:
```bash
codetasker scan . --json
codetasker debt analyze . --json
```
Use the output to prioritize files with high churn + high complexity (`HIGH` or `CRITICAL` risk score).

### 2. Task Injection & Delegation Phase
When discovering unaddressed technical debt during a feature implementation:
```bash
codetasker task inject \
  --repo "<owner/repo>" \
  --file "path/to/file.go" \
  --line <target-line> \
  --type "TODO" \
  --note "Detailed explanation of pending work" \
  --branch "main"
```
This preserves context and creates a trackable GitHub PR without interrupting the current feature branch.

### 3. Task Status Sync Phase
When a task is resolved by a commit, update its status:
```bash
codetasker task update <task-id> --status resolved
codetasker task comment add <task-id> "Resolved in commit <sha>."
```

---

## ❓ Troubleshooting & FAQs

- **Q: `401 Unauthorized` during commands**:
  - *Cause*: Token missing, revoked, or copied with duplicate characters.
  - *Fix*: Run `codetasker auth login --token "<token>"` to refresh.
- **Q: `--json` includes ASCII art?**:
  - *Answer*: When `--json` is supplied, ASCII banners are automatically suppressed for pure JSON parsing.
- **Q: Offline vs Online commands?**:
  - `scan` and `debt analyze .` work 100% offline without any server or internet access.
  - `repo`, `task`, and `notify` require an active internet connection and authentication.

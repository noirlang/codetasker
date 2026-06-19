<div align="center">

<img src="frontend/public/logo-kucuk.png" alt="CodeTasker Logo" width="120" />

# CodeTasker — Two-Way GitHub TODO & Task Management Engine

*Automatically convert inline code comments (TODO, FIXME, BUG) into interactive Kanban tasks, and inject them back into your codebase via automated Pull Requests.*

[Website](https://noirlang.tr) | [GitHub Repository](https://github.com/noirlang/codetasker) | [Contributing](CONTRIBUTING.md)

<video src="" width="700" controls></video>

</div>

## Overview

CodeTasker is an intelligent task synchronization platform that bridges the gap between your codebase and project management tools. It automatically scans your synchronized GitHub repositories for annotations like `// TODO:`, `// FIXME:`, `// BUG:`, `// HACK:`, and `// NOTE:`, maps them onto a visual Kanban board, and allows you to create and inject comments back into your code via automated Git branches and Pull Requests.

It is designed for engineering organizations of all sizes to eliminate technical debt tracking overhead and keep tasks perfectly in sync with the actual code.

## Features

Here is a detailed breakdown of the features provided by CodeTasker. You can place screenshots inside the placeholders below:

### 1. Push-to-Sync (Code to Board)
Automatically converts inline code comments (`// TODO:`, `// FIXME:`, `// BUG:`, `// HACK:`, `// NOTE:`) into interactive tasks. Pushing changes to GitHub immediately syncs the task status on the board without manual intervention.
* **How it works:** Webhook listener processes git commits, taranır and updates MongoDB records.
* **Placeholder:**
  ![Push-to-Sync Screen](./screenshots/push_to_sync.png)

### 2. Task Board (Kanban & List View)
Manage your codebase tasks with a highly interactive Kanban board. Tasks are grouped in columns (`Open`, `In Progress`, `Resolved`) with smooth drag-and-drop operations, or a clean file-grouped list view.
* **How it works:** Enabled with drag-and-drop support, customizable columns, and real-time status syncing.
* **Placeholder:**
  ![Task Board Screen](./screenshots/task_board.png)

### 3. Task Injection (Board to Code)
Inject tasks directly into your codebase from the UI. CodeTasker automatically detects the file extension, writes the comment using the correct language syntax, creates a dedicated branch, and opens a GitHub Pull Request.
* **How it works:** Leverages low-level GitHub Git Trees and Blobs API for non-disruptive, exact-line commenting.
* **Placeholder:**
  ![Task Injection Screen](./screenshots/task_injection.png)

### 4. PR Management & Direct Merge
Review and merge open GitHub Pull Requests directly from the CodeTasker panel. Supports different merge options: standard merge commits, squash and merge, or rebase and merge.
* **How it works:** Uses GitHub PullRequests Merge API to execute merges directly from the UI with custom titles and messages.
* **Placeholder:**
  ![PR Direct Merge Screen](./screenshots/pr_merge.png)

### 5. CODEOWNERS & Maintainer Routing
Automatically routes task ownership using your repository's `.github/CODEOWNERS` rules. When a task in a specific directory is created or completed, notifications are automatically routed to the responsible developer.
* **How it works:** Resolves responsibilities line-by-line using GitHub glob pattern matching on push/sync triggers.
* **Placeholder:**
  ![CODEOWNERS Routing Screen](./screenshots/codeowners.png)

### 6. Telegram & Email Notifications (Multi-Channel Alerts)
In addition to email notifications, users can configure their own custom Telegram Bots. Users register their bot tokens and Chat IDs in their profile settings to receive instant push alerts when assigned a task, mentioned, or when a task is completed.
* **How it works:** Integrates transactional SMTP delivery with Telegram Bot sendMessage API webhooks.
* **Placeholder:**
  ![Notifications Configuration Screen](./screenshots/notifications.png)

### 7. Collaborators & Role Access Control
Manage repository access by setting custom roles (Viewer, Developer, Maintainer) inside CodeTasker to align team permissions, while verifying write access checks on GitHub.
* **How it works:** Secures mutations using middleware-level JWT credentials matched against MongoDB synced collaborator indices.
* **Placeholder:**
  ![Collaborator Settings Screen](./screenshots/collaborators.png)

## System Architecture

CodeTasker is composed of three services:
1. **Frontend:** A responsive Single Page Application built with React, TypeScript, TailwindCSS, and Vite.
2. **Backend:** A high-performance REST API built with Go, Fiber, and the official Google Go-GitHub client.
3. **Database:** MongoDB for storing user sessions, synced repository configurations, collaborators, tasks, and audit logs.

## Requirements

Ensure you have the following installed on your machine:
- **Go** (version 1.21 or later)
- **Node.js** (version 18 or later) & **npm**
- **MongoDB** (running locally or accessible via URI)

## Local Development Setup

### 1. Environment Configuration

Create a `.env` file in the root directory (and copy it to both `backend/` and `frontend/` directories as needed). Define the following variables:

```env
PORT=8080
MONGO_URI=mongodb://localhost:27017
DB_NAME=codetasker
GITHUB_CLIENT_ID=your_github_oauth_client_id
GITHUB_CLIENT_SECRET=your_github_oauth_client_secret
GITHUB_REDIRECT_URL=http://localhost:8080/api/auth/github/callback
JWT_SECRET=your_jwt_signing_secret
WEBHOOK_SECRET=your_github_webhook_hmac_secret
TOKEN_ENCRYPT_KEY=your_aes_32byte_encryption_key
FRONTEND_URL=http://localhost:5173
```

### 2. Run the Backend

```bash
cd backend
go run cmd/server/main.go
```

### 3. Run the Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend will start at `http://localhost:5173/` and proxy API requests to `http://localhost:8080`.

## Quick Installation & Docker Setup

To quickly configure environment variables and launch the entire platform (MongoDB + Go API Backend + React Frontend) using Docker, run the interactive installation script at the project root:

```bash
./setup.sh
```

This script will:
1. Verify system prerequisites (`docker` and `docker compose`).
2. Prompt you step-by-step for configuration settings (Port, Database name, GitHub OAuth credentials, SMTP/email settings, etc.).
3. Auto-generate secure cryptographic keys if left blank (JWT secrets, AES-256 token encryption keys, webhook secrets).
4. Save the configuration to `.env` and offer to run the Docker Compose environment in the background automatically.

Alternatively, you can manually build and start the containers after creating your `.env` configuration:

```bash
docker compose up -d --build
```

## Contributing

Please review the [CONTRIBUTING.md](CONTRIBUTING.md) file for details on our code of conduct and the process for submitting pull requests.

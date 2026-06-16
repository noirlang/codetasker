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

- **Push-to-Sync (Code to Board):** Adding or removing a TODO comment in your files automatically creates, updates, or completes tasks on the CodeTasker Kanban board.
- **Task Injection (Board to Code):** Assign and inject tasks directly into your codebase from the board. CodeTasker creates a dedicated branch, inserts the comment at the exact line with the correct language syntax, and opens a GitHub Pull Request.
- **Language-Sensitive Commenting:** Automatically detects file extensions to write syntactically correct comment blocks (e.g., `//` for Go/TypeScript, `#` for Python/Ruby, `--` for SQL).
- **Collaborative History:** Manage repository collaborators, roles (Viewer, Developer, Maintainer), view branch commit diffs, and inspect build Actions in real-time.
- **GitHub Webhook Sync:** Securely sync branch updates via verified SHA256 HMAC signature webhooks.
- **Security-First Architecture:** Stored GitHub access tokens are encrypted in transit and rest using AES-256-GCM. Session state is managed via secure, HttpOnly, SameSite cookies.

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

## Running with Docker Compose

To start the entire environment (MongoDB, Go API Backend, and React Frontend) with one command:

```bash
docker-compose up --build
```

Ensure you have defined your OAuth credentials (`GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET`) in your root `.env` file before running compose.

## Contributing

Please review the [CONTRIBUTING.md](CONTRIBUTING.md) file for details on our code of conduct and the process for submitting pull requests.

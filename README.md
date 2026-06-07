# CodeTasker

> **Two-Way Sync GitHub TODO & Task Management Platform**  
> A B2B Micro-SaaS that syncs `// TODO:`, `// FIXME:`, and other code annotations into a Kanban board — and lets you inject new TODOs back into your codebase via pull requests.

---

## Architecture Overview

```
codetasker/
├── backend/          # Go 1.22 + Fiber v2 + MongoDB
│   ├── cmd/server/   # Entry point
│   └── internal/
│       ├── config/       # Env config
│       ├── database/     # MongoDB client + index setup
│       ├── domain/       # Pure domain models (User, Task)
│       ├── repository/   # MongoDB repository layer
│       ├── service/      # Business logic (OAuth, GitHub API, tasks)
│       ├── controller/   # HTTP route handlers
│       ├── middleware/   # JWT auth, HMAC webhook verification
│       └── parser/       # Concurrent regex TODO scanner
│
├── frontend/         # React 18 + Vite + TypeScript + Tailwind CSS
│   └── src/
│       ├── api/          # Axios typed API client
│       ├── components/   # Login, Dashboard, CodeViewer, TaskBoard, TaskInjector
│       ├── store/        # Zustand global state
│       └── types/        # Shared TypeScript interfaces
│
└── docker-compose.yml   # Full stack orchestration
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 18, Vite, TypeScript, Tailwind CSS, Monaco Editor |
| Backend | Go 1.22, Fiber v2, go-github/v62 |
| Database | MongoDB 7.0 |
| Auth | GitHub OAuth 2.0 + HS256 JWT (httpOnly cookie) |
| Realtime Sync | GitHub Webhooks (HMAC-SHA256 verified) |
| Task Injection | GitHub Git Data API (blob → tree → commit → branch → PR) |

---

## Prerequisites

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose (optional)
- A GitHub OAuth App ([create one here](https://github.com/settings/developers))
- A GitHub Webhook on each target repository

---

## Quick Start (Local Development)

### 1. Clone the repo

```bash
git clone https://github.com/yourorg/codetasker.git
cd codetasker
```

### 2. Configure environment

```bash
cp backend/.env.example backend/.env
# Edit backend/.env with your GitHub OAuth credentials
```

Required environment variables:

| Variable | Description |
|---|---|
| `GITHUB_CLIENT_ID` | From your GitHub OAuth App |
| `GITHUB_CLIENT_SECRET` | From your GitHub OAuth App |
| `GITHUB_REDIRECT_URL` | `http://localhost:8080/api/auth/github/callback` |
| `JWT_SECRET` | Any random 32+ char string |
| `WEBHOOK_SECRET` | Same value set in GitHub webhook settings |
| `TOKEN_ENCRYPT_KEY` | **Exactly** 32 bytes for AES-256-GCM |
| `MONGO_URI` | `mongodb://localhost:27017` |

### 3. Start with Docker Compose

```bash
docker-compose up --build
```

This starts:
- MongoDB on `localhost:27017`
- Go backend on `localhost:8080`
- React dev server on `localhost:5173` (with HMR)

### 4. Or run manually

**Backend:**
```bash
cd backend
go mod download
go run ./cmd/server/main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

---

## GitHub OAuth App Setup

1. Go to **Settings → Developer settings → OAuth Apps → New OAuth App**
2. Set **Authorization callback URL** to: `http://localhost:8080/api/auth/github/callback`
3. Copy **Client ID** and **Client Secret** to `backend/.env`

---

## GitHub Webhook Setup

For each repository you want to monitor:

1. Go to **Repo → Settings → Webhooks → Add webhook**
2. **Payload URL**: `https://your-domain.com/api/webhooks/github`
3. **Content type**: `application/json`
4. **Secret**: same value as `WEBHOOK_SECRET` in `.env`
5. **Events**: Select `Push` and `Pull requests`

For local development, use [smee.io](https://smee.io) or [ngrok](https://ngrok.com) to proxy webhook deliveries:

```bash
npx smee -u https://smee.io/your-channel -t http://localhost:8080/api/webhooks/github
```

---

## API Reference

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `GET` | `/api/health` | None | Health check |
| `GET` | `/api/auth/github` | None | Initiate GitHub OAuth |
| `GET` | `/api/auth/github/callback` | None | OAuth callback + JWT |
| `POST` | `/api/auth/logout` | JWT | Clear session |
| `GET` | `/api/auth/me` | JWT | Current user info |
| `GET` | `/api/repos` | JWT | List user's repos |
| `GET` | `/api/repos/:owner/:repo/tree` | JWT | File tree |
| `GET` | `/api/repos/:owner/:repo/contents` | JWT | File content (`?path=`) |
| `POST` | `/api/webhooks/github` | HMAC | Push event handler |
| `GET` | `/api/tasks` | JWT | Tasks for repo (`?repo_id=`) |
| `PATCH` | `/api/tasks/:id` | JWT | Update task status |
| `POST` | `/api/tasks/inject` | JWT | Inject TODO → PR |

---

## Security Architecture

### Webhook Verification
Every incoming webhook is verified using HMAC-SHA256 against the raw request body before any processing occurs. Constant-time comparison (`hmac.Equal`) prevents timing attacks.

### Token Encryption
GitHub access tokens are encrypted with **AES-256-GCM** before being stored in MongoDB. The `TOKEN_ENCRYPT_KEY` never leaves the backend process.

### SSRF Prevention
All user-supplied repository owner and name values are validated against `^[a-zA-Z0-9_.-]+$` before being passed to the GitHub API. The go-github client uses a fixed base URL and does not follow arbitrary redirects.

### JWT Security
JWTs are stored in `httpOnly; Secure; SameSite=Strict` cookies — inaccessible to JavaScript, preventing XSS token theft.

---

## Concurrent TODO Scanner

The `parser` package uses a worker pool for high-throughput scanning:

```
Files → [Channel] → [Worker 1]
                  → [Worker 2]  → [Results Channel] → Aggregated []ParsedTask
                  → [Worker N]
```

Workers = `runtime.NumCPU()` by default. Supports patterns:
- `// TODO: ...`
- `// FIXME: ...`
- `// HACK: ...`
- `// BUG: ...`
- `// NOTE: ...`
- `# TODO: ...` (Python/Shell)
- `/* TODO: ... */` (C-style)
- `-- TODO: ...` (SQL)

---

## Two-Way Sync Flow

```
GitHub Push
    └─→ Webhook (HMAC verified)
        └─→ Parse diff for added/modified files
            └─→ Fetch file content via GitHub API
                └─→ Concurrent regex scan (worker pool)
                    └─→ Upsert tasks in MongoDB
                        └─→ Frontend shows updated Kanban

UI Task Inject
    └─→ POST /api/tasks/inject
        └─→ Get latest commit SHA
            └─→ Fetch file blob
                └─→ Insert "// TODO: ..." at line N
                    └─→ Create blob → tree → commit
                        └─→ Create branch codetasker/inject-<ts>
                            └─→ Open Pull Request
                                └─→ Return PR URL to UI
```

---

## License

MIT

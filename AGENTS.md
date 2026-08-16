# AGENTS.md — CodeTasker Project Guidelines

Welcome to **CodeTasker**, an automated technical debt and code annotation management system powered by AI and GitHub integration.

## Project Overview
CodeTasker scans repositories for TODO, FIXME, BUG, HACK, and NOTE annotations, tracks task lifecycle, measures technical debt metrics, enables role-based collaborator access, supports task proposals/discussions with approval voting, and integrates with GitHub webhooks and OAuth.

- **Backend**: Go (Fiber framework, MongoDB driver, Zap logging)
- **Frontend**: TypeScript, React 18, Vite, Tailwind CSS, Monaco Editor
- **Architecture**: Monorepo split into `backend/` and `frontend/`

---

## Agent Instructions & Rules

1. **Obey Explicit Directives**: Always respect exact specifications, API contracts, and user directives.
2. **Code Verification**: Always run `go test ./...` in `backend/` and `./node_modules/.bin/tsc --noEmit` / `npm run build` in `frontend/` before completing tasks.
3. **Immutability & Safety**:
   - Do NOT store plaintext OAuth tokens or secrets in database or JSON API responses.
   - Use AES-256-GCM encryption for stored tokens (`TokenEncryptKey`).
   - Sign git commits using `-s` (`git commit -s`).

---

## Directory Sitemap

- `/backend`: Go API server, MongoDB repositories, services, controllers.
- `/frontend`: Vite + React SPA, UI components, state stores.
- `/setup.sh`: Automated setup and environment file generator.
- `/docker-compose.yml`: Local multi-container deployment config.

---

## Local Development & Testing

```bash
# Backend tests
cd backend && go test ./...

# Frontend typecheck & build
cd frontend && ./node_modules/.bin/tsc --noEmit && npm run build
```

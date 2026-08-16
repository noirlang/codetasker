# AGENTS.md — Backend Architecture & Guidelines

This directory contains the Go backend API service for **CodeTasker**.

## Tech Stack & Dependencies
- **Language**: Go 1.22+
- **HTTP Web Framework**: Fiber v2 (`github.com/gofiber/fiber/v2`)
- **Database**: MongoDB (`go.mongodb.org/mongo-driver`)
- **Logging**: Uber Zap (`go.uber.org/zap`)
- **JWT Auth**: Fiber JWT middleware (`github.com/gofiber/jwt/v3`)
- **GitHub API**: Go GitHub client (`github.com/google/go-github/v62`)

---

## Directory Structure

- `cmd/server/main.go`: Server entry point, dependency injection, Fiber setup, and graceful shutdown.
- `internal/config/config.go`: Environment configuration loader and validator.
- `internal/controller/`: HTTP handlers and Fiber route definitions:
  - `auth_controller.go`: GitHub OAuth flow and JWT cookie handling.
  - `repo_controller.go`: Repository management, collaborators, role permissions, commit diffs, branch management, actions, sandbox sync.
  - `task_controller.go`: TODO injection, task status updates, per-task comments, proposals/discussions (approve/reject).
  - `notification_controller.go`: Notification listing and mark-read handlers.
  - `debt_controller.go`: Technical debt analysis endpoints.
  - `webhook_controller.go`: GitHub HMAC webhook event receivers.
- `internal/domain/models.go`: Domain structs (`User`, `Task`, `Collaborator`, `Comment`, `TaskProposal`, `Notification`, `ActivityLog`).
- `internal/repository/`: MongoDB data access repositories.
- `internal/service/`: Core business logic services (`AuthService`, `GithubService`, `TaskService`, `DebtService`, `EmailService`, `TelegramService`, `CodeOwnerService`).

---

## Key Backend Guidelines

1. **Fiber Context Safety**:
   - Extract user ID using `middleware.GetUserID(c)`.
   - Always return typed JSON maps `fiber.Map{"error": ..., "message": ...}` on failure.
2. **MongoDB Data Integrity**:
   - Use `primitive.ObjectID` for document IDs.
   - Always check for `mongo.ErrNoDocuments` and return `(nil, nil)` where appropriate.
3. **Signed Local Commits**:
   - Sign commits with `-s` (`git commit -s`).

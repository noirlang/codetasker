# AGENTS.md — Frontend Architecture & Guidelines

This directory contains the single-page web application for **CodeTasker**.

## Tech Stack & Core Dependencies
- **Core**: React 18, TypeScript, Vite
- **Styling**: Tailwind CSS, Vanilla CSS, Animate.css, Lucide Icons (`lucide-react`)
- **State Management**: Zustand (`src/store/`)
- **Code Editor**: `@monaco-editor/react`
- **Drag & Drop**: `@hello-pangea/dnd`
- **Routing**: `react-router-dom` v6

---

## Component Sitemap

- `src/App.tsx`: Root router, session hydration (`authStore.fetchUser()`), protected routes.
- `src/api/client.ts`: Shared Axios client with httpOnly cookie credentials & API modules (`authApi`, `reposApi`, `tasksApi`, `notificationsApi`, `commentsApi`, `proposalsApi`, `debtApi`).
- `src/components/`:
  - `Dashboard.tsx`: Repository grid view, organization filter, top bar with `NotificationBell`.
  - `RepoView.tsx`: 3-pane layout (file tree / code editor / task board).
  - `NotificationBell.tsx`: Top-right notification bell dropdown, relative time, unread badge, and task navigation handler.
  - `TaskBoard.tsx`: Kanban & list task board, `TaskDetailModal` with Comments and Proposals/Discussions (Onaylandı/Reddedildi voting).
  - `CollaboratorManager.tsx`: Collaborator roles management, path permissions (`allowed_paths`), direct GitHub invite link, private sandbox repo generator.
  - `CodeViewer.tsx`: Monaco editor integration for code viewing & inline task injection.
  - `TaskInjector.tsx`: Slide-out panel for injecting new TODO annotations into GitHub repos.
  - `CommitHistoryPanel.tsx`, `PullRequestPanel.tsx`, `ActionsPanel.tsx`, `DebtPanel.tsx`, `RepoStatsPanel.tsx`.

---

## Testing & Build Commands

```bash
# Typecheck TypeScript files
./node_modules/.bin/tsc --noEmit

# Build production bundle
npm run build
```

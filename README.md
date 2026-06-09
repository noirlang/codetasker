# CodeTasker — Two-Way GitHub TODO & Task Management Platform

> An intelligent B2B SaaS platform that automatically syncs code annotations (`// TODO:`, `// FIXME:`, `// BUG:`) into trackable Kanban boards, keeping your codebase and project management tools in perfect harmony.

---

## Why CodeTasker? (The Core Problem)

In software development, there is a constant **disconnect** between project planning tools (like Jira, Trello, or GitHub Issues) and the actual code written. Developers often encounter minor bugs, unfinished tasks, or future improvements while coding. To stay focused, they leave quick annotations like `// TODO:` or `// FIXME:` inline.

Unfortunately, these comments frequently get **lost and forgotten** in the depths of the codebase, accumulating as silent technical debt. Opening a Jira ticket for every single detail is tedious, causing administrative overhead and disrupting the developer's flow.

CodeTasker bridges these two worlds. It automates task tracking directly from the developer's natural workspace—the codebase.

---

## Task Distribution in Large & Mid-Sized Organizations

### 1. Large Enterprises
* **The Challenge:** Enterprise companies handle hundreds of developers working across dozens of microservices and repositories. Teams often have no visibility into each other's codebases or transient tasks left in code. Jira boards become bloated, slow, and hard to align with actual development progress.
* **The Solution:** CodeTasker aggregates all inline TODOs and bug reports across all repositories into a single, unified panel. Product Owners and Engineering Directors can track real technical debt and pending tasks automatically, without having to badger developers.

### 2. Mid-Sized & Fast-Growing Teams (Scale-ups)
* **The Challenge:** In mid-sized teams, **velocity is everything**. Reducing bureaucracy, minimizing status meetings, and maintaining focused coding cycles is critical. However, tracking who is working on what—and where code omissions exist—becomes chaotic as the team scales.
* **The Solution:** Developers focus entirely on writing code. The moment they push code to GitHub, CodeTasker detects the changes in the background and updates the Kanban board instantly. If a task status changes in code, the board updates. If they need to assign a task from the board, they can inject a TODO back into the code via a Pull Request. Zero friction, maximum speed.

---

## Key Features

* **Push-to-Sync (Code to Board):** Adding or removing a TODO in your code immediately updates the shared Kanban board.
* **Task Injection (Board to Code):** Assign a task directly to the code from the board. CodeTasker creates a new branch, inserts the comment at the specified line, and opens a **Pull Request (PR)** automatically.
* **Language-Sensitive Commenting:** Automatically detects file types to insert the correct comment syntax (e.g., `#` for Python, `--` for SQL, `//` for Go/TS).
* **Collaborative Commit History:** View branch commits, manage collaborator permissions, merge branches, and include **Co-authors** directly from the UI.
* **Security First:** Features verified webhooks (HMAC-SHA256), encrypted access tokens (AES-256-GCM), and secure authentication cookies.

---

## License

GNU General Public License v3.0 (GPL-3.0) - See the [LICENSE](LICENSE) file for details.

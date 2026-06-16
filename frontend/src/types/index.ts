/**
 * CodeTasker — Shared TypeScript types
 * All interfaces and enums used across the application.
 */

/* ── Auth ─────────────────────────────────────────────────── */

/** Authenticated GitHub user returned by GET /api/auth/me */
export interface User {
  id: string;
  github_id: number;
  username: string;
  email?: string;
  avatar_url: string;
  created_at: string;
}

/* ── Repositories ─────────────────────────────────────────── */

/** A GitHub repository as returned by the backend */
export interface Repo {
  id: number;
  name: string;
  full_name: string;           // e.g. "octocat/Hello-World"
  description: string | null;
  private: boolean;
  language: string | null;
  stargazers_count: number;
  updated_at: string;          // ISO 8601 timestamp
  default_branch: string;
  is_synced?: boolean;
  topics?: string[];
}

export interface Organization {
  login: string;
  avatar_url: string;
}

/* ── Tasks ────────────────────────────────────────────────── */

/** Task status — maps to Kanban column */
export type TaskStatus = 'open' | 'in_progress' | 'resolved';

/** Comment tag type scanned from source files */
export type TaskType = 'TODO' | 'FIXME' | 'HACK' | 'BUG' | 'NOTE';

/** A task extracted from a TODO/FIXME/etc. comment in a repository */
export interface Task {
  id: string;
  repo_id: number;
  repo_name: string;
  file_path: string;           // relative path inside repo, e.g. "src/main.go"
  line_number: number;
  content: string;             // the comment text (excluding the tag)
  type: TaskType;
  status: TaskStatus;
  commit_sha: string;          // SHA of the commit that introduced the comment
  pr_url?: string;             // linked PR url
  created_at: string;
  updated_at: string;
}

/* ── File Tree ────────────────────────────────────────────── */

/** A single node in the repository file tree (from GitHub Trees API) */
export interface FileTreeNode {
  path: string;
  mode: string;
  type: 'blob' | 'tree';
  sha: string;
  size?: number;               // only present for blobs
  url: string;
}

/** Raw file contents fetched from the backend */
export interface FileContent {
  path: string;
  content: string;             // decoded file content (UTF-8 string)
  encoding: string;
  sha: string;
}

/* ── Task Injection ───────────────────────────────────────── */

/** Payload sent to POST /api/tasks/inject to create a PR with a new TODO */
export interface InjectTaskRequest {
  repo_owner: string;
  repo_name: string;
  file_path: string;
  line_number: number;
  description: string;
  branch: string;
  type: string;
}

/* ── API Error ────────────────────────────────────────────── */

/** Normalized error shape returned by the backend */
export interface ApiError {
  error: string;
  message?: string;
}

/* ── Collaborators & Roles ────────────────────────────────── */

export type RepoRole = 'owner' | 'maintainer' | 'developer' | 'viewer';

export interface Collaborator {
  id: string;
  repo_id: number;
  user_id: string;
  username: string;
  avatar_url: string;
  role: RepoRole;
  created_at: string;
}

/* ── Commits & Pull Requests ──────────────────────────────── */

export type CommitCheckState =
  | 'success'
  | 'failure'
  | 'error'
  | 'pending'
  | 'none'
  | 'unknown';

export interface CommitCheckRun {
  name: string;
  status: string;
  conclusion: string;
  details_url: string;
  started_at: string;
  completed_at: string;
}

export interface CommitStatus {
  context: string;
  state: string;
  description: string;
  target_url: string;
}

export interface Commit {
  sha: string;
  message: string;
  author: string;
  author_email: string;
  avatar_url: string;
  committer: string;
  committer_email: string;
  committer_avatar_url: string;
  date: string;
  html_url: string;
  verified: boolean;
  verification_reason: string;
  check_state: CommitCheckState;
  check_total: number;
  check_runs: CommitCheckRun[];
  statuses: CommitStatus[];
  check_error?: string;
}

export interface CommitListResponse {
  commits: Commit[];
  count: number;
  page: number;
  per_page: number;
  next_page: number;
}

export interface PullRequest {
  id: number;
  number: number;
  title: string;
  state: string;
  html_url: string;
  branch: string;
  base: string;
  creator: string;
  avatar_url: string;
  created_at: string;
}

/* ── Comments ─────────────────────────────────────────────────── */

/** A comment on a task */
export interface Comment {
  id: string;
  task_id: string;
  user_id: string;
  username: string;
  avatar_url: string;
  content: string;
  created_at: string;
  updated_at: string;
}

/* ── Notifications ────────────────────────────────────────────── */

/** A user notification from the backend */
export interface Notification {
  id: string;
  user_id: string;
  type: 'task_assigned' | 'comment_added' | 'pr_merged';
  title: string;
  message: string;
  link?: string;
  read: boolean;
  created_at: string;
}

/* ── Activity Log ──────────────────────────────────────────────── */

/** An activity/event in the repository activity feed */
export interface ActivityLog {
  id: string;
  repo_id: number;
  repo_name: string;
  actor_name: string;
  actor_avatar: string;
  action: string;
  target_type: string;
  target_id: string;
  target_label: string;
  meta?: Record<string, string>;
  created_at: string;
}

/* ── Issues ────────────────────────────────────────────────────── */

/** A GitHub issue for a repository */
export interface Issue {
  number: number;
  title: string;
  body: string;
  state: 'open' | 'closed';
  html_url: string;
  user: { login: string; avatar_url: string };
  labels: Array<{ name: string; color: string }>;
  created_at: string;
  updated_at: string;
  comments: number;
}

/* ── Branches ──────────────────────────────────────────────────── */

/** A repository branch */
export interface Branch {
  name: string;
  commit: { sha: string; url: string };
  protected: boolean;
}

/* ── Commit Diff ───────────────────────────────────────────────── */

/** A single file changed in a commit */
export interface CommitFile {
  filename: string;
  status: 'added' | 'modified' | 'removed' | 'renamed';
  additions: number;
  deletions: number;
  changes: number;
  patch?: string;
}

/** Full detail of a commit including file changes */
export interface CommitDetail {
  sha: string;
  commit: {
    message: string;
    author: { name: string; date: string; email: string };
  };
  author?: { login: string; avatar_url: string };
  stats: { total: number; additions: number; deletions: number };
  files: CommitFile[];
}

/* ── Repo Stats ────────────────────────────────────────────────── */

/** Aggregated statistics for a repository's tasks */
export interface RepoStats {
  total: number;
  open: number;
  in_progress: number;
  resolved: number;
  by_type: {
    TODO?: number;
    FIXME?: number;
    HACK?: number;
    BUG?: number;
    NOTE?: number;
  };
}

/* ── GitHub Actions ──────────────────────────────────────────── */

export interface ActionWorkflow {
  id: number;
  name: string;
  path: string;
  state: string;
  html_url: string;
  badge_url: string;
  created_at: string;
  updated_at: string;
}

export interface ActionWorkflowRun {
  id: number;
  name: string;
  display_title: string;
  status: string;
  conclusion: string;
  workflow_id: number;
  run_number: number;
  run_attempt: number;
  event: string;
  head_branch: string;
  head_sha: string;
  html_url: string;
  actor: string;
  actor_avatar_url: string;
  created_at: string;
  updated_at: string;
  run_started_at: string;
}

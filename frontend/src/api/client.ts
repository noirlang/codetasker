/**
 * CodeTasker — Axios API Client
 *
 * Single Axios instance shared across all API modules.
 * - Credentials (httpOnly session cookie) are sent automatically.
 * - 401 responses redirect to /login.
 * - All errors are normalized to ApiError shape.
 */

import axios, { AxiosError } from 'axios';
import type {
  User,
  Repo,
  Task,
  TaskStatus,
  FileTreeNode,
  InjectTaskRequest,
  ApiError,
  Collaborator,
  RepoRole,
  CommitListResponse,
  PullRequest,
  Organization,
  ActionWorkflow,
  ActionWorkflowRun,
  Notification,
  Comment,
  Issue,
  Branch,
  CommitDetail,
  RepoStats,
  ActivityLog,
} from '../types';

// ── Axios instance ─────────────────────────────────────────────────────────

const client = axios.create({
  baseURL: '/api',
  withCredentials: true, // send httpOnly session cookie on every request
  headers: {
    'Content-Type': 'application/json',
  },
});

// ── Response interceptor ───────────────────────────────────────────────────

client.interceptors.response.use(
  // Pass through successful responses unchanged
  (response) => response,

  // Handle errors globally
  (error: AxiosError<ApiError>) => {
    if (error.response?.status === 401) {
      // Session expired or not authenticated → redirect to login
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
      return Promise.reject(error);
    }

    // Normalize the error into a consistent shape
    const apiError: ApiError = {
      error: error.response?.data?.error ?? 'request_failed',
      message:
        error.response?.data?.message ??
        error.message ??
        'An unexpected error occurred.',
    };

    return Promise.reject(apiError);
  }
);

// ── Auth API ───────────────────────────────────────────────────────────────

export const authApi = {
  /**
   * Fetch the currently authenticated user.
   * GET /api/auth/me
   */
  getMe: async (): Promise<User> => {
    const { data } = await client.get<User>('/auth/me');
    return data;
  },

  /**
   * Update the authenticated user's profile settings (such as email).
   * PATCH /api/auth/me
   */
  updateMe: async (email: string): Promise<{ user: User; message: string }> => {
    const { data } = await client.patch<{ user: User; message: string }>('/auth/me', { email });
    return data;
  },

  /**
   * Log the current user out (clears the session cookie server-side).
   * POST /api/auth/logout
   */
  logout: async (): Promise<void> => {
    await client.post('/auth/logout');
  },
};

// ── Repositories API ───────────────────────────────────────────────────────

export const reposApi = {
  /**
   * List all repositories the authenticated user has access to.
   * GET /api/repos
   */
  list: async (): Promise<Repo[]> => {
    const { data } = await client.get<{ repos: Repo[] }>('/repos');
    return data.repos;
  },

  /**
   * Create a webhook for a repository to enable sync.
   * POST /api/repos/:owner/:repo/webhook
   */
  createWebhook: async (
    owner: string,
    repo: string,
    payloadUrl?: string
  ): Promise<{ message: string; payload_url: string; repo_id: number }> => {
    const { data } = await client.post<{
      message: string;
      payload_url: string;
      repo_id: number;
    }>(`/repos/${owner}/${repo}/webhook`, { payload_url: payloadUrl });
    return data;
  },

  /**
   * Fetch the file tree for a given repository and branch.
   * GET /api/repos/:owner/:repo/tree?ref=<branch>
   */
  getTree: async (
    owner: string,
    repo: string,
    branch?: string
  ): Promise<FileTreeNode[]> => {
    const { data } = await client.get<{ entries: FileTreeNode[] }>(
      `/repos/${owner}/${repo}/tree`,
      { params: branch ? { branch } : {} }
    );
    return data.entries;
  },

  /**
   * Fetch the decoded string content of a specific file.
   * GET /api/repos/:owner/:repo/contents?path=<filePath>&ref=<branch>
   */
  getContents: async (
    owner: string,
    repo: string,
    path: string,
    ref?: string
  ): Promise<string> => {
    const { data } = await client.get<{ content: string }>(
      `/repos/${owner}/${repo}/contents`,
      { params: { path, ...(ref ? { ref } : {}) } }
    );
    return data.content;
  },

  /**
   * Update (commit) the content of a specific file back to GitHub.
   * PUT /api/repos/:owner/:repo/contents
   */
  updateContents: async (
    owner: string,
    repo: string,
    path: string,
    content: string,
    branch?: string,
    message?: string,
    coAuthors?: string[]
  ): Promise<{ message: string; commit_sha: string }> => {
    const { data } = await client.put<{ message: string; commit_sha: string }>(
      `/repos/${owner}/${repo}/contents`,
      { path, content, branch, message, co_authors: coAuthors }
    );
    return data;
  },

  /**
   * List all collaborators for a repository.
   * GET /api/repos/:owner/:repo/collaborators
   */
  listCollaborators: async (
    owner: string,
    repo: string
  ): Promise<Collaborator[]> => {
    const { data } = await client.get<{ collaborators: Collaborator[] }>(
      `/repos/${owner}/${repo}/collaborators`
    );
    return data.collaborators;
  },

  /**
   * Add a new collaborator with a role.
   * POST /api/repos/:owner/:repo/collaborators
   */
  addCollaborator: async (
    owner: string,
    repo: string,
    username: string,
    role: RepoRole
  ): Promise<{ collaborator: Collaborator; warning?: string }> => {
    const { data } = await client.post<{ collaborator: Collaborator; warning?: string }>(
      `/repos/${owner}/${repo}/collaborators`,
      { username, role }
    );
    return data;
  },

  /**
   * Update a collaborator's role.
   * PATCH /api/repos/:owner/:repo/collaborators/:id
   */
  updateCollaboratorRole: async (
    owner: string,
    repo: string,
    id: string,
    role: RepoRole
  ): Promise<void> => {
    await client.patch(`/repos/${owner}/${repo}/collaborators/${id}`, {
      role,
    });
  },

  removeCollaborator: async (
    owner: string,
    repo: string,
    id: string
  ): Promise<void> => {
    await client.delete(`/repos/${owner}/${repo}/collaborators/${id}`);
  },

  /**
   * List commits for a repository on a given branch.
   * GET /api/repos/:owner/:repo/commits?branch=<branch>
   */
  listCommits: async (
    owner: string,
    repo: string,
    branch?: string,
    page = 1,
    perPage = 50
  ): Promise<CommitListResponse> => {
    const { data } = await client.get<CommitListResponse>(
      `/repos/${owner}/${repo}/commits`,
      {
        params: {
          ...(branch ? { branch } : {}),
          page,
          per_page: perPage,
        },
      }
    );
    return data;
  },

  /**
   * List GitHub Actions workflows for a repository.
   * GET /api/repos/:owner/:repo/actions/workflows
   */
  listActionWorkflows: async (
    owner: string,
    repo: string
  ): Promise<ActionWorkflow[]> => {
    const { data } = await client.get<{ workflows: ActionWorkflow[] }>(
      `/repos/${owner}/${repo}/actions/workflows`
    );
    return data.workflows;
  },

  /**
   * List recent GitHub Actions workflow runs for a repository.
   * GET /api/repos/:owner/:repo/actions/runs?branch=<branch>&status=<status>
   */
  listActionRuns: async (
    owner: string,
    repo: string,
    params?: { branch?: string; status?: string }
  ): Promise<ActionWorkflowRun[]> => {
    const { data } = await client.get<{ runs: ActionWorkflowRun[] }>(
      `/repos/${owner}/${repo}/actions/runs`,
      { params }
    );
    return data.runs;
  },

  /**
   * List pull requests for a repository.
   * GET /api/repos/:owner/:repo/pulls?state=<state>
   */
  listPulls: async (
    owner: string,
    repo: string,
    state?: string
  ): Promise<PullRequest[]> => {
    const { data } = await client.get<{ pulls: PullRequest[] }>(
      `/repos/${owner}/${repo}/pulls`,
      { params: state ? { state } : {} }
    );
    return data.pulls;
  },

  /**
   * Merge head branch into base branch.
   * POST /api/repos/:owner/:repo/merge
   */
  mergeBranch: async (
    owner: string,
    repo: string,
    base: string,
    head: string,
    commitMessage: string
  ): Promise<{ message: string; commit_sha: string }> => {
    const { data } = await client.post<{ message: string; commit_sha: string }>(
      `/repos/${owner}/${repo}/merge`,
      { base, head, commit_message: commitMessage }
    );
    return data;
  },

  /**
   * Merge an open pull request by its PR number.
   * POST /api/repos/:owner/:repo/pulls/:number/merge
   */
  mergePR: async (
    owner: string,
    repo: string,
    prNumber: number,
    options?: {
      mergeMethod?: 'merge' | 'squash' | 'rebase';
      commitTitle?: string;
      commitMessage?: string;
    }
  ): Promise<{ message: string; commit_sha: string }> => {
    const { data } = await client.post<{ message: string; commit_sha: string }>(
      `/repos/${owner}/${repo}/pulls/${prNumber}/merge`,
      {
        merge_method:   options?.mergeMethod   ?? 'merge',
        commit_title:   options?.commitTitle   ?? '',
        commit_message: options?.commitMessage ?? '',
      }
    );
    return data;
  },



  /**
   * Manually sync tasks for a repository.
   * POST /api/repos/:owner/:repo/sync
   */
  syncTasks: async (owner: string, repo: string): Promise<void> => {
    await client.post(`/repos/${owner}/${repo}/sync`);
  },

  /**
   * List organizations for the authenticated user.
   * GET /api/orgs
   */
  listOrgs: async (): Promise<Organization[]> => {
    const { data } = await client.get<{ orgs: Organization[] }>('/orgs');
    return data.orgs;
  },

  /**
   * List repositories for a specific organization.
   * GET /api/orgs/:org/repos
   */
  listOrgRepos: async (org: string): Promise<Repo[]> => {
    const { data } = await client.get<{ repos: Repo[] }>(`/orgs/${org}/repos`);
    return data.repos;
  },

  /**
   * List GitHub Issues for a repository.
   * GET /api/repos/:owner/:repo/issues?state=<state>
   */
  listIssues: async (owner: string, repo: string, state = 'open'): Promise<Issue[]> => {
    const { data } = await client.get<{ issues: Issue[] }>(
      `/repos/${owner}/${repo}/issues`,
      { params: { state } }
    );
    return data.issues || [];
  },

  /**
   * Create a new GitHub Issue.
   * POST /api/repos/:owner/:repo/issues
   */
  createIssue: async (
    owner: string,
    repo: string,
    issueData: { title: string; body: string; labels: string[] }
  ): Promise<Issue> => {
    const { data } = await client.post<{ issue: Issue }>(
      `/repos/${owner}/${repo}/issues`,
      issueData
    );
    return data.issue;
  },

  /**
   * Update a GitHub Issue's state (open/closed).
   * PATCH /api/repos/:owner/:repo/issues/:number
   */
  updateIssue: async (
    owner: string,
    repo: string,
    number: number,
    state: string
  ): Promise<Issue> => {
    const { data } = await client.patch<{ issue: Issue }>(
      `/repos/${owner}/${repo}/issues/${number}`,
      { state }
    );
    return data.issue;
  },

  /**
   * List branches for a repository.
   * GET /api/repos/:owner/:repo/branches
   */
  listBranches: async (owner: string, repo: string): Promise<Branch[]> => {
    const { data } = await client.get<{ branches: Branch[] }>(
      `/repos/${owner}/${repo}/branches`
    );
    return data.branches || [];
  },

  /**
   * Create a new branch.
   * POST /api/repos/:owner/:repo/branches
   */
  createBranch: async (
    owner: string,
    repo: string,
    branchData: { name: string; from_sha: string }
  ): Promise<Branch> => {
    const { data } = await client.post<{ branch: Branch }>(
      `/repos/${owner}/${repo}/branches`,
      branchData
    );
    return data.branch;
  },

  /**
   * Delete a branch.
   * DELETE /api/repos/:owner/:repo/branches/:branch
   */
  deleteBranch: async (owner: string, repo: string, branch: string): Promise<void> => {
    await client.delete(`/repos/${owner}/${repo}/branches/${branch}`);
  },

  /**
   * Fetch commit detail with file diffs.
   * GET /api/repos/:owner/:repo/commits/:sha
   */
  getCommitDiff: async (owner: string, repo: string, sha: string): Promise<CommitDetail> => {
    const { data } = await client.get<CommitDetail>(
      `/repos/${owner}/${repo}/commits/${sha}`
    );
    return data;
  },

  getStats: async (owner: string, repo: string): Promise<RepoStats> => {
    const { data } = await client.get<{ stats: RepoStats }>(
      `/repos/${owner}/${repo}/stats`
    );
    return data.stats;
  },

  /**
   * Get repository activity feed.
   * GET /api/repos/:owner/:repo/activity
   */
  getActivity: async (owner: string, repo: string): Promise<ActivityLog[]> => {
    const { data } = await client.get<{ activities: ActivityLog[] }>(
      `/repos/${owner}/${repo}/activity`
    );
    return data.activities || [];
  },
};

// ── Tasks API ──────────────────────────────────────────────────────────────

export const tasksApi = {
  /**
   * Fetch all tasks for a specific repository.
   * GET /api/tasks?repo_id=<repoId>
   */
  getByRepo: async (repoId: number): Promise<Task[]> => {
    const { data } = await client.get<{ tasks: Task[] }>('/tasks', {
      params: { repo_id: repoId },
    });
    return data.tasks;
  },

  /**
   * Update the status of a task (e.g. open → in_progress).
   * PATCH /api/tasks/:id
   */
  updateStatus: async (id: string, status: TaskStatus): Promise<Task> => {
    const { data } = await client.patch<Task>(`/tasks/${id}`, { status });
    return data;
  },

  /**
   * Update a task generic fields (e.g. status and/or pr_url).
   * PATCH /api/tasks/:id
   */
  updateTask: async (
    id: string,
    updates: { status?: TaskStatus; pr_url?: string; issue_url?: string; assignee_username?: string; clear_assignee?: boolean }
  ): Promise<Task> => {
    const { data } = await client.patch<Task>(`/tasks/${id}`, updates);
    return data;
  },

  /**
   * Inject a new TODO comment into a file and open a pull request.
   * POST /api/tasks/inject
   * Returns the URL of the created pull request.
   */
  inject: async (req: InjectTaskRequest): Promise<{ pr_url: string }> => {
    const { data } = await client.post<{ pr_url: string }>(
      '/tasks/inject',
      req
    );
    return data;
  },
};

// ── Notifications API ──────────────────────────────────────────────────────

export const notificationsApi = {
  /**
   * List all notifications for the authenticated user.
   * GET /api/notifications
   */
  list: (): Promise<Notification[]> =>
    client.get('/notifications').then((r) => (r.data.notifications as Notification[]) || []),

  /**
   * Get the count of unread notifications.
   * GET /api/notifications/unread-count
   */
  unreadCount: (): Promise<number> =>
    client.get('/notifications/unread-count').then((r) => r.data.count as number),

  /**
   * Mark a single notification as read.
   * PATCH /api/notifications/:id/read
   */
  markRead: (id: string) => client.patch(`/notifications/${id}/read`),

  /**
   * Mark all notifications as read.
   * PATCH /api/notifications/read-all
   */
  markAllRead: () => client.patch('/notifications/read-all'),
};

// ── Comments API ────────────────────────────────────────────────────────────

export const commentsApi = {
  /**
   * List all comments for a task.
   * GET /api/tasks/:id/comments
   */
  list: (taskId: string): Promise<Comment[]> =>
    client.get(`/tasks/${taskId}/comments`).then((r) => (r.data.comments as Comment[]) || []),

  /**
   * Add a comment to a task.
   * POST /api/tasks/:id/comments
   */
  add: (taskId: string, content: string): Promise<Comment> =>
    client
      .post(`/tasks/${taskId}/comments`, { content })
      .then((r) => r.data.comment as Comment),

  /**
   * Delete a comment from a task.
   * DELETE /api/tasks/:id/comments/:commentId
   */
  delete: (taskId: string, commentId: string) =>
    client.delete(`/tasks/${taskId}/comments/${commentId}`),
};

export default client;

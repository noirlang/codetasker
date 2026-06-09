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
  Commit,
  PullRequest,
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
  ): Promise<Collaborator> => {
    const { data } = await client.post<{ collaborator: Collaborator }>(
      `/repos/${owner}/${repo}/collaborators`,
      { username, role }
    );
    return data.collaborator;
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
    branch?: string
  ): Promise<Commit[]> => {
    const { data } = await client.get<{ commits: Commit[] }>(
      `/repos/${owner}/${repo}/commits`,
      { params: branch ? { branch } : {} }
    );
    return data.commits;
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
   * Manually sync tasks for a repository.
   * POST /api/repos/:owner/:repo/sync
   */
  syncTasks: async (owner: string, repo: string): Promise<void> => {
    await client.post(`/repos/${owner}/${repo}/sync`);
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
    updates: { status?: TaskStatus; pr_url?: string }
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

export default client;

/**
 * CodeTasker — Task Zustand Store
 *
 * Manages the task list for the currently-viewed repository.
 * Implements optimistic updates for status changes so the UI
 * feels instant even before the server responds.
 */

import { create } from 'zustand';
import { tasksApi, reposApi } from '../api/client';
import type { Task, TaskStatus, InjectTaskRequest, ApiError } from '../types';

// ── State shape ────────────────────────────────────────────────────────────

interface TaskState {
  /** All tasks for the currently viewed repository. */
  tasks: Task[];

  /** True while a network request is in flight. */
  isLoading: boolean;

  /** Human-readable error message from the last failed operation. */
  error: string | null;

  /**
   * Load tasks for the given repository ID.
   * Replaces the current task list entirely.
   */
  fetchTasks: (repoId: number) => Promise<void>;

  /**
   * Update a task's status via the API.
   * Uses optimistic updates: the local state is updated immediately,
   * and reverted if the API call fails.
   */
  updateTaskStatus: (id: string, status: TaskStatus) => Promise<void>;

  /**
   * Inject a new TODO comment into a repo file and open a pull request.
   * Returns the URL of the created PR on success.
   */
  injectTask: (req: InjectTaskRequest) => Promise<string>;

  /**
   * Link a task to a GitHub Pull Request URL.
   */
  linkTaskToPR: (id: string, prUrl: string) => Promise<void>;

  /**
   * Link a task to a GitHub Issue URL.
   */
  linkTaskToIssue: (id: string, issueUrl: string) => Promise<void>;

  /**
   * Update a task's assignee in the backend and local store.
   */
  updateTaskAssignee: (id: string, username: string | null) => Promise<void>;

  /**
   * Immediately update a task's status in the local store (no API call).
   * Used internally for optimistic updates; also exposed for testing.
   */
  optimisticallyUpdateStatus: (id: string, status: TaskStatus) => void;

  /**
   * Manually trigger sync and refetch tasks.
   */
  syncRepoTasks: (owner: string, repoName: string, repoId: number) => Promise<void>;
}

// ── Store implementation ───────────────────────────────────────────────────

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  isLoading: false,
  error: null,

  // ── Fetch tasks ──────────────────────────────────────────────────────────

  fetchTasks: async (repoId: number) => {
    set({ isLoading: true, error: null });
    try {
      const tasks = await tasksApi.getByRepo(repoId);
      set({ tasks: tasks || [], isLoading: false });
    } catch (err) {
      const apiErr = err as ApiError;
      set({
        error: apiErr.message ?? 'Failed to load tasks.',
        isLoading: false,
      });
    }
  },

  // ── Update task status (optimistic) ─────────────────────────────────────

  updateTaskStatus: async (id: string, status: TaskStatus) => {
    // Snapshot the previous state so we can revert on failure
    const previousTasks = get().tasks;

    // Apply the update immediately in local state
    get().optimisticallyUpdateStatus(id, status);

    try {
      // Confirm with the server
      const updated = await tasksApi.updateStatus(id, status);

      // Replace the optimistic version with the authoritative server response
      set((state) => ({
        tasks: state.tasks.map((t) => (t.id === updated.id ? updated : t)),
      }));
    } catch (err) {
      // Revert the optimistic change
      set({ tasks: previousTasks });

      const apiErr = err as ApiError;
      set({ error: apiErr.message ?? 'Failed to update task status.' });
    }
  },

  // ── Link task to PR (optimistic) ────────────────────────────────────────

  linkTaskToPR: async (id: string, prUrl: string) => {
    const previousTasks = get().tasks;

    // Optimistically update locally
    set((state) => ({
      tasks: state.tasks.map((t) =>
        t.id === id ? { ...t, pr_url: prUrl, updated_at: new Date().toISOString() } : t
      ),
    }));

    try {
      const updated = await tasksApi.updateTask(id, { pr_url: prUrl });
      set((state) => ({
        tasks: state.tasks.map((t) => (t.id === updated.id ? updated : t)),
      }));
    } catch (err) {
      // Revert on failure
      set({ tasks: previousTasks });

      const apiErr = err as ApiError;
      set({ error: apiErr.message ?? 'Failed to link task to Pull Request.' });
    }
  },

  // ── Link task to Issue (optimistic) ─────────────────────────────────────

  linkTaskToIssue: async (id: string, issueUrl: string) => {
    const previousTasks = get().tasks;

    // Optimistically update locally
    set((state) => ({
      tasks: state.tasks.map((t) =>
        t.id === id ? { ...t, issue_url: issueUrl, updated_at: new Date().toISOString() } : t
      ),
    }));

    try {
      const updated = await tasksApi.updateTask(id, { issue_url: issueUrl });
      set((state) => ({
        tasks: state.tasks.map((t) => (t.id === updated.id ? updated : t)),
      }));
    } catch (err) {
      // Revert on failure
      set({ tasks: previousTasks });

      const apiErr = err as ApiError;
      set({ error: apiErr.message ?? 'Failed to link task to GitHub Issue.' });
    }
  },

  // ── Inject task ──────────────────────────────────────────────────────────

  injectTask: async (req: InjectTaskRequest): Promise<string> => {
    const { pr_url } = await tasksApi.inject(req);
    return pr_url;
  },

  updateTaskAssignee: async (id: string, username: string | null) => {
    const previousTasks = get().tasks;

    // Optimistically update locally
    set((state) => ({
      tasks: state.tasks.map((t) =>
        t.id === id
          ? {
              ...t,
              assignee_username: username || '',
              updated_at: new Date().toISOString(),
            }
          : t
      ),
    }));

    try {
      let updated: Task;
      if (username) {
        updated = await tasksApi.updateTask(id, { assignee_username: username });
      } else {
        updated = await tasksApi.updateTask(id, { clear_assignee: true });
      }
      set((state) => ({
        tasks: state.tasks.map((t) => (t.id === updated.id ? updated : t)),
      }));
    } catch (err) {
      // Revert on failure
      set({ tasks: previousTasks });

      const apiErr = err as ApiError;
      set({ error: apiErr.message ?? 'Failed to update task assignee.' });
      throw err;
    }
  },

  // ── Optimistic helper ────────────────────────────────────────────────────

  optimisticallyUpdateStatus: (id: string, status: TaskStatus) => {
    set((state) => ({
      tasks: state.tasks.map((t) =>
        t.id === id ? { ...t, status, updated_at: new Date().toISOString() } : t
      ),
    }));
  },

  // ── Sync repo tasks ──────────────────────────────────────────────────────

  syncRepoTasks: async (owner: string, repoName: string, repoId: number) => {
    set({ isLoading: true, error: null });
    try {
      await reposApi.syncTasks(owner, repoName);
      const tasks = await tasksApi.getByRepo(repoId);
      set({ tasks: tasks || [], isLoading: false });
    } catch (err) {
      const apiErr = err as ApiError;
      set({
        error: apiErr.message ?? 'Failed to sync tasks.',
        isLoading: false,
      });
      throw err;
    }
  },
}));

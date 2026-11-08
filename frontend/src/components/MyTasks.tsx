/**
 * MyTasks — Dedicated view for tasks and TODOs assigned to the current user.
 *
 * Provides:
 * - Summary statistics (Total, Open, In Progress, Resolved, Overdue)
 * - Filtering by status, overdue status, and instant text search
 * - Direct status updating
 * - Integrated TaskDetailModal for viewing/writing comments and managing due dates
 * - Navigation links directly into the repository code viewer
 */

import { useState, useEffect, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircle2,
  Clock,
  AlertCircle,
  Calendar,
  MessageSquare,
  Search,
  RefreshCw,
  ListTodo,
  FileCode,
  FolderGit2,
  User,
  ArrowRight,
} from 'lucide-react';
import { tasksApi } from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { Task, TaskStatus } from '../types';
import Badge from './ui/Badge';
import Spinner from './ui/Spinner';
import TaskDetailModal from './TaskDetailModal';

export default function MyTasks() {
  const navigate = useNavigate();
  const { user } = useAuthStore();

  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [selectedStatus, setSelectedStatus] = useState<'all' | TaskStatus | 'overdue'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  // Selected task for comments & details modal
  const [activeModalTask, setActiveModalTask] = useState<Task | null>(null);

  // Fetch tasks assigned to the current user
  const fetchMyTasks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await tasksApi.getMyTasks();
      setTasks(data || []);
    } catch (err: any) {
      setError(err?.message ?? 'Failed to load your assigned tasks.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMyTasks();
  }, [fetchMyTasks]);

  // Quick status update from card or modal
  const handleStatusChange = async (taskId: string, newStatus: TaskStatus) => {
    // Optimistic update
    setTasks((prev) =>
      prev.map((t) => (t.id === taskId ? { ...t, status: newStatus } : t))
    );
    if (activeModalTask && activeModalTask.id === taskId) {
      setActiveModalTask((prev) => (prev ? { ...prev, status: newStatus } : null));
    }

    try {
      const updated = await tasksApi.updateStatus(taskId, newStatus);
      setTasks((prev) =>
        prev.map((t) => (t.id === taskId ? { ...t, ...updated } : t))
      );
    } catch (err) {
      // Refetch on error
      fetchMyTasks();
    }
  };

  // Due date change
  const handleDueDateChange = async (taskId: string, dueDate: string | null) => {
    setTasks((prev) =>
      prev.map((t) =>
        t.id === taskId ? { ...t, due_date: dueDate || undefined } : t
      )
    );
    if (activeModalTask && activeModalTask.id === taskId) {
      setActiveModalTask((prev) =>
        prev ? { ...prev, due_date: dueDate || undefined } : null
      );
    }

    try {
      if (dueDate) {
        await tasksApi.updateTask(taskId, { due_date: dueDate });
      } else {
        await tasksApi.updateTask(taskId, { clear_due_date: true });
      }
    } catch (err) {
      fetchMyTasks();
    }
  };

  // Re-assign or unassign
  const handleAssign = async (taskId: string, username: string | null) => {
    setTasks((prev) =>
      prev.map((t) =>
        t.id === taskId ? { ...t, assignee_username: username || '' } : t
      )
    );
    if (activeModalTask && activeModalTask.id === taskId) {
      setActiveModalTask((prev) =>
        prev ? { ...prev, assignee_username: username || '' } : null
      );
    }

    try {
      if (username) {
        await tasksApi.updateTask(taskId, { assignee_username: username });
      } else {
        await tasksApi.updateTask(taskId, { clear_assignee: true });
      }
    } catch (err) {
      fetchMyTasks();
    }
  };

  // Check if a task is overdue
  const isTaskOverdue = (task: Task): boolean => {
    if (!task.due_date || task.status === 'resolved') return false;
    const due = new Date(task.due_date);
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    return due < today;
  };

  // Metrics computation
  const stats = useMemo(() => {
    const total = tasks.length;
    const open = tasks.filter((t) => t.status === 'open').length;
    const inProgress = tasks.filter((t) => t.status === 'in_progress').length;
    const resolved = tasks.filter((t) => t.status === 'resolved').length;
    const overdue = tasks.filter(isTaskOverdue).length;
    const completionRate = total > 0 ? Math.round((resolved / total) * 100) : 0;
    return { total, open, inProgress, resolved, overdue, completionRate };
  }, [tasks]);

  // Filtered task list
  const filteredTasks = useMemo(() => {
    return tasks.filter((task) => {
      // Status filter
      if (selectedStatus === 'overdue') {
        if (!isTaskOverdue(task)) return false;
      } else if (selectedStatus !== 'all') {
        if (task.status !== selectedStatus) return false;
      }

      // Search query filter
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        const matchesContent = task.content.toLowerCase().includes(q);
        const matchesRepo = (task.repo_name || '').toLowerCase().includes(q);
        const matchesPath = task.file_path.toLowerCase().includes(q);
        const matchesType = task.type.toLowerCase().includes(q);
        return matchesContent || matchesRepo || matchesPath || matchesType;
      }

      return true;
    });
  }, [tasks, selectedStatus, searchQuery]);

  return (
    <div className="flex flex-col gap-6">
      {/* ── Top Metrics ──────────────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <div className="rounded border border-[#242424] bg-[#111111] p-3.5">
          <div className="flex items-center justify-between text-[#888888]">
            <span className="text-[11px] font-mono uppercase tracking-wider">Assigned to Me</span>
            <ListTodo size={14} />
          </div>
          <p className="mt-2 font-mono text-2xl font-bold text-white">{stats.total}</p>
        </div>

        <div className="rounded border border-[#242424] bg-[#111111] p-3.5">
          <div className="flex items-center justify-between text-blue-400/80">
            <span className="text-[11px] font-mono uppercase tracking-wider">Open</span>
            <span className="h-2 w-2 rounded-full bg-blue-400" />
          </div>
          <p className="mt-2 font-mono text-2xl font-bold text-white">{stats.open}</p>
        </div>

        <div className="rounded border border-[#242424] bg-[#111111] p-3.5">
          <div className="flex items-center justify-between text-amber-400/80">
            <span className="text-[11px] font-mono uppercase tracking-wider">In Progress</span>
            <Clock size={14} className="text-amber-400" />
          </div>
          <p className="mt-2 font-mono text-2xl font-bold text-white">{stats.inProgress}</p>
        </div>

        <div className="rounded border border-[#242424] bg-[#111111] p-3.5">
          <div className="flex items-center justify-between text-emerald-400/80">
            <span className="text-[11px] font-mono uppercase tracking-wider">Resolved</span>
            <CheckCircle2 size={14} className="text-emerald-400" />
          </div>
          <p className="mt-2 font-mono text-2xl font-bold text-white">{stats.resolved}</p>
        </div>

        <div
          className={`rounded border p-3.5 ${
            stats.overdue > 0
              ? 'border-red-500/30 bg-red-500/5'
              : 'border-[#242424] bg-[#111111]'
          }`}
        >
          <div className="flex items-center justify-between">
            <span
              className={`text-[11px] font-mono uppercase tracking-wider ${
                stats.overdue > 0 ? 'text-red-400 font-semibold' : 'text-[#888888]'
              }`}
            >
              Overdue
            </span>
            <AlertCircle
              size={14}
              className={stats.overdue > 0 ? 'text-red-400 animate-pulse' : 'text-[#666666]'}
            />
          </div>
          <p
            className={`mt-2 font-mono text-2xl font-bold ${
              stats.overdue > 0 ? 'text-red-400' : 'text-white'
            }`}
          >
            {stats.overdue}
          </p>
        </div>
      </div>

      {/* ── Controls: Search & Status Filters ───────────────────────────────── */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b border-[#222222] pb-4">
        {/* Status Filter Buttons */}
        <div className="flex flex-wrap items-center gap-1.5">
          {(
            [
              { id: 'all', label: 'All', count: stats.total },
              { id: 'open', label: 'Open', count: stats.open },
              { id: 'in_progress', label: 'In Progress', count: stats.inProgress },
              { id: 'resolved', label: 'Resolved', count: stats.resolved },
              { id: 'overdue', label: 'Overdue', count: stats.overdue },
            ] as const
          ).map((tab) => {
            const isActive = selectedStatus === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setSelectedStatus(tab.id)}
                className={`flex items-center gap-1.5 rounded px-3 py-1.5 text-xs font-mono transition-all cursor-pointer ${
                  isActive
                    ? 'bg-white text-black font-semibold'
                    : 'bg-[#141414] text-[#888888] hover:text-white border border-[#222222] hover:border-[#333333]'
                }`}
              >
                <span>{tab.label}</span>
                <span
                  className={`rounded px-1.5 py-0.2 text-[10px] ${
                    isActive
                      ? 'bg-black/10 text-black font-bold'
                      : tab.id === 'overdue' && tab.count > 0
                      ? 'bg-red-500/20 text-red-400 font-bold'
                      : 'bg-[#222222] text-[#888888]'
                  }`}
                >
                  {tab.count}
                </span>
              </button>
            );
          })}
        </div>

        {/* Search Bar & Refresh */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1 sm:w-64">
            <Search
              size={13}
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[#666666]"
            />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search content, file, repo..."
              className="input pl-8 py-1.5 text-xs w-full font-mono bg-[#111111]"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-[#666666] hover:text-white"
              >
                ×
              </button>
            )}
          </div>

          <button
            onClick={fetchMyTasks}
            disabled={loading}
            title="Refresh assigned tasks"
            className="p-2 bg-[#141414] hover:bg-[#1f1f1f] text-[#a0a0a0] hover:text-white border border-[#2a2a2a] rounded transition-colors disabled:opacity-50 cursor-pointer"
          >
            <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {/* ── Task List / State ────────────────────────────────────────────────── */}
      {loading ? (
        <div className="flex flex-col items-center justify-center py-20 gap-3">
          <Spinner size={28} />
          <p className="text-xs font-mono text-[#666666]">Loading your tasks…</p>
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
          <AlertCircle size={32} className="text-red-400" />
          <p className="text-sm text-red-400">{error}</p>
          <button onClick={fetchMyTasks} className="btn-secondary text-xs">
            Retry
          </button>
        </div>
      ) : filteredTasks.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-[#262626] bg-[#0e0e0e] py-16 px-6 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-[#181818] border border-[#2a2a2a] mb-3 text-[#666666]">
            <ListTodo size={22} />
          </div>
          {tasks.length === 0 ? (
            <>
              <h3 className="text-sm font-semibold text-white">No tasks assigned to you</h3>
              <p className="mt-1 max-w-sm text-xs text-[#888888] leading-relaxed">
                When you or your collaborators assign tasks to{' '}
                <span className="font-mono text-white">@{user?.username || 'you'}</span> in any
                repository, they will show up here.
              </p>
            </>
          ) : (
            <>
              <h3 className="text-sm font-semibold text-white">No tasks match your filter</h3>
              <p className="mt-1 text-xs text-[#888888]">
                Try clearing your search query or selecting a different status tab.
              </p>
              <button
                onClick={() => {
                  setSelectedStatus('all');
                  setSearchQuery('');
                }}
                className="btn-secondary mt-3 text-xs"
              >
                Reset filters
              </button>
            </>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3.5">
          {filteredTasks.map((task) => {
            const isOverdue = isTaskOverdue(task);
            const repoParts = (task.repo_name || '').split('/');
            const repoOwner = repoParts[0] || '';
            const repoName = repoParts[1] || '';

            return (
              <div
                key={task.id}
                className="group relative flex flex-col justify-between rounded-lg border border-[#242424] bg-[#121212] p-4.5 transition-all duration-150 hover:border-[#383838] hover:bg-[#141414] gap-3"
              >
                {/* Header row: Repo + File:Line + Badges */}
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex flex-wrap items-center gap-2">
                    {/* Repo link */}
                    {task.repo_name && (
                      <button
                        onClick={() => navigate(`/repos/${repoOwner}/${repoName}`)}
                        className="inline-flex items-center gap-1 rounded bg-[#1c1c1c] border border-[#2b2b2b] px-2 py-0.5 text-[11px] font-mono text-[#aaaaaa] hover:text-white hover:border-[#444444] transition-colors cursor-pointer"
                        title="Open Repository"
                      >
                        <FolderGit2 size={11} className="text-[#888888]" />
                        <span>{task.repo_name}</span>
                      </button>
                    )}

                    {/* File path & line */}
                    <span
                      className="inline-flex items-center gap-1 font-mono text-xs text-[#888888] truncate max-w-xs md:max-w-md"
                      title={task.file_path}
                    >
                      <FileCode size={12} className="text-[#666666] shrink-0" />
                      <span className="truncate">{task.file_path}</span>
                      <span className="text-[#555555]">:{task.line_number}</span>
                    </span>

                    {/* Annotation Type Badge */}
                    <Badge type={task.type} />
                  </div>

                  {/* Status Dropdown & Due Date Badge */}
                  <div className="flex items-center gap-2">
                    {/* Due date tag */}
                    {task.due_date ? (
                      <div
                        className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-[11px] font-mono ${
                          isOverdue
                            ? 'border-red-500/40 bg-red-500/10 text-red-400 font-semibold'
                            : 'border-[#2a2a2a] bg-[#181818] text-[#888888]'
                        }`}
                        title={isOverdue ? 'This task is overdue!' : 'Target due date'}
                      >
                        <Calendar size={11} className={isOverdue ? 'text-red-400' : 'text-[#666666]'} />
                        <span>
                          {isOverdue && 'Overdue: '}
                          {new Date(task.due_date).toLocaleDateString(undefined, {
                            month: 'short',
                            day: 'numeric',
                            year:
                              new Date(task.due_date).getFullYear() !== new Date().getFullYear()
                                ? 'numeric'
                                : undefined,
                          })}
                        </span>
                      </div>
                    ) : (
                      <button
                        onClick={() => setActiveModalTask(task)}
                        className="text-[10px] font-mono text-[#555555] hover:text-[#888888] transition-colors"
                        title="Click to set target due date"
                      >
                        + Set due date
                      </button>
                    )}

                    {/* Status Dropdown */}
                    <select
                      value={task.status}
                      onChange={(e) => handleStatusChange(task.id, e.target.value as TaskStatus)}
                      className={`rounded border px-2 py-0.5 text-[11px] font-mono focus:outline-none cursor-pointer transition-colors ${
                        task.status === 'resolved'
                          ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400'
                          : task.status === 'in_progress'
                          ? 'border-amber-500/30 bg-amber-500/10 text-amber-400'
                          : 'border-blue-500/30 bg-blue-500/10 text-blue-400'
                      }`}
                    >
                      <option value="open" className="bg-[#181818] text-white">
                        ● Open
                      </option>
                      <option value="in_progress" className="bg-[#181818] text-white">
                        ◐ In Progress
                      </option>
                      <option value="resolved" className="bg-[#181818] text-white">
                        ✓ Resolved
                      </option>
                    </select>
                  </div>
                </div>

                {/* Content */}
                <div
                  onClick={() => setActiveModalTask(task)}
                  className="cursor-pointer group/content"
                >
                  <p className="text-sm text-[#cccccc] leading-relaxed group-hover/content:text-white transition-colors">
                    {task.content}
                  </p>
                </div>

                {/* Bottom Row: Metadata + Action buttons */}
                <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[#1c1c1c] pt-3 text-[11px] text-[#666666]">
                  {/* Creator / Maintainer attribution */}
                  <div className="flex items-center gap-3">
                    {task.created_by_username && (
                      <span className="flex items-center gap-1 font-mono">
                        <User size={10} className="text-[#555555]" />
                        <span>from @{task.created_by_username}</span>
                      </span>
                    )}
                    {task.maintainer_username && (
                      <span className="font-mono text-amber-400/80">
                        maintainer: @{task.maintainer_username}
                      </span>
                    )}
                  </div>

                  {/* Actions: Write Comment & Jump to repo */}
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setActiveModalTask(task)}
                      className="btn-secondary text-[11px] py-1 px-2.5 flex items-center gap-1.5 border-[#282828] hover:border-white transition-all cursor-pointer"
                    >
                      <MessageSquare size={12} />
                      <span>Comments & Details</span>
                    </button>

                    {task.repo_name && (
                      <button
                        onClick={() => navigate(`/repos/${repoOwner}/${repoName}`)}
                        className="btn-ghost text-[11px] py-1 px-2 flex items-center gap-1 text-[#888888] hover:text-white cursor-pointer"
                        title="Jump to repository code"
                      >
                        <span>View in repo</span>
                        <ArrowRight size={11} />
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* ── Task Detail & Comments Modal ─────────────────────────────────────── */}
      {activeModalTask && (
        <TaskDetailModal
          task={activeModalTask}
          currentUsername={user?.username || ''}
          onClose={() => setActiveModalTask(null)}
          onStatusChange={handleStatusChange}
          onDueDateChange={handleDueDateChange}
          onAssign={handleAssign}
          onTaskUpdated={(updated) => {
            setTasks((prev) =>
              prev.map((t) => (t.id === updated.id ? { ...t, ...updated } : t))
            );
            setActiveModalTask(updated);
          }}
        />
      )}
    </div>
  );
}

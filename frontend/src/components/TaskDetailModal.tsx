/**
 * TaskDetailModal — Full modal for inspecting a task, managing its lifecycle status,
 * target due date (son tarih), assignee, comments, and discussion proposals.
 */

import { useState, useEffect } from 'react';
import {
  X,
  User,
  UserMinus,
  MessageSquare,
  Send,
  Trash2,
  Calendar,
  AlertCircle,
  Plus,
  Vote,
  CheckCircle2,
  XCircle,
} from 'lucide-react';
import { commentsApi, proposalsApi, tasksApi } from '../api/client';
import type { Task, Comment, TaskProposal, ProposalStatus, Collaborator, TaskStatus } from '../types';
import Badge from './ui/Badge';

export interface TaskDetailModalProps {
  task: Task;
  collaborators?: Collaborator[];
  currentUsername: string;
  onClose: () => void;
  onAssign?: (taskId: string, username: string | null) => Promise<void>;
  onDueDateChange?: (taskId: string, dueDate: string | null) => Promise<void>;
  onStatusChange?: (taskId: string, status: TaskStatus) => Promise<void>;
  onInjectClick?: (lineNumber?: number, filePath?: string) => void;
  onTaskUpdated?: (updatedTask: Task) => void;
}

export default function TaskDetailModal({
  task,
  collaborators = [],
  currentUsername,
  onClose,
  onAssign,
  onDueDateChange,
  onStatusChange,
  onInjectClick,
  onTaskUpdated,
}: TaskDetailModalProps) {
  // Comments state
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(true);
  const [newComment, setNewComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Assignee state
  const [showAssigneePicker, setShowAssigneePicker] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [assigneeSearch, setAssigneeSearch] = useState('');
  const [localAssignee, setLocalAssignee] = useState<string | null>(
    task.assignee_username ?? null
  );

  // Due Date state
  const [showDatePicker, setShowDatePicker] = useState(false);
  const [savingDueDate, setSavingDueDate] = useState(false);
  const [localDueDate, setLocalDueDate] = useState<string | null>(
    task.due_date ?? null
  );
  const [dateInputVal, setDateInputVal] = useState<string>(() => {
    if (task.due_date) {
      try {
        return new Date(task.due_date).toISOString().slice(0, 10);
      } catch {
        return '';
      }
    }
    return '';
  });

  // Task Status state
  const [localStatus, setLocalStatus] = useState<TaskStatus>(task.status);
  const [updatingStatus, setUpdatingStatus] = useState(false);

  // Proposals state
  const [proposals, setProposals] = useState<TaskProposal[]>([]);
  const [proposalsLoading, setProposalsLoading] = useState(true);
  const [newPropTitle, setNewPropTitle] = useState('');
  const [newPropContent, setNewPropContent] = useState('');
  const [submittingProp, setSubmittingProp] = useState(false);
  const [detailTab, setDetailTab] = useState<'comments' | 'proposals'>('comments');

  // Load comments on mount
  useEffect(() => {
    setCommentsLoading(true);
    commentsApi
      .list(task.id)
      .then(setComments)
      .catch(() => setComments([]))
      .finally(() => setCommentsLoading(false));
  }, [task.id]);

  // Load proposals on mount
  useEffect(() => {
    setProposalsLoading(true);
    proposalsApi
      .list(task.id)
      .then(setProposals)
      .catch(() => setProposals([]))
      .finally(() => setProposalsLoading(false));
  }, [task.id]);

  // ── Handlers: Comments ───────────────────────────────────────────────────

  const handleAddComment = async () => {
    const trimmed = newComment.trim();
    if (!trimmed || submitting) return;
    setSubmitting(true);
    try {
      const comment = await commentsApi.add(task.id, trimmed);
      setComments((prev) => [...prev, comment]);
      setNewComment('');
    } catch {
      // Silently fail
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteComment = async (commentId: string) => {
    setComments((prev) => prev.filter((c) => c.id !== commentId));
    try {
      await commentsApi.delete(task.id, commentId);
    } catch {
      commentsApi.list(task.id).then(setComments).catch(() => {});
    }
  };

  // ── Handlers: Proposals ──────────────────────────────────────────────────

  const handleAddProposal = async () => {
    if (!newPropTitle.trim() || submittingProp) return;
    setSubmittingProp(true);
    try {
      const p = await proposalsApi.add(task.id, newPropTitle.trim(), newPropContent.trim());
      setProposals((prev) => [p, ...prev]);
      setNewPropTitle('');
      setNewPropContent('');
    } catch {
      // Silently fail
    } finally {
      setSubmittingProp(false);
    }
  };

  const handleVoteProposal = async (proposalId: string, status: ProposalStatus) => {
    setProposals((prev) =>
      prev.map((p) => (p.id === proposalId ? { ...p, status } : p))
    );
    try {
      await proposalsApi.updateStatus(task.id, proposalId, status);
    } catch {
      proposalsApi.list(task.id).then(setProposals).catch(() => {});
    }
  };

  const handleDeleteProposal = async (proposalId: string) => {
    setProposals((prev) => prev.filter((p) => p.id !== proposalId));
    try {
      await proposalsApi.delete(task.id, proposalId);
    } catch {
      proposalsApi.list(task.id).then(setProposals).catch(() => {});
    }
  };

  // ── Handlers: Assignee ───────────────────────────────────────────────────

  const handleAssign = async (username: string | null) => {
    setAssigning(true);
    setLocalAssignee(username);
    setShowAssigneePicker(false);
    try {
      if (onAssign) {
        await onAssign(task.id, username);
      } else {
        const updated = await tasksApi.updateTask(task.id, {
          assignee_username: username || undefined,
          clear_assignee: !username,
        });
        if (onTaskUpdated) onTaskUpdated(updated);
      }
    } catch {
      setLocalAssignee(task.assignee_username ?? null);
    } finally {
      setAssigning(false);
    }
  };

  // ── Handlers: Due Date ───────────────────────────────────────────────────

  const handleSaveDueDate = async (dueDateVal: string | null) => {
    setSavingDueDate(true);
    setShowDatePicker(false);
    const newDateIso = dueDateVal ? new Date(dueDateVal + 'T23:59:59Z').toISOString() : null;
    setLocalDueDate(newDateIso);

    try {
      if (onDueDateChange) {
        await onDueDateChange(task.id, newDateIso);
      } else {
        const updated = await tasksApi.updateTask(task.id, {
          due_date: newDateIso,
          clear_due_date: !newDateIso,
        });
        if (onTaskUpdated) onTaskUpdated(updated);
      }
    } catch {
      setLocalDueDate(task.due_date ?? null);
    } finally {
      setSavingDueDate(false);
    }
  };

  // ── Handlers: Status ─────────────────────────────────────────────────────

  const handleStatusChange = async (newStatus: TaskStatus) => {
    if (newStatus === localStatus || updatingStatus) return;
    setUpdatingStatus(true);
    setLocalStatus(newStatus);
    try {
      if (onStatusChange) {
        await onStatusChange(task.id, newStatus);
      } else {
        const updated = await tasksApi.updateStatus(task.id, newStatus);
        if (onTaskUpdated) onTaskUpdated(updated);
      }
    } catch {
      setLocalStatus(task.status);
    } finally {
      setUpdatingStatus(false);
    }
  };

  const shortSha = task.commit_sha ? task.commit_sha.slice(0, 7) : 'HEAD';

  // Format Due Date check
  const isOverdue = localDueDate && localStatus !== 'resolved' && new Date(localDueDate) < new Date();
  const formattedDueDate = localDueDate
    ? new Date(localDueDate).toLocaleDateString(undefined, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      })
    : null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="relative flex flex-col w-full max-w-lg max-h-[88vh] rounded-xl border border-[#222222] bg-[#111111] shadow-2xl shadow-black/80 animate__animated animate__fadeInUp"
        style={{ animationDuration: '0.2s' }}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-3 border-b border-[#1a1a1a] px-5 py-4">
          <div className="flex items-start gap-2.5 min-w-0">
            <Badge type={task.type} />
            <div className="min-w-0">
              <p className="text-sm text-white font-medium leading-snug line-clamp-2">{task.content}</p>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">
                {task.repo_name && (
                  <span className="rounded bg-white/5 border border-white/10 px-1.5 py-0.5 font-mono text-[10px] text-white/80">
                    {task.repo_name}
                  </span>
                )}
                <span className="rounded bg-[#1a1a1a] border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#888888]">
                  {task.file_path}
                </span>
                <span className="rounded bg-[#1a1a1a] border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#888888]">
                  L{task.line_number}
                </span>
                <span className="font-mono text-[10px] text-[#555555]">{shortSha}</span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {onInjectClick && (
              <button
                type="button"
                onClick={() => {
                  onClose();
                  onInjectClick(task.line_number, task.file_path);
                }}
                className="flex items-center gap-1 text-[10px] font-mono text-[#10b981] hover:text-white bg-[#10b981]/10 hover:bg-[#10b981]/30 border border-[#10b981]/30 rounded px-2 py-1 transition-all cursor-pointer"
                title="Add a new line / task annotation to this file"
              >
                <Plus size={10} />
                <span>Add Line</span>
              </button>
            )}
            <button
              onClick={onClose}
              className="shrink-0 rounded p-1 text-[#666666] hover:text-white transition-colors cursor-pointer"
              aria-label="Close modal"
            >
              <X size={15} />
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          {/* Status Row */}
          <div className="border-b border-[#1a1a1a] px-5 py-3 flex items-center justify-between">
            <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Lifecycle Status</span>
            <div className="flex items-center gap-1">
              {(['open', 'in_progress', 'resolved'] as TaskStatus[]).map((st) => (
                <button
                  key={st}
                  disabled={updatingStatus}
                  onClick={() => handleStatusChange(st)}
                  className={`text-[10px] font-mono px-2 py-0.5 rounded border transition-all cursor-pointer capitalize ${
                    localStatus === st
                      ? st === 'resolved'
                        ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40 font-semibold'
                        : st === 'in_progress'
                        ? 'bg-blue-500/20 text-blue-300 border-blue-500/40 font-semibold'
                        : 'bg-white/15 text-white border-white/30 font-semibold'
                      : 'text-[#666666] border-[#222222] hover:text-white hover:border-[#333333]'
                  }`}
                >
                  {st.replace('_', ' ')}
                </button>
              ))}
            </div>
          </div>

          {/* Assignee section */}
          <div className="border-b border-[#1a1a1a] px-5 py-3">
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Assignee</span>
              <button
                onClick={() => setShowAssigneePicker((v) => !v)}
                className="text-[9px] text-[#888888] hover:text-white border border-[#2a2a2a] hover:border-[#3a3a3a] px-1.5 py-0.5 rounded transition-all cursor-pointer font-mono"
                disabled={assigning}
              >
                {assigning ? 'Assigning…' : 'Change'}
              </button>
            </div>

            {/* Current assignee display */}
            <div className="mt-2 flex items-center gap-2">
              {localAssignee ? (
                <>
                  <img
                    src={`https://github.com/${localAssignee}.png`}
                    alt={localAssignee}
                    className="h-5 w-5 rounded-full border border-[#3a3a3a] shrink-0"
                    onError={(e) => {
                      // Fallback icon on image error
                      (e.target as HTMLElement).style.display = 'none';
                    }}
                  />
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-[#2a2a2a] border border-[#3a3a3a] shrink-0">
                    <User size={11} className="text-[#a0a0a0]" />
                  </div>
                  <span className="text-xs text-white font-medium font-mono">@{localAssignee}</span>
                </>
              ) : (
                <>
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a] shrink-0">
                    <User size={11} className="text-[#444444]" />
                  </div>
                  <span className="text-xs text-[#666666] italic font-mono">Unassigned</span>
                </>
              )}
            </div>

            {/* Assignee picker dropdown */}
            {showAssigneePicker && (
              <div className="mt-2 rounded-lg border border-[#222222] bg-[#0d0d0d] p-2 flex flex-col gap-2">
                <div className="flex gap-1.5">
                  <input
                    type="text"
                    value={assigneeSearch}
                    onChange={(e) => setAssigneeSearch(e.target.value)}
                    placeholder="Enter GitHub username..."
                    className="flex-1 rounded border border-[#2a2a2a] bg-[#111111] px-2.5 py-1 text-xs text-white placeholder-[#555555] focus:outline-none focus:border-white font-mono"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && assigneeSearch.trim()) {
                        e.preventDefault();
                        handleAssign(assigneeSearch.trim());
                      }
                    }}
                  />
                  <button
                    onClick={() => {
                      if (assigneeSearch.trim()) {
                        handleAssign(assigneeSearch.trim());
                      }
                    }}
                    disabled={!assigneeSearch.trim()}
                    className="btn-primary py-1 px-2.5 text-[10px] font-mono shrink-0 disabled:opacity-40"
                  >
                    Assign
                  </button>
                </div>

                {/* Direct quick action: Assign to me */}
                {currentUsername && (
                  <button
                    onClick={() => handleAssign(currentUsername)}
                    className="flex items-center gap-2 px-2.5 py-1.5 text-left text-xs text-[#a0a0a0] hover:text-white hover:bg-[#161616] rounded transition-colors cursor-pointer"
                  >
                    <User size={12} className="text-[#a0a0a0]" />
                    <span>Assign to me (@{currentUsername})</span>
                  </button>
                )}

                {/* Clear assignee */}
                {localAssignee && (
                  <button
                    onClick={() => handleAssign(null)}
                    className="flex items-center gap-2 px-2.5 py-1.5 text-left text-xs text-red-400/80 hover:text-red-400 hover:bg-[#161616] rounded transition-colors cursor-pointer"
                  >
                    <UserMinus size={12} />
                    <span>Clear assignee</span>
                  </button>
                )}

                {/* Filtered Collaborator List */}
                {collaborators.length > 0 && (
                  <div className="border-t border-[#1a1a1a] pt-1.5 max-h-36 overflow-y-auto flex flex-col gap-1">
                    {collaborators
                      .filter((c) =>
                        assigneeSearch.trim()
                          ? c.username.toLowerCase().includes(assigneeSearch.trim().toLowerCase())
                          : true
                      )
                      .map((c) => (
                        <button
                          key={c.id}
                          onClick={() => handleAssign(c.username)}
                          className={[
                            'flex w-full items-center gap-2 px-2.5 py-1.5 rounded text-left transition-colors cursor-pointer',
                            localAssignee === c.username
                              ? 'bg-white/[0.08] text-white font-medium'
                              : 'text-[#a0a0a0] hover:bg-[#161616] hover:text-white',
                          ].join(' ')}
                        >
                          <img
                            src={c.avatar_url}
                            alt={c.username}
                            className="h-4 w-4 rounded-full border border-[#2a2a2a] shrink-0"
                          />
                          <span className="text-xs truncate font-mono">@{c.username}</span>
                          <span className="ml-auto text-[9px] font-mono text-[#666666] capitalize">{c.role}</span>
                        </button>
                      ))}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Due Date section (Son Tarih) */}
          <div className="border-b border-[#1a1a1a] px-5 py-3">
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Due Date (Son Tarih)</span>
              <div className="flex items-center gap-1.5">
                {localDueDate && (
                  <button
                    onClick={() => handleSaveDueDate(null)}
                    className="text-[9px] text-red-400/80 hover:text-red-300 border border-red-500/20 hover:border-red-500/40 px-1.5 py-0.5 rounded transition-all cursor-pointer font-mono"
                    disabled={savingDueDate}
                  >
                    Clear
                  </button>
                )}
                <button
                  onClick={() => setShowDatePicker((v) => !v)}
                  className="text-[9px] text-[#888888] hover:text-white border border-[#2a2a2a] hover:border-[#3a3a3a] px-1.5 py-0.5 rounded transition-all cursor-pointer font-mono"
                  disabled={savingDueDate}
                >
                  {savingDueDate ? 'Saving…' : showDatePicker ? 'Cancel' : localDueDate ? 'Edit' : 'Set Date'}
                </button>
              </div>
            </div>

            {/* Current Due Date display */}
            <div className="mt-2 flex items-center gap-2">
              <div
                className={`flex h-5 w-5 items-center justify-center rounded-full border shrink-0 ${
                  isOverdue
                    ? 'bg-red-500/10 border-red-500/30 text-red-400'
                    : localDueDate
                    ? 'bg-amber-500/10 border-amber-500/30 text-amber-300'
                    : 'bg-[#1a1a1a] border-[#2a2a2a] text-[#444444]'
                }`}
              >
                <Calendar size={11} />
              </div>
              {localDueDate ? (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-white font-mono font-medium">{formattedDueDate}</span>
                  {isOverdue && (
                    <span className="flex items-center gap-1 text-[9px] font-mono text-red-400 bg-red-500/10 border border-red-500/30 px-1.5 py-0.5 rounded">
                      <AlertCircle size={10} />
                      Overdue
                    </span>
                  )}
                </div>
              ) : (
                <span className="text-xs text-[#666666] italic font-mono">No deadline set</span>
              )}
            </div>

            {/* Date picker inline editor */}
            {showDatePicker && (
              <div className="mt-2.5 rounded-lg border border-[#222222] bg-[#0d0d0d] p-3 flex flex-col gap-2">
                <label className="text-[10px] text-[#888888] font-mono">Select Deadline</label>
                <div className="flex gap-2">
                  <input
                    type="date"
                    value={dateInputVal}
                    onChange={(e) => setDateInputVal(e.target.value)}
                    className="flex-1 rounded border border-[#2a2a2a] bg-[#111111] px-2.5 py-1 text-xs text-white [color-scheme:dark] focus:outline-none focus:border-white font-mono"
                  />
                  <button
                    onClick={() => handleSaveDueDate(dateInputVal || null)}
                    disabled={!dateInputVal || savingDueDate}
                    className="btn-primary py-1 px-3 text-xs font-mono disabled:opacity-40"
                  >
                    Save
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Authorship & Maintainer metadata */}
          <div className="border-b border-[#1a1a1a] px-5 py-3 flex flex-col gap-2">
            {task.created_by_username && (
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Created by</span>
                <div className="flex items-center gap-1.5">
                  <img
                    src={task.created_by_avatar_url || `https://github.com/${task.created_by_username}.png`}
                    alt={task.created_by_username}
                    className="h-4 w-4 rounded-full border border-[#2a2a2a] shrink-0"
                  />
                  <span className="text-xs text-[#a0a0a0] font-mono">@{task.created_by_username}</span>
                </div>
              </div>
            )}

            {task.completed_by_username && (
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Completed by</span>
                <div className="flex items-center gap-1.5">
                  <img
                    src={task.completed_by_avatar_url || `https://github.com/${task.completed_by_username}.png`}
                    alt={task.completed_by_username}
                    className="h-4 w-4 rounded-full border border-[#2a2a2a] shrink-0"
                  />
                  <span className="text-xs text-emerald-400 font-mono">@{task.completed_by_username}</span>
                </div>
              </div>
            )}

            {task.maintainer_username && (
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Maintainer</span>
                <span className="text-xs text-[#888888] font-mono">@{task.maintainer_username}</span>
              </div>
            )}
          </div>

          {/* Tabs: Comments & Proposals */}
          <div className="border-b border-[#1a1a1a] px-5 flex gap-4 pt-2">
            <button
              onClick={() => setDetailTab('comments')}
              className={`pb-2.5 text-xs font-medium border-b-2 transition-all flex items-center gap-1.5 cursor-pointer ${
                detailTab === 'comments'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#999999]'
              }`}
            >
              <MessageSquare size={12} />
              <span>Comments ({comments.length})</span>
            </button>
            <button
              onClick={() => setDetailTab('proposals')}
              className={`pb-2.5 text-xs font-medium border-b-2 transition-all flex items-center gap-1.5 cursor-pointer ${
                detailTab === 'proposals'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#999999]'
              }`}
            >
              <Vote size={12} />
              <span>Proposals & Discussions ({proposals.length})</span>
            </button>
          </div>

          {/* Tab Content: Comments */}
          {detailTab === 'comments' && (
            <div className="px-5 py-4 flex flex-col gap-3">
              {commentsLoading ? (
                <p className="text-xs text-[#555555] italic">Loading comments…</p>
              ) : comments.length === 0 ? (
                <p className="text-xs text-[#555555] italic py-2">No comments yet. Write the first comment below!</p>
              ) : (
                <div className="flex flex-col gap-2.5 max-h-56 overflow-y-auto pr-1">
                  {comments.map((c) => (
                    <div
                      key={c.id}
                      className="rounded-lg border border-[#1e1e1e] bg-[#141414] p-3 flex flex-col gap-1.5"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <img
                            src={c.avatar_url || `https://github.com/${c.username}.png`}
                            alt={c.username}
                            className="h-4 w-4 rounded-full border border-[#2a2a2a]"
                          />
                          <span className="text-xs font-medium text-white font-mono">@{c.username}</span>
                          <span className="text-[10px] text-[#555555] font-mono">
                            {new Date(c.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                          </span>
                        </div>
                        {currentUsername === c.username && (
                          <button
                            onClick={() => handleDeleteComment(c.id)}
                            className="text-[#555555] hover:text-red-400 p-0.5 rounded transition-colors cursor-pointer"
                            title="Delete comment"
                          >
                            <Trash2 size={11} />
                          </button>
                        )}
                      </div>
                      <p className="text-xs text-[#cccccc] whitespace-pre-wrap leading-relaxed">{c.content}</p>
                    </div>
                  ))}
                </div>
              )}

              {/* Add comment input */}
              <div className="pt-2 border-t border-[#1a1a1a] flex gap-2">
                <input
                  type="text"
                  value={newComment}
                  onChange={(e) => setNewComment(e.target.value)}
                  placeholder="Write a comment..."
                  className="flex-1 rounded-lg border border-[#2a2a2a] bg-[#141414] px-3 py-1.5 text-xs text-white placeholder-[#555555] focus:outline-none focus:border-white"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      handleAddComment();
                    }
                  }}
                />
                <button
                  onClick={handleAddComment}
                  disabled={!newComment.trim() || submitting}
                  className="btn-primary px-3 py-1.5 text-xs flex items-center gap-1 shrink-0 disabled:opacity-40"
                >
                  <Send size={11} />
                  <span>Send</span>
                </button>
              </div>
            </div>
          )}

          {/* Tab Content: Proposals */}
          {detailTab === 'proposals' && (
            <div className="px-5 py-4 flex flex-col gap-3">
              {proposalsLoading ? (
                <p className="text-xs text-[#555555] italic">Loading proposals…</p>
              ) : proposals.length === 0 ? (
                <p className="text-xs text-[#555555] italic py-2">No discussion proposals submitted yet.</p>
              ) : (
                <div className="flex flex-col gap-2.5 max-h-56 overflow-y-auto pr-1">
                  {proposals.map((p) => (
                    <div
                      key={p.id}
                      className="rounded-lg border border-[#1e1e1e] bg-[#141414] p-3 flex flex-col gap-2"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <img
                            src={p.avatar_url || `https://github.com/${p.username}.png`}
                            alt={p.username}
                            className="h-4 w-4 rounded-full border border-[#2a2a2a]"
                          />
                          <span className="text-xs font-semibold text-white">{p.title}</span>
                          <span
                            className={`text-[9px] font-mono px-1.5 py-0.5 rounded capitalize ${
                              p.status === 'approved'
                                ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                                : p.status === 'rejected'
                                ? 'bg-red-500/20 text-red-300 border border-red-500/30'
                                : 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                            }`}
                          >
                            {p.status}
                          </span>
                        </div>
                        {currentUsername === p.username && (
                          <button
                            onClick={() => handleDeleteProposal(p.id)}
                            className="text-[#555555] hover:text-red-400 p-0.5 rounded transition-colors cursor-pointer"
                          >
                            <Trash2 size={11} />
                          </button>
                        )}
                      </div>
                      {p.content && <p className="text-xs text-[#a0a0a0] leading-relaxed">{p.content}</p>}
                      <div className="flex items-center gap-2 pt-1 border-t border-[#1e1e1e]">
                        <span className="text-[10px] text-[#666666] font-mono">Vote:</span>
                        <button
                          onClick={() => handleVoteProposal(p.id, 'approved')}
                          className="flex items-center gap-1 text-[10px] text-emerald-400 hover:text-emerald-300 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20 cursor-pointer"
                        >
                          <CheckCircle2 size={10} />
                          <span>Approve</span>
                        </button>
                        <button
                          onClick={() => handleVoteProposal(p.id, 'rejected')}
                          className="flex items-center gap-1 text-[10px] text-red-400 hover:text-red-300 bg-red-500/10 px-2 py-0.5 rounded border border-red-500/20 cursor-pointer"
                        >
                          <XCircle size={10} />
                          <span>Reject</span>
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Add proposal form */}
              <div className="pt-2 border-t border-[#1a1a1a] flex flex-col gap-2">
                <input
                  type="text"
                  value={newPropTitle}
                  onChange={(e) => setNewPropTitle(e.target.value)}
                  placeholder="Proposal title..."
                  className="rounded-lg border border-[#2a2a2a] bg-[#141414] px-3 py-1.5 text-xs text-white placeholder-[#555555] focus:outline-none focus:border-white font-medium"
                />
                <textarea
                  value={newPropContent}
                  onChange={(e) => setNewPropContent(e.target.value)}
                  placeholder="Details / implementation strategy..."
                  rows={2}
                  className="rounded-lg border border-[#2a2a2a] bg-[#141414] px-3 py-1.5 text-xs text-white placeholder-[#555555] focus:outline-none focus:border-white resize-none"
                />
                <button
                  onClick={handleAddProposal}
                  disabled={!newPropTitle.trim() || submittingProp}
                  className="btn-primary self-end px-3 py-1 text-xs flex items-center gap-1 disabled:opacity-40"
                >
                  <Plus size={11} />
                  <span>Propose</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

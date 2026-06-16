/**
 * TaskBoard — Kanban / List view for repository tasks.
 *
 * Kanban mode: three columns (open → in_progress → resolved) with
 *   @hello-pangea/dnd drag-and-drop. Dropping a card into a column
 *   triggers an optimistic status update.
 *
 * List mode: flat file-grouped list sorted by file_path then line_number.
 *
 * Header contains: title, list/kanban toggle, and "Inject TODO" button.
 *
 * Extended with:
 * - Assignee picker: each task card shows assignee avatar/username;
 *   clicking opens a collaborator picker overlay.
 * - Comments: task detail modal shows comments with add/delete.
 */

import { useEffect, useState, useCallback } from 'react';
import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd';
import {
  LayoutList,
  Columns,
  Plus,
  GitPullRequest,
  User,
  X,
  MessageSquare,
  Send,
  Trash2,
  UserMinus,
} from 'lucide-react';
import { useTaskStore } from '../store/taskStore';
import { commentsApi, reposApi } from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { Task, TaskStatus, PullRequest, Collaborator, Comment } from '../types';
import Badge from './ui/Badge';
import Spinner from './ui/Spinner';

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Format an ISO timestamp as a relative time string */
function timeAgo(isoString: string): string {
  const date = new Date(isoString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (seconds < 60) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return `${Math.floor(days / 7)}w ago`;
}

// ── Props ────────────────────────────────────────────────────────────────────

interface TaskBoardProps {
  repoId: number;
  repoOwner: string;
  repoName: string;
  /** Called when user clicks "Inject TODO" — optionally passes a pre-filled line */
  onInjectClick: (lineNumber?: number) => void;
  /** Called when user clicks a task card to jump to the code */
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  /** List of active pull requests for linking */
  pulls: PullRequest[];
  /** Handler to pair a task to a PR */
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
}

// ── Constants ────────────────────────────────────────────────────────────────

const COLUMNS: { id: TaskStatus; label: string }[] = [
  { id: 'open',        label: 'Open' },
  { id: 'in_progress', label: 'In Progress' },
  { id: 'resolved',    label: 'Resolved' },
];

// Helper to extract PR number from URL
function getPrNumber(url?: string): string | null {
  if (!url) return null;
  const match = url.match(/\/pull\/(\d+)/);
  return match ? `#${match[1]}` : 'PR';
}

// ── Task Detail Modal (comments + assignee) ───────────────────────────────────

function TaskDetailModal({
  task,
  collaborators,
  currentUsername,
  onClose,
  onAssign,
}: {
  task: Task;
  collaborators: Collaborator[];
  currentUsername: string;
  onClose: () => void;
  onAssign: (taskId: string, username: string | null) => Promise<void>;
}) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(true);
  const [newComment, setNewComment] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [showAssigneePicker, setShowAssigneePicker] = useState(false);
  const [assigning, setAssigning] = useState(false);

  // Local assignee state (for optimistic UI)
  const [localAssignee, setLocalAssignee] = useState<string | null>(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (task as any).assignee_username ?? null
  );

  // Load comments on mount
  useEffect(() => {
    setCommentsLoading(true);
    commentsApi
      .list(task.id)
      .then(setComments)
      .catch(() => setComments([]))
      .finally(() => setCommentsLoading(false));
  }, [task.id]);

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
      // Refetch on failure
      commentsApi.list(task.id).then(setComments).catch(() => {});
    }
  };

  const handleAssign = async (username: string | null) => {
    setAssigning(true);
    setLocalAssignee(username);
    setShowAssigneePicker(false);
    try {
      await onAssign(task.id, username);
    } catch {
      setLocalAssignee(null);
    } finally {
      setAssigning(false);
    }
  };

  const displayPath = task.file_path.split('/').slice(-2).join('/');
  const shortSha = task.commit_sha.slice(0, 7);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        className="relative flex flex-col w-full max-w-lg max-h-[85vh] rounded border border-[#222222] bg-[#111111] shadow-2xl shadow-black/80 animate__animated animate__fadeInUp"
        style={{ animationDuration: '0.2s' }}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-3 border-b border-[#1a1a1a] px-5 py-4">
          <div className="flex items-start gap-2 min-w-0">
            <Badge type={task.type} />
            <div className="min-w-0">
              <p className="text-sm text-white font-medium leading-snug line-clamp-2">{task.content}</p>
              <div className="mt-1 flex items-center gap-2">
                <span className="font-mono text-[10px] text-[#666666]">{displayPath}</span>
                <span className="font-mono text-[10px] text-[#666666]">L{task.line_number}</span>
                <span className="font-mono text-[10px] text-[#666666]">{shortSha}</span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="shrink-0 rounded p-1 text-[#666666] hover:text-white transition-colors cursor-pointer"
          >
            <X size={14} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {/* Assignee section */}
          <div className="border-b border-[#1a1a1a] px-5 py-3">
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Assignee</span>
              <button
                onClick={() => setShowAssigneePicker((v) => !v)}
                className="text-[9px] text-[#666666] hover:text-white border border-[#2a2a2a] hover:border-[#3a3a3a] px-1.5 py-0.5 rounded transition-all cursor-pointer font-mono"
                disabled={assigning}
              >
                {assigning ? 'Assigning…' : 'Change'}
              </button>
            </div>

            {/* Current assignee */}
            <div className="mt-2 flex items-center gap-2">
              {localAssignee ? (
                <>
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-[#2a2a2a] border border-[#3a3a3a] shrink-0">
                    <User size={11} className="text-[#a0a0a0]" />
                  </div>
                  <span className="text-[11px] text-white font-medium">{localAssignee}</span>
                </>
              ) : (
                <>
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a] shrink-0">
                    <User size={11} className="text-[#444444]" />
                  </div>
                  <span className="text-[11px] text-[#666666] italic">Unassigned</span>
                </>
              )}
            </div>

            {/* Assignee picker dropdown */}
            {showAssigneePicker && (
              <div className="mt-2 rounded border border-[#222222] bg-[#0d0d0d] overflow-hidden">
                {localAssignee && (
                  <button
                    onClick={() => handleAssign(null)}
                    className="flex w-full items-center gap-2 px-3 py-2 text-left text-[11px] text-[#a0a0a0] hover:bg-[#161616] transition-colors cursor-pointer border-b border-[#1a1a1a]"
                  >
                    <UserMinus size={12} className="text-[#666666]" />
                    Clear assignee
                  </button>
                )}
                {collaborators.map((c) => (
                  <button
                    key={c.id}
                    onClick={() => handleAssign(c.username)}
                    className={[
                      'flex w-full items-center gap-2 px-3 py-2 text-left transition-colors cursor-pointer',
                      localAssignee === c.username
                        ? 'bg-white/[0.04] text-white'
                        : 'text-[#a0a0a0] hover:bg-[#161616]',
                    ].join(' ')}
                  >
                    <img
                      src={c.avatar_url}
                      alt={c.username}
                      className="h-5 w-5 rounded-full border border-[#2a2a2a] shrink-0"
                    />
                    <span className="text-[11px] truncate">{c.username}</span>
                    <span className="ml-auto text-[9px] font-mono text-[#666666] capitalize">{c.role}</span>
                  </button>
                ))}
                {collaborators.length === 0 && (
                  <p className="px-3 py-3 text-[10px] text-[#666666]">No collaborators found</p>
                )}
              </div>
            )}
          </div>

          {/* Comments section */}
          <div className="px-5 py-4">
            <div className="flex items-center gap-2 mb-3">
              <MessageSquare size={12} className="text-[#666666]" />
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Comments</span>
              <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[9px] text-[#666666]">
                {comments.length}
              </span>
            </div>

            {commentsLoading ? (
              <div className="flex justify-center py-4">
                <Spinner size={18} />
              </div>
            ) : (
              <div className="flex flex-col gap-3">
                {comments.length === 0 && (
                  <p className="text-[11px] text-[#666666] py-2 text-center">No comments yet</p>
                )}
                {comments.map((comment) => (
                  <div
                    key={comment.id}
                    className="rounded border border-[#1a1a1a] bg-[#0d0d0d] p-3"
                  >
                    <div className="flex items-center justify-between gap-2 mb-2">
                      <div className="flex items-center gap-2">
                        {comment.avatar_url ? (
                          <img
                            src={comment.avatar_url}
                            alt={comment.username}
                            className="h-5 w-5 rounded-full border border-[#2a2a2a] shrink-0"
                          />
                        ) : (
                          <div className="flex h-5 w-5 items-center justify-center rounded-full bg-[#2a2a2a]">
                            <User size={10} className="text-[#666666]" />
                          </div>
                        )}
                        <span className="text-[11px] font-medium text-white">{comment.username}</span>
                        <span className="text-[9px] text-[#666666] font-mono">{timeAgo(comment.created_at)}</span>
                      </div>
                      {comment.username === currentUsername && (
                        <button
                          onClick={() => handleDeleteComment(comment.id)}
                          className="text-[#444444] hover:text-red-400 transition-colors cursor-pointer"
                          title="Delete comment"
                        >
                          <Trash2 size={11} />
                        </button>
                      )}
                    </div>
                    <p className="text-[11px] text-[#a0a0a0] leading-4 whitespace-pre-wrap">{comment.content}</p>
                  </div>
                ))}
              </div>
            )}

            {/* Add comment */}
            <div className="mt-3 flex flex-col gap-2">
              <textarea
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                placeholder="Add a comment…"
                rows={2}
                className="w-full resize-none rounded border border-[#2a2a2a] bg-[#0d0d0d] px-3 py-2 text-[11px] text-white placeholder-[#444444] focus:outline-none focus:border-[#3a3a3a] transition-colors"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                    e.preventDefault();
                    handleAddComment();
                  }
                }}
              />
              <div className="flex justify-end">
                <button
                  onClick={handleAddComment}
                  disabled={!newComment.trim() || submitting}
                  className="flex items-center gap-1.5 rounded border border-[#2a2a2a] bg-white/[0.04] px-3 py-1.5 text-[10px] font-mono text-[#a0a0a0] hover:text-white hover:border-[#3a3a3a] transition-all cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {submitting ? <Spinner size={11} /> : <Send size={10} />}
                  Comment
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Task Card ────────────────────────────────────────────────────────────────

function TaskCard({
  task,
  index,
  onInjectClick,
  onTaskClick,
  pulls,
  onLinkTaskToPR,
  onOpenDetail,
}: {
  task: Task;
  index: number;
  onInjectClick: (lineNumber?: number) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onOpenDetail: (task: Task) => void;
}) {
  const [showLinkSelect, setShowLinkSelect] = useState(false);
  const displayPath = task.file_path.split('/').slice(-2).join('/');
  const shortSha    = task.commit_sha.slice(0, 7);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const assigneeUsername: string | null = (task as any).assignee_username ?? null;

  return (
    <Draggable draggableId={task.id} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          className={[
            'card flex flex-col gap-2 p-3 cursor-grab active:cursor-grabbing',
            'transition-colors duration-150',
            snapshot.isDragging
              ? 'border-[#3a3a3a] bg-[#242424]'
              : 'hover:border-[#3a3a3a]',
          ].join(' ')}
          onClick={() => onTaskClick ? onTaskClick(task.file_path, task.line_number) : onInjectClick(task.line_number)}
          title="Click to view code at this line"
        >
          {/* Top row: type badge + file path */}
          <div className="flex min-w-0 items-center justify-between gap-2">
            <div className="flex items-center gap-2 min-w-0">
              <Badge type={task.type} />
              <span
                className="min-w-0 truncate font-mono text-[10px] text-[#666666]"
                style={{ fontFamily: "'JetBrains Mono', monospace" }}
                title={task.file_path}
              >
                {displayPath}
              </span>
            </div>
            <span
              className="font-mono text-[10px] text-[#666666] shrink-0"
              style={{ fontFamily: "'JetBrains Mono', monospace" }}
            >
              L{task.line_number}
            </span>
          </div>

          {/* Content */}
          <p className="line-clamp-3 text-xs text-[#a0a0a0]">
            {task.content}
          </p>

          {/* Assignee row */}
          <div
            className="flex items-center gap-1.5"
            onClick={(e) => { e.stopPropagation(); onOpenDetail(task); }}
            title="Click to manage assignee & comments"
          >
            <div className="flex h-4 w-4 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a] shrink-0">
              <User size={9} className={assigneeUsername ? 'text-[#a0a0a0]' : 'text-[#444444]'} />
            </div>
            <span className={`text-[9px] font-mono truncate ${assigneeUsername ? 'text-[#a0a0a0]' : 'text-[#444444] italic'}`}>
              {assigneeUsername ?? 'Unassigned'}
            </span>
          </div>

          {/* Footer: commit SHA + PR Link */}
          <div className="border-t border-[#2a2a2a] pt-2 flex items-center justify-between">
            <span
              className="font-mono text-[9px] text-[#666666]"
              style={{ fontFamily: "'JetBrains Mono', monospace" }}
            >
              {shortSha}
            </span>

            <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
              {task.pr_url ? (
                <a
                  href={task.pr_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-[9px] text-purple-400 bg-purple-400/5 border border-purple-400/10 px-1.5 py-0.5 rounded font-mono hover:bg-purple-400/10 transition-colors"
                >
                  <GitPullRequest size={9} />
                  {getPrNumber(task.pr_url)}
                </a>
              ) : (
                showLinkSelect ? (
                  <select
                    className="bg-[#111111] border border-[#3a3a3a] rounded text-[9px] text-white px-1 py-0.5 focus:outline-none max-w-[120px]"
                    defaultValue=""
                    onChange={async (e) => {
                      const val = e.target.value;
                      if (val && val !== 'cancel') {
                        await onLinkTaskToPR(task.id, val);
                      }
                      setShowLinkSelect(false);
                    }}
                    onBlur={() => setShowLinkSelect(false)}
                  >
                    <option value="" disabled>Link PR...</option>
                    {pulls.map(pr => (
                      <option key={pr.id} value={pr.html_url}>
                        #{pr.number} - {pr.title}
                      </option>
                    ))}
                    <option value="cancel">Cancel</option>
                  </select>
                ) : (
                  <button
                    onClick={() => setShowLinkSelect(true)}
                    className="text-[9px] text-[#666666] hover:text-white border border-[#2a2a2a] hover:border-[#3a3a3a] px-1.5 py-0.5 rounded transition-all font-mono"
                    title="Pair with Pull Request"
                  >
                    Link PR
                  </button>
                )
              )}
            </div>
          </div>
        </div>
      )}
    </Draggable>
  );
}

// ── Kanban Column ────────────────────────────────────────────────────────────

function KanbanColumn({
  columnId,
  label,
  tasks,
  onInjectClick,
  onTaskClick,
  pulls,
  onLinkTaskToPR,
  onOpenDetail,
}: {
  columnId: TaskStatus;
  label: string;
  tasks: Task[];
  onInjectClick: (lineNumber?: number) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onOpenDetail: (task: Task) => void;
}) {
  return (
    <div className="flex min-w-[220px] flex-1 flex-col gap-3">
      {/* Column header */}
      <div className="flex items-center justify-between">
        <span className="section-label">{label}</span>
        <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666]">
          {tasks.length}
        </span>
      </div>

      {/* Drop zone */}
      <Droppable droppableId={columnId}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={[
              'flex min-h-[120px] flex-col gap-2 rounded border p-2 transition-colors duration-150',
              snapshot.isDraggingOver
                ? 'border-[#3a3a3a] bg-[#1a1a1a]'
                : 'border-[#2a2a2a] bg-[#111111]',
            ].join(' ')}
          >
            {tasks.map((task, i) => (
              <TaskCard
                key={task.id}
                task={task}
                index={i}
                onInjectClick={onInjectClick}
                onTaskClick={onTaskClick}
                pulls={pulls}
                onLinkTaskToPR={onLinkTaskToPR}
                onOpenDetail={onOpenDetail}
              />
            ))}
            {provided.placeholder}

            {/* Empty column hint */}
            {tasks.length === 0 && !snapshot.isDraggingOver && (
              <div className="flex flex-1 items-center justify-center py-4">
                <span className="text-[10px] text-[#3a3a3a]">drop here</span>
              </div>
            )}
          </div>
        )}
      </Droppable>
    </div>
  );
}

// ── Task List Item ───────────────────────────────────────────────────────────

function TaskListItem({
  task,
  onInjectClick,
  onTaskClick,
  pulls,
  onLinkTaskToPR,
  onOpenDetail,
}: {
  task: Task;
  onInjectClick: (lineNumber?: number) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onOpenDetail: (task: Task) => void;
}) {
  const [showLinkSelect, setShowLinkSelect] = useState(false);
  const shortSha = task.commit_sha.slice(0, 7);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const assigneeUsername: string | null = (task as any).assignee_username ?? null;

  return (
    <div
      className="card flex items-center justify-between gap-3 px-3 py-2.5 transition-colors duration-150 hover:border-[#3a3a3a] cursor-pointer"
      onClick={() => onTaskClick ? onTaskClick(task.file_path, task.line_number) : onInjectClick(task.line_number)}
    >
      <div className="flex items-center gap-3 min-w-0 flex-1">
        <Badge type={task.type} />
        <Badge type={task.status} className="ml-0" />
        <span
          className="shrink-0 font-mono text-[10px] text-[#666666]"
          style={{ fontFamily: "'JetBrains Mono', monospace" }}
        >
          L{task.line_number}
        </span>
        <p className="flex-1 truncate text-xs text-[#a0a0a0]">
          {task.content}
        </p>
      </div>

      <div className="flex items-center gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
        {/* Assignee chip */}
        <button
          onClick={() => onOpenDetail(task)}
          className="flex items-center gap-1 text-[9px] font-mono border border-[#1a1a1a] hover:border-[#2a2a2a] rounded px-1.5 py-0.5 transition-all cursor-pointer"
          title="Manage assignee & comments"
        >
          <User size={9} className={assigneeUsername ? 'text-[#a0a0a0]' : 'text-[#444444]'} />
          <span className={assigneeUsername ? 'text-[#a0a0a0]' : 'text-[#444444] italic'}>
            {assigneeUsername ?? 'None'}
          </span>
        </button>

        {/* PR Linkage */}
        {task.pr_url ? (
          <a
            href={task.pr_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-[9px] text-purple-400 bg-purple-400/5 border border-purple-400/10 px-1.5 py-0.5 rounded font-mono hover:bg-purple-400/10 transition-colors"
          >
            <GitPullRequest size={9} />
            {getPrNumber(task.pr_url)}
          </a>
        ) : (
          showLinkSelect ? (
            <select
              className="bg-[#111111] border border-[#3a3a3a] rounded text-[9px] text-white px-1 py-0.5 focus:outline-none max-w-[120px]"
              defaultValue=""
              onChange={async (e) => {
                const val = e.target.value;
                if (val && val !== 'cancel') {
                  await onLinkTaskToPR(task.id, val);
                }
                setShowLinkSelect(false);
              }}
              onBlur={() => setShowLinkSelect(false)}
            >
              <option value="" disabled>Link PR...</option>
              {pulls.map(pr => (
                <option key={pr.id} value={pr.html_url}>
                  #{pr.number} - {pr.title}
                </option>
              ))}
              <option value="cancel">Cancel</option>
            </select>
          ) : (
            <button
              onClick={() => setShowLinkSelect(true)}
              className="text-[9px] text-[#666666] hover:text-white border border-[#2a2a2a] hover:border-[#3a3a3a] px-1.5 py-0.5 rounded transition-all font-mono"
            >
              Link PR
            </button>
          )
        )}

        <span
          className="font-mono text-[9px] text-[#666666]"
          style={{ fontFamily: "'JetBrains Mono', monospace" }}
        >
          {shortSha}
        </span>
      </div>
    </div>
  );
}

// ── List View ────────────────────────────────────────────────────────────────

function ListView({
  tasks,
  onInjectClick,
  onTaskClick,
  pulls,
  onLinkTaskToPR,
  onOpenDetail,
}: {
  tasks: Task[];
  onInjectClick: (lineNumber?: number) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onOpenDetail: (task: Task) => void;
}) {
  // Group tasks by file_path, sorted by file then line number
  const sorted = [...tasks].sort((a, b) => {
    const fp = a.file_path.localeCompare(b.file_path);
    if (fp !== 0) return fp;
    return a.line_number - b.line_number;
  });

  const groups = sorted.reduce<Record<string, Task[]>>((acc, task) => {
    if (!acc[task.file_path]) acc[task.file_path] = [];
    acc[task.file_path].push(task);
    return acc;
  }, {});

  return (
    <div className="flex flex-col gap-4">
      {Object.entries(groups).map(([filePath, fileTasks]) => (
        <div key={filePath} className="flex flex-col gap-1">
          {/* File group header */}
          <div className="flex items-center gap-2 pb-1">
            <span
              className="truncate font-mono text-[11px] text-[#a0a0a0]"
              style={{ fontFamily: "'JetBrains Mono', monospace" }}
            >
              {filePath}
            </span>
            <span className="shrink-0 font-mono text-[10px] text-[#666666]">
              ({fileTasks.length})
            </span>
          </div>

          {fileTasks.map((task) => (
            <TaskListItem
              key={task.id}
              task={task}
              onInjectClick={onInjectClick}
              onTaskClick={onTaskClick}
              pulls={pulls}
              onLinkTaskToPR={onLinkTaskToPR}
              onOpenDetail={onOpenDetail}
            />
          ))}
        </div>
      ))}

      {tasks.length === 0 && (
        <p className="py-8 text-center text-xs text-[#666666]">
          No tasks found for this repository.
        </p>
      )}
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

export default function TaskBoard({
  repoId,
  repoOwner,
  repoName,
  onInjectClick,
  onTaskClick,
  pulls,
  onLinkTaskToPR,
}: TaskBoardProps) {
  const { tasks, isLoading, error, fetchTasks, updateTaskStatus, updateTaskAssignee } =
    useTaskStore();

  const currentUser = useAuthStore((s) => s.user);
  const [viewMode, setViewMode] = useState<'kanban' | 'list'>('kanban');

  // Collaborators for assignee picker
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);

  // Task detail modal state
  const [detailTask, setDetailTask] = useState<Task | null>(null);

  // Fetch collaborators for assignee picker
  useEffect(() => {
    if (!repoOwner || !repoName) return;
    reposApi
      .listCollaborators(repoOwner, repoName)
      .then((list) => setCollaborators(list || []))
      .catch(() => {});
  }, [repoOwner, repoName]);

  // Fetch tasks when repoId changes
  useEffect(() => {
    if (repoId > 0) {
      fetchTasks(repoId);
    }
  }, [repoId, fetchTasks]);

  // ── Drag handler ────────────────────────────────────────────────────────

  const handleDragEnd = (result: DropResult) => {
    const { destination, source, draggableId } = result;

    // Dropped outside a column or in the same position
    if (!destination) return;
    if (
      destination.droppableId === source.droppableId &&
      destination.index === source.index
    ) return;

    const newStatus = destination.droppableId as TaskStatus;
    updateTaskStatus(draggableId, newStatus);
  };

  // ── Assign handler ───────────────────────────────────────────────────────

  const handleAssign = useCallback(async (taskId: string, username: string | null) => {
    try {
      await updateTaskAssignee(taskId, username);
    } catch {
      // Silently fail
    }
  }, [updateTaskAssignee]);

  // ── Partition tasks into columns ────────────────────────────────────────

  const byStatus = (status: TaskStatus) =>
    (tasks || []).filter((t) => t.status === status);

  // ── Render states ───────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Spinner size={24} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 p-4 text-center">
        <p className="text-xs text-[#a0a0a0]">{error}</p>
        <button
          className="btn-secondary text-xs"
          onClick={() => fetchTasks(repoId)}
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <>
      <div className="flex flex-col overflow-hidden" style={{ height: '100%' }}>
        {/* Board header */}
        <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] px-4 py-3">
          <div className="flex items-center gap-3">
            <span className="text-xs font-medium text-white">Tasks</span>
            <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666]">
              {tasks.length}
            </span>
          </div>

          <div className="flex items-center gap-2">
            {/* View toggle */}
            <div className="flex rounded border border-[#2a2a2a]">
              <button
                className={`flex items-center gap-1 px-2 py-1 text-[10px] transition-colors duration-150 ${
                  viewMode === 'list'
                    ? 'bg-[#1a1a1a] text-white'
                    : 'text-[#666666] hover:text-white'
                }`}
                onClick={() => setViewMode('list')}
                title="List view"
              >
                <LayoutList size={12} />
              </button>
              <button
                className={`flex items-center gap-1 px-2 py-1 text-[10px] transition-colors duration-150 ${
                  viewMode === 'kanban'
                    ? 'bg-[#1a1a1a] text-white'
                    : 'text-[#666666] hover:text-white'
                }`}
                onClick={() => setViewMode('kanban')}
                title="Kanban view"
              >
                <Columns size={12} />
              </button>
            </div>

            {/* Inject TODO */}
            <button
              className="btn-secondary flex items-center gap-1 px-2 py-1 text-[10px]"
              onClick={() => onInjectClick()}
            >
              <Plus size={11} />
              Inject TODO
            </button>
          </div>
        </div>

        {/* Board body */}
        <div className="flex-1 overflow-auto p-4">
          {viewMode === 'kanban' ? (
            <DragDropContext onDragEnd={handleDragEnd}>
              <div className="flex gap-3" style={{ minWidth: 'max-content' }}>
                {COLUMNS.map((col) => (
                  <KanbanColumn
                    key={col.id}
                    columnId={col.id}
                    label={col.label}
                    tasks={byStatus(col.id)}
                    onInjectClick={onInjectClick}
                    onTaskClick={onTaskClick}
                    pulls={pulls}
                    onLinkTaskToPR={onLinkTaskToPR}
                    onOpenDetail={setDetailTask}
                  />
                ))}
              </div>
            </DragDropContext>
          ) : (
            <ListView
              tasks={tasks}
              onInjectClick={onInjectClick}
              onTaskClick={onTaskClick}
              pulls={pulls}
              onLinkTaskToPR={onLinkTaskToPR}
              onOpenDetail={setDetailTask}
            />
          )}
        </div>
      </div>

      {/* Task detail modal (assignee + comments) */}
      {detailTask && (
        <TaskDetailModal
          task={tasks.find((t) => t.id === detailTask.id) || detailTask}
          collaborators={collaborators}
          currentUsername={currentUser?.username ?? ''}
          onClose={() => setDetailTask(null)}
          onAssign={handleAssign}
        />
      )}
    </>
  );
}

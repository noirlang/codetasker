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
  AlertCircle,
  Calendar,
  MessageSquare,
} from 'lucide-react';
import { useTaskStore } from '../store/taskStore';
import { reposApi } from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { Task, TaskStatus, PullRequest, Collaborator, Issue } from '../types';
import Badge from './ui/Badge';
import Spinner from './ui/Spinner';
import TaskDetailModal from './TaskDetailModal';

// ── Props ────────────────────────────────────────────────────────────────────

interface TaskBoardProps {
  repoId: number;
  repoOwner: string;
  repoName: string;
  /** Called when user clicks "Inject TODO" — optionally passes a pre-filled line and file path */
  onInjectClick: (lineNumber?: number, filePath?: string) => void;
  /** Called when user clicks a task card to jump to the code */
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  /** List of active pull requests for linking */
  pulls: PullRequest[];
  /** List of active issues for linking */
  issues: Issue[];
  /** Handler to pair a task to a PR */
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  /** Handler to pair a task to an Issue */
  onLinkTaskToIssue: (taskId: string, issueUrl: string) => Promise<void>;
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

// Helper to extract Issue number from URL
function getIssueNumber(url?: string): string | null {
  if (!url) return null;
  const match = url.match(/\/issues\/(\d+)/);
  return match ? `#${match[1]}` : 'Issue';
}


// ── Task Card ────────────────────────────────────────────────────────────────

function TaskCard({
  task,
  index,
  onInjectClick,
  onTaskClick,
  pulls,
  issues,
  onLinkTaskToPR,
  onLinkTaskToIssue,
  onOpenDetail,
}: {
  task: Task;
  index: number;
  onInjectClick: (lineNumber?: number, filePath?: string) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  issues: Issue[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onLinkTaskToIssue: (taskId: string, issueUrl: string) => Promise<void>;
  onOpenDetail: (task: Task) => void;
}) {
  const [showLinkSelect, setShowLinkSelect] = useState(false);
  const [showIssueSelect, setShowIssueSelect] = useState(false);
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
          onClick={() => {
            if (onTaskClick) onTaskClick(task.file_path, task.line_number);
            onOpenDetail(task);
          }}
          title="Click to view code, comments & details"
        >
          {/* Top row: type badge + file path & line */}
          <div className="flex min-w-0 items-center justify-between gap-2">
            <div className="flex items-center gap-2 min-w-0">
              <Badge type={task.type} />
              <span
                className="min-w-0 truncate font-mono text-[11px] text-[#888888] hover:text-[#cccccc] transition-colors"
                style={{ fontFamily: "'JetBrains Mono', monospace" }}
                title={task.file_path}
              >
                {displayPath}
              </span>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              <span
                className="rounded bg-[#1a1a1a] border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#888888]"
                style={{ fontFamily: "'JetBrains Mono', monospace" }}
              >
                L{task.line_number}
              </span>
              {task.status === 'open' && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (onTaskClick) onTaskClick(task.file_path, task.line_number);
                    onInjectClick(task.line_number, task.file_path);
                  }}
                  className="flex items-center gap-0.5 text-[9px] font-mono text-[#10b981] hover:text-white bg-[#10b981]/10 hover:bg-[#10b981]/30 border border-[#10b981]/30 rounded px-1.5 py-0.5 transition-all cursor-pointer"
                  title="Add a new line / annotation to this file"
                >
                  <Plus size={9} />
                  <span>Line</span>
                </button>
              )}
            </div>
          </div>

          {/* Content */}
          <p className="line-clamp-3 text-xs text-[#a0a0a0]">
            {task.content}
          </p>

          {/* Assignee + Creator row */}
          <div
            className="flex items-center justify-between gap-1.5"
            onClick={(e) => { e.stopPropagation(); onOpenDetail(task); }}
            title="Click to manage assignee & comments"
          >
            {/* Assignee */}
            <div className="flex items-center gap-1.5 min-w-0">
              <div className="flex h-4 w-4 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a] shrink-0">
                <User size={9} className={assigneeUsername ? 'text-[#a0a0a0]' : 'text-[#444444]'} />
              </div>
              <span className={`text-[9px] font-mono truncate ${assigneeUsername ? 'text-[#a0a0a0]' : 'text-[#444444] italic'}`}>
                {assigneeUsername ?? 'Unassigned'}
              </span>
            </div>

            {/* Due date badge if set */}
            {task.due_date && (
              <div
                className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[9px] font-mono shrink-0 ${
                  new Date(task.due_date) < new Date() && task.status !== 'resolved'
                    ? 'border-red-500/40 bg-red-500/10 text-red-400 font-semibold'
                    : 'border-[#2a2a2a] bg-[#1a1a1a] text-[#888888]'
                }`}
                title={new Date(task.due_date) < new Date() && task.status !== 'resolved' ? 'Overdue!' : 'Due Date'}
              >
                <Calendar size={9} className={new Date(task.due_date) < new Date() && task.status !== 'resolved' ? 'text-red-400' : 'text-[#666666]'} />
                <span>
                  {new Date(task.due_date).toLocaleDateString(undefined, {
                    month: 'short',
                    day: 'numeric',
                  })}
                </span>
              </div>
            )}

            {/* Creator avatar (right side) */}
            {task.created_by_username && (
              <div className="flex items-center gap-1 shrink-0" title={`Created by @${task.created_by_username}`}>
                {task.created_by_avatar_url ? (
                  <img
                    src={task.created_by_avatar_url}
                    alt={task.created_by_username}
                    className="h-4 w-4 rounded-full border border-[#2a2a2a] opacity-70"
                  />
                ) : (
                  <div className="flex h-4 w-4 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a]">
                    <User size={8} className="text-[#444444]" />
                  </div>
                )}
              </div>
            )}

            {/* Completed badge */}
            {task.completed_by_username && (
              <span className="text-[8px] font-mono text-emerald-400 bg-emerald-400/5 border border-emerald-400/10 px-1 py-0.5 rounded shrink-0">
                ✓ done
              </span>
            )}
          </div>


          {/* Footer: commit SHA + PR / Issue Link */}
          <div className="border-t border-[#2a2a2a] pt-2 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span
                className="font-mono text-[9px] text-[#666666]"
                style={{ fontFamily: "'JetBrains Mono', monospace" }}
              >
                {shortSha}
              </span>

              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onOpenDetail(task);
                }}
                className="flex items-center gap-1 text-[9px] font-mono text-[#777777] hover:text-white transition-colors cursor-pointer px-1 py-0.5 rounded border border-transparent hover:border-[#333333]"
                title="Open comments & details"
              >
                <MessageSquare size={9} />
                <span>Comments</span>
              </button>
            </div>

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
                showLinkSelect && (
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
                )
              )}

              {task.issue_url ? (
                <a
                  href={task.issue_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-[9px] text-emerald-400 bg-emerald-400/5 border border-emerald-400/10 px-1.5 py-0.5 rounded font-mono hover:bg-emerald-400/10 transition-colors"
                >
                  <AlertCircle size={9} />
                  {getIssueNumber(task.issue_url)}
                </a>
              ) : (
                showIssueSelect && (
                  <select
                    className="bg-[#111111] border border-[#3a3a3a] rounded text-[9px] text-white px-1 py-0.5 focus:outline-none max-w-[120px]"
                    defaultValue=""
                    onChange={async (e) => {
                      const val = e.target.value;
                      if (val && val !== 'cancel') {
                        await onLinkTaskToIssue(task.id, val);
                      }
                      setShowIssueSelect(false);
                    }}
                    onBlur={() => setShowIssueSelect(false)}
                  >
                    <option value="" disabled>Link Issue...</option>
                    {issues.map(iss => (
                      <option key={iss.number} value={iss.html_url}>
                        #{iss.number} - {iss.title}
                      </option>
                    ))}
                    <option value="cancel">Cancel</option>
                  </select>
                )
              )}

              {!task.pr_url && !showLinkSelect && (
                <button
                  onClick={() => setShowLinkSelect(true)}
                  className="text-[8px] text-[#666666] hover:text-[#a0a0a0] border border-[#2a2a2a] px-1.5 py-0.5 rounded font-mono transition-all"
                >
                  Link PR
                </button>
              )}

              {!task.issue_url && !showIssueSelect && (
                <button
                  onClick={() => setShowIssueSelect(true)}
                  className="text-[8px] text-[#666666] hover:text-[#a0a0a0] border border-[#2a2a2a] px-1.5 py-0.5 rounded font-mono transition-all"
                >
                  Link Issue
                </button>
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
  issues,
  onLinkTaskToPR,
  onLinkTaskToIssue,
  onOpenDetail,
}: {
  columnId: TaskStatus;
  label: string;
  tasks: Task[];
  onInjectClick: (lineNumber?: number, filePath?: string) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  issues: Issue[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onLinkTaskToIssue: (taskId: string, issueUrl: string) => Promise<void>;
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
                issues={issues}
                onLinkTaskToPR={onLinkTaskToPR}
                onLinkTaskToIssue={onLinkTaskToIssue}
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
  issues,
  onLinkTaskToPR,
  onLinkTaskToIssue,
  onOpenDetail,
}: {
  task: Task;
  onInjectClick: (lineNumber?: number, filePath?: string) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  issues: Issue[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onLinkTaskToIssue: (taskId: string, issueUrl: string) => Promise<void>;
  onOpenDetail: (task: Task) => void;
}) {
  const [showLinkSelect, setShowLinkSelect] = useState(false);
  const [showIssueSelect, setShowIssueSelect] = useState(false);
  const shortSha = task.commit_sha.slice(0, 7);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const assigneeUsername: string | null = (task as any).assignee_username ?? null;

  return (
    <div
      className="card flex items-center justify-between gap-3 px-3 py-2.5 transition-colors duration-150 hover:border-[#3a3a3a] cursor-pointer"
      onClick={() => {
        if (onTaskClick) onTaskClick(task.file_path, task.line_number);
        onOpenDetail(task);
      }}
      title="Click to view code, comments & details"
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
        {task.status === 'open' && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              if (onTaskClick) onTaskClick(task.file_path, task.line_number);
              onInjectClick(task.line_number, task.file_path);
            }}
            className="flex items-center gap-0.5 text-[9px] font-mono text-[#10b981] hover:text-white bg-[#10b981]/10 hover:bg-[#10b981]/30 border border-[#10b981]/30 rounded px-1.5 py-0.5 transition-all cursor-pointer shrink-0"
            title="Add a new line / annotation to this file"
          >
            <Plus size={9} />
            <span>+ Line</span>
          </button>
        )}
        <p className="flex-1 truncate text-xs text-[#a0a0a0]">
          {task.content}
        </p>
      </div>

      <div className="flex items-center gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
        {/* Due date badge if set */}
        {task.due_date && (
          <div
            className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[9px] font-mono shrink-0 ${
              new Date(task.due_date) < new Date() && task.status !== 'resolved'
                ? 'border-red-500/40 bg-red-500/10 text-red-400 font-semibold'
                : 'border-[#2a2a2a] bg-[#1a1a1a] text-[#888888]'
            }`}
            title={new Date(task.due_date) < new Date() && task.status !== 'resolved' ? 'Overdue!' : 'Due Date'}
          >
            <Calendar size={9} className={new Date(task.due_date) < new Date() && task.status !== 'resolved' ? 'text-red-400' : 'text-[#666666]'} />
            <span>
              {new Date(task.due_date).toLocaleDateString(undefined, {
                month: 'short',
                day: 'numeric',
              })}
            </span>
          </div>
        )}

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

        {/* Comments button */}
        <button
          onClick={() => onOpenDetail(task)}
          className="flex items-center gap-1 text-[9px] font-mono border border-[#1a1a1a] hover:border-[#2a2a2a] text-[#888888] hover:text-white rounded px-1.5 py-0.5 transition-all cursor-pointer"
          title="Open comments & details"
        >
          <MessageSquare size={9} />
          <span>Comments</span>
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

        {/* Issue Linkage */}
        {task.issue_url ? (
          <a
            href={task.issue_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-[9px] text-emerald-400 bg-emerald-400/5 border border-emerald-400/10 px-1.5 py-0.5 rounded font-mono hover:bg-emerald-400/10 transition-colors"
          >
            <AlertCircle size={9} />
            {getIssueNumber(task.issue_url)}
          </a>
        ) : (
          showIssueSelect ? (
            <select
              className="bg-[#111111] border border-[#3a3a3a] rounded text-[9px] text-white px-1 py-0.5 focus:outline-none max-w-[120px]"
              defaultValue=""
              onChange={async (e) => {
                const val = e.target.value;
                if (val && val !== 'cancel') {
                  await onLinkTaskToIssue(task.id, val);
                }
                setShowIssueSelect(false);
              }}
              onBlur={() => setShowIssueSelect(false)}
            >
              <option value="" disabled>Link Issue...</option>
              {issues.map(iss => (
                <option key={iss.number} value={iss.html_url}>
                  #{iss.number} - {iss.title}
                </option>
              ))}
              <option value="cancel">Cancel</option>
            </select>
          ) : (
            <button
              onClick={() => setShowIssueSelect(true)}
              className="text-[9px] text-[#666666] hover:text-white border border-[#2a2a2a] hover:border-[#3a3a3a] px-1.5 py-0.5 rounded transition-all font-mono"
            >
              Link Issue
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
  issues,
  onLinkTaskToPR,
  onLinkTaskToIssue,
  onOpenDetail,
}: {
  tasks: Task[];
  onInjectClick: (lineNumber?: number, filePath?: string) => void;
  onTaskClick?: (filePath: string, lineNumber: number) => void;
  pulls: PullRequest[];
  issues: Issue[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
  onLinkTaskToIssue: (taskId: string, issueUrl: string) => Promise<void>;
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
              issues={issues}
              onLinkTaskToPR={onLinkTaskToPR}
              onLinkTaskToIssue={onLinkTaskToIssue}
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
  issues,
  onLinkTaskToPR,
  onLinkTaskToIssue,
}: TaskBoardProps) {
  const { tasks, isLoading, error, fetchTasks, updateTaskStatus, updateTaskAssignee, updateTaskDueDate } =
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

  const handleDueDateChange = useCallback(async (taskId: string, dueDate: string | null) => {
    try {
      await updateTaskDueDate(taskId, dueDate);
    } catch {
      // Silently fail
    }
  }, [updateTaskDueDate]);

  const handleStatusChange = useCallback(async (taskId: string, status: TaskStatus) => {
    try {
      await updateTaskStatus(taskId, status);
    } catch {
      // Silently fail
    }
  }, [updateTaskStatus]);

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
                    issues={issues}
                    onLinkTaskToPR={onLinkTaskToPR}
                    onLinkTaskToIssue={onLinkTaskToIssue}
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
              issues={issues}
              onLinkTaskToPR={onLinkTaskToPR}
              onLinkTaskToIssue={onLinkTaskToIssue}
              onOpenDetail={setDetailTask}
            />
          )}
        </div>
      </div>

      {/* Task detail modal (assignee + comments + due date + status) */}
      {detailTask && (
        <TaskDetailModal
          task={tasks.find((t) => t.id === detailTask.id) || detailTask}
          collaborators={collaborators}
          currentUsername={currentUser?.username ?? ''}
          onClose={() => setDetailTask(null)}
          onAssign={handleAssign}
          onDueDateChange={handleDueDateChange}
          onStatusChange={handleStatusChange}
          onInjectClick={onInjectClick}
          onTaskUpdated={(updated) => setDetailTask(updated)}
        />
      )}
    </>
  );
}

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
 */

import { useEffect, useState } from 'react';
import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd';
import { LayoutList, Columns, Plus, GitPullRequest } from 'lucide-react';
import { useTaskStore } from '../store/taskStore';
import type { Task, TaskStatus, PullRequest } from '../types';
import Badge from './ui/Badge';
import Spinner from './ui/Spinner';

// ── Props ────────────────────────────────────────────────────────────────────

interface TaskBoardProps {
  repoId: number;
  repoOwner: string;
  repoName: string;
  /** Called when user clicks "Inject TODO" — optionally passes a pre-filled line */
  onInjectClick: (lineNumber?: number) => void;
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

// ── Task Card ────────────────────────────────────────────────────────────────

function TaskCard({
  task,
  index,
  onInjectClick,
  pulls,
  onLinkTaskToPR,
}: {
  task: Task;
  index: number;
  onInjectClick: (lineNumber?: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
}) {
  const [showLinkSelect, setShowLinkSelect] = useState(false);
  const displayPath = task.file_path.split('/').slice(-2).join('/');
  const shortSha    = task.commit_sha.slice(0, 7);

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
          onClick={() => onInjectClick(task.line_number)}
          title="Click to pre-fill injector with this line"
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
  pulls,
  onLinkTaskToPR,
}: {
  columnId: TaskStatus;
  label: string;
  tasks: Task[];
  onInjectClick: (lineNumber?: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
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
                pulls={pulls}
                onLinkTaskToPR={onLinkTaskToPR}
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
  pulls,
  onLinkTaskToPR,
}: {
  task: Task;
  onInjectClick: (lineNumber?: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
}) {
  const [showLinkSelect, setShowLinkSelect] = useState(false);
  const shortSha = task.commit_sha.slice(0, 7);

  return (
    <div
      className="card flex items-center justify-between gap-3 px-3 py-2.5 transition-colors duration-150 hover:border-[#3a3a3a] cursor-pointer"
      onClick={() => onInjectClick(task.line_number)}
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
  pulls,
  onLinkTaskToPR,
}: {
  tasks: Task[];
  onInjectClick: (lineNumber?: number) => void;
  pulls: PullRequest[];
  onLinkTaskToPR: (taskId: string, prUrl: string) => Promise<void>;
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
              pulls={pulls}
              onLinkTaskToPR={onLinkTaskToPR}
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
  repoOwner: _repoOwner,
  repoName: _repoName,
  onInjectClick,
  pulls,
  onLinkTaskToPR,
}: TaskBoardProps) {
  const { tasks, isLoading, error, fetchTasks, updateTaskStatus } =
    useTaskStore();

  const [viewMode, setViewMode] = useState<'kanban' | 'list'>('kanban');

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
                  pulls={pulls}
                  onLinkTaskToPR={onLinkTaskToPR}
                />
              ))}
            </div>
          </DragDropContext>
        ) : (
          <ListView
            tasks={tasks}
            onInjectClick={onInjectClick}
            pulls={pulls}
            onLinkTaskToPR={onLinkTaskToPR}
          />
        )}
      </div>
    </div>
  );
}


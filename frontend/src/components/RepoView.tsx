/**
 * RepoView — Three-pane layout for a single repository.
 *
 * ┌──────────────┬────────────────────────────┬──────────────────┐
 * │  File Tree   │       Code Viewer           │   Task Board     │
 * │  (240px)     │       (flex-1)              │   (320px)        │
 * └──────────────┴────────────────────────────┴──────────────────┘
 *
 * - Left: recursive file tree fetched from /api/repos/:owner/:repo/tree
 * - Center: Monaco editor showing selected file contents
 * - Right: Kanban/list task board for this repo
 * - Floating: TaskInjector slide-out panel (triggered by "Inject TODO" or line click)
 */

import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Activity,
  ChevronRight,
  ChevronDown,
  FileText,
  Folder,
  FolderOpen,
  ArrowLeft,
  Users,
  GitCommit,
  GitPullRequest,
  RefreshCw,
  BarChart2,
  ShieldAlert,
} from 'lucide-react';
import { reposApi } from '../api/client';
import { useTaskStore } from '../store/taskStore';
import { useAuthStore } from '../store/authStore';
import type { FileTreeNode, ApiError, PullRequest, Collaborator, Issue } from '../types';
import CodeViewer from './CodeViewer';
import TaskBoard from './TaskBoard';
import TaskInjector from './TaskInjector';
import CollaboratorManager from './CollaboratorManager';
import CommitHistoryPanel from './CommitHistoryPanel';
import PullRequestPanel from './PullRequestPanel';
import ActionsPanel from './ActionsPanel';
import RepoStatsPanel from './RepoStatsPanel';
import DebtPanel from './DebtPanel';
import Spinner from './ui/Spinner';

// ── Types ────────────────────────────────────────────────────────────────────

/** A tree node enriched with children for hierarchical rendering */
interface TreeNode {
  name:     string;
  path:     string;
  type:     'blob' | 'tree';
  sha:      string;
  children: TreeNode[];
}

// ── File tree helpers ─────────────────────────────────────────────────────────

/** Convert the flat tree array from the API into a nested structure */
function buildTree(nodes: FileTreeNode[]): TreeNode[] {
  if (!nodes || !Array.isArray(nodes)) return [];
  const root: TreeNode[] = [];

  // Sort: directories first, then files, both alphabetically
  const sorted = [...nodes].sort((a, b) => {
    if (a.type !== b.type) return a.type === 'tree' ? -1 : 1;
    return a.path.localeCompare(b.path);
  });

  sorted.forEach((node) => {
    const parts  = node.path.split('/');
    let current  = root;

    parts.forEach((part, i) => {
      const isLast = i === parts.length - 1;
      let existing = current.find((n) => n.name === part);

      if (!existing) {
        existing = {
          name: part,
          path: parts.slice(0, i + 1).join('/'),
          type: isLast ? node.type : 'tree',
          sha:  isLast ? node.sha : '',
          children: [],
        };
        current.push(existing);
      }

      current = existing.children;
    });
  });

  return root;
}

/** Infer a language display label from filename for the tag */
function langFromPath(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() ?? '';
  const map: Record<string, string> = {
    ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
    go: 'go', py: 'python', rb: 'ruby', rs: 'rust', java: 'java',
    md: 'markdown', json: 'json', yaml: 'yaml', yml: 'yaml',
    css: 'css', html: 'html', sh: 'shell',
  };
  return map[ext] ?? ext;
}

// ── File icon ────────────────────────────────────────────────────────────────

function FileIcon({ path }: { path: string }) {
  const ext = path.split('.').pop()?.toLowerCase() ?? '';
  // All file icons are the same monochrome style — just differ by stroke color shade
  const colors: Record<string, string> = {
    ts: '#a0a0a0', tsx: '#a0a0a0', js: '#a0a0a0', go: '#a0a0a0',
    py: '#a0a0a0', md: '#666666', json: '#666666', yaml: '#666666',
    css: '#666666', html: '#666666',
  };
  const color = colors[ext] ?? '#666666';

  return (
    <FileText size={13} color={color} className="shrink-0" />
  );
}

// ── Tree node component ───────────────────────────────────────────────────────

function TreeNodeView({
  node,
  depth,
  selectedPath,
  onFileSelect,
}: {
  node: TreeNode;
  depth: number;
  selectedPath: string | null;
  onFileSelect: (path: string) => void;
}) {
  const [expanded, setExpanded] = useState(depth < 1); // expand root level by default
  const isSelected = node.path === selectedPath;
  const isFolder   = node.type === 'tree';

  const indent = depth * 12;

  if (isFolder) {
    return (
      <div>
        {/* Folder row */}
        <button
          className="flex w-full items-center gap-1.5 py-0.5 text-left text-xs text-[#a0a0a0] transition-colors duration-150 hover:text-white"
          style={{ paddingLeft: `${12 + indent}px`, paddingRight: '8px' }}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded
            ? <ChevronDown size={11} className="shrink-0 text-[#666666]" />
            : <ChevronRight size={11} className="shrink-0 text-[#666666]" />
          }
          {expanded
            ? <FolderOpen size={13} className="shrink-0" />
            : <Folder size={13} className="shrink-0" />
          }
          <span className="truncate">{node.name}</span>
        </button>

        {/* Children */}
        {expanded && (
          <div
            className="border-l border-[#2a2a2a]"
            style={{ marginLeft: `${20 + indent}px` }}
          >
            {node.children.map((child) => (
              <TreeNodeView
                key={child.path}
                node={child}
                depth={depth + 1}
                selectedPath={selectedPath}
                onFileSelect={onFileSelect}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  // File row
  return (
    <button
      className={[
        'flex w-full items-center gap-1.5 py-0.5 text-left text-xs transition-colors duration-150',
        isSelected
          ? 'bg-[#242424] text-white'
          : 'text-[#666666] hover:bg-[#1a1a1a] hover:text-[#a0a0a0]',
      ].join(' ')}
      style={{ paddingLeft: `${12 + indent}px`, paddingRight: '8px' }}
      onClick={() => onFileSelect(node.path)}
    >
      <FileIcon path={node.path} />
      <span className="truncate">{node.name}</span>
    </button>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

export default function RepoView() {
  const { owner = '', repo: repoName = '' } = useParams<{ owner: string; repo: string }>();
  const navigate = useNavigate();

  // File tree
  const [treeNodes,    setTreeNodes]    = useState<TreeNode[]>([]);
  const [treeLoading,  setTreeLoading]  = useState(true);
  const [treeError,    setTreeError]    = useState<string | null>(null);

  // Selected file
  const [selectedFile,    setSelectedFile]    = useState<string | null>(null);
  const [fileContent,     setFileContent]     = useState<string>('');
  const [fileLoading,     setFileLoading]     = useState(false);

  // Branch tracking
  const [currentBranch,   setCurrentBranch]   = useState('main');
  const [isSaving,        setIsSaving]        = useState(false);

  // Tabs tracking
  const [leftTab,         setLeftTab]         = useState<'files' | 'commits' | 'access'>('files');
  const [rightTab,        setRightTab]        = useState<'tasks' | 'pulls' | 'actions' | 'debt' | 'analytics'>('tasks');
  const [pulls,           setPulls]           = useState<PullRequest[]>([]);
  const [issues,          setIssues]          = useState<Issue[]>([]);

  const currentUser = useAuthStore((s) => s.user);
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);
  const [collabLoading, setCollabLoading] = useState(false);

  const fetchCollaborators = useCallback(async () => {
    if (!owner || !repoName) return;
    setCollabLoading(true);
    try {
      const list = await reposApi.listCollaborators(owner, repoName);
      setCollaborators(list || []);
    } catch (err) {
      console.error('Failed to fetch collaborators:', err);
    } finally {
      setCollabLoading(false);
    }
  }, [owner, repoName]);

  useEffect(() => {
    if (owner && repoName) {
      fetchCollaborators();
    }
  }, [owner, repoName, fetchCollaborators]);

  const currentUserCollab = collaborators.find((c) => c.username === currentUser?.username);
  const currentUserRole = currentUserCollab
    ? currentUserCollab.role
    : owner === currentUser?.username
    ? 'owner'
    : 'none';

  // Default branch — will be populated from a future repo-detail endpoint.
  // For now, 'main' is the safe default; the injector lets users override it.
  const defaultBranch = 'main';

  // Task injector state
  const [injectorOpen,    setInjectorOpen]    = useState(false);
  const [prefilledLine,   setPrefilledLine]   = useState<number | undefined>();

  // Collaborator list state
  const [collabOpen,      setCollabOpen]      = useState(false);

  const tasks = useTaskStore((s) => s.tasks);
  const linkTaskToPR = useTaskStore((s) => s.linkTaskToPR);
  const linkTaskToIssue = useTaskStore((s) => s.linkTaskToIssue);
  const syncRepoTasks = useTaskStore((s) => s.syncRepoTasks);
  const [repoId, setRepoId] = useState<number>(0);
  const [isSyncing, setIsSyncing] = useState(false);

  const handleSync = useCallback(async () => {
    if (isSyncing || !repoId) return;
    setIsSyncing(true);
    try {
      await syncRepoTasks(owner, repoName, repoId);
    } catch (err) {
      const apiErr = err as ApiError;
      alert(apiErr.message ?? 'Failed to sync tasks.');
    } finally {
      setIsSyncing(false);
    }
  }, [owner, repoName, repoId, isSyncing, syncRepoTasks]);

  const [pendingScrollLine, setPendingScrollLine] = useState<number | null>(null);

  // Resolve repository ID from the user's repository list
  useEffect(() => {
    if (!owner || !repoName) return;
    let cancelled = false;
    (async () => {
      try {
        const repos = await reposApi.list();
        const match = repos.find(
          (r) => r.full_name.toLowerCase() === `${owner}/${repoName}`.toLowerCase()
        );
        if (match && !cancelled) {
          setRepoId(match.id);
        }
      } catch (err) {
        console.error('Failed to resolve repo ID:', err);
      }
    })();
    return () => { cancelled = true; };
  }, [owner, repoName]);

  // ── Fetch file tree on mount ────────────────────────────────────────────

  useEffect(() => {
    if (!owner || !repoName) return;

    let cancelled = false;
    setTreeLoading(true);
    setTreeError(null);

    (async () => {
      try {
        const flat = await reposApi.getTree(owner, repoName, currentBranch);
        if (!cancelled) {
          setTreeNodes(buildTree(flat));
        }
      } catch (err) {
        if (!cancelled) {
          const apiErr = err as ApiError;
          setTreeError(apiErr.message ?? 'Failed to load file tree.');
        }
      } finally {
        if (!cancelled) setTreeLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [owner, repoName, currentBranch]);

  // ── Fetch file contents on file selection ────────────────────────────────

  const handleFileSelect = useCallback(async (path: string) => {
    setSelectedFile(path);
    setFileLoading(true);
    setFileContent('');
    try {
      const content = await reposApi.getContents(owner, repoName, path, currentBranch);
      setFileContent(content);
    } catch {
      setFileContent(`// Failed to load file: ${path}`);
    } finally {
      setFileLoading(false);
    }
  }, [owner, repoName, currentBranch]);

  const handleTaskClick = useCallback((filePath: string, lineNumber: number) => {
    setPendingScrollLine(lineNumber);
    handleFileSelect(filePath);
  }, [handleFileSelect]);

  // Fetch pull requests
  const fetchPulls = useCallback(async () => {
    if (!owner || !repoName) return;
    try {
      const data = await reposApi.listPulls(owner, repoName, 'all');
      setPulls(data || []);
    } catch (err) {
      console.error('Failed to load pulls:', err);
    }
  }, [owner, repoName]);

  // Fetch issues
  const fetchIssues = useCallback(async () => {
    if (!owner || !repoName) return;
    try {
      const data = await reposApi.listIssues(owner, repoName, 'open');
      setIssues(data || []);
    } catch (err) {
      console.error('Failed to load issues:', err);
    }
  }, [owner, repoName]);

  useEffect(() => {
    if (owner && repoName) {
      fetchPulls();
      fetchIssues();
    }
  }, [owner, repoName, fetchPulls, fetchIssues]);

  // Handle merge complete
  const handleMergeComplete = useCallback(() => {
    // Refresh tree on default branch
    reposApi.getTree(owner, repoName, defaultBranch).then((flat) => {
      setTreeNodes(buildTree(flat));
      setCurrentBranch(defaultBranch);
    }).catch(err => console.error(err));
  }, [owner, repoName]);

  // ── Handle file save/commit back to GitHub ──────────────────────────────
  const handleSaveFile = useCallback(async (
    newContent: string,
    branch: string,
    message: string,
    coAuthors?: string[]
  ) => {
    if (!owner || !repoName || !selectedFile) return;
    setIsSaving(true);
    try {
      await reposApi.updateContents(owner, repoName, selectedFile, newContent, branch, message, coAuthors);
      setFileContent(newContent);
      setCurrentBranch(branch);
      const flat = await reposApi.getTree(owner, repoName, branch);
      setTreeNodes(buildTree(flat));
      fetchPulls();
    } catch (err) {
      const apiErr = err as ApiError;
      alert(apiErr.message ?? 'Failed to commit changes to GitHub.');
      throw err;
    } finally {
      setIsSaving(false);
    }
  }, [owner, repoName, selectedFile, fetchPulls]);

  // ── Line click from CodeViewer → open injector pre-filled ───────────────

  const handleLineClick = useCallback((lineNumber: number) => {
    setPrefilledLine(lineNumber);
    setInjectorOpen(true);
  }, []);

  // ── Open injector from TaskBoard (no pre-filled line) ───────────────────

  const handleInjectClick = useCallback((lineNumber?: number) => {
    setPrefilledLine(lineNumber);
    setInjectorOpen(true);
  }, []);
  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-[#0a0a0a]">
      {/* Top bar */}
      <div className="flex h-11 shrink-0 items-center justify-between border-b border-[#2a2a2a] bg-[#0a0a0a] px-4">
        <div className="flex items-center gap-3">
          <button
            className="btn-ghost p-1.5 cursor-pointer"
            onClick={() => navigate('/dashboard')}
            title="Back to Dashboard"
          >
            <ArrowLeft size={14} />
          </button>

          <div className="h-4 w-px bg-[#2a2a2a]" />

          <span
            className="font-mono text-sm font-medium text-white"
            style={{ fontFamily: "'JetBrains Mono', monospace" }}
          >
            {owner}/<span className="text-[#a0a0a0]">{repoName}</span>
          </span>
        </div>

        <div className="flex items-center gap-2">
          <button
            className="btn-secondary py-1 px-3 text-xs flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            onClick={handleSync}
            disabled={isSyncing || !repoId}
            title="Sync tasks from codebase"
          >
            <RefreshCw size={12} className={isSyncing ? 'animate-spin' : ''} />
            <span>{isSyncing ? 'Syncing...' : 'Sync codebase'}</span>
          </button>

          <button
            className="btn-secondary py-1 px-3 text-xs flex items-center gap-1.5 cursor-pointer"
            onClick={() => setCollabOpen(true)}
            title="Manage Collaborators"
          >
            <Users size={13} />
            <span>Collaborators</span>
          </button>
        </div>
      </div>

      {/* Three-pane body */}
      <div className="flex flex-1 overflow-hidden">

        {/* ── Left pane: File tree / Commits (240px) ─────────────────────── */}
        <aside
          className="panel flex w-60 shrink-0 flex-col overflow-hidden animate__animated animate__slideInLeft"
          style={{ animationDuration: '0.4s' }}
          aria-label="Left side navigation"
        >
          {/* Left Tabs */}
          <div className="flex h-9 border-b border-[#2a2a2a] px-2 gap-1 bg-[#111111] items-end shrink-0 overflow-x-auto">
            <button
              onClick={() => setLeftTab('files')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer ${
                leftTab === 'files'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              Files
            </button>
            <button
              onClick={() => setLeftTab('commits')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer ${
                leftTab === 'commits'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              <GitCommit size={11} />
              Commits
            </button>
            {currentUserRole !== 'none' && (
              <button
                onClick={() => setLeftTab('access')}
                className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer ${
                  leftTab === 'access'
                    ? 'border-white text-white'
                    : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
                }`}
              >
                <Users size={11} />
                Access
              </button>
            )}
          </div>

          <div className="flex-1 overflow-y-auto py-1">
            {leftTab === 'files' ? (
              <>
                {treeLoading && (
                  <div className="flex items-center justify-center py-8">
                    <Spinner size={20} />
                  </div>
                )}

                {treeError && (
                  <p className="px-3 py-4 text-[11px] text-[#666666]">{treeError}</p>
                )}

                {!treeLoading && !treeError && treeNodes.map((node) => (
                  <TreeNodeView
                    key={node.path}
                    node={node}
                    depth={0}
                    selectedPath={selectedFile}
                    onFileSelect={handleFileSelect}
                  />
                ))}
              </>
            ) : leftTab === 'commits' ? (
              <CommitHistoryPanel
                owner={owner}
                repoName={repoName}
                currentBranch={currentBranch}
              />
            ) : (
              <div className="flex flex-col gap-3 p-3 animate__animated animate__fadeIn" style={{ animationDuration: '0.2s' }}>
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-mono text-[#666666] uppercase tracking-wider">Access Controls</span>
                  {(currentUserRole === 'owner' || currentUserRole === 'maintainer') && (
                    <button
                      onClick={() => setCollabOpen(true)}
                      className="text-[9px] text-[#a0a0a0] hover:text-white border border-[#2a2a2a] px-1.5 py-0.5 rounded transition-all cursor-pointer font-mono"
                    >
                      Manage
                    </button>
                  )}
                </div>
                {collabLoading ? (
                  <div className="flex justify-center py-6">
                    <Spinner size={16} />
                  </div>
                ) : (
                  <div className="flex flex-col gap-2">
                    {collaborators.map((collab) => {
                      const isSelf = collab.username === currentUser?.username;
                      return (
                        <div
                          key={collab.id}
                          className="flex items-center justify-between p-2 rounded bg-[#161616] border border-[#222222]/60"
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <img
                              src={collab.avatar_url}
                              alt={collab.username}
                              className="h-6 w-6 rounded-full border border-[#2a2a2a] shrink-0"
                            />
                            <div className="min-w-0 flex-1">
                              <p className="text-[11px] font-semibold text-white truncate">
                                {collab.username} {isSelf && <span className="text-[9px] text-[#666666]">(You)</span>}
                              </p>
                              <p className="text-[9px] text-[#666666] font-mono capitalize">
                                {collab.role}
                              </p>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        </aside>

        {/* ── Center pane: Code viewer (flex-1) ────────────────────────── */}
        <main className="flex flex-1 flex-col overflow-hidden animate__animated animate__fadeIn" style={{ animationDuration: '0.5s' }}>
          {fileLoading ? (
            <div className="flex flex-1 items-center justify-center">
              <Spinner size={28} />
            </div>
          ) : (
            <CodeViewer
              content={fileContent}
              language={selectedFile ? langFromPath(selectedFile) : ''}
              filePath={selectedFile ?? ''}
              tasks={tasks}
              onLineClick={handleLineClick}
              onSave={handleSaveFile}
              isSaving={isSaving}
              defaultBranch={currentBranch}
              scrollToLine={pendingScrollLine}
              onScrollToLineComplete={() => setPendingScrollLine(null)}
            />
          )}
        </main>

        {/* ── Right pane: Task board / PRs (320px) ──────────────────────── */}
        <aside
          className="panel flex w-80 shrink-0 flex-col overflow-hidden border-l border-r-0 animate__animated animate__slideInRight"
          style={{ animationDuration: '0.4s' }}
          aria-label="Right side panel"
        >
          {/* Right Tabs */}
          <div className="flex h-9 border-b border-[#2a2a2a] px-2 gap-1 bg-[#111111] items-end shrink-0">
            <button
              onClick={() => setRightTab('tasks')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer shrink-0 ${
                rightTab === 'tasks'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              Tasks
            </button>
            <button
              onClick={() => setRightTab('pulls')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer shrink-0 ${
                rightTab === 'pulls'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              <GitPullRequest size={11} />
              Pull Requests
            </button>
            <button
              onClick={() => setRightTab('actions')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer shrink-0 ${
                rightTab === 'actions'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              <Activity size={11} />
              Actions
            </button>
            <button
              onClick={() => setRightTab('debt')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer shrink-0 ${
                rightTab === 'debt'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              <ShieldAlert size={11} />
              Debt
            </button>
            <button
              onClick={() => setRightTab('analytics')}
              className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer shrink-0 ${
                rightTab === 'analytics'
                  ? 'border-white text-white'
                  : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
              }`}
            >
              <BarChart2 size={11} />
              Analytics
            </button>
          </div>

          <div className="flex-1 overflow-hidden flex flex-col">
            {rightTab === 'tasks' ? (
              <TaskBoard
                repoId={repoId}
                repoOwner={owner}
                repoName={repoName}
                onInjectClick={handleInjectClick}
                onTaskClick={handleTaskClick}
                pulls={pulls}
                issues={issues}
                onLinkTaskToPR={linkTaskToPR}
                onLinkTaskToIssue={linkTaskToIssue}
              />
            ) : rightTab === 'pulls' ? (
              <PullRequestPanel
                owner={owner}
                repoName={repoName}
                currentBranch={currentBranch}
                onMergeComplete={handleMergeComplete}
              />
            ) : rightTab === 'actions' ? (
              <ActionsPanel
                owner={owner}
                repoName={repoName}
                currentBranch={currentBranch}
              />
            ) : rightTab === 'debt' ? (
              <DebtPanel
                owner={owner}
                repoName={repoName}
                repoId={repoId}
              />
            ) : (
              <RepoStatsPanel
                owner={owner}
                repoName={repoName}
              />
            )}
          </div>
        </aside>
      </div>

      {/* ── Task Injector slide-out panel ────────────────────────────────── */}
      <TaskInjector
        isOpen={injectorOpen}
        onClose={() => setInjectorOpen(false)}
        repoOwner={owner}
        repoName={repoName}
        defaultBranch={defaultBranch}
        issues={issues}
        prefilledLine={prefilledLine}
        prefilledFile={selectedFile ?? undefined}
      />

      {/* ── Collaborator Manager slide-out panel ────────────────────────── */}
      <CollaboratorManager
        isOpen={collabOpen}
        onClose={() => setCollabOpen(false)}
        repoOwner={owner}
        repoName={repoName}
        onRefresh={fetchCollaborators}
      />
    </div>
  );
}

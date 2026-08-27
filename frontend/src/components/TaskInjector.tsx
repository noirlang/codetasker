/**
 * TaskInjector — Slide-out panel for injecting task annotations into repo files.
 *
 * Supports:
 * - Single or multiple file/line targets in one pull request.
 * - Creating brand new files with task comments.
 * - Distinct per-location descriptions.
 * - Linking GitHub issues and selecting custom branches.
 */

import { useState, useEffect, useRef, useMemo } from 'react';
import {
  X,
  ExternalLink,
  Plus,
  Trash2,
  FilePlus,
  FileCode,
  Folder,
  ChevronRight,
  CornerLeftUp,
  FileText,
} from 'lucide-react';
import { tasksApi, reposApi } from '../api/client';
import type { InjectTaskRequest, TaskLocation, ApiError, Issue, FileTreeNode } from '../types';
import Spinner from './ui/Spinner';

// ── Props ────────────────────────────────────────────────────────────────────

interface TaskInjectorProps {
  isOpen: boolean;
  onClose: () => void;
  repoOwner: string;
  repoName: string;
  defaultBranch: string;
  issues: Issue[];
  /** Pre-fill line number (from clicking a line in CodeViewer) */
  prefilledLine?: number;
  /** Pre-fill file path (from the currently open file) */
  prefilledFile?: string;
  /** Repository tree files for interactive folder/file autocomplete */
  treeFiles?: FileTreeNode[];
}

interface FormLocation {
  id: string;
  filePath: string;
  lineNumber: string;
  description: string;
  isNewFile: boolean;
}

// ── Path Autocomplete & Interactive Folder Navigator ─────────────────────────

interface PathAutocompleteInputProps {
  value: string;
  onChange: (value: string) => void;
  treeFiles?: FileTreeNode[];
  disabled?: boolean;
  placeholder?: string;
  isNewFile?: boolean;
}

function PathAutocompleteInput({
  value,
  onChange,
  treeFiles = [],
  disabled,
  placeholder,
  isNewFile,
}: PathAutocompleteInputProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Index files and directories from flat tree
  const { allFilePaths, allDirPaths } = useMemo(() => {
    if (!treeFiles || treeFiles.length === 0) {
      return { allFilePaths: [], allDirPaths: [] };
    }
    const files: string[] = [];
    const dirs = new Set<string>();

    for (const item of treeFiles) {
      if (item.type === 'blob') {
        files.push(item.path);
      } else if (item.type === 'tree') {
        dirs.add(item.path.endsWith('/') ? item.path : `${item.path}/`);
      }
      // Deriving directories from path segments
      const segments = item.path.split('/');
      for (let i = 1; i < segments.length; i++) {
        dirs.add(segments.slice(0, i).join('/') + '/');
      }
    }

    return {
      allFilePaths: files.sort(),
      allDirPaths: Array.from(dirs).sort(),
    };
  }, [treeFiles]);

  // Compute current directory, search prefix, and suggestions
  const { currentDir, parentDir, breadcrumbs, directDirs, directFiles, otherFiles } = useMemo(() => {
    const trimmed = value.trim();
    const lastSlashIndex = trimmed.lastIndexOf('/');
    const currDir = lastSlashIndex !== -1 ? trimmed.substring(0, lastSlashIndex + 1) : '';
    const prefix = lastSlashIndex !== -1 ? trimmed.substring(lastSlashIndex + 1).toLowerCase() : trimmed.toLowerCase();

    // Breadcrumbs
    const crumbs: { name: string; path: string }[] = [];
    if (currDir) {
      const parts = currDir.split('/').filter(Boolean);
      let acc = '';
      for (const part of parts) {
        acc += `${part}/`;
        crumbs.push({ name: part, path: acc });
      }
    }

    // Parent directory (for Up button)
    const pDir = currDir.replace(/[^/]+\/$/, '');

    // Direct Subdirectories
    const dDirs = allDirPaths
      .filter((dir) => {
        if (!dir.startsWith(currDir) || dir === currDir) return false;
        const rel = dir.substring(currDir.length);
        const parts = rel.split('/').filter(Boolean);
        return parts.length === 1;
      })
      .map((dir) => {
        const rel = dir.substring(currDir.length);
        const name = rel.replace('/', '');
        return {
          fullPath: dir,
          name,
          type: 'dir' as const,
        };
      })
      .filter((d) => !prefix || d.name.toLowerCase().includes(prefix) || d.name.toLowerCase().startsWith(prefix));

    // Direct Files
    const dFiles = allFilePaths
      .filter((file) => {
        if (!file.startsWith(currDir)) return false;
        const rel = file.substring(currDir.length);
        return !rel.includes('/');
      })
      .map((file) => {
        const name = file.substring(currDir.length);
        return {
          fullPath: file,
          name,
          type: 'file' as const,
        };
      })
      .filter((f) => !prefix || f.name.toLowerCase().includes(prefix) || f.name.toLowerCase().startsWith(prefix));

    // Other matches elsewhere in repo (if user typed something and it didn't end with slash)
    let oFiles: { fullPath: string; name: string; fullDisplay: string; type: 'file' }[] = [];
    if (prefix.length >= 2 && !trimmed.endsWith('/')) {
      oFiles = allFilePaths
        .filter((f) => !f.startsWith(currDir) && f.toLowerCase().includes(prefix))
        .slice(0, 8)
        .map((f) => ({
          fullPath: f,
          name: f.split('/').pop() || f,
          fullDisplay: f,
          type: 'file' as const,
        }));
    }

    return {
      currentDir: currDir,
      parentDir: pDir,
      breadcrumbs: crumbs,
      directDirs: dDirs,
      directFiles: dFiles,
      otherFiles: oFiles,
    };
  }, [value, allDirPaths, allFilePaths]);

  const handleSelectDir = (dirPath: string) => {
    onChange(dirPath);
    setIsOpen(true);
    inputRef.current?.focus();
  };

  const handleSelectFile = (filePath: string) => {
    onChange(filePath);
    setIsOpen(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') {
      setIsOpen(false);
    } else if (e.key === 'ArrowDown' && !isOpen) {
      setIsOpen(true);
    }
  };

  const hasSuggestions = directDirs.length > 0 || directFiles.length > 0 || otherFiles.length > 0 || Boolean(currentDir);

  return (
    <div ref={containerRef} className="relative w-full">
      <div className="relative flex items-center">
        <input
          ref={inputRef}
          type="text"
          className="input font-mono text-xs w-full pr-7"
          style={{ fontFamily: "'JetBrains Mono', monospace" }}
          placeholder={placeholder}
          value={value}
          onChange={(e) => {
            onChange(e.target.value);
            setIsOpen(true);
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          autoComplete="off"
          spellCheck={false}
        />
        <button
          type="button"
          tabIndex={-1}
          onClick={() => {
            setIsOpen((prev) => !prev);
            inputRef.current?.focus();
          }}
          className="absolute right-2 text-[#666666] hover:text-[#aaaaaa] p-0.5 cursor-pointer"
          title="Browse repository folders and files"
        >
          <Folder size={12} className={isOpen ? 'text-[#10b981]' : ''} />
        </button>
      </div>

      {/* Autocomplete / Folder Browser Dropdown */}
      {isOpen && hasSuggestions && (
        <div className="absolute left-0 right-0 top-full mt-1.5 z-50 flex flex-col rounded-md border border-[#333333] bg-[#121212] shadow-2xl shadow-black/90 max-h-60 overflow-hidden animate__animated animate__fadeIn animate__faster">
          {/* Breadcrumb Header Bar */}
          <div className="flex items-center justify-between border-b border-[#222222] bg-[#181818] px-2.5 py-1.5 text-[10px] font-mono text-[#888888]">
            <div className="flex items-center gap-1 overflow-x-auto whitespace-nowrap min-w-0">
              <button
                type="button"
                onClick={() => handleSelectDir('')}
                className="hover:text-white transition-colors cursor-pointer text-[#666666]"
                title="Go to root"
              >
                root
              </button>
              {breadcrumbs.map((crumb, idx) => (
                <span key={crumb.path} className="flex items-center gap-0.5">
                  <ChevronRight size={10} className="text-[#444444]" />
                  <button
                    type="button"
                    onClick={() => handleSelectDir(crumb.path)}
                    className={`hover:text-white transition-colors cursor-pointer ${
                      idx === breadcrumbs.length - 1 ? 'text-[#10b981] font-semibold' : 'text-[#a0a0a0]'
                    }`}
                  >
                    {crumb.name}
                  </button>
                </span>
              ))}
            </div>

            {currentDir && (
              <button
                type="button"
                onClick={() => handleSelectDir(parentDir)}
                className="flex items-center gap-1 text-[9px] text-[#888888] hover:text-white bg-[#222222] hover:bg-[#2a2a2a] px-1.5 py-0.5 rounded transition-all ml-2 shrink-0 cursor-pointer"
                title="Up one folder level (..)"
              >
                <CornerLeftUp size={9} />
                <span>Up</span>
              </button>
            )}
          </div>

          {/* Items list */}
          <div className="overflow-y-auto max-h-48 divide-y divide-[#181818]">
            {/* Subdirectories */}
            {directDirs.map((dir) => (
              <button
                key={dir.fullPath}
                type="button"
                onClick={() => handleSelectDir(dir.fullPath)}
                className="w-full flex items-center justify-between px-3 py-1.5 text-left hover:bg-[#1a1a1a] transition-colors cursor-pointer group"
              >
                <div className="flex items-center gap-2 min-w-0">
                  <Folder size={12} className="text-[#f59e0b] shrink-0 group-hover:scale-110 transition-transform" />
                  <span className="font-mono text-xs text-[#e0e0e0] group-hover:text-white truncate">
                    {dir.name}/
                  </span>
                </div>
                <span className="text-[9px] font-mono text-[#555555] group-hover:text-[#888888] uppercase tracking-wider">
                  dir
                </span>
              </button>
            ))}

            {/* Direct Files */}
            {directFiles.map((file) => (
              <button
                key={file.fullPath}
                type="button"
                onClick={() => handleSelectFile(file.fullPath)}
                className="w-full flex items-center justify-between px-3 py-1.5 text-left hover:bg-[#1a1a1a] transition-colors cursor-pointer group"
              >
                <div className="flex items-center gap-2 min-w-0">
                  <FileCode size={12} className="text-[#3b82f6] shrink-0 group-hover:scale-110 transition-transform" />
                  <span className="font-mono text-xs text-[#bbbbbb] group-hover:text-white truncate">
                    {file.name}
                  </span>
                </div>
                <span className="text-[9px] font-mono text-[#555555] group-hover:text-[#888888]">
                  {file.name.split('.').pop() || 'file'}
                </span>
              </button>
            ))}

            {/* Other matches */}
            {otherFiles.length > 0 && (
              <div className="bg-[#0f0f0f]">
                <div className="px-3 py-1 text-[8px] font-mono text-[#555555] uppercase tracking-wider bg-[#141414]">
                  Other matches in repository
                </div>
                {otherFiles.map((file) => (
                  <button
                    key={file.fullPath}
                    type="button"
                    onClick={() => handleSelectFile(file.fullPath)}
                    className="w-full flex items-center justify-between px-3 py-1.5 text-left hover:bg-[#1a1a1a] transition-colors cursor-pointer group"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <FileText size={11} className="text-[#888888] shrink-0" />
                      <span className="font-mono text-[11px] text-[#888888] group-hover:text-white truncate">
                        {file.fullDisplay}
                      </span>
                    </div>
                  </button>
                ))}
              </div>
            )}

            {directDirs.length === 0 && directFiles.length === 0 && otherFiles.length === 0 && (
              <div className="p-3 text-center text-xs text-[#666666] font-mono">
                {isNewFile ? 'No existing folders match — file will be created here.' : 'No files or folders found.'}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Component ────────────────────────────────────────────────────────────────

export default function TaskInjector({
  isOpen,
  onClose,
  repoOwner,
  repoName,
  defaultBranch,
  issues,
  prefilledLine,
  prefilledFile,
  treeFiles,
}: TaskInjectorProps) {
  const [internalTreeFiles, setInternalTreeFiles] = useState<FileTreeNode[]>([]);

  useEffect(() => {
    if (treeFiles && treeFiles.length > 0) {
      setInternalTreeFiles(treeFiles);
    } else if (isOpen && repoOwner && repoName) {
      reposApi
        .getTree(repoOwner, repoName, defaultBranch || 'main')
        .then((flat) => setInternalTreeFiles(flat || []))
        .catch(() => {});
    }
  }, [treeFiles, isOpen, repoOwner, repoName, defaultBranch]);

  const activeTreeFiles = (treeFiles && treeFiles.length > 0) ? treeFiles : internalTreeFiles;

  // ── Form state ────────────────────────────────────────────────────────────
  const [locations, setLocations] = useState<FormLocation[]>([
    {
      id: 'loc-1',
      filePath: '',
      lineNumber: '',
      description: '',
      isNewFile: false,
    },
  ]);
  const [taskType,          setTaskType]          = useState('TODO');
  const [branch,            setBranch]            = useState('');
  const [selectedIssueUrl,  setSelectedIssueUrl]  = useState('');

  // ── UI state ──────────────────────────────────────────────────────────────
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [prUrl,        setPrUrl]        = useState<string | null>(null);
  const [formError,    setFormError]    = useState<string | null>(null);

  /** Ref for the auto-close timer after successful injection */
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Pre-fill whenever panel opens or prefilled values change ──────────────
  useEffect(() => {
    if (isOpen) {
      setLocations([
        {
          id: `loc-${Date.now()}`,
          filePath: prefilledFile ?? '',
          lineNumber: prefilledLine != null ? String(prefilledLine) : '1',
          description: '',
          isNewFile: false,
        },
      ]);
      setTaskType('TODO');
      setBranch(defaultBranch || 'main');
      setSelectedIssueUrl('');
      setPrUrl(null);
      setFormError(null);
      setIsSubmitting(false);
    }
  }, [isOpen, prefilledFile, prefilledLine, defaultBranch]);

  // Clear auto-close timer on unmount
  useEffect(() => {
    return () => {
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  // ── Location management ───────────────────────────────────────────────────

  const handleAddLocation = () => {
    setLocations((prev) => [
      ...prev,
      {
        id: `loc-${Date.now()}-${Math.random().toString(36).substring(2, 6)}`,
        filePath: '',
        lineNumber: '1',
        description: '',
        isNewFile: false,
      },
    ]);
  };

  const handleRemoveLocation = (id: string) => {
    if (locations.length <= 1) return;
    setLocations((prev) => prev.filter((loc) => loc.id !== id));
  };

  const handleUpdateLocation = (id: string, updates: Partial<FormLocation>) => {
    setLocations((prev) =>
      prev.map((loc) => (loc.id === id ? { ...loc, ...updates } : loc))
    );
  };

  // ── Form submit ───────────────────────────────────────────────────────────

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!branch.trim()) {
      setFormError('Target branch name is required.');
      return;
    }

    if (locations.length === 0) {
      setFormError('At least one file location is required.');
      return;
    }

    const payloadLocations: TaskLocation[] = [];

    for (let i = 0; i < locations.length; i++) {
      const loc = locations[i];
      const trimmedPath = loc.filePath.trim();
      const trimmedDesc = loc.description.trim();

      if (!trimmedPath) {
        setFormError(`Location #${i + 1}: File path is required.`);
        return;
      }

      let lineNum = parseInt(loc.lineNumber, 10);
      if (loc.isNewFile) {
        if (isNaN(lineNum) || lineNum < 1) lineNum = 1;
      } else {
        if (isNaN(lineNum) || lineNum < 1) {
          setFormError(`Location #${i + 1} (${trimmedPath}): Line number must be >= 1.`);
          return;
        }
      }

      if (!trimmedDesc) {
        setFormError(`Location #${i + 1} (${trimmedPath}): Description is required.`);
        return;
      }

      payloadLocations.push({
        file_path: trimmedPath,
        line_number: lineNum,
        description: trimmedDesc,
        is_new_file: loc.isNewFile,
      });
    }

    const firstLoc = payloadLocations[0];
    const req: InjectTaskRequest = {
      repo_owner:  repoOwner,
      repo_name:   repoName,
      file_path:   firstLoc.file_path,
      line_number: firstLoc.line_number,
      description: firstLoc.description,
      branch:      branch.trim(),
      type:        taskType,
      issue_url:   selectedIssueUrl || undefined,
      locations:   payloadLocations,
    };

    setIsSubmitting(true);
    try {
      const { pr_url } = await tasksApi.inject(req);
      setPrUrl(pr_url);

      // Auto-close after 4 seconds
      closeTimerRef.current = setTimeout(() => {
        onClose();
      }, 4000);
    } catch (err) {
      const apiErr = err as ApiError;
      setFormError(apiErr.message ?? 'Failed to inject task. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  // ── Keyboard: close on Escape ─────────────────────────────────────────────

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) onClose();
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [isOpen, onClose]);

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <>
      {/* Backdrop */}
      <div
        className={[
          'fixed inset-0 z-40 bg-black transition-opacity duration-200',
          isOpen ? 'opacity-40 pointer-events-auto' : 'opacity-0 pointer-events-none',
        ].join(' ')}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Slide-out panel */}
      <aside
        role="dialog"
        aria-modal="true"
        aria-label="Inject Task"
        className={[
          'fixed right-0 top-0 z-50 flex h-full w-[460px] max-w-full flex-col',
          'border-l border-[#2a2a2a] bg-[#111111] shadow-2xl',
          'transition-transform duration-200 ease-out',
          isOpen ? 'translate-x-0' : 'translate-x-full',
        ].join(' ')}
      >
        {/* Panel header */}
        <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] px-5 py-4">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-white">Inject Task & Open PR</h2>
              {locations.length > 1 && (
                <span className="rounded bg-[#222222] border border-[#333333] px-2 py-0.5 text-[10px] font-mono text-[#10b981]">
                  {locations.length} targets
                </span>
              )}
            </div>
            <p
              className="mt-0.5 font-mono text-[10px] text-[#666666]"
              style={{ fontFamily: "'JetBrains Mono', monospace" }}
            >
              {repoOwner}/{repoName}
            </p>
          </div>
          <button
            onClick={onClose}
            className="btn-ghost p-1.5"
            aria-label="Close panel"
          >
            <X size={14} />
          </button>
        </div>

        {/* Panel body */}
        <div className="flex-1 overflow-y-auto px-5 py-5">
          {/* ── Success state ────────────────────────────────────────────── */}
          {prUrl ? (
            <div className="flex flex-col items-center gap-4 py-12 text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-full border border-[#10b981]/40 bg-[#10b981]/10">
                {/* Checkmark icon */}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-[#10b981]"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </div>
              <div>
                <p className="text-base font-semibold text-white">Pull Request Created</p>
                <p className="mt-1 text-xs text-[#888888]">
                  Injected {locations.length} task annotation{locations.length > 1 ? 's' : ''} into repository.
                </p>
                <p className="mt-1 text-[11px] text-[#555555]">Closing automatically in a few seconds…</p>
              </div>
              <a
                href={prUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="btn-primary mt-2 flex items-center gap-2 text-xs"
              >
                <ExternalLink size={13} />
                View PR on GitHub
              </a>
            </div>
          ) : (
            /* ── Form ──────────────────────────────────────────────────── */
            <form onSubmit={handleSubmit} className="flex flex-col gap-5">
              {/* Task Type & Branch row */}
              <div className="grid grid-cols-2 gap-3">
                <div className="flex flex-col gap-1.5">
                  <label className="section-label" htmlFor="inj-type">
                    Task Type
                  </label>
                  <select
                    id="inj-type"
                    className="input text-xs"
                    value={taskType}
                    onChange={(e) => setTaskType(e.target.value)}
                    disabled={isSubmitting}
                  >
                    <option value="TODO">TODO</option>
                    <option value="FIXME">FIXME</option>
                    <option value="BUG">BUG</option>
                    <option value="HACK">HACK</option>
                    <option value="NOTE">NOTE</option>
                  </select>
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="section-label" htmlFor="inj-branch">
                    Target Branch
                  </label>
                  <input
                    id="inj-branch"
                    type="text"
                    className="input font-mono text-xs"
                    style={{ fontFamily: "'JetBrains Mono', monospace" }}
                    placeholder="main"
                    value={branch}
                    onChange={(e) => setBranch(e.target.value)}
                    disabled={isSubmitting}
                    autoComplete="off"
                    spellCheck={false}
                  />
                </div>
              </div>

              {/* Link Issue (optional) */}
              <div className="flex flex-col gap-1.5">
                <label className="section-label" htmlFor="inj-issue">
                  Link GitHub Issue (Optional)
                </label>
                <select
                  id="inj-issue"
                  className="input text-xs"
                  value={selectedIssueUrl}
                  onChange={(e) => {
                    const val = e.target.value;
                    setSelectedIssueUrl(val);
                    if (val && locations.length === 1 && !locations[0].description) {
                      const matched = issues.find((i) => i.html_url === val);
                      if (matched) {
                        handleUpdateLocation(locations[0].id, {
                          description: `Resolve #${matched.number}: ${matched.title}`,
                        });
                      }
                    }
                  }}
                  disabled={isSubmitting}
                >
                  <option value="">-- No Issue Linked --</option>
                  {issues.map((iss) => (
                    <option key={iss.number} value={iss.html_url}>
                      #{iss.number} - {iss.title}
                    </option>
                  ))}
                </select>
              </div>

              {/* Locations section header */}
              <div className="flex items-center justify-between border-t border-[#222222] pt-4">
                <div>
                  <span className="section-label block">Target Files & Lines</span>
                  <p className="text-[10px] text-[#666666]">
                    Specify files and lines to annotate, or create new files from scratch.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={handleAddLocation}
                  disabled={isSubmitting}
                  className="btn-ghost flex items-center gap-1.5 border border-[#333333] px-2.5 py-1 text-xs text-white hover:border-[#444444]"
                >
                  <Plus size={12} />
                  Add File / Line
                </button>
              </div>

              {/* Location items list */}
              <div className="flex flex-col gap-3.5">
                {locations.map((loc, idx) => (
                  <div
                    key={loc.id}
                    className="relative flex flex-col gap-3 rounded-lg border border-[#2a2a2a] bg-[#161616] p-3.5 transition-all focus-within:border-[#3a3a3a]"
                  >
                    {/* Location item header */}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="flex h-5 w-5 items-center justify-center rounded bg-[#222222] text-[10px] font-mono text-[#888888]">
                          {idx + 1}
                        </span>
                        <span className="text-xs font-medium text-[#cccccc]">
                          {loc.isNewFile ? 'New File' : 'Existing File Location'}
                        </span>
                      </div>

                      <div className="flex items-center gap-2">
                        {/* New file checkbox */}
                        <label className="flex cursor-pointer select-none items-center gap-1.5 text-[11px] text-[#a0a0a0] hover:text-white">
                          <input
                            type="checkbox"
                            checked={loc.isNewFile}
                            onChange={(e) =>
                              handleUpdateLocation(loc.id, {
                                isNewFile: e.target.checked,
                                lineNumber: e.target.checked ? '1' : loc.lineNumber || '1',
                              })
                            }
                            disabled={isSubmitting}
                            className="h-3.5 w-3.5 rounded border-[#3a3a3a] bg-[#111111] accent-[#10b981]"
                          />
                          {loc.isNewFile ? (
                            <span className="flex items-center gap-1 text-[#10b981]">
                              <FilePlus size={11} />
                              Create new file
                            </span>
                          ) : (
                            <span className="flex items-center gap-1">
                              <FileCode size={11} />
                              New file?
                            </span>
                          )}
                        </label>

                        {/* Remove location button */}
                        {locations.length > 1 && (
                          <button
                            type="button"
                            onClick={() => handleRemoveLocation(loc.id)}
                            disabled={isSubmitting}
                            className="btn-ghost p-1 text-[#666666] hover:text-red-400"
                            title="Remove this location"
                          >
                            <Trash2 size={12} />
                          </button>
                        )}
                      </div>
                    </div>

                    {/* File path & Line number inputs */}
                    <div className="grid grid-cols-3 gap-2">
                      <div className={loc.isNewFile ? 'col-span-3' : 'col-span-2'}>
                        <label className="text-[10px] text-[#666666] block mb-1 font-mono">
                          FILE PATH
                        </label>
                        <PathAutocompleteInput
                          value={loc.filePath}
                          onChange={(newPath) => handleUpdateLocation(loc.id, { filePath: newPath })}
                          treeFiles={activeTreeFiles}
                          disabled={isSubmitting}
                          placeholder={loc.isNewFile ? "src/modules/new_service.go" : "src/handlers/auth.go"}
                          isNewFile={loc.isNewFile}
                        />
                      </div>

                      {!loc.isNewFile && (
                        <div className="col-span-1">
                          <label className="text-[10px] text-[#666666] block mb-1 font-mono">
                            LINE #
                          </label>
                          <input
                            type="number"
                            className="input font-mono text-xs"
                            style={{ fontFamily: "'JetBrains Mono', monospace" }}
                            placeholder="42"
                            min={1}
                            value={loc.lineNumber}
                            onChange={(e) => handleUpdateLocation(loc.id, { lineNumber: e.target.value })}
                            disabled={isSubmitting}
                          />
                        </div>
                      )}
                    </div>

                    {/* Description input */}
                    <div>
                      <label className="text-[10px] text-[#666666] block mb-1 font-mono">
                        TASK DESCRIPTION / NOTE
                      </label>
                      <textarea
                        rows={2}
                        className="input resize-none text-xs leading-relaxed"
                        placeholder={
                          loc.isNewFile
                            ? "Describe what this new file should implement…"
                            : "Describe what needs to be refactored or implemented at this line…"
                        }
                        value={loc.description}
                        onChange={(e) => handleUpdateLocation(loc.id, { description: e.target.value })}
                        disabled={isSubmitting}
                      />
                    </div>
                  </div>
                ))}
              </div>

              {/* Add location button */}
              <button
                type="button"
                onClick={handleAddLocation}
                disabled={isSubmitting}
                className="btn-secondary flex w-full items-center justify-center gap-2 text-xs py-2"
              >
                <Plus size={13} />
                Add Another Location or File
              </button>

              {/* Error message */}
              {formError && (
                <div
                  className="rounded border border-red-500/40 bg-red-500/10 px-3.5 py-2.5 text-xs text-red-300"
                >
                  {formError}
                </div>
              )}

              {/* Submit */}
              <button
                type="submit"
                className="btn-primary w-full justify-center py-2.5 text-xs font-semibold"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <>
                    <Spinner size={14} />
                    Creating Pull Request…
                  </>
                ) : (
                  `Inject ${locations.length > 1 ? `${locations.length} Annotations` : `${taskType}`} & Open PR`
                )}
              </button>
            </form>
          )}
        </div>
      </aside>
    </>
  );
}

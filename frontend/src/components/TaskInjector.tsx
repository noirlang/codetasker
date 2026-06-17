/**
 * TaskInjector — Slide-out panel for injecting a TODO comment into a repo file.
 *
 * Slides in from the right edge. Accepts pre-filled file path and line number
 * (e.g. from clicking a line in CodeViewer). On submit, calls the inject API
 * and shows the resulting PR URL with a 3-second auto-close timer.
 */

import { useState, useEffect, useRef } from 'react';
import { X, ExternalLink } from 'lucide-react';
import { tasksApi } from '../api/client';
import type { InjectTaskRequest, ApiError, Issue } from '../types';
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
}: TaskInjectorProps) {
  // ── Form state ────────────────────────────────────────────────────────────
  const [filePath,          setFilePath]          = useState('');
  const [lineNumber,        setLineNumber]        = useState('');
  const [taskType,          setTaskType]          = useState('TODO');
  const [description,       setDescription]       = useState('');
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
      setFilePath(prefilledFile ?? '');
      setLineNumber(prefilledLine != null ? String(prefilledLine) : '');
      setTaskType('TODO');
      setBranch(defaultBranch);
      setDescription('');
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

  // ── Form submit ───────────────────────────────────────────────────────────

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    const lineNum = parseInt(lineNumber, 10);
    if (!filePath.trim()) {
      setFormError('File path is required.');
      return;
    }
    if (isNaN(lineNum) || lineNum < 1) {
      setFormError('Line number must be a positive integer.');
      return;
    }
    if (!description.trim()) {
      setFormError('Description is required.');
      return;
    }
    if (!branch.trim()) {
      setFormError('Branch name is required.');
      return;
    }

    const req: InjectTaskRequest = {
      repo_owner:  repoOwner,
      repo_name:   repoName,
      file_path:   filePath.trim(),
      line_number: lineNum,
      description: description.trim(),
      branch:      branch.trim(),
      type:        taskType,
      issue_url:   selectedIssueUrl || undefined,
    };

    setIsSubmitting(true);
    try {
      const { pr_url } = await tasksApi.inject(req);
      setPrUrl(pr_url);

      // Auto-close after 3 seconds
      closeTimerRef.current = setTimeout(() => {
        onClose();
      }, 3000);
    } catch (err) {
      const apiErr = err as ApiError;
      setFormError(apiErr.message ?? 'Failed to inject TODO. Please try again.');
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
        aria-label="Inject TODO"
        className={[
          'fixed right-0 top-0 z-50 flex h-full w-96 flex-col',
          'border-l border-[#2a2a2a] bg-[#111111]',
          'transition-transform duration-200 ease-out',
          isOpen ? 'translate-x-0' : 'translate-x-full',
        ].join(' ')}
      >
        {/* Panel header */}
        <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] px-5 py-4">
          <div>
            <h2 className="text-sm font-semibold text-white">Inject TODO</h2>
            <p className="mt-0.5 font-mono text-[10px] text-[#666666]"
               style={{ fontFamily: "'JetBrains Mono', monospace" }}>
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
            <div className="flex flex-col items-center gap-4 py-8 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full border border-[#3a3a3a]">
                {/* Checkmark icon */}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-white"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </div>
              <div>
                <p className="text-sm font-medium text-white">Pull Request Created</p>
                <p className="mt-1 text-xs text-[#666666]">Closing in 3 seconds…</p>
              </div>
              <a
                href={prUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="btn-secondary flex items-center gap-2 text-xs"
              >
                <ExternalLink size={12} />
                View PR on GitHub
              </a>
            </div>
          ) : (
            /* ── Form ──────────────────────────────────────────────────── */
            <form onSubmit={handleSubmit} className="flex flex-col gap-5">
              {/* File path */}
              <div className="flex flex-col gap-1.5">
                <label className="section-label" htmlFor="inj-filepath">
                  File Path
                </label>
                <input
                  id="inj-filepath"
                  type="text"
                  className="input font-mono text-xs"
                  style={{ fontFamily: "'JetBrains Mono', monospace" }}
                  placeholder="src/main.go"
                  value={filePath}
                  onChange={(e) => setFilePath(e.target.value)}
                  disabled={isSubmitting}
                  autoComplete="off"
                  spellCheck={false}
                />
              </div>

              {/* Line number */}
              <div className="flex flex-col gap-1.5">
                <label className="section-label" htmlFor="inj-line">
                  Line Number
                </label>
                <input
                  id="inj-line"
                  type="number"
                  className="input font-mono text-xs"
                  style={{ fontFamily: "'JetBrains Mono', monospace" }}
                  placeholder="42"
                  min={1}
                  value={lineNumber}
                  onChange={(e) => setLineNumber(e.target.value)}
                  disabled={isSubmitting}
                />
              </div>

              {/* Task Type */}
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
                    if (val) {
                      const matched = issues.find(i => i.html_url === val);
                      if (matched) {
                        setDescription(`Resolve #${matched.number}: ${matched.title}`);
                      }
                    }
                  }}
                  disabled={isSubmitting}
                >
                  <option value="">-- No Issue Linked --</option>
                  {issues.map(iss => (
                    <option key={iss.number} value={iss.html_url}>
                      #{iss.number} - {iss.title}
                    </option>
                  ))}
                </select>
              </div>

              {/* Description */}
              <div className="flex flex-col gap-1.5">
                <label className="section-label" htmlFor="inj-desc">
                  Description
                </label>
                <textarea
                  id="inj-desc"
                  rows={4}
                  className="input resize-none text-xs leading-relaxed"
                  placeholder="Describe what needs to be done…"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  disabled={isSubmitting}
                />
                <p className="text-[10px] text-[#666666]">
                  This text will appear as the TODO comment body.
                </p>
              </div>

              {/* Target branch */}
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

              {/* Error message */}
              {formError && (
                <p
                  className="rounded border border-[#3a3a3a] px-3 py-2 text-xs"
                  style={{ color: 'rgba(255, 255, 255, 0.7)' }}
                >
                  {formError}
                </p>
              )}

              {/* Submit */}
              <button
                type="submit"
                className="btn-primary w-full justify-center"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <>
                    <Spinner size={14} />
                    Creating PR…
                  </>
                ) : (
                  'Inject TODO & Open PR'
                )}
              </button>
            </form>
          )}
        </div>
      </aside>
    </>
  );
}

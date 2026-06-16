/**
 * CommitDiffPanel — Slide-in drawer showing commit detail + file diffs.
 *
 * Features:
 * - Commit message, author, date
 * - Stats: +additions -deletions
 * - List of files changed with status indicators (added=green, removed=red, modified=yellow)
 * - Expandable patch/diff viewer per file with colored +/- lines
 */

import { useState, useEffect } from 'react';
import {
  X,
  Plus,
  Minus,
  FileText,
  ChevronDown,
  ChevronRight,
  User,
  GitCommit,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { CommitDetail, CommitFile } from '../types';
import Spinner from './ui/Spinner';

// ── Helpers ───────────────────────────────────────────────────────────────────

function timeAgo(isoString: string): string {
  if (!isoString) return '';
  const diff = Date.now() - new Date(isoString).getTime();
  const s = Math.floor(diff / 1000);
  const m = Math.floor(s / 60);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (s < 60) return 'just now';
  if (m < 60) return `${m}m ago`;
  if (h < 24) return `${h}h ago`;
  if (d < 7) return `${d}d ago`;
  return `${Math.floor(d / 7)}w ago`;
}

/** Status color for a file change */
function statusColor(status: CommitFile['status']): string {
  switch (status) {
    case 'added':    return 'text-emerald-400';
    case 'removed':  return 'text-red-400';
    case 'renamed':  return 'text-blue-400';
    default:         return 'text-amber-400'; // modified
  }
}

function statusLabel(status: CommitFile['status']): string {
  switch (status) {
    case 'added':   return 'A';
    case 'removed': return 'D';
    case 'renamed': return 'R';
    default:        return 'M';
  }
}

// ── Diff Viewer ───────────────────────────────────────────────────────────────

function DiffViewer({ patch }: { patch: string }) {
  const lines = patch.split('\n');
  return (
    <pre className="overflow-x-auto rounded bg-[#080808] p-3 text-[10px] leading-5 font-mono">
      {lines.map((line, i) => {
        let cls = 'text-[#666666]';
        if (line.startsWith('+') && !line.startsWith('+++')) cls = 'text-emerald-400 bg-emerald-500/5';
        else if (line.startsWith('-') && !line.startsWith('---')) cls = 'text-red-400 bg-red-500/5';
        else if (line.startsWith('@@')) cls = 'text-blue-400/70';
        return (
          <div key={i} className={`${cls} px-1`}>
            {line || ' '}
          </div>
        );
      })}
    </pre>
  );
}

// ── File Row ──────────────────────────────────────────────────────────────────

function FileRow({ file }: { file: CommitFile }) {
  const [expanded, setExpanded] = useState(false);
  const hasPatch = !!file.patch;

  return (
    <div className="rounded border border-[#1a1a1a] overflow-hidden">
      <button
        className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-[#0d0d0d] transition-colors cursor-pointer"
        onClick={() => hasPatch && setExpanded((v) => !v)}
        disabled={!hasPatch}
      >
        {hasPatch ? (
          expanded ? <ChevronDown size={11} className="shrink-0 text-[#666666]" /> : <ChevronRight size={11} className="shrink-0 text-[#666666]" />
        ) : (
          <FileText size={11} className="shrink-0 text-[#444444]" />
        )}

        <span className={`shrink-0 font-mono text-[9px] font-bold w-3 text-center ${statusColor(file.status)}`}>
          {statusLabel(file.status)}
        </span>

        <span className="flex-1 truncate font-mono text-[10px] text-[#a0a0a0]" title={file.filename}>
          {file.filename}
        </span>

        <div className="flex items-center gap-1 shrink-0">
          {file.additions > 0 && (
            <span className="flex items-center gap-0.5 font-mono text-[9px] text-emerald-400">
              <Plus size={8} />
              {file.additions}
            </span>
          )}
          {file.deletions > 0 && (
            <span className="flex items-center gap-0.5 font-mono text-[9px] text-red-400">
              <Minus size={8} />
              {file.deletions}
            </span>
          )}
        </div>
      </button>

      {expanded && file.patch && (
        <div className="border-t border-[#1a1a1a]">
          <DiffViewer patch={file.patch} />
        </div>
      )}
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

interface CommitDiffPanelProps {
  owner: string;
  repoName: string;
  sha: string;
  onClose: () => void;
}

export default function CommitDiffPanel({
  owner,
  repoName,
  sha,
  onClose,
}: CommitDiffPanelProps) {
  const [detail, setDetail] = useState<CommitDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    reposApi
      .getCommitDiff(owner, repoName, sha)
      .then(setDetail)
      .catch(() => setError('Failed to load commit diff.'))
      .finally(() => setLoading(false));
  }, [owner, repoName, sha]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-end bg-black/50 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        className="flex h-full w-full max-w-xl flex-col bg-[#0d0d0d] border-l border-[#1a1a1a] shadow-2xl shadow-black animate__animated animate__slideInRight"
        style={{ animationDuration: '0.22s' }}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[#1a1a1a] px-5 py-3.5 shrink-0 bg-[#111111]">
          <div className="flex items-center gap-2">
            <GitCommit size={14} className="text-[#a0a0a0]" />
            <span className="font-mono text-[11px] text-white">Commit Diff</span>
            <span className="font-mono text-[10px] text-[#666666]">{sha.slice(0, 7)}</span>
          </div>
          <button
            onClick={onClose}
            className="rounded p-1.5 text-[#666666] hover:text-white transition-colors cursor-pointer"
          >
            <X size={14} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Spinner size={24} />
            </div>
          ) : error || !detail ? (
            <div className="flex flex-col items-center gap-2 py-12">
              <p className="text-[11px] text-[#666666]">{error ?? 'No data'}</p>
              <button
                onClick={() => { setLoading(true); reposApi.getCommitDiff(owner, repoName, sha).then(setDetail).catch(() => setError('Failed')).finally(() => setLoading(false)); }}
                className="rounded border border-[#2a2a2a] px-3 py-1 text-[10px] text-[#666666] hover:text-white transition-all cursor-pointer"
              >
                Retry
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-0">
              {/* Commit meta */}
              <div className="border-b border-[#1a1a1a] px-5 py-4 bg-[#111111]">
                <p className="text-sm font-medium text-white leading-snug">
                  {detail.commit.message.split('\n')[0]}
                </p>
                {detail.commit.message.includes('\n') && (
                  <p className="mt-1 text-[11px] text-[#666666] leading-4 line-clamp-3">
                    {detail.commit.message.split('\n').slice(1).join('\n').trim()}
                  </p>
                )}
                <div className="mt-3 flex items-center gap-3">
                  {detail.author?.avatar_url ? (
                    <img
                      src={detail.author.avatar_url}
                      alt={detail.author.login}
                      className="h-6 w-6 rounded-full border border-[#2a2a2a]"
                    />
                  ) : (
                    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a]">
                      <User size={12} className="text-[#666666]" />
                    </div>
                  )}
                  <span className="text-[11px] text-[#a0a0a0]">
                    {detail.author?.login ?? detail.commit.author.name}
                  </span>
                  <span className="text-[10px] text-[#666666]">
                    {timeAgo(detail.commit.author.date)}
                  </span>
                </div>

                {/* Stats */}
                <div className="mt-3 flex items-center gap-3 rounded border border-[#1a1a1a] bg-[#0d0d0d] px-3 py-2">
                  <span className="font-mono text-[10px] text-[#666666]">
                    {detail.files.length} file{detail.files.length !== 1 ? 's' : ''} changed
                  </span>
                  <span className="flex items-center gap-1 font-mono text-[10px] text-emerald-400">
                    <Plus size={10} />
                    {detail.stats.additions}
                  </span>
                  <span className="flex items-center gap-1 font-mono text-[10px] text-red-400">
                    <Minus size={10} />
                    {detail.stats.deletions}
                  </span>
                </div>
              </div>

              {/* Files */}
              <div className="flex flex-col gap-2 p-4">
                {detail.files.map((file) => (
                  <FileRow key={file.filename} file={file} />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/**
 * PullRequestPanel — Displays Pull Requests and allows branch/PR merging.
 *
 * - Lists all PRs (open, closed, merged) with their status.
 * - Each OPEN PR card has an inline "Merge PR" button that expands a compact
 *   options form (merge method + optional commit title) and merges via GitHub API.
 * - "Merge Branch" button at the top handles arbitrary branch merges.
 */

import { useState, useEffect, useCallback } from 'react';
import {
  GitPullRequest,
  GitMerge,
  ExternalLink,
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  RefreshCw,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { PullRequest, ApiError } from '../types';
import Spinner from './ui/Spinner';

interface PullRequestPanelProps {
  owner: string;
  repoName: string;
  currentBranch: string;
  onMergeComplete: () => void;
}

type MergeMethod = 'merge' | 'squash' | 'rebase';

const MERGE_METHOD_LABELS: Record<MergeMethod, string> = {
  merge:  'Create a merge commit',
  squash: 'Squash and merge',
  rebase: 'Rebase and merge',
};

// ── Inline PR Merge Widget ────────────────────────────────────────────────────

function PrMergeWidget({
  owner,
  repoName,
  pr,
  onMergeComplete,
}: {
  owner: string;
  repoName: string;
  pr: PullRequest;
  onMergeComplete: () => void;
}) {
  const [expanded,    setExpanded]    = useState(false);
  const [method,      setMethod]      = useState<MergeMethod>('merge');
  const [title,       setTitle]       = useState('');
  const [isMerging,   setIsMerging]   = useState(false);
  const [success,     setSuccess]     = useState<string | null>(null);
  const [error,       setError]       = useState<string | null>(null);

  const handleMerge = async () => {
    setIsMerging(true);
    setError(null);
    setSuccess(null);
    try {
      const result = await reposApi.mergePR(owner, repoName, pr.number, {
        mergeMethod:   method,
        commitTitle:   title.trim() || undefined,
      });
      setSuccess(`Merged! Commit: ${result.commit_sha.slice(0, 7)}`);
      setExpanded(false);
      onMergeComplete();
      setTimeout(() => setSuccess(null), 6000);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Merge failed.');
    } finally {
      setIsMerging(false);
    }
  };

  return (
    <div className="mt-1">
      {success && (
        <div className="flex items-center gap-1.5 text-[10px] text-emerald-400 bg-emerald-400/5 border border-emerald-400/10 px-2 py-1 rounded mb-1">
          <CheckCircle2 size={10} className="shrink-0" />
          {success}
        </div>
      )}
      {error && (
        <div className="flex items-center gap-1.5 text-[10px] text-red-400 bg-red-400/5 border border-red-400/10 px-2 py-1 rounded mb-1">
          <AlertCircle size={10} className="shrink-0" />
          {error}
        </div>
      )}

      {/* Toggle button */}
      <button
        onClick={() => setExpanded(v => !v)}
        disabled={isMerging}
        className="flex items-center gap-1 text-[9px] font-mono px-2 py-1 rounded border border-purple-400/20 bg-purple-400/5 text-purple-300 hover:bg-purple-400/10 hover:border-purple-400/30 transition-all cursor-pointer disabled:opacity-40"
      >
        <GitMerge size={9} />
        Merge PR
        {expanded ? <ChevronUp size={9} /> : <ChevronDown size={9} />}
      </button>

      {/* Expanded options */}
      {expanded && (
        <div className="mt-1.5 rounded border border-[#2a2a2a] bg-[#0d0d0d] p-2.5 flex flex-col gap-2 animate-in slide-in-from-top-1 duration-150">
          {/* Merge method selector */}
          <div>
            <label className="block text-[9px] text-[#666666] uppercase tracking-wider mb-1">
              Merge Method
            </label>
            <div className="flex flex-col gap-1">
              {(Object.keys(MERGE_METHOD_LABELS) as MergeMethod[]).map((m) => (
                <label
                  key={m}
                  className="flex items-center gap-2 cursor-pointer group"
                >
                  <input
                    type="radio"
                    name={`merge-method-${pr.number}`}
                    value={m}
                    checked={method === m}
                    onChange={() => setMethod(m)}
                    className="accent-purple-400"
                  />
                  <span className="text-[10px] text-[#a0a0a0] group-hover:text-white transition-colors">
                    {MERGE_METHOD_LABELS[m]}
                  </span>
                </label>
              ))}
            </div>
          </div>

          {/* Commit title (optional) */}
          <div>
            <label className="block text-[9px] text-[#666666] uppercase tracking-wider mb-1">
              Commit Title (Optional)
            </label>
            <input
              type="text"
              className="w-full bg-[#111111] border border-[#2a2a2a] rounded px-2 py-1 text-[10px] text-white placeholder-[#444444] focus:outline-none focus:border-[#3a3a3a] transition-colors"
              placeholder={`${method === 'squash' ? 'Squash' : 'Merge'} PR #${pr.number}: ${pr.title}`}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          {/* Action buttons */}
          <div className="flex gap-1.5 justify-end">
            <button
              onClick={handleMerge}
              disabled={isMerging}
              className="flex items-center gap-1 px-2.5 py-1 text-[10px] font-mono rounded bg-purple-600 hover:bg-purple-500 text-white transition-colors disabled:opacity-50 cursor-pointer"
            >
              {isMerging ? <Spinner size={10} /> : <GitMerge size={10} />}
              {isMerging ? 'Merging…' : 'Confirm Merge'}
            </button>
            <button
              onClick={() => setExpanded(false)}
              disabled={isMerging}
              className="px-2.5 py-1 text-[10px] font-mono rounded border border-[#2a2a2a] text-[#666666] hover:text-white hover:border-[#3a3a3a] transition-colors cursor-pointer"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Main Panel ────────────────────────────────────────────────────────────────

export default function PullRequestPanel({
  owner,
  repoName,
  currentBranch,
  onMergeComplete,
}: PullRequestPanelProps) {
  const [pulls, setPulls] = useState<PullRequest[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Branch-merge form state
  const [showMergeForm, setShowMergeForm]     = useState(false);
  const [baseBranch,    setBaseBranch]         = useState('main');
  const [headBranch,    setHeadBranch]         = useState(currentBranch);
  const [mergeMessage,  setMergeMessage]       = useState('');
  const [isMerging,     setIsMerging]          = useState(false);
  const [mergeSuccess,  setMergeSuccess]       = useState<string | null>(null);
  const [mergeError,    setMergeError]         = useState<string | null>(null);

  const fetchPulls = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await reposApi.listPulls(owner, repoName, 'all');
      setPulls(data || []);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load pull requests.');
    } finally {
      setIsLoading(false);
    }
  }, [owner, repoName]);

  useEffect(() => {
    fetchPulls();
  }, [fetchPulls]);

  // Sync headBranch when currentBranch changes in RepoView
  useEffect(() => {
    setHeadBranch(currentBranch);
  }, [currentBranch]);

  const handleBranchMerge = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsMerging(true);
    setMergeError(null);
    setMergeSuccess(null);

    const msg = mergeMessage.trim() || `Merge branch '${headBranch}' into '${baseBranch}'`;

    try {
      const result = await reposApi.mergeBranch(owner, repoName, baseBranch, headBranch, msg);
      setMergeSuccess(`Branch merged! Commit: ${result.commit_sha.slice(0, 7)}`);
      setMergeMessage('');
      onMergeComplete();
      fetchPulls();
      setTimeout(() => setMergeSuccess(null), 5000);
    } catch (err) {
      const apiErr = err as ApiError;
      setMergeError(apiErr.message ?? 'Failed to merge branches.');
    } finally {
      setIsMerging(false);
    }
  };

  // Derived counts
  const openCount   = pulls.filter(p => p.state === 'open').length;
  const closedCount = pulls.length - openCount;

  return (
    <div className="flex flex-1 flex-col overflow-hidden h-full">
      {/* Header */}
      <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] px-4 py-3 bg-[#111111]">
        <div className="flex items-center gap-2">
          <GitPullRequest size={14} className="text-[#a0a0a0]" />
          <span className="text-xs font-semibold text-white">Pull Requests</span>
          <span className="rounded border border-emerald-400/20 bg-emerald-400/5 px-1.5 py-0.5 font-mono text-[10px] text-emerald-400">
            {openCount} open
          </span>
          {closedCount > 0 && (
            <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666]">
              {closedCount} closed
            </span>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={fetchPulls}
            className="p-1 rounded text-[#666666] hover:text-white transition-colors cursor-pointer"
            title="Refresh"
          >
            <RefreshCw size={11} />
          </button>
          <button
            onClick={() => setShowMergeForm(!showMergeForm)}
            className="btn-secondary py-1 px-2 text-[10px] flex items-center gap-1.5 cursor-pointer"
          >
            <GitMerge size={12} />
            Merge Branch
          </button>
        </div>
      </div>

      {/* Branch Merge Form */}
      {showMergeForm && (
        <form
          onSubmit={handleBranchMerge}
          className="border-b border-[#2a2a2a] bg-[#161616] p-4 flex flex-col gap-3 shrink-0"
        >
          <div className="text-xs font-semibold text-white flex items-center gap-1.5">
            <GitMerge size={12} />
            Merge Branch
          </div>

          {mergeError && (
            <div className="text-[11px] text-red-400 bg-red-400/5 border border-red-400/10 px-3 py-2 rounded flex items-center gap-1.5">
              <AlertCircle size={12} className="shrink-0" />
              <span>{mergeError}</span>
            </div>
          )}
          {mergeSuccess && (
            <div className="text-[11px] text-green-400 bg-green-400/5 border border-green-400/10 px-3 py-2 rounded flex items-center gap-1.5">
              <CheckCircle2 size={12} className="shrink-0" />
              <span>{mergeSuccess}</span>
            </div>
          )}

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="block text-[10px] text-[#a0a0a0] uppercase mb-1">Base Branch</label>
              <input
                type="text"
                className="input text-xs"
                placeholder="main"
                value={baseBranch}
                onChange={(e) => setBaseBranch(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="block text-[10px] text-[#a0a0a0] uppercase mb-1">Head Branch (Merge from)</label>
              <input
                type="text"
                className="input text-xs"
                placeholder="branch-name"
                value={headBranch}
                onChange={(e) => setHeadBranch(e.target.value)}
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-[10px] text-[#a0a0a0] uppercase mb-1">Commit Message (Optional)</label>
            <input
              type="text"
              className="input text-xs"
              placeholder="e.g. Merge branch..."
              value={mergeMessage}
              onChange={(e) => setMergeMessage(e.target.value)}
            />
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="submit"
              disabled={isMerging}
              className="btn-primary py-1.5 px-3 text-xs"
            >
              {isMerging ? <Spinner size={12} /> : 'Merge'}
            </button>
            <button
              type="button"
              onClick={() => setShowMergeForm(false)}
              className="btn-secondary py-1.5 px-3 text-xs"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* Pull Requests list */}
      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-2">
        {isLoading ? (
          <div className="flex flex-1 items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : error ? (
          <div className="text-center py-8">
            <p className="text-xs text-[#a0a0a0] mb-2">{error}</p>
            <button onClick={fetchPulls} className="btn-secondary text-xs">
              Retry
            </button>
          </div>
        ) : pulls.length === 0 ? (
          <p className="py-8 text-center text-xs text-[#666666]">
            No pull requests in this repository.
          </p>
        ) : (
          pulls.map((pr) => {
            const isOpen   = pr.state === 'open';
            const isClosed = pr.state === 'closed';
            const isMerged = pr.state === 'merged';

            const stateColor = isMerged
              ? 'text-purple-400 border-purple-400/20 bg-purple-400/5'
              : isClosed
              ? 'text-[#666666] border-[#2a2a2a]'
              : 'text-emerald-400 border-emerald-400/20 bg-emerald-400/5';

            return (
              <div
                key={pr.id}
                className={[
                  'card p-3 flex flex-col gap-2 transition-all',
                  isOpen ? 'hover:border-purple-400/20' : 'hover:border-[#3a3a3a] opacity-70',
                ].join(' ')}
              >
                {/* Title + external link */}
                <div className="flex items-start justify-between gap-2">
                  <span className="text-xs font-medium text-white line-clamp-2">
                    {pr.title}
                  </span>
                  <a
                    href={pr.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[#666666] hover:text-white shrink-0 mt-0.5"
                    title="View on GitHub"
                  >
                    <ExternalLink size={12} />
                  </a>
                </div>

                {/* Meta row */}
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-mono text-[10px] text-[#666666]">
                    #{pr.number}
                  </span>
                  <span className={`px-1.5 py-0.5 rounded border text-[9px] font-mono ${stateColor}`}>
                    {pr.state}
                  </span>
                  <span
                    className="font-mono text-[9px] text-[#a0a0a0] truncate max-w-[120px]"
                    title={`${pr.base} ← ${pr.branch}`}
                  >
                    {pr.base} ← {pr.branch}
                  </span>
                </div>

                {/* Creator */}
                <div className="flex items-center gap-1.5 border-t border-[#1a1a1a] pt-1.5">
                  <img
                    src={pr.avatar_url}
                    alt={pr.creator}
                    className="h-4 w-4 rounded-full border border-[#3a3a3a]"
                  />
                  <span className="text-[10px] text-[#666666]">
                    by {pr.creator}
                  </span>
                </div>

                {/* Merge widget — only for open PRs */}
                {isOpen && (
                  <PrMergeWidget
                    owner={owner}
                    repoName={repoName}
                    pr={pr}
                    onMergeComplete={() => {
                      onMergeComplete();
                      fetchPulls();
                    }}
                  />
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

/**
 * PullRequestPanel — Displays Pull Requests and allows branch merging.
 */

import { useState, useEffect } from 'react';
import { GitPullRequest, GitMerge, ExternalLink, AlertCircle, CheckCircle2 } from 'lucide-react';
import { reposApi } from '../api/client';
import type { PullRequest, ApiError } from '../types';
import Spinner from './ui/Spinner';

interface PullRequestPanelProps {
  owner: string;
  repoName: string;
  currentBranch: string;
  onMergeComplete: () => void;
}

export default function PullRequestPanel({
  owner,
  repoName,
  currentBranch,
  onMergeComplete,
}: PullRequestPanelProps) {
  const [pulls, setPulls] = useState<PullRequest[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Merge form state
  const [showMergeForm, setShowMergeForm] = useState(false);
  const [baseBranch, setBaseBranch] = useState('main');
  const [headBranch, setHeadBranch] = useState(currentBranch);
  const [mergeMessage, setMergeMessage] = useState('');
  const [isMerging, setIsMerging] = useState(false);
  const [mergeSuccess, setMergeSuccess] = useState<string | null>(null);
  const [mergeError, setMergeError] = useState<string | null>(null);

  const fetchPulls = async () => {
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
  };

  useEffect(() => {
    fetchPulls();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [owner, repoName]);

  // Sync headBranch when currentBranch changes in RepoView
  useEffect(() => {
    setHeadBranch(currentBranch);
  }, [currentBranch]);

  const handleMerge = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsMerging(true);
    setMergeError(null);
    setMergeSuccess(null);

    const msg = mergeMessage.trim() || `Merge branch '${headBranch}' into '${baseBranch}'`;

    try {
      const result = await reposApi.mergeBranch(owner, repoName, baseBranch, headBranch, msg);
      setMergeSuccess(`Branch merged successfully! Commit: ${result.commit_sha.slice(0, 7)}`);
      setMergeMessage('');
      // Refresh branch view & pull requests
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

  return (
    <div className="flex flex-1 flex-col overflow-hidden h-full">
      {/* Header / Merge Trigger */}
      <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] px-4 py-3 bg-[#111111]">
        <div className="flex items-center gap-2">
          <GitPullRequest size={14} className="text-[#a0a0a0]" />
          <span className="text-xs font-semibold text-white">Pull Requests</span>
          <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666]">
            {pulls.length}
          </span>
        </div>

        <button
          onClick={() => setShowMergeForm(!showMergeForm)}
          className="btn-secondary py-1 px-2 text-[10px] flex items-center gap-1.5 cursor-pointer"
        >
          <GitMerge size={12} />
          Merge Branch
        </button>
      </div>

      {/* Merge Form (Slide down / Accordion) */}
      {showMergeForm && (
        <form
          onSubmit={handleMerge}
          className="border-b border-[#2a2a2a] bg-[#161616] p-4 flex flex-col gap-3 shrink-0 animate__animated animate__fadeInDown"
          style={{ animationDuration: '0.2s' }}
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
            <button
              onClick={fetchPulls}
              className="btn-secondary text-xs"
            >
              Retry
            </button>
          </div>
        ) : pulls.length === 0 ? (
          <p className="py-8 text-center text-xs text-[#666666]">
            No pull requests in this repository.
          </p>
        ) : (
          pulls.map((pr) => {
            const isClosed = pr.state === 'closed';
            const isMerged = pr.state === 'merged'; // wait, GitHub lists state as closed even if merged, but backend lists state as merged if we check.
            const stateColor = isClosed
              ? 'text-[#666666] border-[#2a2a2a]'
              : isMerged
              ? 'text-purple-400 border-purple-400/20 bg-purple-400/5'
              : 'text-green-400 border-green-400/20 bg-green-400/5';

            return (
              <div
                key={pr.id}
                className="card p-3 flex flex-col gap-2 hover:border-[#3a3a3a] transition-all"
              >
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

                <div className="flex items-center gap-1.5 border-t border-[#2a2a2a] pt-2">
                  <img
                    src={pr.avatar_url}
                    alt={pr.creator}
                    className="h-4 w-4 rounded-full border border-[#3a3a3a]"
                  />
                  <span className="text-[10px] text-[#666666]">
                    by {pr.creator}
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

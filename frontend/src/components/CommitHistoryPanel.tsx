/**
 * CommitHistoryPanel — Displays the commit history for the current branch.
 */

import { useState, useEffect } from 'react';
import { GitCommit, Calendar, User } from 'lucide-react';
import { reposApi } from '../api/client';
import type { Commit, ApiError } from '../types';
import Spinner from './ui/Spinner';

interface CommitHistoryPanelProps {
  owner: string;
  repoName: string;
  currentBranch: string;
}

export default function CommitHistoryPanel({
  owner,
  repoName,
  currentBranch,
}: CommitHistoryPanelProps) {
  const [commits, setCommits] = useState<Commit[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCommits = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await reposApi.listCommits(owner, repoName, currentBranch);
      setCommits(data || []);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load commit history.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchCommits();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [owner, repoName, currentBranch]);

  // Format date nicely
  const formatDate = (dateStr: string) => {
    if (!dateStr) return '';
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString('tr-TR', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden h-full">
      {/* Panel header */}
      <div className="flex h-9 shrink-0 items-center border-b border-[#2a2a2a] px-3 bg-[#111111] gap-2">
        <GitCommit size={13} className="text-[#a0a0a0]" />
        <span className="text-xs font-semibold text-white">Commits ({currentBranch})</span>
        <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666] ml-auto">
          {commits.length}
        </span>
      </div>

      {/* Commit list */}
      <div className="flex-1 overflow-y-auto py-2 px-3 flex flex-col gap-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : error ? (
          <div className="text-center py-6">
            <p className="text-xs text-[#666666] mb-2">{error}</p>
            <button
              onClick={fetchCommits}
              className="btn-secondary text-[10px] py-1 px-2"
            >
              Yeniden Dene
            </button>
          </div>
        ) : commits.length === 0 ? (
          <p className="py-8 text-center text-xs text-[#666666]">
            Bu dalda commit bulunamadı.
          </p>
        ) : (
          commits.map((commit) => (
            <div
              key={commit.sha}
              className="group border border-[#2a2a2a] rounded bg-[#161616] p-2.5 flex flex-col gap-1.5 hover:border-[#3a3a3a] transition-all"
            >
              {/* Commit Message */}
              <div className="text-xs text-white font-mono break-words line-clamp-2" title={commit.message}>
                {commit.message}
              </div>

              {/* SHA & Date */}
              <div className="flex items-center justify-between text-[9px] font-mono text-[#666666] border-b border-[#2a2a2a]/50 pb-1.5">
                <span className="text-white/60 bg-white/5 border border-white/10 px-1 py-0.5 rounded">
                  {commit.sha.slice(0, 7)}
                </span>
                <span className="flex items-center gap-1">
                  <Calendar size={10} />
                  {formatDate(commit.date)}
                </span>
              </div>

              {/* Author info */}
              <div className="flex items-center gap-2">
                {commit.avatar_url ? (
                  <img
                    src={commit.avatar_url}
                    alt={commit.author}
                    className="h-4 w-4 rounded-full border border-[#3a3a3a] shrink-0"
                  />
                ) : (
                  <User size={12} className="text-[#666666] shrink-0" />
                )}
                <span className="text-[10px] text-[#a0a0a0] truncate font-medium">
                  {commit.author}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

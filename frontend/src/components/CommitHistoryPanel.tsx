/**
 * CommitHistoryPanel — Displays the commit history for the current branch.
 */

import { useState, useEffect, useMemo } from 'react';
import {
  AlertTriangle,
  Calendar,
  CheckCircle2,
  CircleHelp,
  Clock3,
  ExternalLink,
  GitCommit,
  MinusCircle,
  ShieldCheck,
  User,
  UserCheck,
  XCircle,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { Commit, ApiError, CommitCheckState } from '../types';
import Spinner from './ui/Spinner';

interface CommitHistoryPanelProps {
  owner: string;
  repoName: string;
  currentBranch: string;
}

type CommitFilter = 'all' | 'attention' | 'passed';

const checkStateMeta: Record<
  CommitCheckState,
  {
    label: string;
    className: string;
    Icon: typeof CheckCircle2;
  }
> = {
  success: {
    label: 'Passed',
    className: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
    Icon: CheckCircle2,
  },
  failure: {
    label: 'Failed',
    className: 'border-red-500/30 bg-red-500/10 text-red-300',
    Icon: XCircle,
  },
  error: {
    label: 'Error',
    className: 'border-red-500/30 bg-red-500/10 text-red-300',
    Icon: AlertTriangle,
  },
  pending: {
    label: 'Running',
    className: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
    Icon: Clock3,
  },
  none: {
    label: 'No checks',
    className: 'border-[#333333] bg-white/5 text-[#888888]',
    Icon: MinusCircle,
  },
  unknown: {
    label: 'Unknown',
    className: 'border-[#333333] bg-white/5 text-[#888888]',
    Icon: CircleHelp,
  },
};

const filterOptions: { id: CommitFilter; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'attention', label: 'Issues' },
  { id: 'passed', label: 'Passed' },
];

const COMMITS_PER_PAGE = 50;

export default function CommitHistoryPanel({
  owner,
  repoName,
  currentBranch,
}: CommitHistoryPanelProps) {
  const [commits, setCommits] = useState<Commit[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<CommitFilter>('all');
  const [nextPage, setNextPage] = useState<number>(0);

  const fetchCommits = async (page = 1) => {
    const loadingMore = page > 1;
    if (loadingMore) {
      setIsLoadingMore(true);
    } else {
      setIsLoading(true);
      setNextPage(0);
    }
    setError(null);
    try {
      const data = await reposApi.listCommits(
        owner,
        repoName,
        currentBranch,
        page,
        COMMITS_PER_PAGE
      );
      setCommits((current) => {
        if (!loadingMore) return data.commits || [];

        const seen = new Set(current.map((commit) => commit.sha));
        const newCommits = (data.commits || []).filter((commit) => !seen.has(commit.sha));
        return [...current, ...newCommits];
      });
      setNextPage(data.next_page || 0);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load commit history.');
    } finally {
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  useEffect(() => {
    fetchCommits();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [owner, repoName, currentBranch]);

  const visibleCommits = useMemo(() => {
    if (filter === 'all') return commits;
    if (filter === 'passed') {
      return commits.filter((commit) => commit.check_state === 'success');
    }
    return commits.filter((commit) =>
      ['failure', 'error', 'pending', 'unknown'].includes(commit.check_state)
    );
  }, [commits, filter]);

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

  const renderCheckBadge = (state: CommitCheckState | undefined, total: number) => {
    const meta = checkStateMeta[state ?? 'unknown'];
    const Icon = meta.Icon;
    return (
      <span
        className={[
          'inline-flex h-5 items-center gap-1 rounded border px-1.5 text-[9px] font-semibold',
          meta.className,
        ].join(' ')}
        title={total > 0 ? `${total} checks/statuses` : meta.label}
      >
        <Icon size={10} />
        {meta.label}
      </span>
    );
  };

  const renderRunState = (status: string, conclusion: string) => {
    if (status && status !== 'completed') return status;
    return conclusion || status || 'unknown';
  };

  return (
    <div className="flex flex-1 flex-col overflow-hidden h-full">
      {/* Panel header */}
      <div className="flex h-9 shrink-0 items-center border-b border-[#2a2a2a] px-3 bg-[#111111] gap-2">
        <GitCommit size={13} className="text-[#a0a0a0]" />
        <span className="text-xs font-semibold text-white">Commits ({currentBranch})</span>
        <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666] ml-auto">
          {visibleCommits.length}/{commits.length}{nextPage ? '+' : ''}
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
              onClick={() => fetchCommits()}
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
          <>
            <div className="grid grid-cols-3 gap-1 rounded border border-[#2a2a2a] bg-[#111111] p-1">
              {filterOptions.map((option) => (
                <button
                  key={option.id}
                  onClick={() => setFilter(option.id)}
                  className={[
                    'h-6 rounded text-[10px] font-semibold transition-colors',
                    filter === option.id
                      ? 'bg-white text-black'
                      : 'text-[#888888] hover:bg-white/5 hover:text-white',
                  ].join(' ')}
                >
                  {option.label}
                </button>
              ))}
            </div>

            {visibleCommits.length === 0 ? (
              <p className="py-8 text-center text-xs text-[#666666]">
                Bu filtrede commit yok.
              </p>
            ) : (
              visibleCommits.map((commit) => {
                const [subject, ...bodyLines] = commit.message.split('\n');
                const body = bodyLines.join('\n').trim();
                const checkRuns = commit.check_runs ?? [];
                const statuses = commit.statuses ?? [];
                const hasCheckDetails = checkRuns.length > 0 || statuses.length > 0 || commit.check_error;

                return (
                  <div
                    key={commit.sha}
                    className="group border border-[#2a2a2a] rounded bg-[#161616] p-2.5 flex flex-col gap-2 hover:border-[#3a3a3a] transition-all"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <div className="text-xs text-white font-mono break-words line-clamp-2" title={commit.message}>
                          {subject || commit.message}
                        </div>
                        {body && (
                          <p className="mt-1 line-clamp-2 text-[10px] leading-4 text-[#888888]" title={body}>
                            {body}
                          </p>
                        )}
                      </div>
                      {commit.html_url && (
                        <a
                          href={commit.html_url}
                          target="_blank"
                          rel="noreferrer"
                          className="mt-0.5 shrink-0 rounded border border-[#2a2a2a] p-1 text-[#777777] transition-colors hover:border-[#4a4a4a] hover:text-white"
                          title="Open commit"
                        >
                          <ExternalLink size={11} />
                        </a>
                      )}
                    </div>

                    <div className="flex items-center justify-between gap-2 border-b border-[#2a2a2a]/50 pb-1.5">
                      <span className="text-white/70 bg-white/5 border border-white/10 px-1 py-0.5 rounded font-mono text-[9px]">
                        {commit.sha.slice(0, 7)}
                      </span>
                      {renderCheckBadge(commit.check_state, commit.check_total)}
                    </div>

                    <div className="flex items-center justify-between gap-2 text-[9px] font-mono text-[#666666]">
                      <span className="flex min-w-0 items-center gap-1">
                        <Calendar size={10} className="shrink-0" />
                        <span className="truncate">{formatDate(commit.date)}</span>
                      </span>
                      <span
                        className={[
                          'inline-flex shrink-0 items-center gap-1 rounded border px-1 py-0.5',
                          commit.verified
                            ? 'border-emerald-500/25 text-emerald-300'
                            : 'border-[#333333] text-[#777777]',
                        ].join(' ')}
                        title={commit.verification_reason || 'No signature data'}
                      >
                        <ShieldCheck size={9} />
                        {commit.verified ? 'Verified' : 'Unsigned'}
                      </span>
                    </div>

                    <div className="flex flex-col gap-1">
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
                        <span className="text-[10px] text-[#a0a0a0] truncate font-medium" title={commit.author_email}>
                          {commit.author || 'Unknown author'}
                        </span>
                      </div>

                      {commit.committer && commit.committer !== commit.author && (
                        <div className="flex items-center gap-2 pl-0.5">
                          {commit.committer_avatar_url ? (
                            <img
                              src={commit.committer_avatar_url}
                              alt={commit.committer}
                              className="h-3.5 w-3.5 rounded-full border border-[#3a3a3a] shrink-0"
                            />
                          ) : (
                            <UserCheck size={11} className="text-[#666666] shrink-0" />
                          )}
                          <span className="text-[9px] text-[#777777] truncate" title={commit.committer_email}>
                            committed by {commit.committer}
                          </span>
                        </div>
                      )}
                    </div>

                    {hasCheckDetails && (
                      <details className="rounded border border-[#242424] bg-[#111111]">
                        <summary className="flex h-7 cursor-pointer list-none items-center justify-between px-2 text-[10px] font-semibold text-[#a0a0a0] [&::-webkit-details-marker]:hidden">
                          <span>Checks</span>
                          <span className="font-mono text-[#666666]">{commit.check_total}</span>
                        </summary>
                        <div className="flex flex-col gap-1 border-t border-[#242424] p-2">
                          {checkRuns.map((run) => (
                            <a
                              key={`${commit.sha}-${run.name}-${run.details_url}`}
                              href={run.details_url || undefined}
                              target="_blank"
                              rel="noreferrer"
                              className="flex min-h-6 items-center justify-between gap-2 rounded px-1.5 text-[9px] text-[#888888] hover:bg-white/5 hover:text-white"
                            >
                              <span className="truncate">{run.name || 'Unnamed check'}</span>
                              <span className="shrink-0 font-mono">{renderRunState(run.status, run.conclusion)}</span>
                            </a>
                          ))}

                          {statuses.map((status) => (
                            <a
                              key={`${commit.sha}-${status.context}-${status.target_url}`}
                              href={status.target_url || undefined}
                              target="_blank"
                              rel="noreferrer"
                              className="flex min-h-6 items-center justify-between gap-2 rounded px-1.5 text-[9px] text-[#888888] hover:bg-white/5 hover:text-white"
                            >
                              <span className="truncate" title={status.description}>
                                {status.context || 'Commit status'}
                              </span>
                              <span className="shrink-0 font-mono">{status.state || 'unknown'}</span>
                            </a>
                          ))}

                          {commit.check_error && (
                            <p className="rounded border border-amber-500/20 bg-amber-500/10 px-2 py-1 text-[9px] leading-4 text-amber-200">
                              {commit.check_error}
                            </p>
                          )}
                        </div>
                      </details>
                    )}
                  </div>
                );
              })
            )}

            {nextPage > 0 && (
              <button
                onClick={() => fetchCommits(nextPage)}
                disabled={isLoadingMore}
                className="btn-secondary mt-1 min-h-8 justify-center py-1.5 text-[10px] disabled:opacity-50"
              >
                {isLoadingMore ? <Spinner size={12} /> : 'Load more commits'}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

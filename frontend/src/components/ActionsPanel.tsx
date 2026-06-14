/**
 * ActionsPanel — Displays GitHub Actions workflows and recent workflow runs.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock3,
  ExternalLink,
  GitBranch,
  Hash,
  MinusCircle,
  RefreshCw,
  XCircle,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { ActionWorkflow, ActionWorkflowRun, ApiError } from '../types';
import Spinner from './ui/Spinner';

interface ActionsPanelProps {
  owner: string;
  repoName: string;
  currentBranch: string;
}

type BranchFilter = 'all' | 'current';

const branchFilters: { id: BranchFilter; label: string }[] = [
  { id: 'all', label: 'All runs' },
  { id: 'current', label: 'Current branch' },
];

function runStateMeta(status: string, conclusion: string) {
  if (status && status !== 'completed') {
    return {
      label: status.replace(/_/g, ' '),
      className: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
      Icon: Clock3,
    };
  }

  switch (conclusion) {
    case 'success':
      return {
        label: 'passed',
        className: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
        Icon: CheckCircle2,
      };
    case 'failure':
    case 'timed_out':
    case 'action_required':
      return {
        label: conclusion.replace(/_/g, ' '),
        className: 'border-red-500/30 bg-red-500/10 text-red-300',
        Icon: XCircle,
      };
    case 'cancelled':
      return {
        label: 'cancelled',
        className: 'border-[#333333] bg-white/5 text-[#888888]',
        Icon: MinusCircle,
      };
    case 'skipped':
    case 'neutral':
      return {
        label: conclusion,
        className: 'border-[#333333] bg-white/5 text-[#a0a0a0]',
        Icon: MinusCircle,
      };
    default:
      return {
        label: conclusion || status || 'unknown',
        className: 'border-[#333333] bg-white/5 text-[#888888]',
        Icon: AlertTriangle,
      };
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '';
  try {
    return new Date(dateStr).toLocaleDateString('tr-TR', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return dateStr;
  }
}

function RunBadge({ run }: { run?: ActionWorkflowRun }) {
  if (!run) {
    return (
      <span className="inline-flex h-5 items-center gap-1 rounded border border-[#333333] bg-white/5 px-1.5 text-[9px] font-semibold text-[#888888]">
        <MinusCircle size={10} />
        no runs
      </span>
    );
  }

  const meta = runStateMeta(run.status, run.conclusion);
  const Icon = meta.Icon;
  return (
    <span
      className={[
        'inline-flex h-5 items-center gap-1 rounded border px-1.5 text-[9px] font-semibold capitalize',
        meta.className,
      ].join(' ')}
      title={`${run.status || 'unknown'} ${run.conclusion || ''}`.trim()}
    >
      <Icon size={10} />
      {meta.label}
    </span>
  );
}

export default function ActionsPanel({ owner, repoName, currentBranch }: ActionsPanelProps) {
  const [workflows, setWorkflows] = useState<ActionWorkflow[]>([]);
  const [runs, setRuns] = useState<ActionWorkflowRun[]>([]);
  const [branchFilter, setBranchFilter] = useState<BranchFilter>('all');
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const latestRunByWorkflow = useMemo(() => {
    const latest = new Map<number, ActionWorkflowRun>();
    runs.forEach((run) => {
      if (!latest.has(run.workflow_id)) {
        latest.set(run.workflow_id, run);
      }
    });
    return latest;
  }, [runs]);

  const fetchActions = useCallback(async (refreshOnly = false) => {
    if (!owner || !repoName) return;
    if (refreshOnly) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);

    try {
      const params =
        branchFilter === 'current' && currentBranch
          ? { branch: currentBranch }
          : undefined;
      const [workflowData, runData] = await Promise.all([
        reposApi.listActionWorkflows(owner, repoName),
        reposApi.listActionRuns(owner, repoName, params),
      ]);
      setWorkflows(workflowData || []);
      setRuns(runData || []);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load GitHub Actions.');
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, [owner, repoName, branchFilter, currentBranch]);

  useEffect(() => {
    fetchActions();
  }, [fetchActions]);

  return (
    <div className="flex h-full flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] bg-[#111111] px-4 py-3">
        <div className="flex items-center gap-2">
          <Activity size={14} className="text-[#a0a0a0]" />
          <span className="text-xs font-semibold text-white">Actions</span>
          <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[10px] text-[#666666]">
            {workflows.length}
          </span>
        </div>

        <button
          onClick={() => fetchActions(true)}
          disabled={isRefreshing}
          className="btn-secondary px-2 py-1 text-[10px] disabled:opacity-50"
          title="Refresh Actions"
        >
          <RefreshCw size={12} className={isRefreshing ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      <div className="border-b border-[#2a2a2a] bg-[#111111] px-3 py-2">
        <div className="grid grid-cols-2 gap-1 rounded border border-[#2a2a2a] bg-[#0f0f0f] p-1">
          {branchFilters.map((filter) => (
            <button
              key={filter.id}
              onClick={() => setBranchFilter(filter.id)}
              className={[
                'h-7 rounded text-[10px] font-semibold transition-colors',
                branchFilter === filter.id
                  ? 'bg-white text-black'
                  : 'text-[#888888] hover:bg-white/5 hover:text-white',
              ].join(' ')}
            >
              {filter.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-3">
        {isLoading ? (
          <div className="flex flex-1 items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : error ? (
          <div className="py-8 text-center">
            <p className="mb-2 text-xs text-[#a0a0a0]">{error}</p>
            <button onClick={() => fetchActions()} className="btn-secondary text-xs">
              Retry
            </button>
          </div>
        ) : workflows.length === 0 ? (
          <p className="py-8 text-center text-xs text-[#666666]">
            No GitHub Actions workflows in this repository.
          </p>
        ) : (
          <div className="flex flex-col gap-3">
            <section className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">
                  Workflows
                </span>
                <span className="font-mono text-[10px] text-[#666666]">
                  {workflows.length}
                </span>
              </div>

              {workflows.map((workflow) => {
                const latestRun = latestRunByWorkflow.get(workflow.id);
                return (
                  <div
                    key={workflow.id}
                    className="rounded border border-[#2a2a2a] bg-[#161616] p-3 transition-colors hover:border-[#3a3a3a]"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-xs font-semibold text-white" title={workflow.name}>
                          {workflow.name}
                        </p>
                        <p className="mt-1 truncate font-mono text-[9px] text-[#666666]" title={workflow.path}>
                          {workflow.path}
                        </p>
                      </div>
                      {workflow.html_url && (
                        <a
                          href={workflow.html_url}
                          target="_blank"
                          rel="noreferrer"
                          className="shrink-0 rounded border border-[#2a2a2a] p-1 text-[#777777] transition-colors hover:border-[#4a4a4a] hover:text-white"
                          title="Open workflow"
                        >
                          <ExternalLink size={11} />
                        </a>
                      )}
                    </div>

                    <div className="mt-2 flex items-center justify-between gap-2">
                      <span className="rounded border border-[#333333] px-1.5 py-0.5 font-mono text-[9px] text-[#888888]">
                        {workflow.state || 'unknown'}
                      </span>
                      <RunBadge run={latestRun} />
                    </div>
                  </div>
                );
              })}
            </section>

            <section className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">
                  Recent Runs
                </span>
                <span className="font-mono text-[10px] text-[#666666]">{runs.length}</span>
              </div>

              {runs.length === 0 ? (
                <p className="rounded border border-[#242424] bg-[#111111] px-3 py-6 text-center text-xs text-[#666666]">
                  No runs match this filter.
                </p>
              ) : (
                runs.map((run) => (
                  <div
                    key={run.id}
                    className="rounded border border-[#242424] bg-[#111111] p-2.5 transition-colors hover:border-[#333333]"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <p className="line-clamp-2 text-xs font-medium text-white" title={run.display_title || run.name}>
                          {run.display_title || run.name || 'Workflow run'}
                        </p>
                        <p className="mt-1 truncate text-[10px] text-[#888888]">
                          {run.name || 'Actions'}
                        </p>
                      </div>
                      {run.html_url && (
                        <a
                          href={run.html_url}
                          target="_blank"
                          rel="noreferrer"
                          className="mt-0.5 shrink-0 text-[#666666] transition-colors hover:text-white"
                          title="Open run"
                        >
                          <ExternalLink size={12} />
                        </a>
                      )}
                    </div>

                    <div className="mt-2 flex items-center justify-between gap-2">
                      <RunBadge run={run} />
                      <span className="font-mono text-[9px] text-[#666666]">
                        {formatDate(run.run_started_at || run.created_at)}
                      </span>
                    </div>

                    <div className="mt-2 flex flex-wrap items-center gap-2 border-t border-[#242424] pt-2 text-[9px] text-[#777777]">
                      <span className="inline-flex max-w-[110px] items-center gap-1 truncate font-mono" title={run.head_branch}>
                        <GitBranch size={10} className="shrink-0" />
                        <span className="truncate">{run.head_branch || 'unknown'}</span>
                      </span>
                      {run.head_sha && (
                        <span className="inline-flex items-center gap-1 font-mono">
                          <Hash size={10} />
                          {run.head_sha.slice(0, 7)}
                        </span>
                      )}
                      <span className="font-mono">{run.event}</span>
                    </div>

                    {run.actor && (
                      <div className="mt-2 flex items-center gap-1.5">
                        {run.actor_avatar_url && (
                          <img
                            src={run.actor_avatar_url}
                            alt={run.actor}
                            className="h-4 w-4 rounded-full border border-[#3a3a3a]"
                          />
                        )}
                        <span className="truncate text-[10px] text-[#666666]">
                          by {run.actor}
                        </span>
                      </div>
                    )}
                  </div>
                ))
              )}
            </section>
          </div>
        )}
      </div>
    </div>
  );
}

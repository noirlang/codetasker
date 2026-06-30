import { useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  BarChart3,
  DollarSign,
  FileWarning,
  Plus,
  RefreshCw,
  ShieldAlert,
} from 'lucide-react';
import { debtApi } from '../api/client';
import { useTaskStore } from '../store/taskStore';
import type { ApiError, DebtAnalysis, DebtHotspot, DebtLevel } from '../types';
import Spinner from './ui/Spinner';

interface DebtPanelProps {
  owner: string;
  repoName: string;
  repoId: number;
}

const levelStyles: Record<DebtLevel, { text: string; bg: string; bar: string }> = {
  LOW: { text: 'text-emerald-300', bg: 'bg-emerald-400/10 border-emerald-400/20', bar: 'bg-emerald-300' },
  MEDIUM: { text: 'text-amber-300', bg: 'bg-amber-400/10 border-amber-400/20', bar: 'bg-amber-300' },
  HIGH: { text: 'text-orange-300', bg: 'bg-orange-400/10 border-orange-400/20', bar: 'bg-orange-300' },
  CRITICAL: { text: 'text-red-300', bg: 'bg-red-400/10 border-red-400/20', bar: 'bg-red-300' },
};

function formatMoney(value: number) {
  return `$${Math.round(value).toLocaleString()}`;
}

function HotspotCard({
  hotspot,
  owner,
  repoName,
  onCreated,
}: {
  hotspot: DebtHotspot;
  owner: string;
  repoName: string;
  onCreated: () => void;
}) {
  const [creating, setCreating] = useState(false);
  const suggestion = hotspot.suggested_tasks[0];
  const created = suggestion?.status === 'created';
  const style = levelStyles[hotspot.level];

  const handleCreate = async () => {
    if (!suggestion?.id || creating || created) return;
    setCreating(true);
    try {
      await debtApi.createTask(owner, repoName, suggestion.id);
      onCreated();
    } catch (err) {
      const apiErr = err as ApiError;
      alert(apiErr.message ?? 'Failed to create debt task.');
    } finally {
      setCreating(false);
    }
  };

  return (
    <article className="rounded border border-[#242424] bg-[#101010] p-3">
      <div className="mb-2 flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate font-mono text-[11px] text-white" title={hotspot.file}>
            {hotspot.file}
          </p>
          <div className="mt-1 flex items-center gap-1.5">
            <span className={`rounded border px-1.5 py-0.5 text-[9px] font-mono ${style.bg} ${style.text}`}>
              {hotspot.level}
            </span>
            <span className="font-mono text-[9px] text-[#666666]">
              {formatMoney(hotspot.estimated_monthly_cost)}/mo
            </span>
          </div>
        </div>
        <span className="font-mono text-lg font-semibold text-white">{hotspot.debt_score}</span>
      </div>

      <div className="h-1.5 overflow-hidden rounded bg-[#242424]">
        <div
          className={`h-full ${style.bar}`}
          style={{ width: `${Math.min(100, Math.max(0, hotspot.debt_score))}%` }}
        />
      </div>

      <div className="mt-3 grid grid-cols-3 gap-2 text-center">
        <div className="rounded border border-[#1f1f1f] bg-[#0b0b0b] px-2 py-1.5">
          <p className="font-mono text-[11px] text-white">{hotspot.metrics.commit_count}</p>
          <p className="text-[9px] text-[#666666]">commits</p>
        </div>
        <div className="rounded border border-[#1f1f1f] bg-[#0b0b0b] px-2 py-1.5">
          <p className="font-mono text-[11px] text-white">{hotspot.metrics.total_churn}</p>
          <p className="text-[9px] text-[#666666]">churn</p>
        </div>
        <div className="rounded border border-[#1f1f1f] bg-[#0b0b0b] px-2 py-1.5">
          <p className="font-mono text-[11px] text-white">{hotspot.metrics.cyclomatic_complexity_estimate}</p>
          <p className="text-[9px] text-[#666666]">complexity</p>
        </div>
      </div>

      <div className="mt-3 flex flex-col gap-1">
        {hotspot.reasons.slice(0, 3).map((reason) => (
          <div key={reason} className="flex items-start gap-1.5 text-[10px] text-[#a0a0a0]">
            <AlertTriangle size={10} className="mt-0.5 shrink-0 text-[#666666]" />
            <span className="leading-4">{reason}</span>
          </div>
        ))}
      </div>

      {suggestion && (
        <button
          onClick={handleCreate}
          disabled={creating || created}
          className="mt-3 flex w-full items-center justify-center gap-1.5 rounded border border-[#2a2a2a] bg-white/[0.04] px-3 py-2 text-[10px] font-semibold text-[#a0a0a0] transition-all hover:border-[#3a3a3a] hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
        >
          {creating ? <Spinner size={11} /> : <Plus size={11} />}
          {created ? 'Task created' : 'Create task'}
        </button>
      )}
    </article>
  );
}

export default function DebtPanel({ owner, repoName, repoId }: DebtPanelProps) {
  const [analysis, setAnalysis] = useState<DebtAnalysis | null>(null);
  const [loading, setLoading] = useState(true);
  const [analyzing, setAnalyzing] = useState(false);
  const [creatingAll, setCreatingAll] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [days, setDays] = useState(90);
  const [hourlyCost, setHourlyCost] = useState(35);
  const fetchTasks = useTaskStore((s) => s.fetchTasks);

  const highPriorityCount = useMemo(() => {
    if (!analysis) return 0;
    return analysis.hotspots.filter((h) => h.level === 'HIGH' || h.level === 'CRITICAL').length;
  }, [analysis]);

  const loadLatest = async () => {
    if (!owner || !repoName) return;
    setLoading(true);
    setError(null);
    try {
      const latest = await debtApi.getLatest(owner, repoName);
      setAnalysis(latest);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load debt analysis.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLatest();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [owner, repoName]);

  const handleAnalyze = async () => {
    setAnalyzing(true);
    setError(null);
    try {
      const result = await debtApi.analyze(owner, repoName, {
        days,
        hourly_cost: hourlyCost,
      });
      setAnalysis(result);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Debt analysis failed.');
    } finally {
      setAnalyzing(false);
    }
  };

  const handleCreateAll = async () => {
    if (!analysis || creatingAll) return;
    setCreatingAll(true);
    try {
      await debtApi.createAllHighPriorityTasks(owner, repoName);
      if (repoId > 0) {
        await fetchTasks(repoId);
      }
      await loadLatest();
    } catch (err) {
      const apiErr = err as ApiError;
      alert(apiErr.message ?? 'Failed to create debt tasks.');
    } finally {
      setCreatingAll(false);
    }
  };

  const handleTaskCreated = async () => {
    if (repoId > 0) {
      await fetchTasks(repoId);
    }
    await loadLatest();
  };

  const topHotspots = analysis?.hotspots.slice(0, 10) ?? [];

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Spinner size={24} />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="shrink-0 border-b border-[#2a2a2a] p-4">
        <div className="mb-3 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <ShieldAlert size={14} className="text-[#a0a0a0]" />
            <span className="text-xs font-medium text-white">Technical Debt</span>
          </div>
          <button
            onClick={handleAnalyze}
            disabled={analyzing}
            className="btn-secondary flex items-center gap-1.5 px-2 py-1 text-[10px] disabled:opacity-50"
          >
            <RefreshCw size={11} className={analyzing ? 'animate-spin' : ''} />
            Analyze
          </button>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <label className="rounded border border-[#242424] bg-[#0d0d0d] px-2 py-1.5">
            <span className="text-[9px] text-[#666666]">Days</span>
            <input
              type="number"
              min={7}
              max={365}
              value={days}
              onChange={(e) => setDays(Number(e.target.value))}
              className="mt-0.5 w-full bg-transparent font-mono text-xs text-white outline-none"
            />
          </label>
          <label className="rounded border border-[#242424] bg-[#0d0d0d] px-2 py-1.5">
            <span className="text-[9px] text-[#666666]">Hourly cost</span>
            <input
              type="number"
              min={1}
              value={hourlyCost}
              onChange={(e) => setHourlyCost(Number(e.target.value))}
              className="mt-0.5 w-full bg-transparent font-mono text-xs text-white outline-none"
            />
          </label>
        </div>
      </div>

      {error && (
        <div className="border-b border-[#2a2a2a] px-4 py-3">
          <p className="text-xs text-red-300">{error}</p>
        </div>
      )}

      {!analysis ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
          <FileWarning size={26} className="text-[#666666]" />
          <p className="text-xs text-[#a0a0a0]">No debt analysis yet.</p>
          <button
            onClick={handleAnalyze}
            disabled={analyzing}
            className="btn-secondary flex items-center gap-1.5 px-3 py-2 text-xs"
          >
            {analyzing ? <Spinner size={12} /> : <BarChart3 size={12} />}
            Run analysis
          </button>
        </div>
      ) : (
        <>
          <div className="grid shrink-0 grid-cols-3 gap-2 border-b border-[#2a2a2a] p-4">
            <div className="rounded border border-[#242424] bg-[#0d0d0d] p-2">
              <DollarSign size={12} className="mb-1 text-[#666666]" />
              <p className="font-mono text-sm font-semibold text-white">
                {formatMoney(analysis.summary.estimated_monthly_cost)}
              </p>
              <p className="text-[9px] text-[#666666]">monthly</p>
            </div>
            <div className="rounded border border-red-400/20 bg-red-400/10 p-2">
              <p className="font-mono text-sm font-semibold text-red-300">{analysis.summary.critical}</p>
              <p className="text-[9px] text-red-300/70">critical</p>
            </div>
            <div className="rounded border border-orange-400/20 bg-orange-400/10 p-2">
              <p className="font-mono text-sm font-semibold text-orange-300">{analysis.summary.high}</p>
              <p className="text-[9px] text-orange-300/70">high</p>
            </div>
          </div>

          <div className="flex shrink-0 items-center justify-between border-b border-[#2a2a2a] px-4 py-3">
            <div>
              <p className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">
                Top hotspots
              </p>
              <p className="text-[10px] text-[#666666]">{analysis.summary.files_analyzed} files analyzed</p>
            </div>
            <button
              onClick={handleCreateAll}
              disabled={creatingAll || highPriorityCount === 0}
              className="rounded border border-[#2a2a2a] bg-white/[0.04] px-2 py-1 text-[10px] font-semibold text-[#a0a0a0] transition-all hover:border-[#3a3a3a] hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
            >
              {creatingAll ? 'Creating...' : 'Create all high'}
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4">
            <div className="flex flex-col gap-3">
              {topHotspots.map((hotspot) => (
                <HotspotCard
                  key={hotspot.file}
                  hotspot={hotspot}
                  owner={owner}
                  repoName={repoName}
                  onCreated={handleTaskCreated}
                />
              ))}
              {topHotspots.length === 0 && (
                <p className="py-8 text-center text-xs text-[#666666]">No supported source files found.</p>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

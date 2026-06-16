/**
 * RepoStatsPanel — Repository task statistics panel.
 *
 * Displays:
 * - Big numbers for Total, Open, In Progress, Resolved tasks
 * - Horizontal CSS bar chart breakdown by type (TODO/FIXME/HACK/BUG/NOTE)
 * - Bars animate on mount using CSS transitions
 */

import { useState, useEffect, useCallback } from 'react';
import { BarChart2 } from 'lucide-react';
import { reposApi } from '../api/client';
import type { RepoStats } from '../types';
import Spinner from './ui/Spinner';

// ── Helpers ───────────────────────────────────────────────────────────────────

const TYPE_COLORS: Record<string, string> = {
  TODO:  '#ffffff',
  FIXME: '#a0a0a0',
  HACK:  '#666666',
  BUG:   '#cc4444',
  NOTE:  '#4a4a4a',
};

// ── Animated Bar ──────────────────────────────────────────────────────────────

function StatBar({
  label,
  value,
  max,
  color,
}: {
  label: string;
  value: number;
  max: number;
  color: string;
}) {
  const [width, setWidth] = useState('0%');
  const pct = max > 0 ? Math.round((value / max) * 100) : 0;

  useEffect(() => {
    // Slight delay to trigger CSS transition after mount
    const t = setTimeout(() => setWidth(`${pct}%`), 80);
    return () => clearTimeout(t);
  }, [pct]);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] font-semibold text-[#a0a0a0]">{label}</span>
        <span className="font-mono text-[10px] text-[#666666]">{value}</span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-[#1a1a1a] overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-700 ease-out"
          style={{ width, backgroundColor: color }}
        />
      </div>
      <span className="text-[9px] text-[#444444] font-mono">{pct}%</span>
    </div>
  );
}

// ── Stat Card ──────────────────────────────────────────────────────────────────

function StatCard({ label, value, dim }: { label: string; value: number; dim?: boolean }) {
  return (
    <div className="flex flex-col rounded border border-[#1a1a1a] bg-[#0d0d0d] p-4">
      <span className={`font-mono text-2xl font-bold ${dim ? 'text-[#a0a0a0]' : 'text-white'}`}>
        {value}
      </span>
      <span className="mt-1 text-[10px] text-[#666666] font-mono uppercase tracking-wider">{label}</span>
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

interface RepoStatsPanelProps {
  owner: string;
  repoName: string;
}

export default function RepoStatsPanel({ owner, repoName }: RepoStatsPanelProps) {
  const [stats, setStats] = useState<RepoStats | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    try {
      const data = await reposApi.getStats(owner, repoName);
      setStats(data);
    } catch {
      setStats(null);
    } finally {
      setLoading(false);
    }
  }, [owner, repoName]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const types: Array<keyof RepoStats['by_type']> = ['TODO', 'FIXME', 'HACK', 'BUG', 'NOTE'];
  const maxTypeCount = stats
    ? Math.max(...types.map((t) => stats.by_type[t] ?? 0), 1)
    : 1;

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-[#2a2a2a] px-3 py-2.5 shrink-0 bg-[#111111]">
        <BarChart2 size={13} className="text-[#a0a0a0]" />
        <span className="text-xs font-semibold text-white">Statistics</span>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : !stats ? (
          <p className="py-8 text-center text-[11px] text-[#666666]">
            Failed to load statistics
          </p>
        ) : (
          <div className="flex flex-col gap-6">
            {/* Overview cards */}
            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666] mb-3 block">
                Overview
              </span>
              <div className="grid grid-cols-2 gap-2">
                <StatCard label="Total" value={stats.total} />
                <StatCard label="Open" value={stats.open} dim />
                <StatCard label="In Progress" value={stats.in_progress} dim />
                <StatCard label="Resolved" value={stats.resolved} dim />
              </div>
            </div>

            {/* By type */}
            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666] mb-3 block">
                By Type
              </span>
              <div className="flex flex-col gap-3">
                {types.map((type) => {
                  const count = stats.by_type[type] ?? 0;
                  return (
                    <StatBar
                      key={type}
                      label={type}
                      value={count}
                      max={maxTypeCount}
                      color={TYPE_COLORS[type] ?? '#666666'}
                    />
                  );
                })}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

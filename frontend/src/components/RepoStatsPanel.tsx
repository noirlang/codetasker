/**
 * RepoStatsPanel — Repository task statistics and analytics panel.
 *
 * Displays:
 * - Interactive status progress rings (Open, In Progress, Resolved)
 * - Technical Debt Rating (A through F grade based on task weight)
 * - Horizontal CSS bar chart breakdown by annotation type
 * - Collaborators completion leaderboard with progress bars
 */

import { useState, useEffect, useCallback } from 'react';
import { BarChart2, ShieldAlert, Users } from 'lucide-react';
import { reposApi } from '../api/client';
import type { RepoStats } from '../types';
import Spinner from './ui/Spinner';

// ── Helpers ───────────────────────────────────────────────────────────────────

const TYPE_COLORS: Record<string, string> = {
  TODO:  '#3b82f6', // modern blue
  FIXME: '#f59e0b', // modern amber
  HACK:  '#8b5cf6', // modern purple
  BUG:   '#ef4444', // modern red
  NOTE:  '#10b981', // modern emerald
};

// ── Progress Ring Component ────────────────────────────────────────────────────

function ProgressRing({
  percent,
  color,
  label,
  value,
}: {
  percent: number;
  color: string;
  label: string;
  value: number;
}) {
  const radius = 24;
  const stroke = 3.5;
  const normalizedRadius = radius - stroke * 2;
  const circumference = normalizedRadius * 2 * Math.PI;
  const strokeDashoffset = circumference - (percent / 100) * circumference;

  return (
    <div className="flex flex-col items-center gap-1.5 p-2.5 bg-[#0f0f0f] border border-[#1c1c1c] rounded-lg flex-1 min-w-[75px] transition-all duration-300 hover:border-[#2a2a2a]">
      <div className="relative flex items-center justify-center" style={{ width: radius * 2, height: radius * 2 }}>
        <svg height={radius * 2} width={radius * 2} className="transform -rotate-90">
          <circle
            stroke="#161616"
            fill="transparent"
            strokeWidth={stroke}
            r={normalizedRadius}
            cx={radius}
            cy={radius}
          />
          <circle
            stroke={color}
            fill="transparent"
            strokeWidth={stroke}
            strokeDasharray={circumference + ' ' + circumference}
            style={{ strokeDashoffset, transition: 'stroke-dashoffset 0.8s ease-out' }}
            r={normalizedRadius}
            cx={radius}
            cy={radius}
            strokeLinecap="round"
          />
        </svg>
        <span className="absolute font-mono text-[9px] font-bold text-white">
          {percent}%
        </span>
      </div>
      <div className="text-center">
        <p className="text-[10px] font-bold text-white font-mono">{value}</p>
        <p className="text-[7.5px] uppercase tracking-wider text-[#666666] font-mono font-semibold">{label}</p>
      </div>
    </div>
  );
}

// ── Animated Bar Component ─────────────────────────────────────────────────────

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
    const t = setTimeout(() => setWidth(`${pct}%`), 80);
    return () => clearTimeout(t);
  }, [pct]);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] font-semibold text-[#a0a0a0]">{label}</span>
        <span className="font-mono text-[10px] text-[#666666]">{value}</span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-[#161616] overflow-hidden border border-[#222222]/30">
        <div
          className="h-full rounded-full transition-all duration-700 ease-out"
          style={{ width, backgroundColor: color }}
        />
      </div>
      <span className="text-[8px] text-[#444444] font-mono">{pct}% of max</span>
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

  const types = ['TODO', 'FIXME', 'HACK', 'BUG', 'NOTE'];
  const maxTypeCount = stats
    ? Math.max(...types.map((t) => stats.by_type[t as keyof typeof stats.by_type] ?? 0), 1)
    : 1;

  // ── Technical Debt calculations ──────────────────────────────────────────────
  const high = stats?.debt?.high ?? 0;
  const medium = stats?.debt?.medium ?? 0;
  const low = stats?.debt?.low ?? 0;
  const totalTasks = stats?.total ?? 0;
  const openTasks = stats?.open ?? 0;
  const inProgressTasks = stats?.in_progress ?? 0;
  const resolvedTasks = stats?.resolved ?? 0;

  const activeTasks = openTasks + inProgressTasks;
  const totalDebtPoints = (high * 10) + (medium * 4) + (low * 1.5);
  const debtScore = activeTasks > 0 ? Math.min(100, Math.round((totalDebtPoints / activeTasks) * 10)) : 0;

  let grade = 'A+';
  let gradeColor = '#10b981'; // emerald
  let gradeText = 'Clean Codebase';
  let gradeBg = 'rgba(16, 185, 129, 0.1)';

  if (activeTasks > 0) {
    if (debtScore <= 15) {
      grade = 'A';
      gradeColor = '#10b981';
      gradeText = 'Excellent';
      gradeBg = 'rgba(16, 185, 129, 0.1)';
    } else if (debtScore <= 35) {
      grade = 'B';
      gradeColor = '#3b82f6'; // blue
      gradeText = 'Good';
      gradeBg = 'rgba(59, 130, 246, 0.1)';
    } else if (debtScore <= 60) {
      grade = 'C';
      gradeColor = '#f59e0b'; // amber
      gradeText = 'Moderate Debt';
      gradeBg = 'rgba(245, 158, 11, 0.1)';
    } else if (debtScore <= 85) {
      grade = 'D';
      gradeColor = '#f97316'; // orange
      gradeText = 'High Debt';
      gradeBg = 'rgba(249, 115, 22, 0.1)';
    } else {
      grade = 'F';
      gradeColor = '#ef4444'; // red
      gradeText = 'Critical Debt';
      gradeBg = 'rgba(239, 68, 68, 0.1)';
    }
  }

  // Convert assignee map to list and sort by completion rate
  const assigneesList = stats?.by_assignee
    ? Object.values(stats.by_assignee).sort((a, b) => {
        const rateA = a.total > 0 ? a.resolved / a.total : 0;
        const rateB = b.total > 0 ? b.resolved / b.total : 0;
        return rateB - rateA || b.total - a.total;
      })
    : [];

  return (
    <div className="flex flex-col h-full bg-[#0a0a0a]">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-[#2a2a2a] px-3 py-2.5 shrink-0 bg-[#111111]">
        <BarChart2 size={13} className="text-[#a0a0a0]" />
        <span className="text-xs font-semibold text-white">Repository Analytics</span>
      </div>

      <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-6 custom-scrollbar">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size={20} />
          </div>
        ) : !stats ? (
          <p className="py-12 text-center text-[11px] text-[#666666]">
            Failed to load repository analytics
          </p>
        ) : (
          <>
            {/* 1. Status Breakdown (Progress Rings) */}
            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666] mb-3 block">
                Task Status Breakdown
              </span>
              <div className="flex gap-2">
                <ProgressRing
                  label="Open"
                  value={openTasks}
                  percent={totalTasks > 0 ? Math.round((openTasks / totalTasks) * 100) : 0}
                  color="#ef4444"
                />
                <ProgressRing
                  label="Working"
                  value={inProgressTasks}
                  percent={totalTasks > 0 ? Math.round((inProgressTasks / totalTasks) * 100) : 0}
                  color="#3b82f6"
                />
                <ProgressRing
                  label="Resolved"
                  value={resolvedTasks}
                  percent={totalTasks > 0 ? Math.round((resolvedTasks / totalTasks) * 100) : 0}
                  color="#10b981"
                />
              </div>
            </div>

            {/* 2. Technical Debt Grade */}
            <div className="rounded border border-[#1e1e1e] bg-[#0c0c0c] p-3.5 flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">
                  Technical Debt Grade
                </span>
                <span className="flex items-center gap-1 text-[9px] text-[#666666] font-mono">
                  <ShieldAlert size={10} />
                  Score: {debtScore}/100
                </span>
              </div>

              <div className="flex items-center gap-4">
                <div
                  className="w-12 h-12 rounded flex items-center justify-center text-xl font-bold font-mono transition-all border border-[#2a2a2a]"
                  style={{ color: gradeColor, backgroundColor: gradeBg, borderColor: `${gradeColor}30` }}
                >
                  {grade}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-semibold text-white truncate">{gradeText}</p>
                  <p className="text-[10px] text-[#666666] font-mono mt-0.5">
                    {activeTasks} unresolved task{activeTasks !== 1 ? 's' : ''}
                  </p>
                </div>
              </div>

              {/* Severity counts */}
              <div className="grid grid-cols-3 gap-2 mt-1 border-t border-[#1c1c1c] pt-2.5">
                <div className="text-center">
                  <p className="text-[11px] font-bold text-[#ef4444] font-mono">{high}</p>
                  <p className="text-[8px] text-[#666666] font-mono font-semibold">HIGH DEBT</p>
                </div>
                <div className="text-center">
                  <p className="text-[11px] font-bold text-[#f59e0b] font-mono">{medium}</p>
                  <p className="text-[8px] text-[#666666] font-mono font-semibold">MED DEBT</p>
                </div>
                <div className="text-center">
                  <p className="text-[11px] font-bold text-[#10b981] font-mono">{low}</p>
                  <p className="text-[8px] text-[#666666] font-mono font-semibold">LOW DEBT</p>
                </div>
              </div>
            </div>

            {/* 3. By Type breakdown */}
            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666] mb-3 block">
                Annotation Breakdown
              </span>
              <div className="flex flex-col gap-3">
                {types.map((type) => {
                  const count = stats.by_type[type as keyof typeof stats.by_type] ?? 0;
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

            {/* 4. Contributor Completion Leaderboard */}
            <div>
              <div className="flex items-center gap-1.5 mb-3">
                <Users size={11} className="text-[#666666]" />
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">
                  Contributor Board
                </span>
              </div>

              {assigneesList.length === 0 ? (
                <div className="rounded border border-[#1e1e1e]/60 bg-[#0c0c0c]/40 p-4 text-center">
                  <p className="text-[10px] text-[#555555] font-mono">No tasks assigned to contributors yet.</p>
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  {assigneesList.map((collab) => {
                    const rate = collab.total > 0 ? Math.round((collab.resolved / collab.total) * 100) : 0;
                    return (
                      <div
                        key={collab.username}
                        className="p-2.5 rounded border border-[#161616] bg-[#0c0c0c] flex flex-col gap-1.5"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2 min-w-0">
                            <img
                              src={collab.avatar_url}
                              alt={collab.username}
                              className="w-4 h-4 rounded-full border border-[#2a2a2a]"
                            />
                            <span className="text-[11px] font-semibold text-white truncate">{collab.username}</span>
                          </div>
                          <span className="text-[9px] font-mono text-[#a0a0a0]">
                            {collab.resolved}/{collab.total} done
                          </span>
                        </div>

                        {/* Completion rate bar */}
                        <div className="flex items-center gap-2">
                          <div className="h-1 flex-1 rounded bg-[#161616] overflow-hidden">
                            <div
                              className="h-full bg-[#10b981] rounded transition-all duration-500 ease-out"
                              style={{ width: `${rate}%` }}
                            />
                          </div>
                          <span className="text-[9px] font-mono font-bold text-white shrink-0">
                            {rate}%
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

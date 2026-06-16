/**
 * ActivityFeedPanel — Chronological repository activity feed.
 *
 * Features:
 * - Lists repo activities in reverse chronological order
 * - Each entry: actor avatar, actor name, action text, target label, time ago
 * - Auto-refreshes every 60 seconds
 */

import { useState, useEffect, useCallback } from 'react';
import {
  Activity,
  GitCommit,
  MessageSquare,
  GitMerge,
  Tag,
  User,
  Zap,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { ActivityLog } from '../types';
import Spinner from './ui/Spinner';

// ── Helpers ───────────────────────────────────────────────────────────────────

function timeAgo(isoString: string): string {
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

/** Return an icon for a given action string */
function ActionIcon({ action }: { action: string }) {
  const a = action.toLowerCase();
  const cls = 'shrink-0 text-[#666666]';
  if (a.includes('commit'))  return <GitCommit size={13} className={cls} />;
  if (a.includes('comment')) return <MessageSquare size={13} className={cls} />;
  if (a.includes('merge') || a.includes('pr')) return <GitMerge size={13} className={cls} />;
  if (a.includes('tag') || a.includes('release')) return <Tag size={13} className={cls} />;
  if (a.includes('assign')) return <User size={13} className={cls} />;
  return <Zap size={13} className={cls} />;
}

// ── Main Component ────────────────────────────────────────────────────────────

interface ActivityFeedPanelProps {
  owner: string;
  repoName: string;
}

export default function ActivityFeedPanel({ owner, repoName }: ActivityFeedPanelProps) {
  const [activities, setActivities] = useState<ActivityLog[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchActivity = useCallback(async () => {
    setLoading(true);
    try {
      const data = await reposApi.getActivity(owner, repoName);
      setActivities(data);
    } catch {
      setActivities([]);
    } finally {
      setLoading(false);
    }
  }, [owner, repoName]);

  useEffect(() => {
    fetchActivity();
    // Auto-refresh every 60 seconds
    const interval = setInterval(fetchActivity, 60_000);
    return () => clearInterval(interval);
  }, [fetchActivity]);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-[#2a2a2a] px-3 py-2.5 shrink-0 bg-[#111111]">
        <Activity size={13} className="text-[#a0a0a0]" />
        <span className="text-xs font-semibold text-white">Activity</span>
        <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[9px] text-[#666666]">
          {activities.length}
        </span>
        <span className="ml-auto font-mono text-[9px] text-[#444444]">auto-refresh 60s</span>
      </div>

      {/* Activity list */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : activities.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 py-10">
            <Activity size={24} className="text-[#333333]" />
            <p className="text-[11px] text-[#666666]">No activity yet</p>
          </div>
        ) : (
          <div className="flex flex-col divide-y divide-[#1a1a1a]">
            {activities.map((activity) => (
              <div
                key={activity.id}
                className="flex items-start gap-3 px-3 py-3 hover:bg-[#0d0d0d] transition-colors"
              >
                {/* Actor avatar */}
                <div className="shrink-0 mt-0.5">
                  {activity.actor_avatar ? (
                    <img
                      src={activity.actor_avatar}
                      alt={activity.actor_name}
                      className="h-6 w-6 rounded-full border border-[#2a2a2a]"
                    />
                  ) : (
                    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-[#1a1a1a] border border-[#2a2a2a]">
                      <User size={12} className="text-[#666666]" />
                    </div>
                  )}
                </div>

                <div className="flex-1 min-w-0">
                  {/* Action row */}
                  <div className="flex items-start gap-1.5 flex-wrap">
                    <span className="text-[11px] font-medium text-white">{activity.actor_name}</span>
                    <div className="flex items-center gap-1">
                      <ActionIcon action={activity.action} />
                      <span className="text-[11px] text-[#a0a0a0]">{activity.action}</span>
                    </div>
                  </div>

                  {/* Target */}
                  <div className="mt-0.5 flex items-center gap-1.5">
                    <span className="text-[9px] font-mono text-[#666666] uppercase tracking-wider">
                      {activity.target_type}
                    </span>
                    <span className="text-[10px] text-[#a0a0a0] truncate">
                      {activity.target_label}
                    </span>
                  </div>

                  {/* Timestamp */}
                  <span className="mt-1 block font-mono text-[9px] text-[#444444]">
                    {timeAgo(activity.created_at)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

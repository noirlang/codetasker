/**
 * IssuesPanel — GitHub Issues browser for a repository.
 *
 * Features:
 * - Filter by state: open / closed
 * - List issues with number, title, labels, author avatar, comment count, time ago
 * - "New Issue" button (for owner/maintainer/developer)
 * - Click issue → opens GitHub URL in new tab
 * - Inline close/reopen button for owner/maintainer
 */

import { useState, useEffect, useCallback } from 'react';
import {
  CircleDot,
  Plus,
  ExternalLink,
  MessageSquare,
  X,
  Send,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { Issue, RepoRole } from '../types';
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

// ── New Issue Modal ───────────────────────────────────────────────────────────

function NewIssueModal({
  onClose,
  onSubmit,
}: {
  onClose: () => void;
  onSubmit: (title: string, body: string, labels: string[]) => Promise<void>;
}) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [labelsStr, setLabelsStr] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    const t = title.trim();
    if (!t || submitting) return;
    setSubmitting(true);
    const labels = labelsStr
      .split(',')
      .map((l) => l.trim())
      .filter(Boolean);
    try {
      await onSubmit(t, body.trim(), labels);
      onClose();
    } catch {
      // Silently fail
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        className="w-full max-w-md rounded border border-[#222222] bg-[#111111] shadow-2xl animate__animated animate__fadeInUp"
        style={{ animationDuration: '0.18s' }}
      >
        <div className="flex items-center justify-between border-b border-[#1a1a1a] px-5 py-3.5">
          <div className="flex items-center gap-2">
            <CircleDot size={13} className="text-[#a0a0a0]" />
            <span className="text-sm font-semibold text-white">New Issue</span>
          </div>
          <button onClick={onClose} className="text-[#666666] hover:text-white transition-colors cursor-pointer">
            <X size={14} />
          </button>
        </div>

        <div className="flex flex-col gap-4 p-5">
          <div className="flex flex-col gap-1.5">
            <label className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Title *</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Issue title…"
              className="w-full rounded border border-[#2a2a2a] bg-[#0d0d0d] px-3 py-2 text-[12px] text-white placeholder-[#444444] focus:outline-none focus:border-[#3a3a3a] transition-colors"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Body</label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Describe the issue…"
              rows={4}
              className="w-full resize-none rounded border border-[#2a2a2a] bg-[#0d0d0d] px-3 py-2 text-[12px] text-white placeholder-[#444444] focus:outline-none focus:border-[#3a3a3a] transition-colors"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Labels (comma-separated)</label>
            <input
              type="text"
              value={labelsStr}
              onChange={(e) => setLabelsStr(e.target.value)}
              placeholder="bug, enhancement…"
              className="w-full rounded border border-[#2a2a2a] bg-[#0d0d0d] px-3 py-2 text-[12px] text-white placeholder-[#444444] focus:outline-none focus:border-[#3a3a3a] transition-colors"
            />
          </div>
          <div className="flex justify-end gap-2">
            <button
              onClick={onClose}
              className="rounded border border-[#2a2a2a] px-4 py-1.5 text-[11px] text-[#666666] hover:text-white hover:border-[#3a3a3a] transition-all cursor-pointer"
            >
              Cancel
            </button>
            <button
              onClick={handleSubmit}
              disabled={!title.trim() || submitting}
              className="flex items-center gap-1.5 rounded border border-[#2a2a2a] bg-white/[0.05] px-4 py-1.5 text-[11px] text-white hover:bg-white/10 transition-all cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {submitting ? <Spinner size={11} /> : <Send size={11} />}
              Create Issue
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

interface IssuesPanelProps {
  owner: string;
  repoName: string;
  currentUserRole: RepoRole | 'none';
}

export default function IssuesPanel({ owner, repoName, currentUserRole }: IssuesPanelProps) {
  const [issues, setIssues] = useState<Issue[]>([]);
  const [stateFilter, setStateFilter] = useState<'open' | 'closed'>('open');
  const [loading, setLoading] = useState(true);
  const [showNewIssue, setShowNewIssue] = useState(false);
  const [togglingId, setTogglingId] = useState<number | null>(null);

  const canWrite = ['owner', 'maintainer', 'developer'].includes(currentUserRole);
  const canManage = ['owner', 'maintainer'].includes(currentUserRole);

  const fetchIssues = useCallback(async () => {
    setLoading(true);
    try {
      const data = await reposApi.listIssues(owner, repoName, stateFilter);
      setIssues(data);
    } catch {
      setIssues([]);
    } finally {
      setLoading(false);
    }
  }, [owner, repoName, stateFilter]);

  useEffect(() => {
    fetchIssues();
  }, [fetchIssues]);

  const handleCreateIssue = async (title: string, body: string, labels: string[]) => {
    const newIssue = await reposApi.createIssue(owner, repoName, { title, body, labels });
    setIssues((prev) => [newIssue, ...prev]);
  };

  const handleToggleState = async (issue: Issue) => {
    setTogglingId(issue.number);
    const newState = issue.state === 'open' ? 'closed' : 'open';
    try {
      const updated = await reposApi.updateIssue(owner, repoName, issue.number, newState);
      if (stateFilter !== newState) {
        // Remove from current filter view
        setIssues((prev) => prev.filter((i) => i.number !== issue.number));
      } else {
        setIssues((prev) => prev.map((i) => (i.number === issue.number ? updated : i)));
      }
    } catch {
      // Silently fail
    } finally {
      setTogglingId(null);
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-[#2a2a2a] px-3 py-2.5 shrink-0 bg-[#111111]">
        <div className="flex items-center gap-2">
          <CircleDot size={13} className="text-[#a0a0a0]" />
          <span className="text-xs font-semibold text-white">Issues</span>
          <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[9px] text-[#666666]">
            {issues.length}
          </span>
        </div>
        {canWrite && (
          <button
            onClick={() => setShowNewIssue(true)}
            className="flex items-center gap-1 rounded border border-[#2a2a2a] bg-transparent px-2 py-1 text-[10px] text-[#a0a0a0] hover:text-white hover:border-[#3a3a3a] transition-all cursor-pointer font-mono"
          >
            <Plus size={10} />
            New Issue
          </button>
        )}
      </div>

      {/* Filter tabs */}
      <div className="flex gap-0 border-b border-[#1a1a1a] px-3 bg-[#0d0d0d] shrink-0">
        {(['open', 'closed'] as const).map((s) => (
          <button
            key={s}
            onClick={() => setStateFilter(s)}
            className={`px-3 py-1.5 text-[10px] font-semibold border-b-2 transition-all duration-150 flex items-center gap-1 cursor-pointer capitalize ${
              stateFilter === s
                ? 'border-white text-white'
                : 'border-transparent text-[#666666] hover:text-[#a0a0a0]'
            }`}
          >
            <CircleDot size={10} />
            {s}
          </button>
        ))}
      </div>

      {/* Issues list */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : issues.length === 0 ? (
          <p className="py-8 text-center text-[11px] text-[#666666]">
            No {stateFilter} issues found
          </p>
        ) : (
          <div className="flex flex-col divide-y divide-[#1a1a1a]">
            {issues.map((issue) => (
              <div
                key={issue.number}
                className="flex items-start gap-3 px-3 py-3 hover:bg-[#0d0d0d] transition-colors"
              >
                <CircleDot
                  size={14}
                  className={`mt-0.5 shrink-0 ${issue.state === 'open' ? 'text-emerald-400' : 'text-[#666666]'}`}
                />

                <div className="flex-1 min-w-0">
                  {/* Title */}
                  <a
                    href={issue.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[11px] font-medium text-white hover:text-[#a0a0a0] transition-colors line-clamp-2 leading-snug"
                  >
                    {issue.title}
                    <ExternalLink size={9} className="inline ml-1 opacity-50" />
                  </a>

                  {/* Meta */}
                  <div className="mt-1 flex flex-wrap items-center gap-1.5">
                    <span className="font-mono text-[9px] text-[#666666]">#{issue.number}</span>
                    <span className="text-[9px] text-[#666666]">
                      by {issue.user.login} · {timeAgo(issue.created_at)}
                    </span>

                    {/* Labels */}
                    {issue.labels.map((label) => (
                      <span
                        key={label.name}
                        className="rounded px-1.5 py-0.5 text-[9px] font-mono border"
                        style={{
                          borderColor: `#${label.color}44`,
                          backgroundColor: `#${label.color}18`,
                          color: `#${label.color}`,
                        }}
                      >
                        {label.name}
                      </span>
                    ))}

                    {/* Comment count */}
                    {issue.comments > 0 && (
                      <span className="flex items-center gap-0.5 text-[9px] text-[#666666]">
                        <MessageSquare size={9} />
                        {issue.comments}
                      </span>
                    )}
                  </div>
                </div>

                {/* Close/Reopen button */}
                {canManage && (
                  <button
                    onClick={() => handleToggleState(issue)}
                    disabled={togglingId === issue.number}
                    className="shrink-0 rounded border border-[#2a2a2a] px-1.5 py-0.5 text-[9px] font-mono text-[#666666] hover:text-white hover:border-[#3a3a3a] transition-all cursor-pointer disabled:opacity-40"
                  >
                    {togglingId === issue.number ? (
                      <Spinner size={9} />
                    ) : issue.state === 'open' ? (
                      'Close'
                    ) : (
                      'Reopen'
                    )}
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* New issue modal */}
      {showNewIssue && (
        <NewIssueModal
          onClose={() => setShowNewIssue(false)}
          onSubmit={handleCreateIssue}
        />
      )}
    </div>
  );
}

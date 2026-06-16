/**
 * BranchesPanel — Branch management panel for a repository.
 *
 * Features:
 * - List all branches with name, commit SHA (7 chars), protected badge
 * - Highlight current branch
 * - "New Branch" button for owner/maintainer → modal with name + from (select base)
 * - Delete button (trash icon) for non-protected branches, owner/maintainer only
 * - Confirm before delete
 */

import { useState, useEffect, useCallback } from 'react';
import {
  GitBranch,
  Plus,
  Trash2,
  ShieldCheck,
  X,
  Send,
} from 'lucide-react';
import { reposApi } from '../api/client';
import type { Branch, RepoRole } from '../types';
import Spinner from './ui/Spinner';

// ── New Branch Modal ──────────────────────────────────────────────────────────

function NewBranchModal({
  branches,
  onClose,
  onSubmit,
}: {
  branches: Branch[];
  onClose: () => void;
  onSubmit: (name: string, fromSha: string) => Promise<void>;
}) {
  const [name, setName] = useState('');
  const [fromSha, setFromSha] = useState(branches[0]?.commit.sha ?? '');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    const n = name.trim();
    if (!n || !fromSha || submitting) return;
    setSubmitting(true);
    try {
      await onSubmit(n, fromSha);
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
        className="w-full max-w-sm rounded border border-[#222222] bg-[#111111] shadow-2xl animate__animated animate__fadeInUp"
        style={{ animationDuration: '0.18s' }}
      >
        <div className="flex items-center justify-between border-b border-[#1a1a1a] px-5 py-3.5">
          <div className="flex items-center gap-2">
            <GitBranch size={13} className="text-[#a0a0a0]" />
            <span className="text-sm font-semibold text-white">New Branch</span>
          </div>
          <button onClick={onClose} className="text-[#666666] hover:text-white transition-colors cursor-pointer">
            <X size={14} />
          </button>
        </div>

        <div className="flex flex-col gap-4 p-5">
          <div className="flex flex-col gap-1.5">
            <label className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">Branch Name *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="feature/my-new-branch"
              className="w-full rounded border border-[#2a2a2a] bg-[#0d0d0d] px-3 py-2 text-[12px] text-white placeholder-[#444444] focus:outline-none focus:border-[#3a3a3a] transition-colors font-mono"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[10px] font-mono uppercase tracking-wider text-[#666666]">From Branch *</label>
            <select
              value={fromSha}
              onChange={(e) => setFromSha(e.target.value)}
              className="w-full rounded border border-[#2a2a2a] bg-[#0d0d0d] px-3 py-2 text-[12px] text-white focus:outline-none focus:border-[#3a3a3a] transition-colors cursor-pointer font-mono"
            >
              {branches.map((b) => (
                <option key={b.name} value={b.commit.sha}>
                  {b.name} ({b.commit.sha.slice(0, 7)})
                </option>
              ))}
            </select>
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
              disabled={!name.trim() || !fromSha || submitting}
              className="flex items-center gap-1.5 rounded border border-[#2a2a2a] bg-white/[0.05] px-4 py-1.5 text-[11px] text-white hover:bg-white/10 transition-all cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {submitting ? <Spinner size={11} /> : <Send size={11} />}
              Create
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

interface BranchesPanelProps {
  owner: string;
  repoName: string;
  currentBranch: string;
  currentUserRole: RepoRole | 'none';
}

export default function BranchesPanel({
  owner,
  repoName,
  currentBranch,
  currentUserRole,
}: BranchesPanelProps) {
  const [branches, setBranches] = useState<Branch[]>([]);
  const [loading, setLoading] = useState(true);
  const [showNewBranch, setShowNewBranch] = useState(false);
  const [deletingName, setDeletingName] = useState<string | null>(null);

  const canManage = ['owner', 'maintainer'].includes(currentUserRole);

  const fetchBranches = useCallback(async () => {
    setLoading(true);
    try {
      const data = await reposApi.listBranches(owner, repoName);
      setBranches(data);
    } catch {
      setBranches([]);
    } finally {
      setLoading(false);
    }
  }, [owner, repoName]);

  useEffect(() => {
    fetchBranches();
  }, [fetchBranches]);

  const handleCreateBranch = async (name: string, fromSha: string) => {
    const newBranch = await reposApi.createBranch(owner, repoName, { name, from_sha: fromSha });
    setBranches((prev) => [...prev, newBranch].sort((a, b) => a.name.localeCompare(b.name)));
  };

  const handleDeleteBranch = async (branchName: string) => {
    if (!window.confirm(`Delete branch "${branchName}"? This cannot be undone.`)) return;
    setDeletingName(branchName);
    try {
      await reposApi.deleteBranch(owner, repoName, branchName);
      setBranches((prev) => prev.filter((b) => b.name !== branchName));
    } catch {
      // Silently fail
    } finally {
      setDeletingName(null);
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-[#2a2a2a] px-3 py-2.5 shrink-0 bg-[#111111]">
        <div className="flex items-center gap-2">
          <GitBranch size={13} className="text-[#a0a0a0]" />
          <span className="text-xs font-semibold text-white">Branches</span>
          <span className="rounded border border-[#2a2a2a] px-1.5 py-0.5 font-mono text-[9px] text-[#666666]">
            {branches.length}
          </span>
        </div>
        {canManage && (
          <button
            onClick={() => setShowNewBranch(true)}
            className="flex items-center gap-1 rounded border border-[#2a2a2a] bg-transparent px-2 py-1 text-[10px] text-[#a0a0a0] hover:text-white hover:border-[#3a3a3a] transition-all cursor-pointer font-mono"
          >
            <Plus size={10} />
            New Branch
          </button>
        )}
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner size={20} />
          </div>
        ) : branches.length === 0 ? (
          <p className="py-8 text-center text-[11px] text-[#666666]">No branches found</p>
        ) : (
          <div className="flex flex-col divide-y divide-[#1a1a1a]">
            {branches.map((branch) => {
              const isCurrent = branch.name === currentBranch;
              const isDeleting = deletingName === branch.name;

              return (
                <div
                  key={branch.name}
                  className={[
                    'flex items-center justify-between gap-3 px-3 py-2.5 transition-colors',
                    isCurrent ? 'bg-white/[0.02]' : 'hover:bg-[#0d0d0d]',
                  ].join(' ')}
                >
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <GitBranch
                      size={12}
                      className={isCurrent ? 'text-white shrink-0' : 'text-[#666666] shrink-0'}
                    />
                    <div className="min-w-0">
                      <div className="flex items-center gap-1.5">
                        <span
                          className={`font-mono text-[11px] truncate ${isCurrent ? 'text-white font-semibold' : 'text-[#a0a0a0]'}`}
                        >
                          {branch.name}
                        </span>
                        {isCurrent && (
                          <span className="shrink-0 rounded border border-white/20 px-1 py-0.5 font-mono text-[8px] text-white/60">
                            current
                          </span>
                        )}
                        {branch.protected && (
                          <span title="Protected branch" className="shrink-0 flex items-center">
                            <ShieldCheck size={10} className="text-amber-400/70" />
                          </span>
                        )}
                      </div>
                      <span className="font-mono text-[9px] text-[#666666]">
                        {branch.commit.sha.slice(0, 7)}
                      </span>
                    </div>
                  </div>

                  {/* Delete button — only for non-protected, non-current branches by owner/maintainer */}
                  {canManage && !branch.protected && !isCurrent && (
                    <button
                      onClick={() => handleDeleteBranch(branch.name)}
                      disabled={isDeleting}
                      className="shrink-0 rounded p-1 text-[#444444] hover:text-red-400 transition-colors cursor-pointer disabled:opacity-40"
                      title={`Delete branch ${branch.name}`}
                    >
                      {isDeleting ? <Spinner size={12} /> : <Trash2 size={12} />}
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* New Branch modal */}
      {showNewBranch && (
        <NewBranchModal
          branches={branches}
          onClose={() => setShowNewBranch(false)}
          onSubmit={handleCreateBranch}
        />
      )}
    </div>
  );
}

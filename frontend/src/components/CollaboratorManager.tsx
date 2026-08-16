import { useEffect, useState } from 'react';
import { X, UserPlus, Shield, Trash2, Users, ExternalLink, GitBranch, Key } from 'lucide-react';
import { reposApi } from '../api/client';
import { useAuthStore } from '../store/authStore';
import type { Collaborator, RepoRole, ApiError } from '../types';
import Spinner from './ui/Spinner';

interface CollaboratorManagerProps {
  isOpen: boolean;
  onClose: () => void;
  repoOwner: string;
  repoName: string;
  onRefresh?: () => void;
}

export default function CollaboratorManager({
  isOpen,
  onClose,
  repoOwner,
  repoName,
  onRefresh,
}: CollaboratorManagerProps) {
  const currentUser = useAuthStore((s) => s.user);
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);
  const [sandboxNotice, setSandboxNotice] = useState<string | null>(null);

  // Form state
  const [username, setUsername] = useState('');
  const [role, setRole] = useState<RepoRole>('developer');
  const [allowedPathsStr, setAllowedPathsStr] = useState('*');

  // Fetch collaborators
  const fetchCollaborators = async () => {
    if (!repoOwner || !repoName) return;
    setLoading(true);
    setError(null);
    try {
      const list = await reposApi.listCollaborators(repoOwner, repoName);
      setCollaborators(list);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load collaborators.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      fetchCollaborators();
      setUsername('');
      setRole('developer');
      setAllowedPathsStr('*');
      setError(null);
      setWarning(null);
      setSandboxNotice(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, repoOwner, repoName]);

  // Determine current user's role on this repo
  const currentUserCollab = collaborators.find((c) => c.username === currentUser?.username);
  const currentUserRole: RepoRole | 'none' = currentUserCollab
    ? currentUserCollab.role
    : repoOwner === currentUser?.username
    ? 'owner'
    : 'none';

  const canManage = currentUserRole === 'owner' || currentUserRole === 'maintainer';

  const githubInviteUrl = `https://github.com/${repoOwner}/${repoName}/settings/access`;

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) return;

    const allowedPaths = allowedPathsStr
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);

    setSubmitting(true);
    setError(null);
    setWarning(null);
    try {
      const { collaborator, warning: warnMsg } = await reposApi.addCollaborator(
        repoOwner,
        repoName,
        username.trim(),
        role,
        allowedPaths
      );
      setCollaborators((prev) => [...prev, collaborator]);
      setUsername('');
      if (warnMsg) {
        setWarning(warnMsg);
      }
      if (onRefresh) onRefresh();
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to add collaborator.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRoleChange = async (collabId: string, newRole: RepoRole) => {
    setError(null);
    try {
      const target = collaborators.find((c) => c.id === collabId);
      await reposApi.updateCollaboratorRole(
        repoOwner,
        repoName,
        collabId,
        newRole,
        target?.allowed_paths,
        target?.private_repo
      );
      setCollaborators((prev) =>
        prev.map((c) => (c.id === collabId ? { ...c, role: newRole } : c))
      );
      if (onRefresh) onRefresh();
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to update role.');
    }
  };

  const handleSyncSandbox = async (collabId: string) => {
    setError(null);
    try {
      const res = await reposApi.syncSandboxRepo(repoOwner, repoName, collabId);
      setSandboxNotice(`Private sandbox configured: ${res.private_repo}`);
      setCollaborators((prev) =>
        prev.map((c) => (c.id === collabId ? { ...c, private_repo: res.private_repo } : c))
      );
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to sync sandbox repo.');
    }
  };

  const handleRemove = async (collabId: string) => {
    if (!confirm('Are you sure you want to remove this collaborator?')) return;
    setError(null);
    try {
      await reposApi.removeCollaborator(repoOwner, repoName, collabId);
      setCollaborators((prev) => prev.filter((c) => c.id !== collabId));
      if (onRefresh) onRefresh();
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to remove collaborator.');
    }
  };

  return (
    <>
      {/* Backdrop overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm transition-opacity duration-300"
          onClick={onClose}
        />
      )}

      {/* Slide-out Panel */}
      <div
        className={`fixed right-0 top-0 z-50 h-full w-[420px] border-l border-[#2a2a2a] bg-[#111111] p-6 transition-transform duration-300 ease-in-out ${
          isOpen ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex h-full flex-col gap-6">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-[#2a2a2a] pb-4">
            <div className="flex items-center gap-2">
              <Users size={16} className="text-[#a0a0a0]" />
              <h2 className="text-sm font-semibold text-white">Repository Collaborators & Roles</h2>
            </div>
            <button
              onClick={onClose}
              className="rounded p-1 text-[#666666] hover:bg-[#1a1a1a] hover:text-white cursor-pointer"
            >
              <X size={16} />
            </button>
          </div>

          {/* Direct GitHub invite action */}
          <div className="flex items-center justify-between rounded bg-[#161616] border border-[#2a2a2a] p-3">
            <div className="min-w-0 flex-1">
              <p className="text-xs font-semibold text-white">GitHub Collaboration Sync</p>
              <p className="text-[10px] text-[#666666] truncate">Invite directly on GitHub for repo access</p>
            </div>
            <a
              href={githubInviteUrl}
              target="_blank"
              rel="noreferrer"
              className="btn-secondary py-1 px-3 text-xs flex items-center gap-1.5 shrink-0 hover:border-white"
            >
              <ExternalLink size={12} />
              <span>GitHub Invite</span>
            </a>
          </div>

          {error && (
            <div className="text-xs text-red-400 bg-red-400/5 border border-red-400/10 px-3 py-2 rounded">
              {error}
            </div>
          )}

          {warning && (
            <div className="text-xs text-amber-400 bg-amber-400/5 border border-amber-400/10 px-3 py-2 rounded leading-relaxed">
              {warning}
            </div>
          )}

          {sandboxNotice && (
            <div className="text-xs text-emerald-400 bg-emerald-400/5 border border-emerald-400/10 px-3 py-2 rounded leading-relaxed">
              {sandboxNotice}
            </div>
          )}

          {/* Add Collaborator Form (Only for Owner / Maintainer) */}
          {canManage && (
            <form onSubmit={handleAdd} className="flex flex-col gap-3 border-b border-[#2a2a2a] pb-6">
              <h3 className="text-xs font-semibold text-white flex items-center gap-1.5">
                <UserPlus size={13} /> Add Collaborator & Define Role
              </h3>
              <div>
                <label className="block text-[10px] text-[#666666] uppercase mb-1 font-mono">GitHub Username</label>
                <input
                  type="text"
                  className="input w-full"
                  placeholder="e.g. octocat"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  disabled={submitting}
                  required
                />
              </div>

              <div>
                <label className="block text-[10px] text-[#666666] uppercase mb-1 font-mono flex items-center gap-1">
                  <Key size={10} /> Allowed File Paths (Glob / Role Rules)
                </label>
                <input
                  type="text"
                  className="input w-full font-mono text-xs"
                  placeholder="e.g. frontend/**, public/** or *"
                  value={allowedPathsStr}
                  onChange={(e) => setAllowedPathsStr(e.target.value)}
                  disabled={submitting}
                />
                <span className="text-[9px] text-[#666666] mt-0.5 block">Comma separated paths (e.g., `frontend/**, backend/**`)</span>
              </div>

              <div className="flex gap-2 items-end">
                <div className="flex-1">
                  <label className="block text-[10px] text-[#666666] uppercase mb-1 font-mono">Role</label>
                  <select
                    className="input w-full bg-[#111111] border border-[#3a3a3a] rounded px-3 py-2 text-white text-sm"
                    value={role}
                    onChange={(e) => setRole(e.target.value as RepoRole)}
                    disabled={submitting}
                  >
                    <option value="developer">Developer (Read/Write)</option>
                    <option value="maintainer">Maintainer (Admin)</option>
                    <option value="viewer">Viewer (Read Only)</option>
                  </select>
                </div>
                <button
                  type="submit"
                  disabled={submitting}
                  className="btn-primary py-2 px-4 h-[38px] text-xs font-medium"
                >
                  {submitting ? <Spinner size={12} /> : 'Add Collaborator'}
                </button>
              </div>
            </form>
          )}

          {/* Collaborator List */}
          <div className="flex-1 overflow-y-auto">
            <h3 className="text-xs font-semibold text-white mb-3 flex items-center gap-1.5">
              <Shield size={13} /> Active Team ({collaborators.length})
            </h3>

            {loading ? (
              <div className="flex justify-center py-8">
                <Spinner size={20} />
              </div>
            ) : (
              <div className="flex flex-col gap-3">
                {collaborators.map((collab) => {
                  const isSelf = collab.username === currentUser?.username;
                  const isOwner = collab.role === 'owner';

                  const isEditable =
                    !isOwner &&
                    canManage &&
                    !(currentUserRole === 'maintainer' && collab.role === 'maintainer');

                  return (
                    <div
                      key={collab.id}
                      className="flex flex-col gap-2 p-3 rounded bg-[#161616] border border-[#222222]"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2.5 min-w-0">
                          <img
                            src={collab.avatar_url}
                            alt={collab.username}
                            className="h-7 w-7 rounded-full border border-[#2a2a2a] shrink-0"
                          />
                          <div className="min-w-0">
                            <p className="text-xs font-semibold text-white truncate">
                              {collab.username}{' '}
                              {isSelf && <span className="text-[10px] text-[#666666]">(You)</span>}
                            </p>
                            <p className="text-[9px] text-[#666666] font-mono capitalize">
                              {collab.role}
                            </p>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 shrink-0">
                          {canManage && !isOwner && (
                            <button
                              onClick={() => handleSyncSandbox(collab.id)}
                              className="p-1 rounded border border-[#2a2a2a] text-[10px] text-[#a0a0a0] hover:text-white hover:border-white transition-colors cursor-pointer flex items-center gap-1 font-mono"
                              title="Generate Private Sandbox Repo Sync"
                            >
                              <GitBranch size={10} />
                              <span>Sandbox</span>
                            </button>
                          )}
                          {isEditable ? (
                            <>
                              <select
                                className="bg-[#111111] border border-[#2a2a2a] rounded text-[11px] text-[#a0a0a0] py-1 px-2 focus:border-white focus:outline-none"
                                value={collab.role}
                                onChange={(e) =>
                                  handleRoleChange(collab.id, e.target.value as RepoRole)
                                }
                              >
                                <option value="viewer">Viewer</option>
                                <option value="developer">Developer</option>
                                <option value="maintainer">Maintainer</option>
                              </select>
                              <button
                                onClick={() => handleRemove(collab.id)}
                                className="p-1 rounded text-[#666666] hover:bg-[#222222] hover:text-red-400 transition-colors cursor-pointer"
                                title="Remove Collaborator"
                              >
                                <Trash2 size={13} />
                              </button>
                            </>
                          ) : (
                            <span className="text-[10px] font-mono text-[#666666] border border-[#2a2a2a] px-2 py-0.5 rounded uppercase">
                              {collab.role}
                            </span>
                          )}
                        </div>
                      </div>

                      {collab.allowed_paths && collab.allowed_paths.length > 0 && (
                        <div className="text-[9px] font-mono text-[#666666] bg-[#0d0d0d] p-1.5 rounded border border-[#222222] truncate">
                          Paths: {collab.allowed_paths.join(', ')}
                        </div>
                      )}

                      {collab.private_repo && (
                        <div className="text-[9px] font-mono text-emerald-400/80 bg-emerald-500/5 p-1.5 rounded border border-emerald-500/10 truncate flex items-center justify-between">
                          <span>Private Repo: {collab.private_repo}</span>
                          <a
                            href={`https://github.com/${collab.private_repo}`}
                            target="_blank"
                            rel="noreferrer"
                            className="underline hover:text-emerald-300 ml-1"
                          >
                            Open
                          </a>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

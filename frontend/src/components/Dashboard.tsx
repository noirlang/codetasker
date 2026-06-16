/**
 * Dashboard — Repository listing page.
 *
 * Fetches repos from /api/repos on mount and renders them in a responsive
 * grid with a sleek left sidebar. Each card links to the RepoView.
 */

import { useEffect, useState, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Activity,
  CheckCircle2,
  Star,
  Lock,
  RefreshCw,
  ChevronRight,
  LayoutDashboard,
  GitFork,
  Settings,
  BookOpen,
  LogOut,
  GitCommit,
  GitPullRequest,
  Github,
  ShieldCheck,
  Mail,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import ScrollReveal from 'scrollreveal';
import { useAuthStore } from '../store/authStore';
import { reposApi } from '../api/client';
import type { Repo, ApiError, Organization, User } from '../types';
import Spinner from './ui/Spinner';

// ── 3D Tilt Card Helper Component ──────────────────────────────────────────
function TiltCard({
  children,
  className = '',
  intensity = 10,
}: {
  children: React.ReactNode;
  className?: string;
  intensity?: number;
}) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [tilt, setTilt] = useState({ x: 0, y: 0 });
  const [isHovered, setIsHovered] = useState(false);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current) return;
    const rect = cardRef.current.getBoundingClientRect();
    const width = rect.width;
    const height = rect.height;
    const mouseX = e.clientX - rect.left - width / 2;
    const mouseY = e.clientY - rect.top - height / 2;
    const rX = (mouseY / (height / 2)) * -intensity;
    const rY = (mouseX / (width / 2)) * intensity;
    setTilt({ x: rX, y: rY });
  };

  const handleMouseLeave = () => {
    setTilt({ x: 0, y: 0 });
    setIsHovered(false);
  };

  return (
    <div
      ref={cardRef}
      onMouseMove={handleMouseMove}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={handleMouseLeave}
      className={`transition-transform duration-200 ease-out ${className}`}
      style={{
        transform: `perspective(1000px) rotateX(${tilt.x}deg) rotateY(${tilt.y}deg) scale(${isHovered ? 1.01 : 1})`,
        transformStyle: 'preserve-3d',
      }}
    >
      <div style={{ transform: 'translateZ(20px)', transformStyle: 'preserve-3d', height: '100%' }}>
        {children}
      </div>
    </div>
  );
}

// ── Helpers ─────────────────────────────────────────────────────────────────

/** Format an ISO timestamp as a relative time string ("3 days ago") */
function relativeTime(isoString: string): string {
  const date = new Date(isoString);
  const now  = new Date();
  const diff = now.getTime() - date.getTime();

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours   = Math.floor(minutes / 60);
  const days    = Math.floor(hours / 24);
  const weeks   = Math.floor(days / 7);
  const months  = Math.floor(days / 30);

  if (seconds < 60)  return 'just now';
  if (minutes < 60)  return `${minutes}m ago`;
  if (hours   < 24)  return `${hours}h ago`;
  if (days    < 7)   return `${days}d ago`;
  if (weeks   < 5)   return `${weeks}w ago`;
  return `${months}mo ago`;
}

// ── Sub-components ───────────────────────────────────────────────────────────

/** Redesigned Repository Card showing description and topics/tags */
function RepoCard({
  repo,
  onView,
  onSync,
  isSyncing,
}: {
  repo: Repo;
  onView: () => void;
  onSync: () => void;
  isSyncing: boolean;
}) {
  const [owner] = repo.full_name.split('/');

  return (
    <TiltCard intensity={4} className="reveal-card w-full">
      <article className="card flex h-full flex-col justify-between gap-5 p-6 transition-all duration-200 hover:border-[#3a3a3a] bg-[#131313] border border-[#222222]">
        <div>
          {/* Header */}
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h2
                className="truncate font-mono text-sm font-semibold text-white tracking-tight hover:text-white/80 transition-colors"
                style={{ fontFamily: "'JetBrains Mono', monospace" }}
                title={repo.full_name}
              >
                {repo.name}
              </h2>
              <p className="mt-0.5 text-[11px] text-[#666666] font-mono">{owner}</p>
            </div>
            {repo.private && (
              <span className="flex shrink-0 items-center gap-1 rounded bg-[#1c1c1c] border border-[#2e2e2e] px-2 py-0.5 text-[9px] text-[#666666] uppercase font-mono tracking-wider">
                <Lock size={8} />
                private
              </span>
            )}
          </div>

          {/* Description ("hakkında") */}
          <p className="mt-3 text-xs text-[#a0a0a0] leading-relaxed line-clamp-3 min-h-[3.25rem]">
            {repo.description ?? (
              <span className="italic text-[#444444]">No description provided</span>
            )}
          </p>

          {/* Topics/Tags ("tag kısmı") */}
          {repo.topics && repo.topics.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-4">
              {repo.topics.slice(0, 5).map((topic) => (
                <span
                  key={topic}
                  className="inline-block text-[10px] font-mono px-2 py-0.5 rounded border border-[#222222] text-[#888888] bg-[#0c0c0c] hover:border-white/20 transition-colors duration-150"
                >
                  #{topic}
                </span>
              ))}
              {repo.topics.length > 5 && (
                <span className="inline-block text-[10px] font-mono px-1.5 py-0.5 text-[#555555]">
                  +{repo.topics.length - 5} more
                </span>
              )}
            </div>
          )}
        </div>

        <div>
          {/* Meta row */}
          <div className="flex items-center gap-3 mb-4 text-[11px] text-[#555555]">
            {repo.language && (
              <span className="tag text-[#a0a0a0] border-[#3a3a3a] px-2 py-0.5 bg-transparent">{repo.language}</span>
            )}
            <span className="flex items-center gap-1">
              <Star size={10} className="text-[#666666]" />
              {repo.stargazers_count.toLocaleString()}
            </span>
            <span className="ml-auto font-mono text-[10px]">
              {relativeTime(repo.updated_at)}
            </span>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2 border-t border-[#222222] pt-4">
            <button
              onClick={onView}
              className="btn-secondary flex-1 justify-center text-xs py-2 border-[#2a2a2a] hover:border-white transition-all duration-150"
            >
              View Tasks
              <ChevronRight size={12} className="ml-1" />
            </button>
            {repo.is_synced ? (
              <span className="inline-flex items-center gap-1.5 px-3 py-2 border border-white/10 bg-white/5 rounded text-[11px] font-mono text-white/80 select-none">
                <span className="h-1.5 w-1.5 rounded-full bg-white animate-pulse" />
                Synced
              </span>
            ) : (
              <button
                onClick={onSync}
                disabled={isSyncing}
                className="btn-ghost shrink-0 text-xs flex items-center gap-1.5 border border-[#2a2a2a] px-3 py-2 hover:border-white transition-all duration-150 h-9"
                title="Activate Webhook Sync"
              >
                {isSyncing ? (
                  <Spinner size={12} />
                ) : (
                  <RefreshCw size={12} />
                )}
                Sync
              </button>
            )}
          </div>
        </div>
      </article>
    </TiltCard>
  );
}

function DashboardInfoCard({
  icon: Icon,
  title,
  children,
}: {
  icon: LucideIcon;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded border border-[#2a2a2a] bg-[#111111] p-5">
      <div className="mb-3 flex items-center gap-2">
        <Icon size={15} className="text-[#a0a0a0]" />
        <h2 className="text-sm font-semibold text-white">{title}</h2>
      </div>
      {children}
    </section>
  );
}

function DocsContent() {
  const docs = [
    {
      icon: RefreshCw,
      title: 'Sync repositories',
      text: 'Use Sync on a repository card to register the webhook and let CodeTasker scan TODO, FIXME, BUG, HACK, and NOTE annotations.',
    },
    {
      icon: GitCommit,
      title: 'Review commits',
      text: 'Open a repository, switch to Commits, and use Load more commits when the branch has more history than the first page.',
    },
    {
      icon: Activity,
      title: 'Inspect Actions',
      text: 'Use the Actions tab inside a repository to see workflows, recent runs, branch filters, status badges, and links back to GitHub.',
    },
    {
      icon: GitPullRequest,
      title: 'Inject and merge',
      text: 'Create code annotations from the task board, review pull requests, and merge branches from the right-side PR panel.',
    },
  ];

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {docs.map((item) => (
        <DashboardInfoCard key={item.title} icon={item.icon} title={item.title}>
          <p className="text-sm leading-6 text-[#a0a0a0]">{item.text}</p>
        </DashboardInfoCard>
      ))}

      <section className="rounded border border-[#2a2a2a] bg-[#111111] p-5 lg:col-span-2">
        <div className="mb-4 flex items-center gap-2">
          <ShieldCheck size={15} className="text-[#a0a0a0]" />
          <h2 className="text-sm font-semibold text-white">Sign-in help</h2>
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          {[
            'Open CodeTasker from http://localhost:5173 during local testing.',
            'Use the same browser session for CodeTasker and GitHub approval.',
            'If state mismatch appears, close the stale callback tab and sign in again.',
          ].map((step) => (
            <div key={step} className="rounded border border-[#242424] bg-[#0d0d0d] p-3">
              <div className="mb-2 flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-wider text-[#666666]">
                <CheckCircle2 size={11} />
                Check
              </div>
              <p className="text-xs leading-5 text-[#a0a0a0]">{step}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function SettingsContent({
  user,
  repos,
  orgs,
}: {
  user: User | null;
  repos: Repo[];
  orgs: Organization[];
}) {
  const syncedCount = repos.filter((repo) => repo.is_synced).length;
  const privateCount = repos.filter((repo) => repo.private).length;
  const { updateEmail } = useAuthStore();

  const [email, setEmail] = useState(user?.email || '');
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Sync internal state when user profile is loaded/updated
  useEffect(() => {
    if (user?.email) {
      setEmail(user.email);
    }
  }, [user?.email]);

  const handleSaveEmail = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMessage(null);
    try {
      await updateEmail(email);
      setMessage({ type: 'success', text: 'Email settings saved successfully.' });
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Failed to save email settings.' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <DashboardInfoCard icon={Github} title="GitHub account">
        {user ? (
          <div className="flex items-center gap-3">
            <img
              src={user.avatar_url}
              alt={user.username}
              className="h-10 w-10 rounded-full border border-[#3a3a3a]"
            />
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-white">{user.username}</p>
              <p className="text-xs text-[#666666]">Signed in with GitHub</p>
            </div>
          </div>
        ) : (
          <p className="text-sm text-[#a0a0a0]">No authenticated user loaded.</p>
        )}
      </DashboardInfoCard>

      <DashboardInfoCard icon={Mail} title="Email notifications">
        <form onSubmit={handleSaveEmail} className="flex flex-col gap-3">
          <p className="text-xs text-[#a0a0a0] leading-relaxed">
            Enter your email address to receive notifications when you are assigned to a task or when someone comments on your tasks.
          </p>
          <div className="flex gap-2">
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="e.g. name@company.com"
              className="input flex-1"
              required
            />
            <button
              type="submit"
              disabled={saving}
              className="btn-primary shrink-0 text-xs py-2 px-3"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
          {message && (
            <p className={`text-xs ${message.type === 'success' ? 'text-emerald-400' : 'text-red-400'}`}>
              {message.text}
            </p>
          )}
        </form>
      </DashboardInfoCard>

      <DashboardInfoCard icon={GitFork} title="Connected sources">
        <div className="grid grid-cols-2 gap-2">
          {[
            ['Repositories', repos.length],
            ['Synced repos', syncedCount],
            ['Organizations', orgs.length],
            ['Private repos', privateCount],
          ].map(([label, value]) => (
            <div key={label} className="rounded border border-[#242424] bg-[#0d0d0d] p-3">
              <span className="text-xs text-[#888888]">{label}</span>
              <p className="mt-1 font-mono text-lg font-semibold text-white">{value}</p>
            </div>
          ))}
        </div>
      </DashboardInfoCard>

      <DashboardInfoCard icon={ShieldCheck} title="Security">
        <div className="grid gap-2">
          {[
            'Your GitHub connection is protected.',
            'Only repositories your GitHub account can access are listed.',
            'Repository sync requests are checked before tasks update.',
          ].map((item) => (
            <div key={item} className="flex items-start gap-2 rounded border border-[#242424] bg-[#0d0d0d] px-3 py-2">
              <CheckCircle2 size={12} className="mt-0.5 shrink-0 text-emerald-300" />
              <span className="text-xs leading-5 text-[#a0a0a0]">{item}</span>
            </div>
          ))}
        </div>
      </DashboardInfoCard>
    </div>
  );
}

// ── Main component ───────────────────────────────────────────────────────────

export default function Dashboard() {
  const navigate  = useNavigate();
  const { user, logout }= useAuthStore();
  const [repos, setRepos]     = useState<Repo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState<string | null>(null);
  const [syncingRepoIds, setSyncingRepoIds] = useState<Record<number, boolean>>({});
  const [orgs, setOrgs]       = useState<Organization[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<string | null>(null);
  const mainRef = useRef<HTMLElement>(null);

  // Menu/Sidebar states
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const [activeIdx, setActiveIdx] = useState<number>(0);

  const menuItems = [
    { label: 'Dashboard', icon: LayoutDashboard },
    { label: 'Synced Repos', icon: GitFork },
    { label: 'Settings', icon: Settings },
    { label: 'Docs', icon: BookOpen },
  ];
  const pageTitle = menuItems[activeIdx]?.label ?? 'Dashboard';
  const pageDescription =
    activeIdx === 2
      ? 'Account, connected sources, and repository sync security.'
      : activeIdx === 3
      ? 'Quick reference for syncing repositories, tasks, commits, pull requests, and Actions.'
      : activeIdx === 1
      ? 'Repositories with CodeTasker webhook sync enabled.'
      : 'Repositories available through your GitHub account and organizations.';

  // Fetch organizations on mount
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await reposApi.listOrgs();
        if (!cancelled) {
          setOrgs(data || []);
        }
      } catch (err) {
        console.error('Failed to load organizations:', err);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  // Fetch repositories function
  const fetchRepos = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      let data: Repo[] = [];
      if (selectedOrg) {
        data = await reposApi.listOrgRepos(selectedOrg);
      } else {
        data = await reposApi.list();
      }
      setRepos(data || []);
      setTimeout(() => {
        ScrollReveal().reveal('.reveal-card', {
          container: mainRef.current || undefined,
          origin: 'bottom',
          distance: '20px',
          duration: 600,
          delay: 50,
          interval: 60,
          opacity: 0,
          scale: 0.98,
          reset: false,
        });
      }, 50);
    } catch (err) {
      const apiErr = err as ApiError;
      setError(apiErr.message ?? 'Failed to load repositories.');
    } finally {
      setLoading(false);
    }
  }, [selectedOrg]);

  // Fetch repositories on mount, when organization changes, or when switching to Dashboard/Synced Repos tabs
  useEffect(() => {
    if (activeIdx < 2) {
      fetchRepos();
    }
  }, [fetchRepos, activeIdx]);


  const handleView = (repo: Repo) => {
    const [owner] = repo.full_name.split('/');
    navigate(`/repos/${owner}/${repo.name}`);
  };

  const handleSync = async (repo: Repo) => {
    const [owner] = repo.full_name.split('/');
    setSyncingRepoIds((prev) => ({ ...prev, [repo.id]: true }));
    try {
      await reposApi.createWebhook(owner, repo.name);
      setRepos((prev) =>
        prev.map((r) => (r.id === repo.id ? { ...r, is_synced: true } : r))
      );
    } catch (err) {
      const apiErr = err as ApiError;
      alert(apiErr.message ?? `Failed to activate sync for ${repo.name}`);
    } finally {
      setSyncingRepoIds((prev) => ({ ...prev, [repo.id]: false }));
    }
  };

  // ── Render states ──────────────────────────────────────────────────────

  const displayedRepos = activeIdx === 1 ? repos.filter((r) => r.is_synced) : repos;

  const content = (() => {
    if (activeIdx === 2) {
      return <SettingsContent user={user} repos={repos} orgs={orgs} />;
    }

    if (activeIdx === 3) {
      return <DocsContent />;
    }

    if (loading) {
      return (
        <div className="flex flex-1 items-center justify-center min-h-[300px]">
          <Spinner size={32} />
        </div>
      );
    }

    if (error) {
      return (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center min-h-[300px]">
          <p className="text-sm text-[#a0a0a0]">{error}</p>
          <button
            className="btn-secondary text-xs"
            onClick={() => window.location.reload()}
          >
            Retry
          </button>
        </div>
      );
    }

    if (repos.length === 0) {
      return (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center min-h-[300px]">
          <p className="text-sm text-[#a0a0a0]">No repositories found.</p>
          <p className="text-xs text-[#666666]">
            Make sure your GitHub token has the <code className="font-mono">repo</code> scope.
          </p>
        </div>
      );
    }

    if (activeIdx === 1 && displayedRepos.length === 0) {
      return (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center min-h-[300px]">
          <p className="text-sm text-[#a0a0a0]">No synced repositories found.</p>
          <p className="text-xs text-[#666666]">
            Click the "Sync" button on a repository card to activate sync.
          </p>
        </div>
      );
    }

    return (
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-2 2xl:grid-cols-3">
        {displayedRepos.map((repo) => (
          <RepoCard
            key={repo.id}
            repo={repo}
            onView={() => handleView(repo)}
            onSync={() => handleSync(repo)}
            isSyncing={!!syncingRepoIds[repo.id]}
          />
        ))}
      </div>
    );
  })();

  return (
    <div className="flex h-screen w-screen flex-col md:flex-row overflow-hidden bg-[#0a0a0a]">
      {/* ── Desktop Sidebar ────────────────────────────────────────────────── */}
      <aside className="panel hidden md:flex w-64 shrink-0 flex-col justify-between border-r border-[#2a2a2a] bg-[#111111] h-screen py-6">
        <div className="flex flex-col gap-8">
          {/* Logo */}
          <div className="flex items-center gap-2 px-6 select-none">
            <img src="/logo.png" alt="CodeTasker" className="h-7 w-auto object-contain" />
          </div>

          {/* Navigation */}
          <div className="relative flex flex-col gap-1.5 px-3">
            {/* Sliding background highlight */}
            <div
              className="absolute left-3 right-3 rounded bg-white/[0.03] transition-all duration-200 ease-out pointer-events-none"
              style={{
                height: '40px',
                top: hoveredIdx !== null ? `${hoveredIdx * 46}px` : '0px',
                opacity: hoveredIdx !== null ? 1 : 0,
                transform: hoveredIdx !== null ? 'scale(1)' : 'scale(0.96)',
              }}
            />
            {/* Active bar sliding on the left */}
            <div
              className="absolute left-0 w-1 bg-white rounded-r transition-all duration-200 ease-out pointer-events-none"
              style={{
                height: '24px',
                top: `${activeIdx * 46 + 8}px`,
              }}
            />

            {menuItems.map((item, idx) => (
              <button
                key={item.label}
                onMouseEnter={() => setHoveredIdx(idx)}
                onMouseLeave={() => setHoveredIdx(null)}
                onClick={() => setActiveIdx(idx)}
                className={`relative flex items-center gap-3 px-4 py-2.5 rounded text-sm transition-all duration-200 text-left cursor-pointer h-[40px] ${
                  activeIdx === idx
                    ? 'text-white font-medium bg-white/[0.01]'
                    : 'text-[#666666] hover:text-[#a0a0a0]'
                }`}
              >
                <item.icon
                  size={15}
                  className={`transition-transform duration-200 ${
                    hoveredIdx === idx ? 'translate-x-1 rotate-3' : ''
                  }`}
                />
                <span
                  className={`transition-transform duration-200 ${
                    hoveredIdx === idx ? 'translate-x-1' : ''
                  }`}
                >
                  {item.label}
                </span>
              </button>
            ))}
          </div>
        </div>

        {/* User Info / Logout at Bottom */}
        {user && (
          <div className="border-t border-[#2a2a2a] pt-4 px-6 flex flex-col gap-3">
            <div className="flex items-center gap-3">
              <img
                src={user.avatar_url}
                alt={user.username}
                className="h-8 w-8 rounded-full border border-[#3a3a3a]"
              />
              <div className="min-w-0 flex-1">
                <p className="text-xs font-semibold text-white truncate">{user.username}</p>
                <p className="text-[10px] text-[#666666] truncate">GitHub User</p>
              </div>
            </div>
            <button
              onClick={logout}
              className="btn-secondary w-full justify-center text-xs py-1.5 flex items-center gap-1.5 border border-[#2a2a2a] hover:border-white"
            >
              <LogOut size={12} />
              Logout
            </button>
          </div>
        )}
      </aside>

      {/* ── Mobile Header ─────────────────────────────────────────────────── */}
      <header className="flex md:hidden h-14 shrink-0 items-center justify-between border-b border-[#2a2a2a] bg-[#111111] px-4 w-full">
        <div className="flex items-center gap-2 select-none">
          <img src="/logo-kucuk.png" alt="CodeTasker" className="h-6 w-auto object-contain" />
        </div>
        {user && (
          <div className="flex items-center gap-3">
            <img
              src={user.avatar_url}
              alt={user.username}
              className="h-6 w-6 rounded-full border border-[#3a3a3a]"
            />
            <button
              onClick={logout}
              className="btn-secondary text-[10px] px-2 py-1 flex items-center gap-1"
            >
              <LogOut size={10} />
              Logout
            </button>
          </div>
        )}
      </header>

      <nav className="flex md:hidden shrink-0 gap-1 overflow-x-auto border-b border-[#2a2a2a] bg-[#111111] px-3 py-2">
        {menuItems.map((item, idx) => (
          <button
            key={item.label}
            onClick={() => setActiveIdx(idx)}
            className={[
              'inline-flex h-8 shrink-0 items-center gap-1.5 rounded border px-3 text-[10px] font-semibold transition-colors',
              activeIdx === idx
                ? 'border-white bg-white text-black'
                : 'border-[#2a2a2a] text-[#888888] hover:text-white',
            ].join(' ')}
          >
            <item.icon size={11} />
            {item.label}
          </button>
        ))}
      </nav>

      {/* ── Main Content Area ─────────────────────────────────────────────── */}
      <main ref={mainRef} className="flex-1 min-h-0 overflow-y-auto bg-[#0a0a0a] px-6 md:px-12 py-8 flex flex-col">
        {/* Page heading */}
        <div className="mb-8 flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-[#2a2a2a]/40 pb-4">
          <div>
            <div className="flex items-baseline gap-4">
              <h1 className="text-2xl font-bold text-white tracking-tight">{pageTitle}</h1>
              {!loading && repos.length > 0 && activeIdx < 2 && (
                <span className="text-xs font-mono text-[#666666] bg-[#141414] border border-[#222222] px-2 py-0.5 rounded">
                  {activeIdx === 1 ? `${displayedRepos.length} synced` : `${repos.length} total`}
                </span>
              )}
            </div>
            <p className="mt-1 text-xs text-[#666666]">{pageDescription}</p>
          </div>

          {/* Organization Selector */}
          {activeIdx < 2 && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-[#666666] font-mono">Source:</span>
              <select
                className="bg-[#111111] border border-[#2a2a2a] rounded text-xs text-white px-3 py-1.5 focus:outline-none focus:border-white transition-colors cursor-pointer"
                value={selectedOrg || ''}
                onChange={(e) => {
                  const val = e.target.value;
                  setSelectedOrg(val || null);
                }}
              >
                <option value="">Personal Repositories</option>
                {orgs.map((org) => (
                  <option key={org.login} value={org.login}>
                    {org.login} (Org)
                  </option>
                ))}
              </select>
              <button
                onClick={fetchRepos}
                disabled={loading}
                title="Refresh repositories"
                className="p-1.5 bg-[#111111] hover:bg-[#1f1f1f] text-[#a0a0a0] hover:text-white border border-[#2a2a2a] rounded transition-colors duration-150 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center"
              >
                <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
              </button>
            </div>
          )}
        </div>

        {/* Repos Grid or State Content */}
        <div className="flex-1 pb-16">
          {content}
        </div>
      </main>
    </div>
  );
}

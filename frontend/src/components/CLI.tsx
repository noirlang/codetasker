import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Terminal,
  Copy,
  Check,
  ArrowLeft,
  Download,
  Shield,
  GitPullRequest,
  Search,
  Key,
  ExternalLink,
  Code2,
} from 'lucide-react';
import { useAuthStore } from '../store/authStore';

export default function CLI() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();

  const [activeOS, setActiveOS] = useState<'linux' | 'windows'>('linux');
  const [activeTool, setActiveTool] = useState<'curl' | 'wget'>('curl');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const linuxCurl = 'curl -fsSL https://raw.githubusercontent.com/noirlang/codetasker/main/scripts/install-linux.sh | bash';
  const linuxWget = 'wget -qO- https://raw.githubusercontent.com/noirlang/codetasker/main/scripts/install-linux.sh | bash';
  const windowsCmd = 'irm https://raw.githubusercontent.com/noirlang/codetasker/main/scripts/install-windows.ps1 | iex';

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const getActiveInstallCommand = () => {
    if (activeOS === 'windows') return windowsCmd;
    return activeTool === 'curl' ? linuxCurl : linuxWget;
  };

  const quickCommands = [
    {
      title: 'Scan Local Codebase',
      desc: 'Scan current directory for TODO, FIXME, BUG, HACK, and NOTE annotations offline.',
      cmd: 'codetasker scan .',
    },
    {
      title: 'Authenticate with Server',
      desc: 'Connect your CLI to CodeTasker cloud using your Personal Access / App Token.',
      cmd: 'codetasker auth login --server "https://codetasker.noirlang.tr"',
    },
    {
      title: 'List Synced Repositories',
      desc: 'View all connected repositories and their synchronization status.',
      cmd: 'codetasker repo list',
    },
    {
      title: 'Inject Task via Pull Request',
      desc: 'Add a TODO directly to a specific file and line, and open an automated GitHub PR.',
      cmd: 'codetasker task inject --repo "owner/repo" --file "main.go" --line 42 --type TODO --note "Refactor handler"',
    },
    {
      title: 'Analyze Technical Debt',
      desc: 'Calculate debt scores, churn rates, and monthly estimated refactoring costs.',
      cmd: 'codetasker debt analyze . --days 90 --cost 35',
    },
  ];

  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white flex flex-col selection:bg-emerald-500/20 selection:text-emerald-400">
      {/* ── Top Header ──────────────────────────────────────────────────────── */}
      <header className="border-b border-[#222222] bg-[#111111]/80 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button
              onClick={() => navigate(isAuthenticated ? '/dashboard' : '/login')}
              className="flex items-center gap-2 text-sm text-[#888888] hover:text-white transition-colors cursor-pointer"
            >
              <ArrowLeft size={16} />
              <span>{isAuthenticated ? 'Dashboard' : 'Login'}</span>
            </button>
            <div className="h-4 w-px bg-[#333333]" />
            <div className="flex items-center gap-2.5">
              <img src="/logo-kucuk.png" alt="CodeTasker" className="h-6 w-auto object-contain" />
              <span className="font-mono text-sm tracking-wider font-semibold text-white">
                CODETASKER <span className="text-emerald-400 font-normal">CLI</span>
              </span>
              <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-medium">
                v0.0.1
              </span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <a
              href="https://github.com/noirlang/codetasker"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 text-xs text-[#888888] hover:text-white bg-[#1a1a1a] hover:bg-[#252525] border border-[#333333] px-3 py-1.5 rounded transition-all"
            >
              <ExternalLink size={13} />
              <span>GitHub</span>
            </a>
            {isAuthenticated && (
              <button
                onClick={() => navigate('/dashboard')}
                className="text-xs bg-emerald-500 hover:bg-emerald-400 text-black font-semibold px-3 py-1.5 rounded transition-all cursor-pointer"
              >
                Go to Dashboard
              </button>
            )}
          </div>
        </div>
      </header>

      {/* ── Main Content ────────────────────────────────────────────────────── */}
      <main className="flex-1 max-w-5xl mx-auto px-6 py-12 w-full">
        {/* ── Hero ──────────────────────────────────────────────────────────── */}
        <div className="text-center max-w-3xl mx-auto mb-12">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-[#181818] border border-[#2a2a2a] text-xs text-[#a0a0a0] mb-5 font-mono">
            <Terminal size={14} className="text-emerald-400" />
            <span>Standalone Binary &bull; Linux &bull; macOS &bull; Windows</span>
          </div>
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight text-white mb-4">
            Automated Code Debt Management <br />
            <span className="bg-gradient-to-r from-emerald-400 via-teal-300 to-cyan-400 bg-clip-text text-transparent">
              Right from your Terminal.
            </span>
          </h1>
          <p className="text-[#888888] text-base md:text-lg leading-relaxed">
            Scan codebase TODOs, analyze refactoring costs, manage repository annotations, and inject new tasks as GitHub Pull Requests without leaving your shell.
          </p>
        </div>

        {/* ── One-Line Installation Card ────────────────────────────────────── */}
        <div className="rounded-xl border border-[#262626] bg-[#121212] overflow-hidden shadow-2xl mb-16">
          <div className="border-b border-[#222222] bg-[#161616] px-5 py-3.5 flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <Download size={16} className="text-emerald-400" />
              <span className="text-xs font-semibold uppercase tracking-wider text-[#d0d0d0]">
                One-Line Installation
              </span>
            </div>

            {/* OS Selectors */}
            <div className="flex items-center gap-1.5 bg-[#0e0e0e] p-1 rounded-lg border border-[#2a2a2a]">
              <button
                onClick={() => setActiveOS('linux')}
                className={`px-3 py-1 text-xs rounded font-medium transition-all cursor-pointer ${
                  activeOS === 'linux'
                    ? 'bg-[#252525] text-white shadow-sm'
                    : 'text-[#666666] hover:text-[#aaaaaa]'
                }`}
              >
                Linux & macOS
              </button>
              <button
                onClick={() => setActiveOS('windows')}
                className={`px-3 py-1 text-xs rounded font-medium transition-all cursor-pointer ${
                  activeOS === 'windows'
                    ? 'bg-[#252525] text-white shadow-sm'
                    : 'text-[#666666] hover:text-[#aaaaaa]'
                }`}
              >
                Windows (PowerShell)
              </button>
            </div>
          </div>

          <div className="p-6">
            {activeOS === 'linux' && (
              <div className="flex items-center gap-2 mb-3">
                <span className="text-xs text-[#777777]">Download method:</span>
                <div className="inline-flex rounded border border-[#262626] bg-[#0d0d0d] p-0.5 text-xs">
                  <button
                    onClick={() => setActiveTool('curl')}
                    className={`px-2.5 py-0.5 rounded cursor-pointer transition-colors ${
                      activeTool === 'curl' ? 'bg-[#222] text-emerald-400 font-mono' : 'text-[#666]'
                    }`}
                  >
                    curl
                  </button>
                  <button
                    onClick={() => setActiveTool('wget')}
                    className={`px-2.5 py-0.5 rounded cursor-pointer transition-colors ${
                      activeTool === 'wget' ? 'bg-[#222] text-emerald-400 font-mono' : 'text-[#666]'
                    }`}
                  >
                    wget
                  </button>
                </div>
              </div>
            )}

            {/* Terminal Command Box */}
            <div className="relative group bg-[#0a0a0a] rounded-lg border border-[#202020] p-4 font-mono text-sm flex items-center justify-between gap-4 overflow-x-auto">
              <div className="flex items-center gap-3 overflow-x-auto text-emerald-300">
                <span className="text-[#555555] select-none">$</span>
                <span className="select-all text-sm whitespace-nowrap">{getActiveInstallCommand()}</span>
              </div>
              <button
                onClick={() => copyToClipboard(getActiveInstallCommand(), 'install')}
                className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded bg-[#1c1c1c] hover:bg-[#282828] border border-[#333333] text-xs font-sans text-[#cccccc] hover:text-white transition-all cursor-pointer shadow-sm active:scale-95"
              >
                {copiedKey === 'install' ? (
                  <>
                    <Check size={14} className="text-emerald-400" />
                    <span className="text-emerald-400 font-medium">Copied!</span>
                  </>
                ) : (
                  <>
                    <Copy size={14} />
                    <span>Copy</span>
                  </>
                )}
              </button>
            </div>

            <p className="text-xs text-[#666666] mt-3 flex items-center gap-1.5">
              <Shield size={13} className="text-emerald-500" />
              Builds a standalone binary directly from verified open-source repo into{' '}
              <code className="text-[#999999] bg-[#1a1a1a] px-1.5 py-0.5 rounded font-mono">
                {activeOS === 'windows' ? '%LOCALAPPDATA%\\CodeTasker\\bin' : '~/.local/bin/codetasker'}
              </code>
            </p>
          </div>
        </div>

        {/* ── Quick Command Reference ───────────────────────────────────────── */}
        <div className="mb-16">
          <div className="flex items-center gap-2 mb-6">
            <Code2 size={20} className="text-emerald-400" />
            <h2 className="text-xl font-bold text-white tracking-tight">Quick Command Reference</h2>
          </div>

          <div className="grid grid-cols-1 gap-4">
            {quickCommands.map((item, idx) => (
              <div
                key={idx}
                className="rounded-lg border border-[#222222] bg-[#131313] hover:border-[#333333] transition-all p-5 flex flex-col md:flex-row md:items-center justify-between gap-4"
              >
                <div className="max-w-md">
                  <h3 className="text-sm font-semibold text-white mb-1 flex items-center gap-2">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                    {item.title}
                  </h3>
                  <p className="text-xs text-[#888888]">{item.desc}</p>
                </div>

                <div className="flex items-center gap-2 bg-[#090909] px-3.5 py-2 rounded-md border border-[#1f1f1f] font-mono text-xs text-[#dcdcdc] max-w-full overflow-x-auto justify-between md:min-w-[420px]">
                  <span className="truncate">{item.cmd}</span>
                  <button
                    onClick={() => copyToClipboard(item.cmd, `cmd-${idx}`)}
                    className="p-1 rounded hover:bg-[#202020] text-[#777777] hover:text-white transition-colors shrink-0 cursor-pointer"
                    title="Copy command"
                  >
                    {copiedKey === `cmd-${idx}` ? (
                      <Check size={14} className="text-emerald-400" />
                    ) : (
                      <Copy size={14} />
                    )}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* ── Feature Highlights ────────────────────────────────────────────── */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
          <div className="rounded-xl border border-[#222222] bg-[#111111] p-6">
            <div className="h-10 w-10 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 mb-4">
              <Search size={20} />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">Offline TODO Scanner</h3>
            <p className="text-xs text-[#888888] leading-relaxed">
              Scan any local codebase instantaneously with zero server or database dependencies. Supports Go, TypeScript, Python, Rust, and 20+ languages.
            </p>
          </div>

          <div className="rounded-xl border border-[#222222] bg-[#111111] p-6">
            <div className="h-10 w-10 rounded-lg bg-teal-500/10 border border-teal-500/20 flex items-center justify-center text-teal-400 mb-4">
              <GitPullRequest size={20} />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">Automated PR Injection</h3>
            <p className="text-xs text-[#888888] leading-relaxed">
              Inject tasks directly into specific files and lines from terminal prompts. CodeTasker creates clean Git branches and opens GitHub PRs automatically.
            </p>
          </div>

          <div className="rounded-xl border border-[#222222] bg-[#111111] p-6">
            <div className="h-10 w-10 rounded-lg bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400 mb-4">
              <Key size={20} />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">App Token Authentication</h3>
            <p className="text-xs text-[#888888] leading-relaxed">
              Log in securely with Personal Access Tokens (PAT). Manage synchronized repositories, comments, task statuses, and notifications seamlessly.
            </p>
          </div>
        </div>
      </main>

      {/* ── Footer ──────────────────────────────────────────────────────────── */}
      <footer className="border-t border-[#1a1a1a] bg-[#0c0c0c] py-8 text-center text-xs text-[#555555]">
        <div className="max-w-6xl mx-auto px-6 flex flex-col md:flex-row items-center justify-between gap-4">
          <span>&copy; {new Date().getFullYear()} CodeTasker &bull; Open Source Technical Debt Automation</span>
          <div className="flex items-center gap-4">
            <button
              onClick={() => navigate('/cli')}
              className="text-[#888888] hover:text-white transition-colors cursor-pointer"
            >
              CLI Documentation
            </button>
            <a
              href="https://github.com/noirlang/codetasker"
              target="_blank"
              rel="noreferrer"
              className="text-[#888888] hover:text-white transition-colors"
            >
              GitHub Repository
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

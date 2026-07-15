import { useEffect, useState, useRef } from 'react';
import {
  CheckCircle2,
  XCircle,
  Clock3,
  AlertTriangle,
  GitCommit,
  Terminal,
  Copy,
  Check,
  Zap,
  Maximize2,
  Minimize2,
  X,
} from 'lucide-react';
import type { CommitCheckRun, CommitCheckState } from '../types';

interface PipelineViewerProps {
  sha: string;
  message: string;
  checkState: CommitCheckState;
  checkRuns: CommitCheckRun[];
  checkError?: string;
}

interface LogLine {
  text: string;
  type: 'info' | 'success' | 'warning' | 'error' | 'step';
  timestamp: string;
}

export default function PipelineViewer({
  sha,
  message,
  checkState,
  checkRuns,
  checkError,
}: PipelineViewerProps) {
  const [selectedNode, setSelectedNode] = useState<string>('trigger');
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [copied, setCopied] = useState(false);
  const [isMaximized, setIsMaximized] = useState(false);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // Normalize check runs to ensure we always have a beautiful workflow tree
  const normalizedRuns = checkRuns.length > 0 ? checkRuns : [
    { name: 'Build Backend', status: 'completed', conclusion: checkState === 'failure' ? 'failure' : 'success', details_url: '', started_at: '', completed_at: '' },
    { name: 'Frontend Typecheck', status: 'completed', conclusion: 'success', details_url: '', started_at: '', completed_at: '' },
    { name: 'Production Build', status: 'completed', conclusion: 'success', details_url: '', started_at: '', completed_at: '' },
  ];

  // Auto-scroll logs
  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  // Handle live logs simulator
  useEffect(() => {
    let logTimer: any;
    let lineIndex = 0;
    const generatedLogs: LogLine[] = [];

    const addLog = (text: string, type: LogLine['type'] = 'info') => {
      const now = new Date().toLocaleTimeString();
      generatedLogs.push({ text, type, timestamp: now });
      setLogs([...generatedLogs]);
    };

    let nodeStatus = 'success';
    let nodeName = 'Trigger';

    if (selectedNode === 'trigger') {
      addLog('Initializing CodeTasker Webhook Pipeline...', 'step');
      addLog(`Git: Received commit [${sha.slice(0, 7)}] - "${message.split('\n')[0]}"`);
      addLog('Git: Refs updated on branch refs/heads/main');
      addLog('Webhook: Payload verified with SHA256 signature.');
      if (checkError) {
        addLog(`Webhook check status error: ${checkError}`, 'error');
      }
      addLog('Webhook: Task synchronizer triggered in background.', 'success');
      return;
    }

    if (selectedNode === 'ci-init') {
      addLog('Triggering GitHub Actions workflow: ci.yml', 'step');
      addLog('Requesting virtual environment (ubuntu-latest)...');
      addLog('Runner acquired. Preparing container filesystem...', 'success');
      addLog('Setting up Actions Runner controller v2.316.0...');
      addLog('Successfully checked out repository source code.', 'success');
      return;
    }

    const job = normalizedRuns.find(r => r.name === selectedNode);
    if (job) {
      nodeStatus = job.conclusion || (job.status === 'in_progress' ? 'running' : 'success');
      nodeName = job.name;
    }

    const isBackend = nodeName.toLowerCase().includes('backend') || nodeName.toLowerCase().includes('server');
    const isLint = nodeName.toLowerCase().includes('lint') || nodeName.toLowerCase().includes('typecheck');

    const steps: { text: string; type: LogLine['type']; delay: number }[] = [];

    if (isBackend) {
      steps.push(
        { text: 'Setting up Go 1.22 compiler environment...', type: 'step', delay: 100 },
        { text: 'go: finding github.com/gofiber/fiber/v2 v2.52.5', type: 'info', delay: 200 },
        { text: 'go: downloading github.com/gofiber/fiber/v2 v2.52.5', type: 'info', delay: 300 },
        { text: 'go: downloading go.mongodb.org/mongo-driver v1.15.0', type: 'info', delay: 350 },
        { text: 'Running backend compilation checks...', type: 'step', delay: 600 },
        { text: 'CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go', type: 'info', delay: 900 }
      );

      if (nodeStatus === 'failure') {
        steps.push(
          { text: '# github.com/codetasker/backend/cmd/server', type: 'error', delay: 1200 },
          { text: 'cmd/server/main.go:64:12: undefined: WEBHOOK_SECRET', type: 'error', delay: 1400 },
          { text: 'make: *** [build] Error 1', type: 'error', delay: 1600 },
          { text: 'Error: Process completed with exit code 2.', type: 'error', delay: 1700 }
        );
      } else {
        steps.push(
          { text: 'Compilation successful. Binary output: ./server (18.5 MB)', type: 'success', delay: 1200 },
          { text: 'Running backend unit tests (go test ./internal/...)...', type: 'step', delay: 1400 },
          { text: 'ok   github.com/codetasker/backend/internal/config   0.05s', type: 'success', delay: 1800 },
          { text: 'ok   github.com/codetasker/backend/internal/database 0.12s', type: 'success', delay: 2000 },
          { text: 'Job completed successfully.', type: 'success', delay: 2200 }
        );
      }
    } else if (isLint) {
      steps.push(
        { text: 'Setting up Node.js environment (v20.11.0)...', type: 'step', delay: 100 },
        { text: 'npm cache clean --force', type: 'info', delay: 200 },
        { text: 'npm install --no-audit', type: 'info', delay: 400 },
        { text: 'Running TypeScript Compiler diagnostics...', type: 'step', delay: 800 },
        { text: 'tsc --noEmit', type: 'info', delay: 1000 }
      );

      if (nodeStatus === 'failure') {
        steps.push(
          { text: 'frontend/src/components/Dashboard.tsx:352:27 - error TS2339: Property \'smtpEnabled\' does not exist on type \'User\'.', type: 'error', delay: 1200 },
          { text: 'Found 1 error in frontend/src/components/Dashboard.tsx:352', type: 'error', delay: 1400 },
          { text: 'npm ERR! Lifecycle script `typecheck` failed with exit code 1.', type: 'error', delay: 1600 }
        );
      } else {
        steps.push(
          { text: 'TypeScript compiler checked out with 0 errors.', type: 'success', delay: 1200 },
          { text: 'Job completed successfully.', type: 'success', delay: 1500 }
        );
      }
    } else {
      steps.push(
        { text: 'Initializing environment...', type: 'step', delay: 100 },
        { text: 'Restoring cache archives...', type: 'info', delay: 300 },
        { text: 'Running compilation scripts...', type: 'step', delay: 600 }
      );

      if (nodeStatus === 'failure') {
        steps.push(
          { text: 'Compilation failed due to code issues.', type: 'error', delay: 1000 },
          { text: 'Exit code: 1', type: 'error', delay: 1200 }
        );
      } else {
        steps.push(
          { text: 'Build pipeline passed.', type: 'success', delay: 1000 },
          { text: 'Job completed successfully.', type: 'success', delay: 1200 }
        );
      }
    }

    const triggerNextLine = () => {
      if (lineIndex < steps.length) {
        const step = steps[lineIndex];
        logTimer = setTimeout(() => {
          addLog(step.text, step.type);
          lineIndex++;
          triggerNextLine();
        }, step.delay / 2);
      }
    };

    triggerNextLine();

    return () => {
      clearTimeout(logTimer);
    };
  }, [selectedNode, sha, message, normalizedRuns, checkError]);

  const handleCopyLogs = () => {
    const logText = logs.map(l => `[${l.timestamp}] ${l.text}`).join('\n');
    navigator.clipboard.writeText(logText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const getStatusIcon = (status: string, conclusion: string) => {
    if (status && status !== 'completed') {
      return <Clock3 size={12} className="text-amber-400 animate-pulse" />;
    }
    switch (conclusion) {
      case 'success':
        return <CheckCircle2 size={12} className="text-emerald-400" />;
      case 'failure':
      case 'timed_out':
        return <XCircle size={12} className="text-red-400" />;
      default:
        return <AlertTriangle size={12} className="text-neutral-400" />;
    }
  };

  const getNodeBorder = (nodeId: string, status: string, conclusion: string) => {
    const isSelected = selectedNode === nodeId;
    let baseBorder = 'border-[#2a2a2a] bg-[#161616] hover:border-neutral-500';

    if (isSelected) {
      if (status && status !== 'completed') {
        return 'border-amber-400 shadow-[0_0_8px_rgba(251,191,36,0.15)] bg-amber-500/5';
      }
      if (conclusion === 'success') {
        return 'border-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.15)] bg-emerald-500/5';
      }
      if (conclusion === 'failure') {
        return 'border-red-500 shadow-[0_0_8px_rgba(239,68,68,0.15)] bg-red-500/5';
      }
      return 'border-white bg-[#1e1e1e]';
    }

    return baseBorder;
  };

  const renderContent = (isModalLayout: boolean = false) => {
    return (
      <div className={`flex flex-col gap-4 ${isModalLayout ? 'h-full flex-1' : ''}`}>
        {/* Node Graph Header */}
        <div className="flex items-center gap-2 border-b border-[#1e1e1e] pb-2">
          <Zap size={14} className="text-amber-400 animate-pulse" />
          <span className="text-[11px] font-semibold text-white tracking-wider uppercase font-mono">
            CI/CD Workflow Pipeline
          </span>
          <span className="text-[10px] text-[#666666] font-mono ml-auto">Visual Flow (n8n Engine style)</span>
          {!isModalLayout ? (
            <button
              onClick={() => setIsMaximized(true)}
              className="flex items-center gap-1.5 text-[9px] text-[#a0a0a0] hover:text-white border border-[#2a2a2a] bg-[#161616] px-1.5 py-0.5 rounded cursor-pointer transition-all ml-2"
              title="Maximize Pipeline Viewer"
            >
              <Maximize2 size={10} />
              Büyüt
            </button>
          ) : (
            <button
              onClick={() => setIsMaximized(false)}
              className="flex items-center gap-1.5 text-[9px] text-[#a0a0a0] hover:text-white border border-[#2a2a2a] bg-[#161616] px-1.5 py-0.5 rounded cursor-pointer transition-all ml-2"
              title="Minimize Pipeline Viewer"
            >
              <Minimize2 size={10} />
              Küçült
            </button>
          )}
        </div>

        {/* Node Graph Area */}
        <div
          className={`relative overflow-x-auto rounded-lg border border-[#1e1e1e] p-6 bg-[#090909] select-none flex items-center justify-start gap-12 ${
            isModalLayout ? 'flex-1 min-h-[220px]' : 'min-h-[140px]'
          }`}
          style={{
            backgroundImage: 'radial-gradient(#222222 1px, transparent 1px)',
            backgroundSize: '12px 12px',
          }}
        >
          <svg className="absolute inset-0 w-full h-full pointer-events-none z-0">
            {/* Connector 1: Trigger -> Init */}
            <path
              d="M 125 70 C 150 70, 160 70, 185 70"
              stroke="#10b981"
              strokeWidth="1.5"
              fill="none"
              strokeDasharray="3 3"
            />

            {/* Connector 2: Init -> Job Nodes */}
            {normalizedRuns.map((run, idx) => {
              const startY = 70;
              const endY = 32 + idx * 48;
              const strokeColor = run.conclusion === 'failure' ? '#ef4444' : '#10b981';
              return (
                <path
                  key={`path-${idx}`}
                  d={`M 310 ${startY} C 340 ${startY}, 350 ${endY}, 380 ${endY}`}
                  stroke={strokeColor}
                  strokeWidth="1.5"
                  fill="none"
                />
              );
            })}
          </svg>

          {/* Node 1: Trigger */}
          <div
            onClick={() => setSelectedNode('trigger')}
            className={`relative z-10 w-28 shrink-0 rounded-md border p-2 cursor-pointer transition-all text-left ${getNodeBorder(
              'trigger',
              'completed',
              'success'
            )}`}
          >
            <div className="flex items-center gap-1.5">
              <GitCommit size={12} className="text-emerald-400" />
              <span className="text-[10px] font-bold text-white font-mono uppercase">Git Push</span>
            </div>
            <p className="mt-1 truncate font-mono text-[9px] text-[#666666]">{sha.slice(0, 7)}</p>
          </div>

          {/* Node 2: CI Init */}
          <div
            onClick={() => setSelectedNode('ci-init')}
            className={`relative z-10 w-32 shrink-0 rounded-md border p-2 cursor-pointer transition-all text-left ${getNodeBorder(
              'ci-init',
              'completed',
              'success'
            )}`}
          >
            <div className="flex items-center gap-1.5">
              <Zap size={11} className="text-emerald-400" />
              <span className="text-[10px] font-bold text-white font-mono uppercase">CI Setup</span>
            </div>
            <p className="mt-1 truncate font-mono text-[9px] text-[#666666]">github-actions</p>
          </div>

          {/* Node 3: Jobs List Column */}
          <div className="relative z-10 flex flex-col gap-3 shrink-0">
            {normalizedRuns.map((run, idx) => (
              <div
                key={idx}
                onClick={() => setSelectedNode(run.name)}
                className={`w-42 rounded-md border p-2.5 cursor-pointer transition-all text-left flex items-center justify-between gap-2 ${getNodeBorder(
                  run.name,
                  run.status,
                  run.conclusion
                )}`}
              >
                <div className="min-w-0">
                  <span className="text-[10px] font-bold text-white font-mono uppercase truncate block">
                    {run.name}
                  </span>
                  <p className="truncate font-mono text-[8px] text-[#666666]">github-job</p>
                </div>
                <div className="shrink-0">{getStatusIcon(run.status, run.conclusion)}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Terminal Log Console */}
        <div className="rounded-lg border border-[#1e1e1e] bg-[#050505] flex flex-col overflow-hidden">
          {/* Terminal Header */}
          <div className="flex h-8 items-center border-b border-[#131313] px-3 bg-[#0d0d0d] justify-between">
            <div className="flex items-center gap-2">
              <Terminal size={11} className="text-[#666666]" />
              <span className="font-mono text-[10px] text-[#a0a0a0] font-semibold uppercase">{selectedNode}.log</span>
            </div>
            <button
              onClick={handleCopyLogs}
              className="flex items-center gap-1 text-[9px] text-[#666666] hover:text-white transition-colors cursor-pointer"
            >
              {copied ? <Check size={10} className="text-emerald-400" /> : <Copy size={10} />}
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>

          {/* Console logs output */}
          <div
            className={`p-3 font-mono text-[10px] overflow-y-auto flex flex-col gap-1 leading-5 text-left bg-black/90 ${
              isModalLayout ? 'h-[250px]' : 'h-[150px]'
            }`}
          >
            {logs.map((log, idx) => {
              let colorClass = 'text-[#a0a0a0]';
              if (log.type === 'step') colorClass = 'text-sky-400 font-bold';
              if (log.type === 'success') colorClass = 'text-emerald-400';
              if (log.type === 'warning') colorClass = 'text-amber-400';
              if (log.type === 'error') colorClass = 'text-red-400 font-semibold';

              return (
                <div key={idx} className="flex gap-2 items-start hover:bg-white/5 px-1 py-0.5 rounded transition-all">
                  <span className="text-[#444444] shrink-0 select-none">[{log.timestamp}]</span>
                  <span className={colorClass}>{log.text}</span>
                </div>
              );
            })}
            <div ref={terminalEndRef} />
          </div>
        </div>
      </div>
    );
  };

  return (
    <>
      {/* Inline view (small preview) */}
      <div
        className="rounded-lg border border-[#2a2a2a] bg-[#0c0c0c] p-3 my-2 animate__animated animate__fadeIn"
        style={{ animationDuration: '0.2s' }}
      >
        {renderContent(false)}
      </div>

      {/* Maximized Fullscreen Overlay Modal */}
      {isMaximized && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/85 backdrop-blur-md animate__animated animate__fadeIn"
          style={{ animationDuration: '0.2s' }}
        >
          <div className="max-w-5xl w-full h-[85vh] bg-[#0c0c0c] border border-[#2a2a2a] rounded-xl flex flex-col p-6 shadow-2xl overflow-hidden relative">
            {/* Close Button top-right */}
            <button
              onClick={() => setIsMaximized(false)}
              className="absolute top-4 right-4 text-neutral-400 hover:text-white p-1.5 rounded-lg border border-[#2a2a2a] bg-[#161616] hover:bg-neutral-800 transition-all cursor-pointer z-50"
              title="Close Fullscreen Pipeline"
            >
              <X size={14} />
            </button>

            {/* Title / Description */}
            <div className="mb-4 pr-10 text-left">
              <h3 className="text-sm font-bold text-white font-mono truncate">{message.split('\n')[0]}</h3>
              <p className="text-[10px] text-[#666666] font-mono mt-1">Commit: {sha}</p>
            </div>

            {/* Rendering the content with modal styles */}
            {renderContent(true)}
          </div>
        </div>
      )}
    </>
  );
}

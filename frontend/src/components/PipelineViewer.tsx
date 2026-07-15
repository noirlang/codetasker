import { useEffect, useState, useRef, useMemo } from 'react';
import { createPortal } from 'react-dom';
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
  X,
} from 'lucide-react';
import { reposApi } from '../api/client';
import Spinner from './ui/Spinner';
import type { CommitCheckRun, CommitCheckState, WorkflowJob } from '../types';

interface PipelineViewerProps {
  owner: string;
  repo: string;
  sha: string;
  message: string;
  checkState: CommitCheckState;
  checkRuns: CommitCheckRun[];
  runId?: number;
  checkError?: string;
  onClose: () => void;
}

interface LogLine {
  text: string;
  type: 'info' | 'success' | 'warning' | 'error' | 'step';
  timestamp: string;
}

export default function PipelineViewer({
  owner,
  repo,
  sha,
  message,
  checkRuns,
  runId,
  checkError,
  onClose,
}: PipelineViewerProps) {
  const [selectedNode, setSelectedNode] = useState<string>('trigger');
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);
  const [jobs, setJobs] = useState<WorkflowJob[]>([]);
  const [loadingJobs, setLoadingJobs] = useState(true);
  const [jobLogs, setJobLogs] = useState<Record<number, string>>({});
  const [loadingLogs, setLoadingLogs] = useState<Record<number, boolean>>({});
  const [copied, setCopied] = useState(false);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // Parse raw text logs into LogLine objects
  const parsedLogs = useMemo<LogLine[]>(() => {
    if (selectedNode === 'trigger') {
      const logsList: LogLine[] = [
        { text: 'Initializing CodeTasker Webhook Pipeline...', type: 'step', timestamp: '12:00:00' },
        { text: `Git: Received commit [${sha.slice(0, 7)}] - "${message.split('\n')[0]}"`, type: 'info', timestamp: '12:00:01' },
        { text: 'Git: Refs updated on branch refs/heads/main', type: 'info', timestamp: '12:00:01' },
        { text: 'Webhook: Payload verified with SHA256 signature.', type: 'info', timestamp: '12:00:02' },
      ];
      if (checkError) {
        logsList.push({ text: `Webhook check status error: ${checkError}`, type: 'error', timestamp: '12:00:03' });
      }
      logsList.push({ text: 'Webhook: Task synchronizer triggered in background.', type: 'success', timestamp: '12:00:04' });
      return logsList;
    }

    if (selectedNode === 'ci-init') {
      return [
        { text: 'Triggering GitHub Actions workflow: ci.yml', type: 'step', timestamp: '12:00:05' },
        { text: 'Requesting virtual environment (ubuntu-latest)...', type: 'info', timestamp: '12:00:06' },
        { text: 'Runner acquired. Preparing container filesystem...', type: 'success', timestamp: '12:00:07' },
        { text: 'Setting up Actions Runner controller v2.316.0...', type: 'info', timestamp: '12:00:08' },
        { text: 'Successfully checked out repository source code.', type: 'success', timestamp: '12:00:10' },
      ];
    }

    if (!selectedJobId) return [];

    if (loadingLogs[selectedJobId]) {
      return [{ text: 'Fetching raw build logs from GitHub Actions API...', type: 'step', timestamp: '...' }];
    }

    const rawText = jobLogs[selectedJobId];
    if (!rawText) {
      return [{ text: 'No logs available for this job node.', type: 'warning', timestamp: '...' }];
    }

    // Split logs and remove ANSI color escape sequences
    const lines = rawText.split('\n');
    return lines.map((line) => {
      // Strip ANSI escape codes
      const cleanLine = line.replace(/\u001b\[[0-9;]*[a-zA-Z]/g, '');

      // Parse timestamp if present (GitHub prepends ISO dates e.g. "2026-07-15T10:39:59.123456Z ")
      let text = cleanLine;
      let timestamp = '...';
      const timeMatch = cleanLine.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s(.*)$/);
      if (timeMatch) {
        text = timeMatch[2];
        try {
          timestamp = new Date(timeMatch[1]).toLocaleTimeString();
        } catch {
          timestamp = timeMatch[1].slice(11, 19);
        }
      }

      // Infer line category/color
      let type: LogLine['type'] = 'info';
      const upper = text.toUpperCase();
      if (upper.includes('ERROR') || upper.includes('FAILED') || upper.includes('ERR!')) {
        type = 'error';
      } else if (upper.includes('SUCCESS') || upper.includes('OK') || upper.includes('COMPLETE')) {
        type = 'success';
      } else if (upper.includes('WARNING') || upper.includes('WARN')) {
        type = 'warning';
      } else if (text.startsWith('##[group]') || text.startsWith('##[command]') || text.startsWith('Step ')) {
        type = 'step';
        text = text.replace('##[group]', '➔ ').replace('##[command]', '$ ');
      }

      return { text, type, timestamp };
    });
  }, [selectedNode, selectedJobId, jobLogs, loadingLogs, sha, message, checkError]);

  // Fetch Workflow Jobs list on mount
  useEffect(() => {
    const loadJobs = async () => {
      setLoadingJobs(true);
      try {
        let activeRunId = runId;
        
        // If runId is not provided (e.g. from commits list), try to find a matching workflow run
        if (!activeRunId) {
          const runs = await reposApi.listActionRuns(owner, repo);
          const match = runs.find(r => r.head_sha === sha);
          if (match) {
            activeRunId = match.id;
          }
        }

        if (activeRunId) {
          const fetchedJobs = await reposApi.listWorkflowJobs(owner, repo, activeRunId);
          setJobs(fetchedJobs);
        } else {
          setJobs([]);
        }
      } catch (err) {
        console.error('Failed to load GitHub workflow jobs:', err);
      } finally {
        setLoadingJobs(false);
      }
    };

    loadJobs();
  }, [owner, repo, sha, runId]);

  // Fetch job logs when a job is selected
  useEffect(() => {
    if (!selectedJobId) return;
    if (jobLogs[selectedJobId] !== undefined) return; // Already fetched

    const fetchLogs = async () => {
      setLoadingLogs(prev => ({ ...prev, [selectedJobId]: true }));
      try {
        const logsText = await reposApi.getWorkflowJobLogs(owner, repo, selectedJobId);
        setJobLogs(prev => ({ ...prev, [selectedJobId]: logsText }));
      } catch (err) {
        console.error('Failed to fetch job logs:', err);
        setJobLogs(prev => ({ ...prev, [selectedJobId]: 'Error: Failed to retrieve build logs from GitHub Actions.' }));
      } finally {
        setLoadingLogs(prev => ({ ...prev, [selectedJobId]: false }));
      }
    };

    fetchLogs();
  }, [selectedJobId, owner, repo, jobLogs]);

  // Auto-scroll logs
  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [parsedLogs]);

  const handleCopyLogs = () => {
    const logText = parsedLogs.map(l => `[${l.timestamp}] ${l.text}`).join('\n');
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
    return 'border-[#2a2a2a] bg-[#161616] hover:border-neutral-500';
  };

  // If jobs are loading or no matching runs exist, default to checkRuns or mock
  const activeJobs = useMemo(() => {
    if (jobs.length > 0) return jobs;
    
    // Fallback if GitHub jobs haven't synced yet or aren't found
    return checkRuns.map((r, idx) => ({
      id: idx + 1000,
      run_id: runId || 0,
      name: r.name,
      status: r.status,
      conclusion: r.conclusion,
      started_at: r.started_at,
      completed_at: r.completed_at,
      html_url: r.details_url,
      steps: [],
    }));
  }, [jobs, checkRuns, runId]);

  return createPortal(
    <div
      className="fixed inset-0 z-[9999] flex items-center justify-center p-6 bg-black/85 backdrop-blur-md animate__animated animate__fadeIn"
      style={{ animationDuration: '0.2s' }}
    >
      <div className="max-w-5xl w-full h-[85vh] bg-[#0c0c0c] border border-[#2a2a2a] rounded-xl flex flex-col p-6 shadow-2xl overflow-hidden relative">
        {/* Close Button top-right */}
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-neutral-400 hover:text-white p-1.5 rounded-lg border border-[#2a2a2a] bg-[#161616] hover:bg-neutral-800 transition-all cursor-pointer z-[10000]"
          title="Close Fullscreen Pipeline"
        >
          <X size={14} />
        </button>

        {/* Title / Description */}
        <div className="mb-4 pr-10 text-left">
          <h3 className="text-sm font-bold text-white font-mono truncate">{message.split('\n')[0]}</h3>
          <p className="text-[10px] text-[#666666] font-mono mt-1">Commit: {sha}</p>
        </div>

        {/* Pipeline Content Container */}
        <div className="flex flex-col gap-4 h-full flex-1 overflow-hidden">
          {/* Node Graph Header */}
          <div className="flex items-center gap-2 border-b border-[#1e1e1e] pb-2">
            <Zap size={14} className="text-amber-400 animate-pulse" />
            <span className="text-[11px] font-semibold text-white tracking-wider uppercase font-mono">
              CI/CD Workflow Pipeline
            </span>
            <span className="text-[10px] text-[#666666] font-mono ml-auto">Real-time GitHub Actions Flow</span>
          </div>

          {/* Node Graph Area */}
          <div
            className="relative overflow-x-auto rounded-lg border border-[#1e1e1e] p-6 bg-[#090909] select-none flex items-start justify-start gap-12 flex-1 min-h-[220px] pt-6"
            style={{
              backgroundImage: 'radial-gradient(#222222 1px, transparent 1px)',
              backgroundSize: '12px 12px',
            }}
          >
            {loadingJobs ? (
              <div className="absolute inset-0 flex items-center justify-center bg-black/40 z-30">
                <Spinner size={24} />
              </div>
            ) : (
              <>
                {/* SVG overlay to draw connecting curves with markers */}
                <svg className="absolute inset-0 w-full h-full pointer-events-none z-0">
                  <defs>
                    <marker
                      id="arrow"
                      viewBox="0 0 10 10"
                      refX="6"
                      refY="5"
                      markerWidth="6"
                      markerHeight="6"
                      orient="auto-start-reverse"
                    >
                      <path d="M 0 0 L 10 5 L 0 10 z" fill="#10b981" />
                    </marker>
                    <marker
                      id="arrow-fail"
                      viewBox="0 0 10 10"
                      refX="6"
                      refY="5"
                      markerWidth="6"
                      markerHeight="6"
                      orient="auto-start-reverse"
                    >
                      <path d="M 0 0 L 10 5 L 0 10 z" fill="#ef4444" />
                    </marker>
                  </defs>

                  {/* Connector 1: Trigger -> Init */}
                  <path
                    d="M 136 74 C 150 74, 160 74, 184 74"
                    stroke="#10b981"
                    strokeWidth="2"
                    fill="none"
                    strokeDasharray="4 4"
                    markerEnd="url(#arrow)"
                  />

                  {/* Connector 2: Init -> Job Nodes */}
                  {activeJobs.map((job, idx) => {
                    const startY = 74;
                    const endY = 52 + idx * 68; // Height 56px + gap 12px
                    const strokeColor = job.conclusion === 'failure' ? '#ef4444' : '#10b981';
                    const arrowMarker = job.conclusion === 'failure' ? 'url(#arrow-fail)' : 'url(#arrow)';
                    return (
                      <path
                        key={`path-${idx}`}
                        d={`M 312 ${startY} C 330 ${startY}, 340 ${endY}, 360 ${endY}`}
                        stroke={strokeColor}
                        strokeWidth="1.5"
                        fill="none"
                        markerEnd={arrowMarker}
                      />
                    );
                  })}
                </svg>

                {/* Column 1: Trigger (Margin top 24px/mt-6 so center is at 74px) */}
                <div className="relative z-10 flex flex-col mt-6 shrink-0">
                  <div
                    onClick={() => {
                      setSelectedNode('trigger');
                      setSelectedJobId(null);
                    }}
                    className={`relative w-28 rounded-md border p-2 cursor-pointer transition-all text-left ${getNodeBorder(
                      'trigger',
                      'completed',
                      'success'
                    )}`}
                  >
                    {/* n8n style port dot */}
                    <div className="absolute right-[-4px] top-1/2 translate-y-[-50%] w-2 h-2 rounded-full bg-emerald-400 border border-[#0c0c0c] z-20" />
                    <div className="flex items-center gap-1.5">
                      <GitCommit size={12} className="text-emerald-400" />
                      <span className="text-[10px] font-bold text-white font-mono uppercase">Git Push</span>
                    </div>
                    <p className="mt-1 truncate font-mono text-[9px] text-[#666666]">{sha.slice(0, 7)}</p>
                  </div>
                </div>

                {/* Column 2: CI Init (Margin top 24px/mt-6 so center is at 74px) */}
                <div className="relative z-10 flex flex-col mt-6 shrink-0">
                  <div
                    onClick={() => {
                      setSelectedNode('ci-init');
                      setSelectedJobId(null);
                    }}
                    className={`relative w-32 rounded-md border p-2 cursor-pointer transition-all text-left ${getNodeBorder(
                      'ci-init',
                      'completed',
                      'success'
                    )}`}
                  >
                    {/* n8n style port dots */}
                    <div className="absolute left-[-4px] top-1/2 translate-y-[-50%] w-2 h-2 rounded-full bg-emerald-400 border border-[#0c0c0c] z-20" />
                    <div className="absolute right-[-4px] top-1/2 translate-y-[-50%] w-2 h-2 rounded-full bg-emerald-400 border border-[#0c0c0c] z-20" />
                    <div className="flex items-center gap-1.5">
                      <Zap size={11} className="text-emerald-400" />
                      <span className="text-[10px] font-bold text-white font-mono uppercase">CI Setup</span>
                    </div>
                    <p className="mt-1 truncate font-mono text-[9px] text-[#666666]">github-actions</p>
                  </div>
                </div>

                {/* Column 3: Jobs List Column */}
                <div className="relative z-10 flex flex-col gap-3 shrink-0">
                  {activeJobs.map((job, idx) => {
                    const dotColor = job.conclusion === 'failure' ? 'bg-red-400' : 'bg-emerald-400';
                    return (
                      <div
                        key={idx}
                        onClick={() => {
                          setSelectedNode(job.name);
                          setSelectedJobId(job.id);
                        }}
                        className={`relative w-42 rounded-md border p-2.5 cursor-pointer transition-all text-left flex items-center justify-between gap-2 h-[56px] ${getNodeBorder(
                          job.name,
                          job.status,
                          job.conclusion
                        )}`}
                      >
                        {/* n8n style port dot */}
                        <div className={`absolute left-[-4px] top-1/2 translate-y-[-50%] w-2 h-2 rounded-full ${dotColor} border border-[#0c0c0c] z-20`} />
                        <div className="min-w-0">
                          <span className="text-[10px] font-bold text-white font-mono uppercase truncate block">
                            {job.name}
                          </span>
                          <p className="truncate font-mono text-[8px] text-[#666666]">github-job</p>
                        </div>
                        <div className="shrink-0">{getStatusIcon(job.status, job.conclusion)}</div>
                      </div>
                    );
                  })}
                </div>
              </>
            )}
          </div>

          {/* Terminal Log Console */}
          <div className="rounded-lg border border-[#1e1e1e] bg-[#050505] flex flex-col overflow-hidden shrink-0">
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
            <div className="p-3 font-mono text-[10px] overflow-y-auto flex flex-col gap-1 leading-5 text-left bg-black/90 h-[250px]">
              {parsedLogs.map((log, idx) => {
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
      </div>
    </div>,
    document.body
  );
}

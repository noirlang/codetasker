/**
 * CodeViewer — Monaco Editor wrapper for read-only source file display.
 *
 * Features:
 * - Infers language from file extension
 * - Highlights lines that have associated TODO tasks (gray bg decoration)
 * - Fires onLineClick when the user clicks a line
 * - Shows a breadcrumb + language tag in the header bar
 * - Empty state when no file is selected
 */

import { useRef, useCallback, useState, useEffect } from 'react';
import Editor, { type OnMount } from '@monaco-editor/react';
import type { editor as MonacoEditor } from 'monaco-editor';
import { Edit3, GitCommit, X } from 'lucide-react';
import type { Task } from '../types';
import Spinner from './ui/Spinner';

// ── Props ────────────────────────────────────────────────────────────────────

interface CodeViewerProps {
  /** Decoded source content to display. Empty string = empty state. */
  content: string;
  /** Monaco language ID (will be auto-inferred if not set) */
  language?: string;
  /** Relative path of the file (used in the breadcrumb) */
  filePath: string;
  /** Tasks associated with this file (for line highlighting) */
  tasks: Task[];
  /** Called when the user clicks a line in the editor */
  onLineClick: (lineNumber: number) => void;
  /** Called when saving code modifications */
  onSave: (
    newContent: string,
    branch: string,
    message: string,
    coAuthors?: string[]
  ) => Promise<void>;
  /** Saving state indicator */
  isSaving: boolean;
  /** Current active branch */
  defaultBranch: string;
}

// ── Language inference ───────────────────────────────────────────────────────

/** Map file extensions to Monaco language IDs */
const EXT_TO_LANG: Record<string, string> = {
  ts:   'typescript',
  tsx:  'typescript',
  js:   'javascript',
  jsx:  'javascript',
  go:   'go',
  py:   'python',
  rb:   'ruby',
  rs:   'rust',
  java: 'java',
  kt:   'kotlin',
  cs:   'csharp',
  cpp:  'cpp',
  c:    'c',
  h:    'c',
  hpp:  'cpp',
  sh:   'shell',
  bash: 'shell',
  zsh:  'shell',
  yaml: 'yaml',
  yml:  'yaml',
  json: 'json',
  toml: 'toml',
  md:   'markdown',
  mdx:  'markdown',
  css:  'css',
  scss: 'scss',
  html: 'html',
  xml:  'xml',
  sql:  'sql',
  proto:'proto',
  tf:   'hcl',
  hcl:  'hcl',
  dockerfile: 'dockerfile',
};

function inferLanguage(filePath: string): string {
  const parts    = filePath.toLowerCase().split('/');
  const filename = parts[parts.length - 1];

  // Special whole-filename matches
  if (filename === 'dockerfile') return 'dockerfile';
  if (filename === 'makefile')   return 'makefile';

  const ext = filename.split('.').pop() ?? '';
  return EXT_TO_LANG[ext] ?? 'plaintext';
}

// ── Breadcrumb helper ────────────────────────────────────────────────────────

function FileBreadcrumb({ path }: { path: string }) {
  const parts = path.split('/');
  return (
    <div className="flex min-w-0 items-center gap-1 font-mono text-xs text-[#a0a0a0]"
         style={{ fontFamily: "'JetBrains Mono', monospace" }}>
      {parts.map((part, i) => (
        <span key={i} className="flex items-center gap-1">
          {i > 0 && <span className="text-[#3a3a3a]">/</span>}
          <span className={i === parts.length - 1 ? 'text-white' : ''}>
            {part}
          </span>
        </span>
      ))}
    </div>
  );
}

// ── Component ────────────────────────────────────────────────────────────────

export default function CodeViewer({
  content,
  filePath,
  tasks,
  onLineClick,
  onSave,
  isSaving,
  defaultBranch,
}: CodeViewerProps) {
  // Keep a ref to the Monaco editor instance for decorations + event listeners
  const editorRef = useRef<MonacoEditor.IStandaloneCodeEditor | null>(null);
  const decorationsRef = useRef<string[]>([]);

  const language = inferLanguage(filePath);

  // ── States ──────────────────────────────────────────────────────────────────
  const [isEditing, setIsEditing] = useState(false);
  const [editedValue, setEditedValue] = useState(content);
  const [showCommitForm, setShowCommitForm] = useState(false);
  const [commitMessage, setCommitMessage] = useState('');
  const [commitBranch, setCommitBranch] = useState(defaultBranch);
  const [coAuthorsInput, setCoAuthorsInput] = useState('');
  const [error, setError] = useState<string | null>(null);

  // Keep isEditing ref to prevent closure capture in Monaco mouse down event
  const isEditingRef = useRef(isEditing);
  useEffect(() => {
    isEditingRef.current = isEditing;
  }, [isEditing]);

  // Reset states on file switch or branch change
  useEffect(() => {
    setIsEditing(false);
    setShowCommitForm(false);
    setEditedValue(content);
    setCommitMessage('');
    setCommitBranch(defaultBranch);
    setCoAuthorsInput('');
    setError(null);
  }, [filePath, content, defaultBranch]);

  // ── Editor mount handler ──────────────────────────────────────────────────

  const handleEditorMount: OnMount = useCallback(
    (editor) => {
      editorRef.current = editor;

      // Apply task line decorations
      applyDecorations(editor);

      // Detect line clicks via mousedown
      editor.onMouseDown((e) => {
        // Only trigger line click / TODO injector if we are NOT in editing mode!
        if (isEditingRef.current) return;
        const lineNumber = e.target.position?.lineNumber;
        if (lineNumber != null) {
          onLineClick(lineNumber);
        }
      });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [tasks, onLineClick]
  );

  // ── Decoration application ────────────────────────────────────────────────

  const applyDecorations = (editor: MonacoEditor.IStandaloneCodeEditor) => {
    // Only highlight lines for the current file
    const fileTaskLines = tasks
      .filter((t) => t.file_path === filePath)
      .map((t) => t.line_number);

    const newDecorations: MonacoEditor.IModelDeltaDecoration[] = fileTaskLines.map(
      (line) => ({
        range: {
          startLineNumber: line,
          startColumn: 1,
          endLineNumber: line,
          endColumn: Number.MAX_SAFE_INTEGER,
        },
        options: {
          isWholeLine: true,
          className: 'task-line-highlight',
          // Inline style via Monaco's overviewRuler for the mini scrollbar marker
          overviewRuler: {
            color: '#3a3a3a',
            position: 1, // OverviewRulerLane.Left
          },
          minimap: {
            color: '#3a3a3a',
            position: 1,
          },
        },
      })
    );

    decorationsRef.current = editor.deltaDecorations(
      decorationsRef.current,
      newDecorations
    );
  };

  // Re-apply decorations when tasks or filePath change
  useEffect(() => {
    if (editorRef.current) {
      applyDecorations(editorRef.current);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tasks, filePath]);

  const handleConfirmCommit = async () => {
    setError(null);
    const msg = commitMessage.trim() || `Update ${filePath.split('/').pop()}`;
    const branch = commitBranch.trim() || defaultBranch;
    const coAuthors = coAuthorsInput
      .split(',')
      .map((s) => s.trim())
      .filter((s) => s.length > 0);

    try {
      await onSave(editedValue, branch, msg, coAuthors);
      setIsEditing(false);
      setShowCommitForm(false);
    } catch (err: any) {
      setError(err?.message ?? 'Failed to commit changes');
    }
  };

  // ── Empty state ───────────────────────────────────────────────────────────

  if (!filePath || !content) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 bg-[#0a0a0a] text-center">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="32"
          height="32"
          viewBox="0 0 24 24"
          fill="none"
          stroke="#3a3a3a"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z" />
          <polyline points="13 2 13 9 20 9" />
        </svg>
        <p className="text-sm text-[#666666]">Select a file from the tree</p>
      </div>
    );
  }

  // ── Editor render ─────────────────────────────────────────────────────────

  return (
    <div className="flex flex-1 flex-col overflow-hidden bg-[#0a0a0a]">
      {/* Top bar: breadcrumb + action buttons */}
      <div className="flex h-10 shrink-0 items-center justify-between border-b border-[#2a2a2a] bg-[#111111] px-4">
        <div className="flex items-center gap-2">
          <FileBreadcrumb path={filePath} />
          {isEditing && (
            <span className="rounded bg-white/5 border border-white/10 px-1.5 py-0.5 text-[9px] font-mono text-white/60 uppercase tracking-wider">
              Editing
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button
                onClick={() => setShowCommitForm(true)}
                className="btn-primary flex items-center gap-1.5 py-1 px-3 text-xs"
              >
                <GitCommit size={12} />
                Commit
              </button>
              <button
                onClick={() => {
                  setIsEditing(false);
                  setShowCommitForm(false);
                  setEditedValue(content);
                }}
                className="btn-secondary flex items-center gap-1.5 py-1 px-3 text-xs"
              >
                <X size={12} />
                Cancel
              </button>
            </>
          ) : (
            <>
              <button
                onClick={() => {
                  setIsEditing(true);
                  setEditedValue(content);
                }}
                className="btn-secondary flex items-center gap-1.5 py-1 px-3 text-xs"
              >
                <Edit3 size={12} />
                Edit File
              </button>
              <span className="tag ml-2 shrink-0">{language}</span>
            </>
          )}
        </div>
      </div>

      {/* Commit changes slide-down panel */}
      {showCommitForm && (
        <div className="flex flex-col gap-3 border-b border-[#2a2a2a] bg-[#161616] p-4 animate__animated animate__fadeInDown" style={{ animationDuration: '0.2s' }}>
          <div className="text-xs font-semibold text-white">Commit Changes to GitHub</div>
          
          {error && (
            <div className="text-xs text-red-400 bg-red-400/5 border border-red-400/10 px-3 py-2 rounded">
              {error}
            </div>
          )}

          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div className="md:col-span-2">
                <label className="block text-[10px] text-[#a0a0a0] uppercase mb-1">Commit Message</label>
                <input
                  type="text"
                  className="input w-full"
                  placeholder={`Update ${filePath.split('/').pop()}`}
                  value={commitMessage}
                  onChange={(e) => setCommitMessage(e.target.value)}
                />
              </div>
              <div>
                <label className="block text-[10px] text-[#a0a0a0] uppercase mb-1">Branch</label>
                <input
                  type="text"
                  className="input w-full"
                  placeholder={defaultBranch}
                  value={commitBranch}
                  onChange={(e) => setCommitBranch(e.target.value)}
                />
              </div>
            </div>

            <div className="flex flex-col md:flex-row md:items-end gap-3">
              <div className="flex-1">
                <label className="block text-[10px] text-[#a0a0a0] uppercase mb-1">
                  Co-authors (comma separated)
                </label>
                <input
                  type="text"
                  className="input w-full"
                  placeholder="e.g. John Doe <john@doe.com>, Jane Smith <jane@smith.com>"
                  value={coAuthorsInput}
                  onChange={(e) => setCoAuthorsInput(e.target.value)}
                />
              </div>
              <div className="flex gap-2 shrink-0 self-end">
                <button
                  onClick={handleConfirmCommit}
                  disabled={isSaving}
                  className="btn-primary py-2 px-4 text-xs h-9"
                >
                  {isSaving ? <Spinner size={12} /> : 'Commit'}
                </button>
                <button
                  onClick={() => setShowCommitForm(false)}
                  className="btn-secondary py-2 px-4 text-xs h-9"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Monaco editor */}
      <div className="relative flex-1 overflow-hidden">
        {/* Inject CSS for task line decoration */}
        <style>{`
          .task-line-highlight {
            background-color: rgba(255, 255, 255, 0.04) !important;
          }
        `}</style>

        <Editor
          key={`${filePath}-${isEditing}`} // remount when file or edit mode changes to refresh readOnly state properly
          value={isEditing ? editedValue : content}
          language={language}
          theme="vs-dark"
          onMount={handleEditorMount}
          onChange={(val) => {
            if (isEditing) setEditedValue(val ?? '');
          }}
          loading={
            <div className="flex h-full items-center justify-center">
              <Spinner size={28} />
            </div>
          }
          options={{
            readOnly: !isEditing,
            minimap: { enabled: false },
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            fontSize: 13,
            fontFamily: "'JetBrains Mono', Menlo, monospace",
            fontLigatures: true,
            renderLineHighlight: 'line',
            renderLineHighlightOnlyWhenFocus: false,
            cursorStyle: 'line',
            scrollbar: {
              verticalScrollbarSize: 6,
              horizontalScrollbarSize: 6,
            },
            overviewRulerLanes: 2,
            padding: { top: 12, bottom: 12 },
            wordWrap: 'off',
            contextmenu: isEditing, // Enable default context menu in edit mode
            quickSuggestions: isEditing,
            parameterHints: { enabled: isEditing },
            suggestOnTriggerCharacters: isEditing,
            acceptSuggestionOnEnter: isEditing ? 'on' : 'off',
            tabCompletion: isEditing ? 'on' : 'off',
            wordBasedSuggestions: isEditing ? 'allDocuments' : 'off',
          }}
        />
      </div>
    </div>
  );
}

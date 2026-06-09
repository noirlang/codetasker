/**
 * Badge — Pill label for TaskType and TaskStatus.
 *
 * Uses monospace font and subtle borders only — no vibrant colors.
 * The design system intentionally keeps all badges near-grayscale
 * so they don't compete with code content.
 */

import type { TaskType, TaskStatus } from '../../types';

type BadgeValue = TaskType | TaskStatus;

interface BadgeProps {
  type: BadgeValue;
  className?: string;
}

// ── Style maps ──────────────────────────────────────────────────────────────

/**
 * Maps each badge value to a Tailwind class string.
 * All backgrounds are transparent — depth comes from the border only.
 */
const BADGE_STYLES: Record<BadgeValue, string> = {
  // ── Task types ────────────────────────────────────────────────────────────
  TODO:       'text-white border-white',
  FIXME:      'text-[#a0a0a0] border-[#3a3a3a]',
  HACK:       'text-[#666666] border-[#2a2a2a]',
  BUG:        'text-white border-[#4a4a4a]',
  NOTE:       'text-[#666666] border-[#2a2a2a]',

  // ── Task statuses ─────────────────────────────────────────────────────────
  open:       'text-white border-[#3a3a3a]',
  in_progress:'text-[#a0a0a0] border-[#3a3a3a]',
  resolved:   'text-[#666666] border-[#2a2a2a] line-through',
};

/** Human-readable label (mainly for status values with underscores) */
const BADGE_LABELS: Record<BadgeValue, string> = {
  TODO:        'TODO',
  FIXME:       'FIXME',
  HACK:        'HACK',
  BUG:         'BUG',
  NOTE:        'NOTE',
  open:        'open',
  in_progress: 'in progress',
  resolved:    'resolved',
};

// ── Component ───────────────────────────────────────────────────────────────

export default function Badge({ type, className = '' }: BadgeProps) {
  const styles = BADGE_STYLES[type] ?? 'text-[#a0a0a0] border-[#3a3a3a]';
  const label  = BADGE_LABELS[type] ?? type;

  return (
    <span
      className={[
        'inline-flex items-center px-1.5 py-0.5',
        'text-[10px] font-mono font-medium tracking-wide',
        'rounded border bg-transparent',
        'select-none whitespace-nowrap',
        styles,
        className,
      ].join(' ')}
    >
      {label}
    </span>
  );
}

/**
 * Spinner — Animated loading indicator.
 *
 * Renders a rotating SVG circle with a white stroke on a dark track.
 * Keeps it purely CSS-based for zero-dependency animation.
 */

interface SpinnerProps {
  /** Diameter in pixels. Default: 20 */
  size?: number;
  /** Additional Tailwind / class names for positioning */
  className?: string;
}

export default function Spinner({ size = 20, className = '' }: SpinnerProps) {
  const r = (size - 4) / 2; // radius with 2px stroke clearance on each side
  const cx = size / 2;
  const cy = size / 2;
  const circumference = 2 * Math.PI * r;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className={`animate-spin ${className}`}
      aria-label="Loading…"
      role="status"
    >
      {/* Background track */}
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke="#2a2a2a"
        strokeWidth={2}
      />
      {/* Spinning arc — 75% of the circumference visible */}
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke="#ffffff"
        strokeWidth={2}
        strokeLinecap="round"
        strokeDasharray={`${circumference * 0.75} ${circumference * 0.25}`}
        strokeDashoffset={0}
      />
    </svg>
  );
}

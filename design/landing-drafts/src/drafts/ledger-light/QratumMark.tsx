// QratumMark.tsx
// Drop-in logo mark for the ledger-light site. Inline SVG, no deps.
// Geometry is the locked family grammar: 100-grid, 9-unit stroke,
// square caps, miter joins, a dot = "the unit of work".
//
// Usage:
//   <QratumMark size={26} />                       // ink strokes, yellow dot (default)
//   <QratumMark size={16} tone="mono" />           // inherits currentColor (favicon on dark, etc.)
//   <QratumMark size={20} dot="#0b0b0b" />         // custom dot

type Props = {
  size?: number;
  /** "ledger" = ink strokes + yellow dot (default). "mono" = all currentColor. */
  tone?: "ledger" | "mono";
  stroke?: string;
  dot?: string;
  className?: string;
  title?: string;
};

export function QratumMark({
  size = 24,
  tone = "ledger",
  stroke,
  dot,
  className,
  title = "Qratum",
}: Props) {
  const strokeColor = stroke ?? (tone === "mono" ? "currentColor" : "#0b0b0b");
  const dotColor = dot ?? (tone === "mono" ? "currentColor" : "#f5d90a");
  return (
    <svg
      viewBox="0 0 100 100"
      width={size}
      height={size}
      role="img"
      aria-label={title}
      className={className}
      style={{ display: "block", flexShrink: 0 }}
    >
      <g
        fill="none"
        stroke={strokeColor}
        strokeWidth={9}
        strokeLinecap="square"
        strokeLinejoin="miter"
      >
        <path d="M 36 18 L 30 78" />
        <path d="M 58 18 L 52 78" />
        <path d="M 20 40 L 70 40" />
        <path d="M 18 60 L 68 60" />
      </g>
      <circle cx="84" cy="72" r="7" fill={dotColor} />
    </svg>
  );
}

// ── Sibling marks, for the shared ecosystem footer ──────────────────
// Same grammar; render in mono (currentColor) so the footer controls color.
export function EdictumMark({ size = 20, color = "currentColor", title = "Edictum" }: { size?: number; color?: string; title?: string }) {
  return (
    <svg viewBox="0 0 100 100" width={size} height={size} role="img" aria-label={title} style={{ display: "block", flexShrink: 0, color }}>
      <g fill="none" stroke="currentColor" strokeWidth={9} strokeLinecap="square" strokeLinejoin="miter">
        <path d="M 30 18 L 14 18 L 14 82 L 30 82" />
        <path d="M 70 18 L 86 18 L 86 82 L 70 82" />
      </g>
      <circle cx="50" cy="50" r="9" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function DuctumMark({ size = 20, color = "currentColor", title = "Ductum" }: { size?: number; color?: string; title?: string }) {
  return (
    <svg viewBox="0 0 100 100" width={size} height={size} role="img" aria-label={title} style={{ display: "block", flexShrink: 0, color }}>
      <g fill="none" stroke="currentColor" strokeWidth={9} strokeLinecap="square" strokeLinejoin="miter">
        <path d="M 16 50 L 40 50" />
        <path d="M 40 50 L 64 26" />
        <path d="M 40 50 L 64 74" />
      </g>
      <circle cx="70" cy="22" r="7" fill="currentColor" stroke="none" />
      <circle cx="70" cy="78" r="7" fill="currentColor" stroke="none" />
      <circle cx="14" cy="50" r="7" fill="currentColor" stroke="none" />
    </svg>
  );
}

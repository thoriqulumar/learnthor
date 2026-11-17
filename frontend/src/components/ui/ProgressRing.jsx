/**
 * ProgressRing — circular SVG progress indicator with animated fill.
 * Extracted from HomePage's MasteryRing so it's reusable on dashboards,
 * module cards, and result screens.
 *
 * Props:
 *   pct   — 0–100 fill value
 *   size  — pixel diameter (default 40)
 *   label — text to display in the center (defaults to "{pct}%")
 */
export default function ProgressRing({ pct, size = 40, label }) {
  const r = 14;
  const circ = 2 * Math.PI * r;
  const dash = (Math.max(0, Math.min(100, pct)) / 100) * circ;
  const complete = pct >= 100;
  const inProgress = pct > 0 && !complete;

  return (
    <svg width={size} height={size} viewBox="0 0 32 32" className="shrink-0">
      <circle cx="16" cy="16" r={r} fill="none" stroke="#e5e7eb" strokeWidth="3" className="dark:stroke-slate-600" />
      <circle
        cx="16"
        cy="16"
        r={r}
        fill="none"
        stroke={complete ? '#10b981' : inProgress ? '#6366f1' : '#d1d5db'}
        strokeWidth="3"
        strokeDasharray={`${dash} ${circ}`}
        strokeLinecap="round"
        transform="rotate(-90 16 16)"
        style={{ transition: 'stroke-dasharray 0.5s ease, stroke 0.3s ease' }}
      />
      <text
        x="16"
        y="20"
        textAnchor="middle"
        fontSize="7"
        fontWeight="600"
        fill={complete ? '#059669' : inProgress ? '#4f46e5' : '#9ca3af'}
      >
        {label ?? `${pct}%`}
      </text>
    </svg>
  );
}
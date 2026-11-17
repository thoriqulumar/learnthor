const COLORS = {
  indigo: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300',
  sakura: 'bg-sakura-100 text-sakura-700 dark:bg-sakura-900/40 dark:text-sakura-300',
  matcha: 'bg-matcha-100 text-matcha-700 dark:bg-matcha-900/40 dark:text-matcha-300',
  sun: 'bg-sun-100 text-sun-700 dark:bg-sun-900/40 dark:text-sun-300',
  emerald: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300',
  amber: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
  gray: 'bg-gray-100 text-gray-600 dark:bg-slate-700 dark:text-slate-300',
  premium: 'bg-gradient-to-r from-amber-400 to-sakura-400 text-white',
};

/**
 * Badge — a compact pill for tags, status indicators, and counts.
 * Colors: indigo | sakura | matcha | sun | emerald | amber | gray | premium.
 */
export default function Badge({ color = 'gray', children, className = '' }) {
  return (
    <span
      className={`inline-flex items-center gap-1 text-xs font-semibold px-2.5 py-0.5 rounded-full ${COLORS[color] ?? COLORS.gray} ${className}`}
    >
      {children}
    </span>
  );
}
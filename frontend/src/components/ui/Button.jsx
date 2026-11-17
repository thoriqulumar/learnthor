import { motion } from 'framer-motion';

// ── Button variants ─────────────────────────────────────────────────────────
const VARIANTS = {
  primary:
    'bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 shadow-soft',
  secondary:
    'bg-white text-gray-700 border border-gray-200 hover:bg-gray-50 dark:bg-slate-800 dark:text-slate-200 dark:border-slate-700 dark:hover:bg-slate-700',
  ghost:
    'text-gray-600 hover:bg-gray-100 dark:text-slate-300 dark:hover:bg-slate-800',
  danger:
    'bg-red-500 text-white hover:bg-red-600 dark:bg-red-600 dark:hover:bg-red-700 shadow-soft',
  premium:
    'bg-gradient-to-r from-amber-400 to-sakura-400 text-white hover:from-amber-500 hover:to-sakura-500 shadow-soft',
};

const SIZES = {
  sm: 'px-3 py-1.5 text-sm rounded-lg gap-1.5',
  md: 'px-4 py-2.5 text-sm rounded-xl gap-2',
  lg: 'px-6 py-3.5 text-base rounded-2xl gap-2',
};

/**
 * Button — the shared button primitive. Polymorphic via `as` so it can
 * render as a motion.button for animations or a plain button.
 * Variants: primary | secondary | ghost | danger | premium.
 * Sizes: sm | md | lg.
 */
export default function Button({
  variant = 'primary',
  size = 'md',
  children,
  className = '',
  disabled,
  ...props
}) {
  return (
    <motion.button
      whileTap={{ scale: disabled ? 1 : 0.96 }}
      whileHover={{ scale: disabled ? 1 : 1.02 }}
      transition={{ type: 'spring', stiffness: 400, damping: 17 }}
      disabled={disabled}
      className={`inline-flex items-center justify-center font-semibold transition-colors select-none
        ${VARIANTS[variant] ?? VARIANTS.primary}
        ${SIZES[size] ?? SIZES.md}
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        ${className}`}
      {...props}
    >
      {children}
    </motion.button>
  );
}
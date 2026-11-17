import { motion } from 'framer-motion';

/**
 * Card — the shared surface primitive. Soft shadow, generous radius,
 * optional hover lift. Supports dark mode automatically.
 *
 * Props:
 *   hover    — add a subtle lift + shadow on hover (default true)
 *   padded   — adds internal padding (default false; set true or pass a
 *              custom padding class string)
 *   motion — framer-motion props to merge (e.g. layout, initial, animate)
 */
export default function Card({
  children,
  className = '',
  hover = false,
  padded = false,
  motion: motionProps,
  ...props
}) {
  const padding = padded === true ? 'p-5' : padded ? padded : '';
  return (
    <motion.div
      whileHover={hover ? { y: -2, boxShadow: '0 8px 32px -4px rgb(0 0 0 / 0.08), 0 4px 12px -4px rgb(0 0 0 / 0.1)' } : undefined}
      transition={{ type: 'spring', stiffness: 300, damping: 20 }}
      className={`bg-white dark:bg-slate-800 dark:border-slate-700 border border-gray-200 shadow-soft rounded-3xl ${padding} ${className}`}
      {...(motionProps || {})}
      {...props}
    >
      {children}
    </motion.div>
  );
}
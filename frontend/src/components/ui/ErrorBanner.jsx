import { friendlyErrorParts } from '../../lib/errors';

/**
 * ErrorBanner — the shared error display primitive. Renders a friendly,
 * bilingual-aware error message with optional retry / go-back affordance.
 *
 * Props:
 *   err     — the axios error (or any { response: { status, data: { code } } })
 *   lang    — preferred_gloss value ('en' | 'id' | 'both', default 'en')
 *   onRetry — called when the user taps "Try again" (hidden if omitted)
 *   onBack  — called when the user taps "Go back" (hidden if omitted)
 *
 * If err is null/undefined the banner renders nothing (no-op), so it's safe
 * to always render <ErrorBanner err={maybeErr} /> in a component.
 */
export default function ErrorBanner({ err, lang = 'en', onRetry, onBack }) {
  if (!err) return null;

  const parts = friendlyErrorParts(err);
  const message = lang === 'id' ? parts.id : parts.en;
  const sub = lang === 'id' ? parts.en : parts.id;

  return (
    <div
      className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-2xl p-4 text-center"
      role="alert"
    >
      <div className="text-2xl mb-2">😅</div>
      <p className="text-sm font-semibold text-red-700 dark:text-red-300 mb-1">
        {message}
      </p>
      {sub && sub !== message && lang !== 'en' ? (
        <p className="text-xs text-red-500 dark:text-red-400 mb-2">{sub}</p>
      ) : null}
      {(onRetry || onBack) && (
        <div className="flex items-center justify-center gap-2 mt-3">
          {onRetry && (
            <button
              onClick={onRetry}
              className="text-sm px-3 py-1.5 rounded-lg bg-red-600 text-white font-medium hover:bg-red-700 transition-colors"
            >
              Try again
            </button>
          )}
          {onBack && (
            <button
              onClick={onBack}
              className="text-sm px-3 py-1.5 rounded-lg border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 font-medium hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
            >
              Go back
            </button>
          )}
        </div>
      )}
    </div>
  );
}
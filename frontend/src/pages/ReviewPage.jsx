import Meaning from '../components/Meaning';
import { useLocale } from '../lib/locale';
import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../lib/api';
import { friendlyError } from '../lib/errors';
import { useAuth } from '../context/AuthContext';

// ── FSRS rating buttons ────────────────────────────────────────────────────
const RATINGS = [
  { value: 1, label: 'Again', hint: '< 1m', cls: 'bg-red-500 hover:bg-red-600' },
  { value: 2, label: 'Hard', hint: '~10m', cls: 'bg-amber-500 hover:bg-amber-600' },
  { value: 3, label: 'Good', hint: 'tomorrow', cls: 'bg-indigo-500 hover:bg-indigo-600' },
  { value: 4, label: 'Easy', hint: '4d+', cls: 'bg-emerald-500 hover:bg-emerald-600' },
];

function Spinner() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center">
      <div className="w-8 h-8 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

export default function ReviewPage() {
  const { tr, language } = useLocale();
  const navigate = useNavigate();
  const { user } = useAuth();
  const gloss = user?.preferred_gloss ?? 'both';

  const [cards, setCards] = useState([]);
  const [index, setIndex] = useState(0);
  const [revealed, setRevealed] = useState(false);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [stats, setStats] = useState({ reviewed: 0, correct: 0 });

  const loadQueue = useCallback(() => {
    setLoading(true);
    setError('');
    api
      .get('/reviews/due')
      .then(({ data }) => {
        setCards(data.cards ?? []);
        setIndex(0);
        setRevealed(false);
        setDone((data.cards ?? []).length === 0);
        setStats({ reviewed: 0, correct: 0 });
      })
      .catch(setError)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    loadQueue(); // eslint-disable-line react-hooks/set-state-in-effect -- fetch on mount
  }, [loadQueue]);

  const current = cards[index];

  function grade(rating) {
    if (!current || submitting) return;
    setSubmitting(true);
    api
      .post('/reviews/submit', {
        vocab_item_id: current.vocab_item_id,
        rating,
      })
      .then(() => {
        const isCorrect = rating >= 3;
        setStats((s) => ({
          reviewed: s.reviewed + 1,
          correct: s.correct + (isCorrect ? 1 : 0),
        }));
        if (index + 1 < cards.length) {
          setIndex(index + 1);
          setRevealed(false);
        } else {
          setDone(true);
        }
      })
      .catch(setError)
      .finally(() => setSubmitting(false));
  }

  if (loading) return <Spinner />;

  if (error) {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <p className="text-red-500 mb-4">{typeof error === 'string' ? tr(error) : friendlyError(error, language)}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-indigo-600 hover:underline"
        >{tr("← Go back")}</button>
      </div>
    );
  }

  if (done) {
    const pct =
      stats.reviewed > 0
        ? Math.round((stats.correct / stats.reviewed) * 100)
        : 0;
    return (
      <div className="max-w-lg mx-auto px-4 py-12 text-center">
        <div className="text-6xl mb-4">✅</div>
        <h1 className="text-2xl font-bold text-gray-900 mb-2">{tr("Review complete!")}</h1>
        <p className="text-gray-500 mb-6">{tr('You reviewed {count} cards.', { count: stats.reviewed })}<br />{tr("Accuracy:")}<strong>{pct}%</strong>
        </p>
        <div className="flex flex-col gap-3">
          <button
            onClick={loadQueue}
            className="w-full py-3.5 rounded-2xl bg-indigo-600 text-white font-semibold text-base hover:bg-indigo-700 transition-colors"
          >{tr("Check for more due cards →")}</button>
          <button
            onClick={() => navigate('/')}
            className="w-full py-3 rounded-2xl border border-gray-200 text-gray-600 font-medium hover:bg-gray-50 transition-colors"
          >{tr("← Back to home")}</button>
        </div>
      </div>
    );
  }

  if (!current) return <Spinner />;

  const progressPct = Math.round((index / cards.length) * 100);

  return (
    <div className="max-w-lg mx-auto px-4 py-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => navigate('/')}
          className="text-sm text-gray-400 hover:text-gray-600"
        >{tr("← Back")}</button>
        <div className="text-center">
          <p className="text-xs text-gray-400 uppercase tracking-wider">{tr("Daily Review")}</p>
        </div>
        <span className="text-sm text-gray-400">
          {index + 1}/{cards.length}
        </span>
      </div>

      {/* Progress bar */}
      <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden mb-6">
        <div
          className="h-full bg-indigo-500 rounded-full transition-all duration-300"
          style={{ width: `${progressPct}%` }}
        />
      </div>

      {/* Card */}
      <div
        className={`bg-white border-2 rounded-3xl p-8 text-center shadow-sm transition-all duration-300 cursor-pointer select-none
          ${revealed ? 'border-indigo-300 shadow-indigo-100' : 'border-gray-200 hover:border-indigo-200 hover:shadow-md'}`}
        onClick={() => !revealed && setRevealed(true)}
      >
        <p className="font-bold text-gray-900 leading-tight mb-1 text-5xl">
          {current.kanji ?? current.kana}
        </p>

        {current.kanji && current.kana && current.kanji !== current.kana && (
          <p className="text-xl text-gray-500 mb-1">{current.kana}</p>
        )}

        <p className="text-base text-indigo-500 font-medium mb-4">
          {current.romaji}
        </p>

        {current.part_of_speech && current.part_of_speech !== 'character' && (
          <span className="text-xs px-2.5 py-0.5 rounded-full font-medium bg-gray-100 text-gray-600">
            {current.part_of_speech}
          </span>
        )}

        {!revealed && (
          <p className="mt-6 text-sm text-gray-400 animate-pulse">{tr("Tap to reveal meaning")}</p>
        )}
      </div>

      {/* Reveal + rating buttons */}
      {revealed ? (
        <div className="mt-4 space-y-3">
          {/* Meaning(s) */}
          <div className="bg-indigo-50 border border-indigo-100 rounded-2xl p-5 text-center">
            <Meaning item={current} gloss={gloss} />
            {current.example_sentence && (
              <div className="mt-3 text-left bg-white rounded-xl px-4 py-3 border border-indigo-100">
                <p className="text-xs text-gray-400 uppercase tracking-widest mb-1">{tr("Example")}</p>
                <p className="text-gray-700 text-sm leading-relaxed">
                  {current.example_sentence}
                </p>
              </div>
            )}
          </div>

          {/* FSRS rating buttons */}
          <p className="text-center text-xs text-gray-400 mb-1">{tr("How well did you recall it?")}</p>
          <div className="grid grid-cols-4 gap-2">
            {RATINGS.map((r) => (
              <button
                key={r.value}
                onClick={() => grade(r.value)}
                disabled={submitting}
                className={`flex flex-col items-center py-3 rounded-xl text-white font-semibold transition-colors disabled:opacity-50 ${r.cls}`}
              >
                <span className="text-sm">{tr(r.label)}</span>
                
              </button>
            ))}
          </div>
        </div>
      ) : (
        <p className="text-center text-xs text-gray-400 mt-2">{tr("Tap the card to reveal, then rate your recall")}</p>
      )}
    </div>
  );
}

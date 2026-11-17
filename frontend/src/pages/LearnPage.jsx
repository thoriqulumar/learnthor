import Meaning from '../components/Meaning';
import { useLocale } from '../lib/locale';
import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate, useLocation, Link } from 'react-router-dom';
import api from '../lib/api';
import { friendlyError } from '../lib/errors';
import { useAuth } from '../context/AuthContext';

// ── Part-of-speech colours ─────────────────────────────────────────────────
const POS_COLORS = {
  noun: 'bg-sky-100 text-sky-700',
  verb: 'bg-emerald-100 text-emerald-700',
  'i-adj': 'bg-violet-100 text-violet-700',
  'na-adj': 'bg-pink-100 text-pink-700',
  adv: 'bg-amber-100 text-amber-700',
  character: 'bg-indigo-100 text-indigo-700',
};



function Spinner() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center">
      <div className="w-8 h-8 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

// ── Stage components ───────────────────────────────────────────────────────

/**
 * Intro — "You're about to learn N words" splash before cards begin.
 */
function IntroStage({ count, title, onStart }) {
  const { tr } = useLocale();
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center px-6">
      <div className="text-6xl mb-5">📖</div>
      <h1 className="text-2xl font-bold text-gray-900 mb-2">{title}</h1>
      <p className="text-gray-500 mb-8">{tr('You are about to learn {count} words.', { count })}<br />{tr("Take your time — tap each card to reveal it.")}</p>
      <button
        onClick={onStart}
        className="px-8 py-3 bg-indigo-600 text-white rounded-2xl font-semibold text-lg hover:bg-indigo-700 transition-colors"
      >{tr("Start learning →")}</button>
    </div>
  );
}

/**
 * WordCard — the main learning card for a single vocabulary item.
 * Tapping reveals meaning; a second tap advances.
 */
function WordCard({ item, index, total, onNext, isCharacter, gloss }) {
  const { tr } = useLocale();
  const [revealed, setRevealed] = useState(false);

  const isLast = index === total - 1;

  return (
    <div className="flex flex-col h-full">
      {/* Progress dots */}
      <div className="flex gap-1 mb-6 flex-wrap justify-center">
        {Array.from({ length: total }).map((_, i) => (
          <div
            key={i}
            className={`h-1.5 rounded-full transition-all duration-300 ${
              i < index
                ? 'bg-emerald-400'
                : i === index
                  ? 'bg-indigo-500 w-6'
                  : 'bg-gray-200'
            }`}
            style={{ width: i === index ? undefined : '6px' }}
          />
        ))}
      </div>

      {/* Counter */}
      <p className="text-xs text-gray-400 text-center mb-4 uppercase tracking-widest">{tr('Word {current} of {total}', { current: index + 1, total })}
      </p>

      {/* Card front — always visible */}
      <div
        className={`bg-white border-2 rounded-3xl p-8 text-center shadow-sm transition-all duration-300 cursor-pointer select-none
          ${revealed ? 'border-indigo-300 shadow-indigo-100' : 'border-gray-200 hover:border-indigo-200 hover:shadow-md'}`}
        onClick={() => !revealed && setRevealed(true)}
      >
        {/* Main character */}
        <p
          className={`font-bold text-gray-900 leading-tight mb-1 ${
            isCharacter ? 'text-7xl' : 'text-5xl'
          }`}
        >
          {item.kanji ?? item.kana}
        </p>

        {/* Kana reading (only for non-character or when kanji differs from kana) */}
        {item.kanji && item.kana && item.kanji !== item.kana && (
          <p className="text-xl text-gray-500 mb-1">{item.kana}</p>
        )}

        {/* Romaji */}
        <p className="text-base text-indigo-500 font-medium mb-4">
          {item.romaji}
        </p>

        {/* POS badge */}
        {item.part_of_speech && item.part_of_speech !== 'character' && (
          <span
            className={`text-xs px-2.5 py-0.5 rounded-full font-medium ${
              POS_COLORS[item.part_of_speech] ?? 'bg-gray-100 text-gray-600'
            }`}
          >
            {item.part_of_speech}
          </span>
        )}

        {/* Tap hint when unrevealed */}
        {!revealed && (
          <p className="mt-6 text-sm text-gray-400 animate-pulse">{tr("Tap to reveal meaning")}</p>
        )}
      </div>

      {/* Revealed section — slides down */}
      <div
        className={`overflow-hidden transition-all duration-500 ${
          revealed ? 'max-h-96 mt-4 opacity-100' : 'max-h-0 opacity-0'
        }`}
      >
        <div className="bg-indigo-50 border border-indigo-100 rounded-2xl p-5 text-center">
          <Meaning item={item} gloss={gloss} />
          {/* Example sentence with translated meanings */}
          {item.example_sentence && (
            <div className="mt-3 text-left bg-white rounded-xl px-4 py-3 border border-indigo-100">
              <p className="text-xs text-gray-400 uppercase tracking-widest mb-1">{tr("Example")}</p>
              <p className="text-gray-700 text-sm leading-relaxed">
                {item.example_sentence}
              </p>
              <div className="mt-2 space-y-0.5 text-xs">
                {item.english_meaning && (
                  <p className="text-indigo-600">
                    <span className="text-gray-400 mr-1">{tr("EN:")}</span>
                    {item.english_meaning}
                  </p>
                )}
                {item.indonesian_meaning && (
                  <p className="text-amber-600">
                    <span className="text-gray-400 mr-1">{tr("ID:")}</span>
                    {item.indonesian_meaning}
                  </p>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Next / Finish */}
        <button
          onClick={onNext}
          className="w-full mt-4 py-3.5 rounded-2xl bg-indigo-600 text-white font-semibold text-base hover:bg-indigo-700 transition-colors"
        >
          {isLast ? tr("Finish lesson ✓") : tr("Next word →")}
        </button>
      </div>
    </div>
  );
}

/**
 * Summary — shown after all words have been reviewed.
 */
function SummaryStage({ items, moduleId, title, onReview }) {
  const { user } = useAuth();
  const { tr } = useLocale();
  const navigate = useNavigate();
  const isCharacter =
    items.length > 0 && items[0].part_of_speech === 'character';

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      <div className="text-center mb-8">
        <div className="text-6xl mb-3">🎉</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-1">{tr("Lesson complete!")}</h2>
        <p className="text-gray-500">{tr('You reviewed {count} words in {title}.', { count: items.length, title })}
        </p>
      </div>

      {/* Word summary grid */}
      <div className="bg-white border border-gray-200 rounded-2xl overflow-hidden mb-6 shadow-sm">
        <div className="px-4 py-3 border-b border-gray-100">
          <p className="text-xs text-gray-400 font-semibold uppercase tracking-widest">{tr("Words covered")}</p>
        </div>
        <div className="divide-y divide-gray-50">
          {items.map((item) => (
            <div key={item.id} className="flex items-center gap-3 px-4 py-3">
              <span className="text-2xl font-bold text-gray-900 w-16 shrink-0 text-center">
                {item.kanji ?? item.kana}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm text-gray-500">
                  {item.kana !== item.kanji ? item.kana : ''}{' '}
                  {item.romaji && (
                    <span className="text-indigo-500">{item.romaji}</span>
                  )}
                </p>
                <Meaning item={item} gloss={user?.preferred_gloss} />
              </div>
              {item.part_of_speech && item.part_of_speech !== 'character' && (
                <span
                  className={`text-xs px-2 py-0.5 rounded-full font-medium shrink-0 ${
                    POS_COLORS[item.part_of_speech] ??
                    'bg-gray-100 text-gray-600'
                  }`}
                >
                  {item.part_of_speech}
                </span>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex flex-col gap-3">
        <Link
          to={`/modules/${moduleId}/quiz?type=mixed&limit=10`}
          state={{
            title,
            skillTypes: isCharacter
              ? ['kanji_to_kana', 'kana_to_kanji']
              : ['kanji_to_kana', 'kana_to_kanji', 'fill_blank'],
          }}
          className="w-full py-3.5 rounded-2xl bg-indigo-600 text-white font-semibold text-center text-base hover:bg-indigo-700 transition-colors"
        >{tr("Practice with quiz →")}</Link>
        <Link
          to="/review"
          className="w-full py-3 rounded-2xl border border-violet-200 bg-violet-50 text-violet-700 font-medium text-center hover:bg-violet-100 transition-colors"
        >{tr("🔄 Daily review (SRS)")}</Link>
        <button
          onClick={onReview}
          className="w-full py-3 rounded-2xl border border-gray-200 text-gray-600 font-medium hover:bg-gray-50 transition-colors"
        >{tr("Review words again")}</button>
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-gray-400 hover:text-gray-600 text-center mt-1"
        >{tr("← Back to module")}</button>
      </div>
    </div>
  );
}

// ── Main page ──────────────────────────────────────────────────────────────

export default function LearnPage() {
  const { tr, language } = useLocale();
  const { moduleId } = useParams();
  const navigate = useNavigate();
  const { state: navState } = useLocation();
  const { user } = useAuth();
  const gloss = user?.preferred_gloss ?? 'both';
  const moduleTitle = navState?.title ?? 'Lesson';

  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // stage: 'intro' | 'cards' | 'summary'
  const [stage, setStage] = useState('intro');
  const [cardIndex, setCardIndex] = useState(0);

  useEffect(() => {
    api
      .get(`/modules/${moduleId}/vocab`)
      .then(({ data }) => {
        const loaded = data.items ?? [];
        setItems(loaded);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => setLoading(false));
  }, [moduleId]);

  const handleStart = useCallback(() => {
    setCardIndex(0);
    setStage('cards');
  }, []);

  const handleNext = useCallback(() => {
    if (cardIndex + 1 >= items.length) {
      setStage('summary');
    } else {
      setCardIndex((i) => i + 1);
    }
  }, [cardIndex, items.length]);

  const handleReview = useCallback(() => {
    setCardIndex(0);
    setStage('cards');
  }, []);

  if (loading) return <Spinner />;

  if (error) {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <div className="text-5xl mb-4">🔒</div>
        <p className="text-gray-700 font-semibold mb-2">{tr("Can't load lesson")}</p>
        <p className="text-gray-500 text-sm mb-6">{typeof error === 'string' ? tr(error) : friendlyError(error, language)}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-indigo-600 hover:underline text-sm"
        >{tr("← Go back")}</button>
      </div>
    );
  }

  const isCharacter =
    items.length > 0 && items[0].part_of_speech === 'character';

  return (
    <div className="max-w-lg mx-auto px-4 py-6">
      {/* Top bar (hidden on intro so it doesn't clutter the splash) */}
      {stage !== 'intro' && stage !== 'summary' && (
        <div className="flex items-center justify-between mb-2">
          <button
            onClick={() => navigate(-1)}
            className="text-sm text-gray-400 hover:text-gray-600"
          >{tr("← Back")}</button>
          <p className="text-sm font-medium text-gray-700 truncate max-w-[200px]">
            {moduleTitle}
          </p>
          <span className="text-sm text-gray-400">
            {cardIndex + 1}/{items.length}
          </span>
        </div>
      )}

      {stage === 'intro' && (
        <IntroStage
          count={items.length}
          title={moduleTitle}
          onStart={handleStart}
        />
      )}

      {stage === 'cards' && items[cardIndex] && (
        <WordCard
          key={items[cardIndex].id}
          item={items[cardIndex]}
          index={cardIndex}
          total={items.length}
          onNext={handleNext}
          isCharacter={isCharacter}
          gloss={gloss}
        />
      )}

      {stage === 'summary' && (
        <SummaryStage
          items={items}
          moduleId={moduleId}
          title={moduleTitle}
          onReview={handleReview}
        />
      )}
    </div>
  );
}

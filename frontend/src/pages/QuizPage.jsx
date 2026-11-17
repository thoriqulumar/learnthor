import { friendlyError } from '../lib/errors';
import { useLocale } from '../lib/locale';
import { useEffect, useRef, useState } from 'react';
import {
  useParams,
  useNavigate,
  useSearchParams,
  useLocation,
} from 'react-router-dom';
import api from '../lib/api';
import { useAuth } from '../context/AuthContext';

// ─── SRS Constants ────────────────────────────────────────────────────────────
const MASTERY_THRESHOLD = 0.85; // item is "mastered" at this level
const CORRECT_GAIN = 0.15; // mastery gained per correct answer
const WRONG_PENALTY = 0.25; // mastery lost per wrong answer
const MAX_DELAY = 6; // max positions to defer after a correct answer
const ADVANCE_MS = 1400; // ms before auto-advancing

// ─── Skill labels ─────────────────────────────────────────────────────────────
const SKILL_LABELS = {
  kanji_to_kana: 'Romaji → Char',
  kana_to_kanji: 'Char → Romaji',
  fill_blank: 'Fill the Blank',
};

const TYPE_LABELS = {
  kanji_to_kana: 'Romaji → Character',
  kana_to_kanji: 'Character → Romaji',
  fill_blank: 'Fill the Blank',
  mixed: 'Mixed',
};

const PROMPT_HINT = {
  kanji_to_kana: 'What is the reading?',
  kana_to_kanji: 'Which matches this reading?',
  fill_blank: 'Choose the missing word',
};

// Skill types applicable per module flavour
const CHARACTER_SKILLS = ['kanji_to_kana', 'kana_to_kanji'];
const VOCAB_SKILLS = ['kanji_to_kana', 'kana_to_kanji', 'fill_blank'];

// Skill type → badge colour class
const SKILL_COLOR = {
  kanji_to_kana: 'bg-blue-100 text-blue-700',
  kana_to_kanji: 'bg-purple-100 text-purple-700',
  fill_blank: 'bg-amber-100 text-amber-700',
};

function Spinner() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center">
      <div className="w-8 h-8 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

/** Six dots visualising mastery level (0 → 1). */
function MasteryDots({ mastery }) {
  const filled = Math.min(6, Math.round((mastery / 1.0) * 6));
  return (
    <div className="flex gap-1.5 items-center">
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className={`w-2.5 h-2.5 rounded-full transition-colors duration-300 ${
            i < filled ? 'bg-indigo-500' : 'bg-gray-200'
          }`}
        />
      ))}
    </div>
  );
}

/**
 * Pick the next item to display.
 * - Must be unmastered (mastery < threshold) AND available (nextAppear <= counter)
 * - Falls back to lowest-mastery unmastered item if all are waiting
 * - Randomises among bottom 40% mastery candidates for variety
 */
function selectNext(states, counter) {
  const eligible = states.filter(
    (s) => s.mastery < MASTERY_THRESHOLD && s.nextAppear <= counter,
  );
  const pool =
    eligible.length > 0
      ? eligible
      : states.filter((s) => s.mastery < MASTERY_THRESHOLD);
  if (pool.length === 0) return null;
  pool.sort((a, b) => a.mastery - b.mastery);
  const candidateCount = Math.max(1, Math.ceil(pool.length * 0.4));
  return pool[Math.floor(Math.random() * candidateCount)];
}

export default function QuizPage() {
  const { tr, language } = useLocale();
  const { moduleId } = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { state: navState } = useLocation();
  const { user } = useAuth();
  const moduleTitle = navState?.title ?? 'Quiz';

  const quizType = searchParams.get('type') || 'mixed';

  // localStorage key unique per user + module + quiz type
  const saveKey = `quiz_state_${user?.id ?? 'anon'}_${moduleId}_${quizType}`;

  // Per-item SRS state: { ...question, stateId, mastery, nextAppear, timesCorrect, timesWrong }
  const [itemStates, setItemStates] = useState([]);
  const [currentItem, setCurrentItem] = useState(null);
  const [sessionCounter, setSessionCounter] = useState(0);
  const [hasSavedState, setHasSavedState] = useState(false);

  const [selected, setSelected] = useState(null);
  const [confirmed, setConfirmed] = useState(false);
  const [result, setResult] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const autoTimer = useRef(null);

  // ── Persist state to localStorage whenever it meaningfully changes ──────────
  useEffect(() => {
    if (!itemStates.length || result) return;
    try {
      localStorage.setItem(
        saveKey,
        JSON.stringify({
          itemStates,
          currentItemStateId: currentItem?.stateId,
          sessionCounter,
          savedAt: Date.now(),
        }),
      );
    } catch {
      // quota exceeded — ignore silently
    }
  }, [itemStates, currentItem, sessionCounter]); // eslint-disable-line react-hooks/exhaustive-deps

  function clearSavedState() {
    localStorage.removeItem(saveKey);
  }

  // ── Load ────────────────────────────────────────────────────────────────────

  function loadQuiz(forceNew = false) {
    // Check for saved state unless user explicitly restarted
    if (!forceNew) {
      try {
        const raw = localStorage.getItem(saveKey);
        if (raw) {
          const saved = JSON.parse(raw);
          // Discard saves older than 7 days
          if (
            Date.now() - saved.savedAt < 7 * 24 * 60 * 60 * 1000 &&
            saved.itemStates?.length
          ) {
            const restoredCurrent =
              saved.itemStates.find(
                (s) => s.stateId === saved.currentItemStateId,
              ) ?? saved.itemStates[0];
            setItemStates(saved.itemStates);
            setCurrentItem(restoredCurrent);
            setSessionCounter(saved.sessionCounter ?? 0);
            setHasSavedState(true);
            setLoading(false);
            setResult(null);
            setSelected(null);
            setConfirmed(false);
            return; // resume from saved state
          }
        }
      } catch {
        // corrupt save — ignore
      }
    }

    setHasSavedState(false);
    setLoading(true);
    setError('');
    setResult(null);
    setSelected(null);
    setConfirmed(false);
    setSessionCounter(0);
    clearSavedState();

    // Fetch all items (limit 50) so SRS can cycle through every vocab item
    api
      .get(`/modules/${moduleId}/quiz`, {
        params: { type: quizType, limit: 50 },
      })
      .then(({ data }) => {
        if (!data.questions?.length) {
          setError('No quiz questions available for this module.');
          return;
        }
        const states = data.questions.map((q, idx) => ({
          ...q,
          stateId: idx,
          mastery: 0,
          nextAppear: 0,
          timesCorrect: 0,
          timesWrong: 0,
        }));
        setItemStates(states);
        const first = states[Math.floor(Math.random() * states.length)];
        setCurrentItem(first);
      })
      .catch((err) => {
        if (err.response?.status === 403) {
          setError(
            'This module is locked. Complete the previous module first.',
          );
        } else {
          setError('Failed to load quiz.');
        }
      })
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    loadQuiz();
    return () => clearTimeout(autoTimer.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [moduleId, quizType]);

  // ── Answer ──────────────────────────────────────────────────────────────────

  function handleSelect(optionId) {
    if (confirmed || !currentItem) return;

    const isCorrect = optionId === currentItem.correct_id;
    setSelected(optionId);
    setConfirmed(true);

    const newCounter = sessionCounter + 1;

    const updatedStates = itemStates.map((s) => {
      if (s.stateId !== currentItem.stateId) return s;
      const newMastery = isCorrect
        ? Math.min(1.0, s.mastery + CORRECT_GAIN)
        : Math.max(0, s.mastery - WRONG_PENALTY);
      // Correct → delay scales with mastery; wrong → back next cycle
      const delay = isCorrect ? 1 + Math.floor(newMastery * MAX_DELAY) : 1;
      return {
        ...s,
        mastery: newMastery,
        nextAppear: newCounter + delay,
        timesCorrect: s.timesCorrect + (isCorrect ? 1 : 0),
        timesWrong: s.timesWrong + (isCorrect ? 0 : 1),
      };
    });

    setSessionCounter(newCounter);
    setItemStates(updatedStates);

    const allMastered = updatedStates.every(
      (s) => s.mastery >= MASTERY_THRESHOLD,
    );

    autoTimer.current = setTimeout(() => {
      if (allMastered) {
        submitResults(updatedStates);
      } else {
        const next = selectNext(updatedStates, newCounter);
        setCurrentItem(next ?? updatedStates[0]);
        setSelected(null);
        setConfirmed(false);
      }
    }, ADVANCE_MS);
  }

  // ── Submit ──────────────────────────────────────────────────────────────────

  function submitResults(states) {
    setSubmitting(true);
    clearSavedState(); // progress submitted — no need to resume
    const answers = states
      .filter((s) => s.timesCorrect + s.timesWrong > 0) // only submit attempted items
      .map((s) => ({
        vocab_item_id: s.vocab_item_id,
        skill_type: s.type, // per-skill tracking (was missing)
        is_correct: s.timesCorrect > s.timesWrong, // actual performance, not full mastery
      }));
    api
      .post(`/modules/${moduleId}/quiz/finish`, { answers })
      .then(({ data }) => setResult({ ...data, itemStates: states }))
      .catch(() => {
        const mastered = states.filter(
          (s) => s.mastery >= MASTERY_THRESHOLD,
        ).length;
        const score = Math.round((mastered / states.length) * 100);
        setResult({
          score,
          correct: mastered,
          total: states.length,
          high_score: score,
          passed: false,
          is_new_high: false,
          pass_threshold: 90,
          itemStates: states,
        });
      })
      .finally(() => setSubmitting(false));
  }

  function handleRestart() {
    clearTimeout(autoTimer.current);
    loadQuiz(true); // forceNew = true — discard saved state
  }

  function handleSubmitEarly() {
    clearTimeout(autoTimer.current);
    submitResults(itemStates);
  }

  // ── Guards ──────────────────────────────────────────────────────────────────

  if (loading || submitting) return <Spinner />;

  if (error) {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <p className="text-red-500 mb-4">{typeof error === 'string' ? tr(error) : friendlyError(error, language)}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-indigo-600 hover:underline text-sm"
        >{tr("← Go back")}</button>
      </div>
    );
  }

  if (result) {
    return (
      <ResultScreen
        result={result}
        onRestart={handleRestart}
        onBack={() => navigate(-1)}
      />
    );
  }

  if (!currentItem) return <Spinner />;

  const masteredCount = itemStates.filter(
    (s) => s.mastery >= MASTERY_THRESHOLD,
  ).length;
  const totalItems = itemStates.length;
  const masteryPct = Math.round((masteredCount / totalItems) * 100);
  const curState = itemStates.find((s) => s.stateId === currentItem.stateId);
  const currentMastery = curState?.mastery ?? 0;
  const isCorrectPick = selected === currentItem.correct_id;

  // Per-skill progress summary (for multi-skill / mixed sessions)
  const skillTypes_used = [...new Set(itemStates.map((s) => s.type))];
  const skillMastery = Object.fromEntries(
    skillTypes_used.map((t) => {
      const items = itemStates.filter((s) => s.type === t);
      const mastered = items.filter(
        (s) => s.mastery >= MASTERY_THRESHOLD,
      ).length;
      return [t, Math.round((mastered / items.length) * 100)];
    }),
  );

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      {/* Resumed session banner */}
      {hasSavedState && (
        <div className="flex items-center justify-between bg-indigo-50 border border-indigo-200 rounded-xl px-4 py-2 mb-4 text-sm">
          <span className="text-indigo-700">{tr("↩ Resuming saved session")}</span>
          <button
            onClick={() => loadQuiz(true)}
            className="text-xs text-indigo-500 hover:text-indigo-700 underline"
          >{tr("Start fresh")}</button>
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => navigate(-1)}
          className="text-gray-400 hover:text-gray-600 text-xl"
        >
          ←
        </button>
        <div className="text-center">
          <p className="text-sm font-medium text-gray-700 truncate max-w-[200px]">
            {moduleTitle}
          </p>
          <p className="text-xs text-gray-400">
            {masteredCount} / {totalItems}{' '}{tr("mastered")}{' '}</p>
        </div>
        <button
          onClick={handleSubmitEarly}
          className="text-xs text-gray-400 hover:text-gray-600 px-2 py-1 rounded-lg hover:bg-gray-100 transition-colors"
          title={tr("Submit current progress")}
        >{tr("Finish")}</button>
      </div>

      {/* Mastery progress bar — green fill = items mastered */}
      <div className="h-2 bg-gray-100 rounded-full mb-4 overflow-hidden">
        <div
          className="h-full bg-emerald-500 rounded-full transition-all duration-500"
          style={{ width: `${masteryPct}%` }}
        />
      </div>

      {/* Per-skill mini progress bars (only shown in mixed or multi-skill mode) */}
      {skillTypes_used.length > 1 && (
        <div className="flex gap-2 mb-4">
          {skillTypes_used.map((t) => (
            <div key={t} className="flex-1">
              <div className="flex justify-between items-center mb-0.5">
                <span
                  className={`text-xs px-1.5 py-0.5 rounded font-medium ${SKILL_COLOR[t] ?? 'bg-gray-100 text-gray-600'}`}
                >
                  {tr(SKILL_LABELS[t] ?? t)}
                </span>
                <span className="text-xs text-gray-400">
                  {skillMastery[t]}%
                </span>
              </div>
              <div className="h-1 bg-gray-100 rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all duration-500 ${
                    skillMastery[t] >= 85 ? 'bg-emerald-500' : 'bg-indigo-400'
                  }`}
                  style={{ width: `${skillMastery[t]}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Current item mastery dots */}
      <div className="flex items-center justify-between mb-4 px-1">
        <MasteryDots mastery={currentMastery} />
        <span className="text-xs text-gray-400">
          {Math.round(currentMastery * 100)}{' '}{tr("% mastery")}{' '}</span>
      </div>

      {/* Prompt card */}
      <div className="bg-white border border-gray-200 rounded-2xl p-6 mb-6 text-center shadow-sm">
        <p className="text-xs text-gray-400 uppercase tracking-wide mb-3">
          {tr(PROMPT_HINT[currentItem.type] ?? tr("Choose the correct answer"))}
        </p>
        <p className="text-4xl font-bold text-gray-900 leading-relaxed">
          {currentItem.prompt}
        </p>
      </div>

      {/* Options */}
      <div className="grid grid-cols-2 gap-3 mb-6">
        {currentItem.options.map((opt) => {
          const isThisCorrect = opt.id === currentItem.correct_id;
          const isThisSelected = opt.id === selected;
          let cls =
            'border border-gray-200 bg-white text-gray-800 hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer';
          let icon = null;
          if (confirmed) {
            if (isThisCorrect) {
              cls =
                'border-2 border-emerald-500 bg-emerald-50 text-emerald-800 cursor-default';
              icon = <span className="text-emerald-500 text-sm ml-1">✓</span>;
            } else if (isThisSelected) {
              cls =
                'border-2 border-red-400 bg-red-50 text-red-700 cursor-default';
              icon = <span className="text-red-400 text-sm ml-1">✗</span>;
            } else {
              cls =
                'border border-gray-100 bg-white text-gray-300 cursor-default';
            }
          } else if (isThisSelected) {
            cls = 'border-2 border-indigo-500 bg-indigo-50 text-indigo-800';
          }
          return (
            <button
              key={opt.id}
              onClick={() => handleSelect(opt.id)}
              disabled={confirmed}
              className={`rounded-xl px-4 py-4 text-center font-medium transition-colors text-lg flex items-center justify-center ${cls}`}
            >
              {opt.text}
              {icon}
            </button>
          );
        })}
      </div>

      {/* Feedback */}
      {confirmed ? (
        <div
          className={`p-3 rounded-xl text-sm font-medium text-center ${
            isCorrectPick
              ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
              : 'bg-red-50 text-red-700 border border-red-200'
          }`}
        >
          {isCorrectPick ? (
            <>{tr("✓ Correct!")}</>
          ) : (
            <>{tr("✗ Correct answer:")}{' '}
              <strong>
                {
                  currentItem.options.find(
                    (o) => o.id === currentItem.correct_id,
                  )?.text
                }
              </strong>
            </>
          )}
        </div>
      ) : (
        <p className="text-center text-xs text-gray-400">{tr("Tap an answer to continue")}</p>
      )}
    </div>
  );
}

function ResultScreen({ result, onRestart, onBack }) {
  const { tr } = useLocale();
  const {
    score,
    correct,
    high_score,
    passed,
    is_new_high,
    pass_threshold,
    skill_scores, // from backend: { kanji_to_kana: 85, kana_to_kanji: 72 }
    itemStates,
  } = result;

  const total = itemStates?.length ?? correct ?? 0;
  const totalAttempts = itemStates
    ? itemStates.reduce((sum, s) => sum + s.timesCorrect + s.timesWrong, 0)
    : total;
  const totalCorrectAttempts = itemStates
    ? itemStates.reduce((sum, s) => sum + s.timesCorrect, 0)
    : correct;
  const accuracy =
    totalAttempts > 0
      ? Math.round((totalCorrectAttempts / totalAttempts) * 100)
      : score;

  // Per-skill breakdown from session itemStates
  const sessionSkillTypes = itemStates
    ? [...new Set(itemStates.map((s) => s.type))]
    : [];
  const sessionSkillScores = Object.fromEntries(
    sessionSkillTypes.map((t) => {
      const items = itemStates.filter((s) => s.type === t);
      const mastered = items.filter(
        (s) => s.mastery >= MASTERY_THRESHOLD,
      ).length;
      return [t, Math.round((mastered / items.length) * 100)];
    }),
  );
  // Merge backend all-time high with session scores (backend wins if available)
  const displaySkillScores = { ...sessionSkillScores, ...(skill_scores ?? {}) };
  const skillEntries = Object.entries(displaySkillScores);

  // Hardest items: sorted by wrong answers, top 5, annotated with skill type
  const hardItems = itemStates
    ? [...itemStates]
        .sort((a, b) => b.timesWrong - a.timesWrong)
        .filter((s) => s.timesWrong > 0)
        .slice(0, 5)
    : [];

  return (
    <div className="max-w-lg mx-auto px-4 py-12 text-center">
      {/* Score ring */}
      <div className="relative inline-flex items-center justify-center mb-6">
        <svg width="120" height="120" className="-rotate-90">
          <circle
            cx="60"
            cy="60"
            r="52"
            fill="none"
            stroke="#e5e7eb"
            strokeWidth="10"
          />
          <circle
            cx="60"
            cy="60"
            r="52"
            fill="none"
            stroke={passed ? '#10b981' : '#f59e0b'}
            strokeWidth="10"
            strokeDasharray={`${2 * Math.PI * 52}`}
            strokeDashoffset={`${2 * Math.PI * 52 * (1 - score / 100)}`}
            strokeLinecap="round"
          />
        </svg>
        <div className="absolute flex flex-col items-center">
          <span className="text-3xl font-bold text-gray-900">{score}%</span>
          <span className="text-xs text-gray-400">{' '}{tr("mastered")}{' '}</span>
        </div>
      </div>

      <h2 className="text-2xl font-bold text-gray-900 mb-1">
        {passed ? tr("🎉 All Mastered!") : tr("📚 Keep Practicing")}
      </h2>
      <p className="text-gray-500 mb-1">
        {correct} / {total}{' '}{tr("skills mastered")}{' '}</p>
      <p className="text-gray-400 text-sm mb-1">
        {totalAttempts}{' '}{tr("attempts ·")}{' '}{accuracy}{' '}{tr("% accuracy")}{' '}</p>
      {is_new_high && (
        <p className="text-emerald-600 font-semibold text-sm mb-1">{tr("⭐ New high score!")}</p>
      )}
      <p className="text-gray-400 text-sm mb-6">{' '}{tr("Module best:")}{' '}{high_score}{tr("% · Pass threshold:")}{' '}
        {pass_threshold}%
      </p>

      {/* Per-skill score breakdown */}
      {skillEntries.length > 0 && (
        <div className="mb-6 text-left bg-white border border-gray-200 rounded-2xl p-4">
          <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">{tr("Skill breakdown (all-time best)")}</p>
          <div className="space-y-3">
            {skillEntries.map(([t, pct]) => (
              <div key={t}>
                <div className="flex justify-between items-center mb-1">
                  <span
                    className={`text-xs px-1.5 py-0.5 rounded font-medium ${SKILL_COLOR[t] ?? 'bg-gray-100 text-gray-600'}`}
                  >
                    {tr(SKILL_LABELS[t] ?? t)}
                  </span>
                  <span
                    className={`text-xs font-semibold ${pct >= pass_threshold ? 'text-emerald-600' : 'text-amber-600'}`}
                  >
                    {pct}%
                  </span>
                </div>
                <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-all duration-700 ${pct >= pass_threshold ? 'bg-emerald-500' : 'bg-amber-400'}`}
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {!passed && (
        <p className="text-amber-600 text-sm mb-5 bg-amber-50 border border-amber-200 rounded-xl px-4 py-3">{' '}{tr("You need")}{' '}{pass_threshold}{' '}{tr("% to pass and unlock the next module.")}{' '}</p>
      )}

      {/* Items that need more practice */}
      {hardItems.length > 0 && (
        <div className="text-left mb-8 bg-gray-50 border border-gray-200 rounded-2xl p-4">
          <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">{tr("Needs more practice")}</p>
          <div className="space-y-2">
            {hardItems.map((s) => (
              <div
                key={s.stateId}
                className="flex items-center justify-between text-sm"
              >
                <div className="flex items-center gap-2">
                  <span
                    className={`text-xs px-1 py-0.5 rounded ${SKILL_COLOR[s.type] ?? 'bg-gray-100 text-gray-500'}`}
                  >
                    {tr(SKILL_LABELS[s.type] ?? s.type)}
                  </span>
                  <span className="font-medium text-gray-800">{s.prompt}</span>
                </div>
                <span className="text-xs text-gray-400">
                  {s.timesCorrect}✓ {s.timesWrong}✗ ·{' '}
                  {Math.round(s.mastery * 100)}%
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-3 justify-center">
        <button
          onClick={onRestart}
          className="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold rounded-xl transition-colors"
        >{tr("Try Again")}</button>
        <button
          onClick={onBack}
          className="px-6 py-3 bg-gray-100 hover:bg-gray-200 text-gray-700 font-semibold rounded-xl transition-colors"
        >{tr("Back")}</button>
      </div>
    </div>
  );
}

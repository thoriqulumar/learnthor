import { useEffect, useState } from 'react';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';

import api from '../lib/api';
import { friendlyError } from '../lib/errors';
import { useLocale } from '../lib/locale';

const SKILL_LABELS = {
  kanji_to_kana: 'Romaji \u2192 Char',
  kana_to_kanji: 'Char \u2192 Romaji',
  fill_blank: 'Fill the Blank',
};

function getSkillTypesList(isCharacter) {
  return isCharacter
    ? ['kanji_to_kana', 'kana_to_kanji']
    : ['kanji_to_kana', 'kana_to_kanji', 'fill_blank'];
}

function getQuizTypes(isCharacter, sectionTitle) {
  if (isCharacter) {
    const script = /katakana/i.test(sectionTitle)
      ? 'Katakana'
      : /hiragana/i.test(sectionTitle)
        ? 'Hiragana'
        : 'Character';
    return [
      {
        type: 'kanji_to_kana',
        label: `Romaji \u2192 ${script}`,
        desc: `See the romaji, pick the ${script.toLowerCase()} character`,
        symbol: 'Aa',
      },
      {
        type: 'kana_to_kanji',
        label: `${script} \u2192 Romaji`,
        desc: `See the ${script.toLowerCase()} character, pick the romaji`,
        symbol: script === 'Katakana' ? '\u30a2' : '\u3042',
      },
      {
        type: 'mixed',
        label: 'Mixed',
        desc: 'Random combination of both directions',
        symbol: 'MIX',
      },
    ];
  }
  return [
    {
      type: 'kanji_to_kana',
      label: 'Kanji \u2192 Kana',
      desc: 'See the kanji, pick the reading',
      symbol: '\u6f22',
    },
    {
      type: 'kana_to_kanji',
      label: 'Kana \u2192 Kanji',
      desc: 'See the reading, pick the kanji',
      symbol: '\u304b',
    },
    {
      type: 'fill_blank',
      label: 'Fill the Blank',
      desc: 'Complete the example sentence',
      symbol: '\uff3f',
    },
    {
      type: 'mixed',
      label: 'Mixed',
      desc: 'Random combination of all types',
      symbol: 'MIX',
    },
  ];
}

export default function ModulePage() {
  const { tr, language } = useLocale();
  const { moduleId } = useParams();
  const navigate = useNavigate();
  const { state: navState } = useLocation();
  const moduleTitle = navState?.title ?? 'Module';
  const sectionTitle = navState?.sectionTitle ?? '';
  const [vocabCount, setVocabCount] = useState(null);
  const [isCharacter, setIsCharacter] = useState(false);
  const [skillScores] = useState(navState?.skillScores ?? {});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .get(`/modules/${moduleId}/vocab`)
      .then(({ data }) => {
        const items = data.items ?? [];
        setVocabCount(data.count ?? items.length);
        setIsCharacter(items.length > 0 && items[0].part_of_speech === 'character');
      })
      .catch(setError)
      .finally(() => setLoading(false));
  }, [moduleId]);

  if (loading) return <Spinner />;

  if (error) {
    return (
      <div className="page-panel text-center">
        <p className="font-semibold mb-2">{tr('Module Locked')}</p>
        <p className="text-sm mb-6">
          {typeof error === 'string' ? tr(error) : friendlyError(error, language)}
        </p>
        <button onClick={() => navigate(-1)} className="module-error-back">
          &larr; {tr('Go back')}
        </button>
      </div>
    );
  }

  const quizTypes = getQuizTypes(isCharacter, sectionTitle);
  const recordedScores = Object.values(skillScores).filter(Number.isFinite);
  const averageScore = recordedScores.length
    ? Math.round(recordedScores.reduce((total, score) => total + score, 0) / recordedScores.length)
    : null;

  return (
    <div className="module-workspace">
      <header className="module-page-header">
        <button onClick={() => navigate(-1)} className="module-back" aria-label={tr('Go back')}>
          <span aria-hidden="true">&larr;</span>
        </button>
        <div className="module-title-block">
          <p>{tr('Module workspace')}</p>
          <h1>{moduleTitle}</h1>
          {sectionTitle && <span>{sectionTitle}</span>}
        </div>
        <div className="module-kind">
          {tr(isCharacter ? 'Character practice' : 'Vocabulary practice')}
        </div>
      </header>

      <div className="module-page-layout">
        <main className="module-page-content">
          <section className="module-launchpad" aria-labelledby="learning-path-title">
            <ModuleSectionHeading
              eyebrow={`01 / ${tr('Learning path')}`}
              title={tr('Learn and review')}
              description={tr('Start with a guided lesson, then review the cards.')}
              id="learning-path-title"
            />

            <div className="module-action-grid">
              <Link
                to={`/modules/${moduleId}/learn`}
                state={{ title: moduleTitle, sectionTitle }}
                className="module-action module-action-primary"
              >
                <span className="module-action-index">01</span>
                <div>
                  <small>{tr('Guided lesson')}</small>
                  <strong>{tr('Learn')}</strong>
                  <p>{tr('Step-by-step word introduction')}</p>
                </div>
                <b aria-hidden="true">&rarr;</b>
              </Link>

              <Link to={`/modules/${moduleId}/vocab`} className="module-action module-action-study">
                <span className="module-action-index">02</span>
                <div>
                  <small>{tr('Reference')}</small>
                  <strong>{tr('Study Cards')}</strong>
                  <p>{tr('Flip-card reference')}</p>
                </div>
                <b aria-hidden="true">&rarr;</b>
              </Link>
            </div>
          </section>

          <section className="module-quiz-area" aria-labelledby="quiz-arena-title">
            <ModuleSectionHeading
              eyebrow={`02 / ${tr('Quiz arena')}`}
              title={tr('Choose a quiz type')}
              description={tr('Focused ways to test this module.')}
              id="quiz-arena-title"
            />

            <div className="quiz-choice-grid">
              {quizTypes.map(({ type, label, desc, symbol }, index) => {
                const typeSkills = type === 'mixed' ? getSkillTypesList(isCharacter) : [type];
                const score = type === 'mixed' ? averageScore : skillScores[type];
                return (
                  <Link
                    key={type}
                    to={`/modules/${moduleId}/quiz?type=${type}&limit=10`}
                    state={{
                      title: moduleTitle,
                      sectionTitle,
                      skillTypes: getSkillTypesList(isCharacter),
                      skillScores,
                    }}
                    className="quiz-choice-card"
                  >
                    <div className="quiz-choice-topline">
                      <span>{String(index + 1).padStart(2, '0')}</span>
                      {Number.isFinite(score) && <strong>{score}%</strong>}
                    </div>
                    <div className="quiz-choice-body">
                      <b aria-hidden="true">{symbol}</b>
                      <div>
                        <h3>{tr(label)}</h3>
                        <p>{tr(desc)}</p>
                      </div>
                    </div>
                    {typeSkills.some((skill) => skill in skillScores) && (
                      <div className="quiz-skill-summary">
                        {typeSkills.filter((skill) => skill in skillScores).map((skill) => (
                          <div key={skill}>
                            <span>{tr(SKILL_LABELS[skill])}</span>
                            <i><em style={{ width: `${skillScores[skill]}%` }} /></i>
                          </div>
                        ))}
                      </div>
                    )}
                    <span className="quiz-choice-arrow" aria-hidden="true">&rarr;</span>
                  </Link>
                );
              })}
            </div>
          </section>
        </main>

        <aside className="module-sidebar" aria-label={tr('Module stats')}>
          <section className="module-word-stat">
            <p>{tr('Words in this module')}</p>
            <strong>{vocabCount ?? '—'}</strong>
            <span>{tr('{count} words', { count: vocabCount ?? 0 })}</span>
          </section>

          <section className="module-readiness">
            <div>
              <p>{tr('Readiness')}</p>
              <strong>{averageScore === null ? '—' : `${averageScore}%`}</strong>
            </div>
            <span>{tr(averageScore === null ? 'No quiz score yet' : 'Based on your saved quiz scores')}</span>
            <div className="readiness-meter">
              <i style={{ width: `${averageScore ?? 0}%` }} />
            </div>
          </section>

          <section className="module-practice-plan">
            <p>{tr('Practice plan')}</p>
            <ol>
              <li><b>1</b><span>{tr('Learn the words')}</span></li>
              <li><b>2</b><span>{tr('Review with study cards')}</span></li>
              <li><b>3</b><span>{tr('Test one skill or go mixed')}</span></li>
            </ol>
          </section>
        </aside>
      </div>
    </div>
  );
}

function ModuleSectionHeading({ eyebrow, title, description, id }) {
  return (
    <div className="module-section-heading">
      <div>
        <p>{eyebrow}</p>
        <h2 id={id}>{title}</h2>
      </div>
      <span>{description}</span>
    </div>
  );
}

function Spinner() {
  return (
    <div className="page-loader">
      <span />
    </div>
  );
}

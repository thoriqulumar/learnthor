import { useEffect, useMemo, useState } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';

import { ErrorBanner } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import api from '../lib/api';
import { useLocale } from '../lib/locale';

const LEVELS = {
  beginner: { api: null, title: 'Beginner', eyebrow: 'START / FOUNDATION', description: 'Build your reading foundation before entering the JLPT route.' },
  n5: { api: 'N5', title: 'JLPT N5', eyebrow: 'LEVEL 01 / FOUNDATION', description: 'Build useful everyday Japanese and learn the shape of JLPT questions.' },
  n4: { api: 'N4', title: 'JLPT N4', eyebrow: 'LEVEL 02 / EVERYDAY', description: 'Connect ideas, read practical texts, and handle a wider range of grammar.' },
  n3: { api: 'N3', title: 'JLPT N3', eyebrow: 'LEVEL 03 / INTERMEDIATE', description: 'Read for nuance, reason from context, and control intermediate grammar.' },
};

const CATEGORIES = [
  { key: 'vocabulary', label: 'Vocabulary', japanese: '語彙', description: 'Vocabulary modules' },
  { key: 'particles', label: 'Particles', japanese: '助詞', description: 'Understand the small words that define each sentence relationship.' },
  { key: 'grammar', label: 'Grammar', japanese: '文法', description: 'Learn patterns through explanations, contrasts, and examples.' },
  { key: 'reading', label: 'Reading / Dokkai', japanese: '読解', description: 'Read passages and answer questions from evidence in the text.' },
];

function sectionLevel(section) {
  if (/N3/i.test(section.title)) return 'n3';
  if (/N4/i.test(section.title)) return 'n4';
  if (/N5/i.test(section.title)) return 'n5';
  return 'beginner';
}

function ModuleCard({ mod, sectionTitle, bypass }) {
  const { tr } = useLocale();
  const locked = mod.locked && !bypass;
  const content = <>
    <span className="module-status" aria-hidden="true">{locked ? '×' : mod.passed ? '✓' : '↗'}</span>
    <span className="module-copy"><strong>{mod.title}</strong><small>{tr('{count} words', { count: mod.vocab_count })}{mod.high_score > 0 && ` · ${tr('Best: {score}%', { score: mod.high_score })}`}</small></span>
  </>;
  return locked ? <div className="module-tile is-locked" aria-disabled="true">{content}</div> :
    <Link className={`module-tile${mod.passed ? ' is-passed' : ''}`} to={`/modules/${mod.id}`}
      state={{ title: mod.title, sectionTitle, skillScores: mod.skill_scores ?? {} }}>{content}</Link>;
}

export default function LearningLevelPage() {
  const { level } = useParams();
  const config = LEVELS[level];
  const { user } = useAuth();
  const { tr, language } = useLocale();
  const [sections, setSections] = useState([]);
  const [lessons, setLessons] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeCategory, setActiveCategory] = useState('vocabulary');
  const category = CATEGORIES.find(item => item.key === activeCategory);
  const categoryLessons = lessons.filter(lesson => lesson.category === activeCategory);
  const privileged = ['admin', 'editor'].includes(user?.role);
  const premium = privileged || user?.subscription_tier === 'premium';
  const bypass = privileged || (premium && user?.progression_bypass === true);

  useEffect(() => {
    if (!config) return;
    const requests = [api.get('/sections')];
    if (config.api) requests.push(api.get('/jlpt/lessons', { params: { level: config.api } }));
    Promise.all(requests).then(([sectionResponse, lessonResponse]) => {
      setSections((sectionResponse.data.sections ?? []).filter(section => sectionLevel(section) === level));
      setLessons(lessonResponse?.data.lessons ?? []);
    }).catch(setError).finally(() => setLoading(false));
  }, [config, level]);

  const stats = useMemo(() => {
    const total = sections.reduce((sum, section) => sum + section.modules_total, 0);
    const passed = sections.reduce((sum, section) => sum + section.modules_passed, 0);
    return { total, passed, progress: total ? Math.round(passed * 100 / total) : 0 };
  }, [sections]);

  if (!config) return <Navigate to="/learn" replace />;
  if (loading) return <div className="page-loader" role="status"><span /></div>;
  if (error) return <div className="page-panel"><ErrorBanner err={error} lang={language} /></div>;

  return <div className="level-page">
    <header className="level-page-hero">
      <Link to="/learn" className="level-back" aria-label={tr('Back to levels')}>←</Link>
      <div><p>{config.eyebrow}</p><h1>{config.title}</h1><span>{tr(config.description)}</span></div>
      <div className="level-hero-stat"><strong>{stats.progress}%</strong><small>{tr('vocabulary route')}</small></div>
    </header>

    {config.api && <section className="jlpt-skills" aria-labelledby="jlpt-skills-title">
      <div className="level-section-heading"><div><p>01 / {tr('JLPT skills')}</p><h2 id="jlpt-skills-title">{tr('Study by skill')}</h2></div>
        <Link className="level-exam-launch" to={`/learn/${level}/exam`}>{tr('JLPT practice exam')} ↗</Link></div>
      <div className="skill-category-nav" role="group" aria-label={tr('JLPT skills')}>{CATEGORIES.map(item =>
        <button key={item.key} type="button" aria-pressed={activeCategory === item.key}
          onClick={() => setActiveCategory(item.key)}>
          <b aria-hidden="true">{item.japanese}</b><span>{tr(item.label)}</span>
          <small>{item.key === 'vocabulary' ? stats.total : lessons.filter(lesson => lesson.category === item.key).length}</small>
        </button>
      )}</div>
      {activeCategory !== 'vocabulary' && <div className="skill-library" aria-live="polite">
        <header className="skill-library-heading"><div><h3>{tr(category.label)}</h3><p>{tr(category.description)}</p></div>
          <span>{tr('{count} lessons', { count: categoryLessons.length })}</span>
        </header>
        <div className="skill-lesson-grid">{categoryLessons.map((lesson, index) => {
            const title = language === 'id' ? lesson.title_id : lesson.title_en;
            const summary = language === 'id' ? lesson.summary_id : lesson.summary_en;
            const content = <><div className="skill-lesson-top"><span>{String(index + 1).padStart(2, '0')}</span>
              <b>{lesson.locked ? tr('Premium lesson') : lesson.attempt_count > 0 ? `${lesson.best_score}%` : '→'}</b></div>
              <h4>{title}</h4><p>{summary}</p></>;
            return lesson.locked ? <div key={lesson.id} className="skill-lesson-card is-locked" aria-disabled="true">{content}</div> :
              <Link key={lesson.id} className="skill-lesson-card" to={`/learn/${level}/lessons/${lesson.id}`}>{content}</Link>;
        })}</div>
        {categoryLessons.length === 0 && <p>{tr('No lessons available yet.')}</p>}
      </div>}
    </section>}

    {(activeCategory === 'vocabulary' || !config.api) && <section className="level-vocabulary" aria-labelledby="level-vocabulary-title">
      <div className="level-section-heading"><div><p>{config.api ? '02' : '01'} / {tr('Vocabulary route')}</p><h2 id="level-vocabulary-title">{tr(config.api ? 'Vocabulary modules' : 'Beginner foundations')}</h2></div><span>{tr('{passed} of {total} modules complete', stats)}</span></div>
      <div className="section-list">{sections.map((section, index) => {
        const premiumLocked = section.is_premium && !premium;
        const sectionLocked = !bypass && (section.locked || premiumLocked);
        return <details key={section.id} className={`course-section level-course-section${sectionLocked ? ' section-locked' : ''}`} open={sections.length === 1 || index === 0}>
          <summary className="section-heading"><span className="section-number">{String(index + 1).padStart(2, '0')}</span><div><p className="section-kicker">{section.is_premium ? 'PREMIUM' : 'FOUNDATION'}</p><h3>{section.title}</h3><small>{tr('{count} modules', { count: section.modules_total })}</small></div><span className="section-score">{section.mastery_pct}%</span></summary>
          <div className="section-meter"><span style={{ width: `${section.mastery_pct}%` }} /></div>
          {sectionLocked && <div className="section-lock-message"><strong>{tr(premiumLocked ? 'Premium section' : 'Locked')}</strong><p>{tr(premiumLocked ? 'Upgrade to open this level.' : 'Master the previous section to unlock')}</p>{premiumLocked && <Link to="/upgrade">{tr('Upgrade to Premium →')}</Link>}</div>}
          <div className="module-grid">{(section.modules ?? []).map(mod => <ModuleCard key={mod.id} mod={{ ...mod, locked: sectionLocked || mod.locked }} sectionTitle={section.title} bypass={bypass} />)}</div>
          {!sectionLocked && (section.mastery_pct === 100 || bypass) && section.modules_total > 0 && <Link className="section-test" to={`/sections/${section.id}/quiz`} state={{ sectionTitle: section.title }}>{tr('Section Test')} <span>↗</span></Link>}
        </details>;
      })}</div>
    </section>}
  </div>;
}

import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import api from '../lib/api';
import { ErrorBanner } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import { useLocale } from '../lib/locale';

const LEVELS = [
  { key: 'beginner', label: 'Beginner', marker: 'START', description: 'Kana, greetings, numbers, and essential foundations.' },
  { key: 'n5', label: 'JLPT N5', marker: 'N5', description: 'Foundation vocabulary, particles, grammar, reading, and exam practice.' },
  { key: 'n4', label: 'JLPT N4', marker: 'N4', description: 'Everyday Japanese with broader grammar and reading comprehension.' },
  { key: 'n3', label: 'JLPT N3', marker: 'N3', description: 'Intermediate Japanese, inference, nuance, and exam strategy.' },
];

function levelForSection(section) {
  if (/N3/i.test(section.title)) return 'n3';
  if (/N4/i.test(section.title)) return 'n4';
  if (/N5/i.test(section.title)) return 'n5';
  return 'beginner';
}

export default function HomePage() {
  const { user, refreshUser } = useAuth();
  const { tr, language } = useLocale();
  const privileged = ['admin', 'editor'].includes(user?.role);
  const premium = privileged || user?.subscription_tier === 'premium';
  const bypass = privileged || (premium && user?.progression_bypass === true);
  const [sections, setSections] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [reviewStats, setReviewStats] = useState(null);
  const [glossSaving, setGlossSaving] = useState(false);
  const [preferenceError, setPreferenceError] = useState(null);
  const gloss = user?.preferred_gloss ?? 'both';
  useEffect(() => {
    api.get('/sections').then(({ data }) => setSections(data.sections ?? []))
      .catch(setError).finally(() => setLoading(false));
    api.get('/reviews/stats').then(({ data }) => setReviewStats(data)).catch(() => {});
  }, []);
  function updateGloss(next) {
    if (next === gloss || glossSaving) return;
    setGlossSaving(true); setPreferenceError(null);
    api.put('/me/preferences', { preferred_gloss: next })
      .then(() => refreshUser({ preferred_gloss: next }))
      .catch(setPreferenceError).finally(() => setGlossSaving(false));
  }
  if (loading) return <div className="page-loader" role="status" aria-label={tr('Loading…')}><span /></div>;
  if (error) return <div className="page-panel"><ErrorBanner err={error} lang={language} onRetry={() => window.location.reload()} /></div>;
  const effective = sections.map((s, i) => ({ ...s,
    premiumLocked: s.is_premium && !premium,
    locked: !bypass && (s.locked || (s.is_premium && !premium) || (i > 0 && (sections[i - 1].mastery_pct ?? 0) < 100)),
  }));
  const total = sections.reduce((n, s) => n + s.modules_total, 0);
  const passed = sections.reduce((n, s) => n + s.modules_passed, 0);
  const nextSection = effective.find(s => !s.locked && s.modules?.some(m => !m.passed && (!m.locked || bypass)));
  const next = nextSection?.modules.find(m => !m.passed && (!m.locked || bypass));
  return <div className="dashboard">
    <header className="study-hero">
      <div className="hero-copy"><p className="eyebrow">LEARNTHOR / 日本語</p><h1>{tr('Make Japanese yours.')}</h1>
        <p className="hero-description">{tr('A new word. A new possibility.')}</p>
        {next && <Link className="hero-cta" to={'/modules/' + next.id} state={{ title: next.title, sectionTitle: nextSection.title, skillScores: next.skill_scores }}>{tr('Continue learning')} <span>↗</span></Link>}
      </div>
      <div className="hero-art" aria-hidden="true"><span className="art-star">✦</span><span className="art-kanji">学</span><span className="art-sticker">LEVEL UP!</span></div>
    </header>
    <div className="dashboard-strip">
      <div><span className="strip-label">{tr('Learning progress')}</span><strong>{passed}<span> / {total}</span></strong><small>{tr('modules')}</small></div>
      <div className="gloss-control"><label htmlFor="gloss">{tr('Vocabulary language')}</label><select id="gloss" value={gloss} onChange={e => updateGloss(e.target.value)} disabled={glossSaving}>
        <option value="en">English</option><option value="id">Bahasa Indonesia</option><option value="both">EN + ID</option>
      </select></div>
    </div>
    {preferenceError && <ErrorBanner err={preferenceError} lang={language} />}
    {user?.progression_bypass && premium && <p className="personal-notice"><strong>★ {tr('Personal access')}</strong> {tr('All lessons available. Your scores still reflect your practice.')}</p>}
    {reviewStats?.due > 0 && <Link to="/review" className="review-banner"><div><span className="eyebrow">{tr('Your next move')}</span><h2>{tr('Daily Review')}</h2><p>{tr('{count} cards due', { count: reviewStats.due })}</p></div><span aria-hidden="true">↗</span></Link>}
    {!premium && <Link to="/upgrade" className="upgrade-banner"><strong>{tr('👑 Upgrade to Premium')}</strong><span>{tr('Unlock N4, N3 & Daily Vocab — one-time payment, lifetime access')} ↗</span></Link>}
    <div className="route-heading"><h2>{tr('Choose your level')}</h2><span>01 — 04</span></div>
    <div className="level-route-grid">{LEVELS.map((level, index) => {
      const group = effective.filter(section => levelForSection(section) === level.key);
      const groupTotal = group.reduce((sum, section) => sum + section.modules_total, 0);
      const groupPassed = group.reduce((sum, section) => sum + section.modules_passed, 0);
      const progress = groupTotal ? Math.round(groupPassed * 100 / groupTotal) : 0;
      const locked = group.length > 0 && group.every(section => section.locked) && !bypass;
      return <Link key={level.key} to={`/learn/${level.key}`} className={'level-route-card' + (locked ? ' is-locked' : '')}>
        <div className="level-route-top"><span>{String(index + 1).padStart(2, '0')}</span><b>{level.marker}</b></div>
        <h3>{tr(level.label)}</h3><p>{tr(level.description)}</p>
        <div className="level-route-progress"><i><em style={{ width: `${progress}%` }} /></i><strong>{progress}%</strong></div>
        <small>{tr('{passed} of {total} modules complete', { passed: groupPassed, total: groupTotal })}</small>
        <span className="level-route-arrow" aria-hidden="true">→</span>
      </Link>;
    })}</div>
  </div>;
}

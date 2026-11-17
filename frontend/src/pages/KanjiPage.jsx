import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import { ErrorBanner } from '../components/ui';
import api from '../lib/api';
import { useLocale } from '../lib/locale';

const LEVELS = ['ALL', 'N5', 'N4', 'N3'];

export default function KanjiPage() {
  const { tr, language } = useLocale();
  const [items, setItems] = useState([]);
  const [level, setLevel] = useState('ALL');
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.get('/kanji').then(({ data }) => setItems(data.items ?? []))
      .catch(setError).finally(() => setLoading(false));
  }, []);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return items.filter((item) => {
      if (level !== 'ALL' && item.jlpt_level !== level) return false;
      if (!needle) return true;
      return [item.character, item.english_meaning, item.indonesian_meaning, ...(item.on_readings ?? []), ...(item.kun_readings ?? [])]
        .some((value) => value?.toLowerCase().includes(needle));
    });
  }, [items, level, query]);

  if (loading) return <div className="page-loader" role="status"><span /></div>;
  if (error) return <div className="page-panel"><ErrorBanner err={error} lang={language} onRetry={() => window.location.reload()} /></div>;

  return <div className="kanji-library">
    <header className="kanji-library-header">
      <div><p className="eyebrow">KANJI ARCHIVE / 漢字</p><h1>{tr('Kanji library')}</h1><p>{tr('Study characters from N5 through N3. Open a card to see how to write it.')}</p></div>
      <strong><span>{items.length}</span>{tr('characters')}</strong>
    </header>
    <div className="kanji-toolbar">
      <div className="kanji-levels" aria-label={tr('JLPT level')}>{LEVELS.map((item) => <button key={item} className={level === item ? 'active' : ''} onClick={() => setLevel(item)}>{item === 'ALL' ? tr('All') : item}</button>)}</div>
      <label className="kanji-search"><span>{tr('Search')}</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={tr('Kanji, reading, or meaning')} /></label>
    </div>
    <div className="kanji-result-line"><strong>{filtered.length}</strong> {tr('characters shown')}</div>
    {filtered.length === 0 ? <div className="kanji-empty">{tr('No kanji matches your search.')}</div> : <div className="kanji-grid">
      {filtered.map((item) => <Link to={`/kanji/${encodeURIComponent(item.character)}`} className="kanji-card" key={item.character} state={{ item }}>
        <span className={`kanji-level level-${item.jlpt_level.toLowerCase()}`}>{item.jlpt_level}</span>
        <b>{item.character}</b>
        <p>{language === 'id' ? item.indonesian_meaning : item.english_meaning}</p>
        <small>{item.stroke_count} {tr('strokes')} · {(item.on_readings?.[0] || item.kun_readings?.[0] || '—')}</small>
      </Link>)}
    </div>}
    <p className="kanji-attribution">{tr('Kanji data: OpenJLPT / KANJIDIC2. Stroke diagrams: KanjiVG.')}</p>
  </div>;
}

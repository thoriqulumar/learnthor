import { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';

import { ErrorBanner } from '../components/ui';
import api from '../lib/api';
import { useLocale } from '../lib/locale';

export default function KanjiDetailPage() {
  const { character } = useParams();
  const { state } = useLocation();
  const { tr, language } = useLocale();
  const [item, setItem] = useState(state?.item ?? null);
  const [loading, setLoading] = useState(!state?.item);
  const [error, setError] = useState(null);
  const [word, setWord] = useState(null);
  const [wordItems, setWordItems] = useState(null);
  const [wordError, setWordError] = useState(null);
  const dialogRef = useRef(null);

  useEffect(() => {
    if (!word) return;
    let active = true;
    dialogRef.current.showModal();
    api.get('/kanji/word', { params: { q: word } })
      .then(({ data }) => { if (active) setWordItems(data.items); })
      .catch(err => { if (active) setWordError(err); });
    return () => { active = false; };
  }, [word]);

  function openWord(value) {
    setWordItems(null);
    setWordError(null);
    setWord(value);
  }

  function closeWord() {
    dialogRef.current?.close();
    setWord(null);
  }

  useEffect(() => {
    if (item?.character === character) return;
    api.get(`/kanji/${encodeURIComponent(character)}`).then(({ data }) => setItem(data))
      .catch(setError).finally(() => setLoading(false));
  }, [character, item?.character]);

  if (loading) return <div className="page-loader" role="status"><span /></div>;
  if (error || !item) return <div className="page-panel"><ErrorBanner err={error} lang={language} /></div>;

  return <article className="kanji-detail">
    <header className="kanji-detail-header">
      <Link to="/kanji" aria-label={tr('Back to kanji library')}>←</Link>
      <div><p className="eyebrow">{item.jlpt_level} / {item.stroke_count} {tr('strokes')}</p><h1>{item.character}</h1></div>
      <span>{item.jlpt_level}</span>
    </header>
    <div className="kanji-detail-layout">
      <section className="stroke-panel">
        <p>{tr('How to write')}</p>
        <div className="stroke-diagram"><img src={item.stroke_svg} alt={tr('Stroke order for {character}', { character: item.character })} /></div>
        <small>{tr('Follow the numbered strokes from first to last.')}</small>
      </section>
      <div className="kanji-facts">
        <section className="meaning-panel"><p className="fact-label">{tr('Meaning')}</p><div><span>EN</span><strong>{item.english_meaning}</strong></div><div><span>ID</span><strong>{item.indonesian_meaning}</strong></div></section>
        <section><p className="fact-label">{tr("On'yomi")}</p><div className="reading-chips">{item.on_readings.length ? item.on_readings.map(reading => <span key={reading}>{reading}</span>) : <span>—</span>}</div></section>
        <section><p className="fact-label">{tr("Kun'yomi")}</p><div className="reading-chips">{item.kun_readings.length ? item.kun_readings.map(reading => <span key={reading}>{reading}</span>) : <span>—</span>}</div></section>
        <section className="kanji-meta"><div><span>{tr('Radical')}</span><strong>{item.radical || '—'}</strong></div><div><span>{tr('School grade')}</span><strong>{item.grade || '—'}</strong></div><div><span>{tr('Frequency')}</span><strong>{item.frequency || '—'}</strong></div></section>
        <section><p className="fact-label">{tr('Example words')}</p><div className="example-word-list">{item.example_words.map(example => <button type="button" key={example} onClick={() => openWord(example)}>{example}</button>)}</div></section>
      </div>
    </div>
    <p className="kanji-attribution">{tr('Kanji data: OpenJLPT / KANJIDIC2. Stroke diagrams: KanjiVG.')}</p>
    {word && <dialog ref={dialogRef} className="kanji-word-dialog" aria-labelledby="word-title" onCancel={closeWord} onClick={event => { if (event.target === event.currentTarget) closeWord(); }}>
      <div className="word-dialog-content">
        <button type="button" className="word-dialog-close" onClick={closeWord} aria-label={tr('Close')}>×</button>
        <p className="fact-label">{tr('Example words')}</p><h2 id="word-title" lang="ja">{word}</h2>
        {wordError && <ErrorBanner err={wordError} lang={language} />}
        {!wordItems && !wordError && <p role="status">{tr('Loading…')}</p>}
        {wordItems?.length === 0 && <p>{tr('This word’s reading and meaning are not yet in the vocabulary catalog.')}</p>}
        {wordItems?.map(entry => <section key={entry.id} className="word-definition">
          <p className="fact-label">{tr('Reading')}</p><p className="word-kana" lang="ja">{entry.kana}</p>
          {entry.romaji && <p className="word-romaji">{entry.romaji}</p>}
          <dl><dt>EN</dt><dd>{entry.english_meaning}</dd><dt>ID</dt><dd>{entry.indonesian_meaning || tr('Translation not available yet.')}</dd></dl>
          {entry.part_of_speech && <p>{entry.part_of_speech}</p>}
          {entry.example_sentence && <blockquote lang="ja">{entry.example_sentence}</blockquote>}
        </section>)}
      </div>
    </dialog>}
  </article>;
}

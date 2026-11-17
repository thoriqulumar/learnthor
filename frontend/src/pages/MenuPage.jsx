import { Link } from 'react-router-dom';

import { useLocale } from '../lib/locale';

export default function MenuPage() {
  const { tr } = useLocale();
  return <div className="main-menu">
    <header className="menu-hero">
      <div>
        <p className="eyebrow">LEARNTHOR / 日本語</p>
        <h1>{tr('Choose your path.')}</h1>
        <p>{tr('Build vocabulary through the learning route, or study individual kanji and their stroke order.')}</p>
      </div>
      <span aria-hidden="true">進</span>
    </header>
    <section className="menu-routes" aria-label={tr('Learning menu')}>
      <Link className="menu-route menu-route-learning" to="/learn">
        <span className="menu-route-number">01</span>
        <div><small>{tr('Structured course')}</small><h2>{tr('Learning route')}</h2><p>{tr('Lessons, vocabulary quizzes, section tests, and review progress.')}</p></div>
        <b aria-hidden="true">学</b><i aria-hidden="true">↗</i>
      </Link>
      <Link className="menu-route menu-route-kanji" to="/kanji">
        <span className="menu-route-number">02</span>
        <div><small>{tr('Character library')}</small><h2>{tr('Kanji')}</h2><p>{tr('All N5–N3 kanji with readings, meanings, examples, and stroke order.')}</p></div>
        <b aria-hidden="true">漢</b><i aria-hidden="true">↗</i>
      </Link>
    </section>
  </div>;
}

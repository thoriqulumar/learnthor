import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import api from '../lib/api';
import { ErrorBanner } from '../components/ui';
import { useLocale } from '../lib/locale';

const LABELS = { vocabulary: 'Vocabulary', particles: 'Particles', grammar: 'Grammar', exam: 'Sentence order and context', reading: 'Reading / Dokkai' };

export default function JLPTExamPage() {
  const { level } = useParams();
  const { tr, language } = useLocale();
  const [paper, setPaper] = useState(null);
  const [answers, setAnswers] = useState({});
  const [result, setResult] = useState(null);
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    api.get('/jlpt/exam', { params: { level } }).then(({ data }) => setPaper(data)).catch(setError);
  }, [level]);
  function finish() {
    setBusy(true);
    api.post('/jlpt/exam/finish', { token: paper.token, answers: paper.questions.map(q => ({ question_id: q.id, selected_index: answers[q.id] ?? -1 })) })
      .then(({ data }) => setResult(data)).catch(setError).finally(() => setBusy(false));
  }
  return <div className="level-page">
    <header className="level-page-hero"><Link className="level-back" to={`/learn/${level}`}>←</Link>
      <div><p>{level.toUpperCase()} / {tr('Exam Practice')}</p><h1>{tr('JLPT practice exam')}</h1><span>{tr('Vocabulary → Grammar and particles → Reading. Results appear after finishing.')}</span></div></header>
    <p className="exam-policy">{tr('Practice score only; this is not an official JLPT score.')}</p>
    {error && <ErrorBanner err={error} lang={language} />}
    {!paper && !error && <div className="page-loader" role="status"><span /></div>}
    {result && <div className="exam-result-summary"><strong>{result.score}%</strong><span>{tr('{correct} of {total} correct', result)}</span>
      {Object.entries(LABELS).map(([category,label]) => {
        const questions = paper.questions.filter(q => q.category === category);
        const correct = result.results.filter(r => r.correct && questions.some(q => q.id === r.question_id)).length;
        return questions.length > 0 && <p key={category}>{tr(label)}: {correct} / {questions.length}</p>;
      })}</div>}
    {paper && Object.entries(LABELS).map(([category,label],categoryIndex) => {
      const questions = paper.questions.filter(q => q.category === category);
      if (!questions.length) return null;
      return <section key={category} className="exam-skill-section">
        <h2>{String(categoryIndex + 1).padStart(2,'0')} / {tr(label)}</h2>
        <div className="exam-paper-grid">{questions.map(q => {
          const grade = result?.results.find(r => r.question_id === q.id);
          return <fieldset className="jlpt-question exam-paper-question" key={q.id}>
            <legend>{language === 'id' ? q.prompt_id : q.prompt_en}</legend>
            {q.passage && <blockquote lang="ja">{q.passage}</blockquote>}
            <div>{q.options.map((option,index) => <label key={index} className={grade?.correct_index === index ? 'is-correct' : answers[q.id] === index ? 'is-selected' : ''}>
              <input type="radio" name={q.id} checked={answers[q.id] === index} disabled={Boolean(result)} onChange={() => setAnswers(current => ({...current,[q.id]:index}))} /><span>{option}</span>
            </label>)}</div>
            {grade && <p className="jlpt-explanation">{language === 'id' ? grade.explanation_id : grade.explanation_en}</p>}
          </fieldset>;
        })}</div>
      </section>;
    })}
    {paper && !result && <button className="jlpt-submit" disabled={busy} onClick={finish}>{tr(busy ? 'Checking…' : 'Finish quiz')}</button>}
    {result && <Link className="section-test" to={`/learn/${level}`}>{tr('Back to level')} →</Link>}
  </div>;
}

import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';

import { ErrorBanner } from '../components/ui';
import api from '../lib/api';
import { useLocale } from '../lib/locale';

export default function JLPTLessonPage() {
  const { level, lessonId } = useParams();
  const navigate = useNavigate();
  const { tr, language } = useLocale();
  const [lesson, setLesson] = useState(null);
  const [answers, setAnswers] = useState({});
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.get(`/jlpt/lessons/${lessonId}`).then(({ data }) => setLesson(data))
      .catch(setError).finally(() => setLoading(false));
  }, [lessonId]);

  function submitQuiz() {
    setSubmitting(true); setError(null);
    const payload = lesson.questions.map(question => ({ question_id: question.id, selected_index: answers[question.id] ?? -1 }));
    api.post(`/jlpt/lessons/${lessonId}/finish`, { answers: payload }).then(({ data }) => {
      setResult(data);
      document.getElementById('jlpt-quiz')?.scrollIntoView({ behavior: 'smooth' });
    }).catch(setError).finally(() => setSubmitting(false));
  }

  if (loading) return <div className="page-loader" role="status"><span /></div>;
  if (error && !lesson) return <div className="page-panel"><ErrorBanner err={error} lang={language} /><button className="module-error-back" onClick={() => navigate(-1)}>← {tr('Go back')}</button></div>;
  if (!lesson) return null;

  const title = language === 'id' ? lesson.title_id : lesson.title_en;
  const summary = language === 'id' ? lesson.summary_id : lesson.summary_en;
  const resultByQuestion = new Map((result?.results ?? []).map(item => [item.question_id, item]));
  const allAnswered = lesson.questions.every(question => Number.isInteger(answers[question.id]));

  return <article className="jlpt-lesson-page">
    <header className="jlpt-lesson-hero">
      <Link to={`/learn/${level}`} className="level-back" aria-label={tr('Back to level')}>←</Link>
      <div><p>{lesson.level} / {tr(lesson.category)}</p><h1>{title}</h1><span>{summary}</span></div>
      <b>{lesson.level}</b>
    </header>

    <main className="jlpt-material-layout">
      <div className="jlpt-material-content">{lesson.content.map((section, index) => <section key={index} className="jlpt-material-block">
        <div className="material-index">{String(index + 1).padStart(2, '0')}</div>
        <h2>{language === 'id' ? section.heading_id : section.heading_en}</h2>
        <p>{language === 'id' ? section.body_id : section.body_en}</p>
        {section.japanese_text && <blockquote lang="ja">{section.japanese_text}</blockquote>}
        {section.examples?.map((example, exampleIndex) => <div className="jlpt-example" key={exampleIndex}>
          <strong lang="ja">{example.japanese}</strong><span>{language === 'id' ? example.indonesian : example.english}</span>
        </div>)}
      </section>)}</div>

      <section className="jlpt-quiz" id="jlpt-quiz">
        <div className="jlpt-quiz-heading"><div><p>{tr('CHECKPOINT')}</p><h2>{tr('Lesson quiz')}</h2></div>{result && <strong>{result.score}%</strong>}</div>
        {result && <div className={`jlpt-result-banner${result.score >= 67 ? ' is-pass' : ''}`}><strong>{tr('{correct} of {total} correct', result)}</strong><span>{tr('Best score: {score}%', { score: result.best_score })}</span></div>}
        <div className="jlpt-questions">{lesson.questions.map((question, questionIndex) => {
          const graded = resultByQuestion.get(question.id);
          return <fieldset key={question.id} className="jlpt-question">
            <legend><span>{String(questionIndex + 1).padStart(2, '0')}</span>{language === 'id' ? question.prompt_id : question.prompt_en}</legend>
            <div>{question.options.map((option, optionIndex) => {
              const selected = answers[question.id] === optionIndex;
              const correct = graded?.correct_index === optionIndex;
              const wrong = graded && selected && !correct;
              return <label key={optionIndex} className={`${selected ? 'is-selected ' : ''}${correct ? 'is-correct ' : ''}${wrong ? 'is-wrong' : ''}`.trim()}>
                <input type="radio" name={question.id} checked={selected} disabled={Boolean(result)} onChange={() => setAnswers(current => ({ ...current, [question.id]: optionIndex }))} />
                <span>{option}</span>
              </label>;
            })}</div>
            {graded && <p className="jlpt-explanation">{language === 'id' ? graded.explanation_id : graded.explanation_en}</p>}
          </fieldset>;
        })}</div>
        {error && <ErrorBanner err={error} lang={language} />}
        {!result ? <button className="jlpt-submit" disabled={!allAnswered || submitting} onClick={submitQuiz}>{submitting ? tr('Checking…') : tr('Finish quiz')}</button> :
          <button className="jlpt-submit" onClick={() => { setAnswers({}); setResult(null); }}>{tr('Try again')}</button>}
      </section>
    </main>
  </article>;
}

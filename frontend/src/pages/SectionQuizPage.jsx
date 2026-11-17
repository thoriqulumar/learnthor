import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import api from '../lib/api';
import { friendlyError } from '../lib/errors';
import { useLocale } from '../lib/locale';

const EXAM_SECONDS = 30 * 60;

const PROMPT_HINT = {
  kanji_to_kana: 'What is the reading?',
  kana_to_kanji: 'Which matches this reading?',
  fill_blank: 'Choose the missing word',
};

const SKILL_LABELS = {
  kanji_to_kana: 'Kanji → Kana',
  kana_to_kanji: 'Kana → Kanji',
  fill_blank: 'Fill the Blank',
};

function questionKey(question) {
  return `${question.vocab_item_id}:${question.type}`;
}

function formatTime(seconds) {
  const minutes = Math.floor(seconds / 60);
  return `${String(minutes).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}`;
}

function Spinner() {
  return <div className="page-loader" role="status" aria-label="Loading"><span /></div>;
}

function ResultScreen({ result, sectionTitle, onRestart, onBack }) {
  const { tr } = useLocale();
  const passed = result.score >= 80;
  return (
    <div className="section-exam result-view">
      <p className="exam-kicker">{tr('Section Test Complete')}</p>
      <h1>{sectionTitle}</h1>
      <div className={passed ? 'result-score passed' : 'result-score'}>{result.score}%</div>
      <p>{result.correct} / {result.total} {tr('correct')}</p>
      <p className="result-note">{passed ? tr('Great work!') : tr("Keep reviewing — you'll get there!")}</p>
      <div className="exam-actions">
        <button type="button" className="secondary-action" onClick={onBack}>{tr('Back')}</button>
        <button type="button" className="primary-action" onClick={onRestart}>{tr('Try again')}</button>
      </div>
    </div>
  );
}

export default function SectionQuizPage() {
  const { tr, language } = useLocale();
  const { sectionId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const sectionTitle = location.state?.sectionTitle ?? tr('Section Test');
  const [questions, setQuestions] = useState([]);
  const [answers, setAnswers] = useState({});
  const [index, setIndex] = useState(0);
  const [secondsLeft, setSecondsLeft] = useState(EXAM_SECONDS);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [result, setResult] = useState(null);
  const questionsRef = useRef([]);
  const answersRef = useRef({});
  const submittedRef = useRef(false);

  const submitExam = useCallback(async () => {
    if (submittedRef.current || questionsRef.current.length === 0) return;
    submittedRef.current = true;
    setSubmitting(true);
    setError(null);
    const payload = questionsRef.current.flatMap((question) => {
      const selected = answersRef.current[questionKey(question)];
      return selected ? [{
        vocab_item_id: question.vocab_item_id,
        skill_type: question.type,
        selected_option_id: selected,
      }] : [];
    });
    try {
      const { data } = await api.post(`/sections/${sectionId}/quiz/finish`, { answers: payload });
      setResult(data);
    } catch (err) {
      submittedRef.current = false;
      setError(err);
    } finally {
      setSubmitting(false);
    }
  }, [sectionId]);

  const loadExam = useCallback(() => {
    submittedRef.current = false;
    answersRef.current = {};
    questionsRef.current = [];
    setLoading(true);
    setSubmitting(false);
    setError(null);
    setResult(null);
    setQuestions([]);
    setAnswers({});
    setIndex(0);
    setSecondsLeft(EXAM_SECONDS);
    api.get(`/sections/${sectionId}/quiz`)
      .then(({ data }) => {
        const loaded = data.questions ?? [];
        questionsRef.current = loaded;
        setQuestions(loaded);
        setSecondsLeft(data.timer_seconds ?? EXAM_SECONDS);
      })
      .catch(setError)
      .finally(() => setLoading(false));
  }, [sectionId]);

  useEffect(() => {
    loadExam(); // eslint-disable-line react-hooks/set-state-in-effect -- load exam on route entry
  }, [loadExam]);

  useEffect(() => {
    if (loading || result || submitting || questions.length === 0) return undefined;
    const timer = window.setInterval(() => {
      setSecondsLeft((current) => {
        if (current <= 1) {
          window.clearInterval(timer);
          queueMicrotask(submitExam);
          return 0;
        }
        return current - 1;
      });
    }, 1000);
    return () => window.clearInterval(timer);
  }, [loading, questions.length, result, submitExam, submitting]);

  function selectAnswer(optionId) {
    const key = questionKey(questions[index]);
    const next = { ...answersRef.current, [key]: optionId };
    answersRef.current = next;
    setAnswers(next);
  }

  if (loading || submitting) return <Spinner />;
  if (error) return (
    <div className="section-exam error-view">
      <p>{friendlyError(error, language)}</p>
      <button type="button" className="secondary-action" onClick={() => navigate(-1)}>{tr('Back')}</button>
    </div>
  );
  if (result) return <ResultScreen result={result} sectionTitle={sectionTitle} onRestart={loadExam} onBack={() => navigate('/')} />;
  if (questions.length === 0) return <div className="section-exam error-view"><p>{tr('No questions are available for this section.')}</p></div>;

  const current = questions[index];
  const selected = answers[questionKey(current)];
  const answered = Object.keys(answers).length;
  return (
    <div className="section-exam">
      <header className="exam-header">
        <button type="button" onClick={() => navigate(-1)}>{tr('Back')}</button>
        <div><p>{tr('Section Test')}</p><h1>{sectionTitle}</h1></div>
        <time className={secondsLeft <= 300 ? 'urgent' : ''}>{formatTime(secondsLeft)}</time>
      </header>

      <div className="exam-overview">
        <span>{tr('Question {current} of {total}', { current: index + 1, total: questions.length })}</span>
        <span>{tr('{answered} answered', { answered })}</span>
      </div>
      <div className="exam-progress"><span style={{ width: `${((index + 1) / questions.length) * 100}%` }} /></div>

      <main className="exam-question">
        <p className="exam-kicker">{tr(SKILL_LABELS[current.type] ?? current.type)}</p>
        <p className="question-hint">{tr(PROMPT_HINT[current.type] ?? 'Choose the correct answer')}</p>
        <h2>{current.prompt}</h2>
        <div className="exam-options">
          {current.options.map((option, optionIndex) => (
            <button type="button" key={option.id}
              className={selected === option.id ? 'selected' : ''}
              aria-pressed={selected === option.id}
              onClick={() => selectAnswer(option.id)}>
              <span>{String.fromCharCode(65 + optionIndex)}</span>{option.text}
            </button>
          ))}
        </div>
      </main>

      <nav className="exam-navigation" aria-label={tr('Question navigation')}>
        <button type="button" className="secondary-action" disabled={index === 0} onClick={() => setIndex(index - 1)}>{tr('Previous')}</button>
        <div className="question-dots">
          {questions.map((question, questionIndex) => (
            <button type="button" key={questionKey(question)} aria-label={tr('Go to question {number}', { number: questionIndex + 1 })}
              className={`${questionIndex === index ? 'current' : ''} ${answers[questionKey(question)] ? 'answered' : ''}`}
              onClick={() => setIndex(questionIndex)}>{questionIndex + 1}</button>
          ))}
        </div>
        {index < questions.length - 1 ?
          <button type="button" className="primary-action" onClick={() => setIndex(index + 1)}>{tr('Next')}</button> :
          <button type="button" className="finish-action" onClick={submitExam}>{tr('Finish Test')}</button>}
      </nav>
      <button type="button" className="finish-exam" onClick={submitExam}>{tr('Finish Test')} · {answered}/{questions.length}</button>
      <p className="exam-policy">{tr('Answers are graded only after you finish or time runs out.')}</p>
    </div>
  );
}

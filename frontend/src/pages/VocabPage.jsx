import { useAuth } from '../context/AuthContext';
import { friendlyError } from '../lib/errors';
import Meaning from '../components/Meaning';
import { useLocale } from '../lib/locale';
import { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import api from '../lib/api';

const POS_COLORS = {
  noun: 'bg-sky-100 text-sky-700',
  verb: 'bg-emerald-100 text-emerald-700',
  'i-adj': 'bg-violet-100 text-violet-700',
  'na-adj': 'bg-pink-100 text-pink-700',
  adv: 'bg-amber-100 text-amber-700',
};

export default function VocabPage() {
  const { user } = useAuth();
  const { tr, language } = useLocale();
  const { moduleId } = useParams();
  const navigate = useNavigate();
  const [vocab, setVocab] = useState([]);
  const [moduleTitle, setModuleTitle] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [flip, setFlip] = useState({});

  useEffect(() => {
    api
      .get(`/modules/${moduleId}/vocab`)
      .then(({ data }) => {
        setVocab(data.items ?? []);
        setModuleTitle(data.module_title ?? '');
      })
      .catch(setError)
      .finally(() => setLoading(false));
  }, [moduleId]);

  const filtered = vocab.filter((v) => {
    const q = search.toLowerCase();
    return (
      v.kana?.includes(q) ||
      v.kanji?.includes(q) ||
      v.romaji?.toLowerCase().includes(q) ||
      v.english_meaning?.toLowerCase().includes(q) || v.indonesian_meaning?.toLowerCase().includes(q)
    );
  });

  if (loading) return <Spinner />;
  if (error) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-16 text-center">
        <p className="text-red-500 mb-4">{friendlyError(error, language)}</p>
        <button
          onClick={() => navigate(-1)}
          className="text-indigo-600 hover:underline text-sm"
        >{tr("← Go back")}</button>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-8">
      <div className="flex items-center gap-3 mb-6">
        <button
          onClick={() => navigate(-1)}
          className="text-gray-400 hover:text-gray-600 text-xl"
        >
          ←
        </button>
        <div>
          <h1 className="text-xl font-bold text-gray-900">
            {moduleTitle || tr("Vocabulary")}
          </h1>
          <p className="text-sm text-gray-500">{tr('{count} words', { count: vocab.length })}</p>
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        <input
          type="text"
          placeholder={tr("Search kanji, kana, English…")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <Link
          to={`/modules/${moduleId}/quiz`}
          className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg transition-colors"
        >{tr("Quiz →")}</Link>
      </div>

      <div className="space-y-2">
        {filtered.map((v) => (
          <button
            key={v.id}
            onClick={() => setFlip((f) => ({ ...f, [v.id]: !f[v.id] }))}
            className="w-full text-left bg-white border border-gray-200 rounded-xl p-4 hover:border-indigo-200 transition-colors"
          >
            {!flip[v.id] ? (
              <div className="flex items-baseline gap-3">
                <span className="text-2xl font-bold text-gray-900">
                  {v.kanji ?? v.kana}
                </span>
                {v.kanji && (
                  <span className="text-gray-400 text-sm">{v.kana}</span>
                )}
                <span className="text-gray-400 text-sm ml-auto">
                  {v.romaji}
                </span>
                {v.part_of_speech && (
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full font-medium ${POS_COLORS[v.part_of_speech] ?? 'bg-gray-100 text-gray-600'}`}
                  >
                    {v.part_of_speech}
                  </span>
                )}
              </div>
            ) : (
              <div className="space-y-1">
                <div className="font-semibold text-indigo-700"><Meaning item={v} gloss={user?.preferred_gloss} />
                </div>
                {v.example_sentence && (
                  <p className="text-sm text-gray-500 italic">
                    {v.example_sentence}
                  </p>
                )}
                <p className="text-xs text-gray-400">{tr("tap to flip back")}</p>
              </div>
            )}
          </button>
        ))}
        {filtered.length === 0 && (
          <p className="text-center text-gray-400 py-8">{tr('No words match “{query}”', { query: search })}
          </p>
        )}
      </div>
    </div>
  );
}

function Spinner() {
  return (
    <div className="min-h-[60vh] flex items-center justify-center">
      <div className="w-8 h-8 border-4 border-indigo-600 border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

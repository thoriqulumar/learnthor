import { vocabularyMeanings } from '../lib/locale';

export default function Meaning({ item, gloss }) {
  return <div className="meaning-list">{vocabularyMeanings(item, gloss).map(meaning => (
    <p key={meaning.language} lang={meaning.language}>
      <span className="meaning-language">{meaning.language.toUpperCase()}</span>{meaning.text}
    </p>
  ))}</div>;
}

import { useLocale } from '../lib/locale';

export default function LanguageSwitch() {
  const { language, setLanguage, tr } = useLocale();
  return (
    <div className="language-switch" role="group" aria-label={tr('Interface language')}>
      {['en', 'id'].map(value => (
        <button key={value} type="button" aria-pressed={language === value}
          onClick={() => setLanguage(value)} lang={value}>
          {value === 'en' ? 'EN' : 'ID'}
        </button>
      ))}
    </div>
  );
}

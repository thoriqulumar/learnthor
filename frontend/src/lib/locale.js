import { useCallback, useSyncExternalStore } from 'react';
import id from './translations-id.json';

const key = 'learnthor-ui-language';
function currentLanguage() {
  try { return localStorage.getItem(key) === 'id' ? 'id' : 'en'; }
  catch { return 'en'; }
}
function subscribe(callback) {
  window.addEventListener('storage', callback);
  window.addEventListener('learnthor-language', callback);
  return () => {
    window.removeEventListener('storage', callback);
    window.removeEventListener('learnthor-language', callback);
  };
}

export function translate(message, language, values = {}) {
  const template = language === 'id' ? (id[message] ?? message) : message;
  return String(template ?? '').replace(/\{(\w+)\}/g, (match, name) => values[name] ?? match);
}

export function useLocale() {
  const language = useSyncExternalStore(subscribe, currentLanguage, () => 'en');
  const tr = useCallback((message, values) => translate(message, language, values), [language]);
  return { language, tr, setLanguage };
}

export function setLanguage(language) {
  if (!['en', 'id'].includes(language)) return;
  try { localStorage.setItem(key, language); } catch { /* Browser storage may be disabled. */ }
  document.documentElement.lang = language;
  window.dispatchEvent(new Event('learnthor-language'));
}

export function vocabularyMeanings(item, gloss = 'both') {
  const en = item.english_meaning?.trim();
  const id = item.indonesian_meaning?.trim();
  if (gloss === 'en') return en ? [{ language: 'en', text: en }] : id ? [{ language: 'id', text: id }] : [];
  if (gloss === 'id') return id ? [{ language: 'id', text: id }] : en ? [{ language: 'en', text: en }] : [];
  return [...(en ? [{ language: 'en', text: en }] : []), ...(id ? [{ language: 'id', text: id }] : [])];
}

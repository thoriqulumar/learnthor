import { describe, it, expect } from 'vitest';
import { translate, vocabularyMeanings } from './locale';

describe('localization', () => {
  it('translates Indonesian UI and interpolates complete count messages', () => {
    expect(translate('Log in', 'id')).toBe('Masuk');
    expect(translate('{count} words', 'id', { count: 12 })).toBe('12 kata');
    expect(translate('Log in', 'en')).toBe('Log in');
    expect(translate('日本語', 'id')).toBe('日本語');
  });
  it('falls back to available vocabulary translations without hiding the meaning', () => {
    expect(vocabularyMeanings({ english_meaning: 'bridge' }, 'id')).toEqual([{ language: 'en', text: 'bridge' }]);
    expect(vocabularyMeanings({ indonesian_meaning: 'jembatan' }, 'en')).toEqual([{ language: 'id', text: 'jembatan' }]);
    expect(vocabularyMeanings({ english_meaning: 'bridge', indonesian_meaning: 'jembatan' }, 'both')).toHaveLength(2);
  });
});

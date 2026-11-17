import { beforeEach, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import LanguageSwitch from './LanguageSwitch';
import { useLocale } from '../lib/locale';

function Example() {
  const { tr } = useLocale();
  return <><LanguageSwitch /><p>{tr('Log in')}</p></>;
}
beforeEach(() => localStorage.clear());
it('changes UI immediately and persists language independently from vocabulary', () => {
  localStorage.setItem('user', JSON.stringify({ preferred_gloss: 'both' }));
  render(<Example />);
  fireEvent.click(screen.getByRole('button', { name: 'ID' }));
  expect(screen.getByText('Masuk')).toBeInTheDocument();
  expect(document.documentElement.lang).toBe('id');
  expect(localStorage.getItem('learnthor-ui-language')).toBe('id');
  expect(JSON.parse(localStorage.getItem('user')).preferred_gloss).toBe('both');
});

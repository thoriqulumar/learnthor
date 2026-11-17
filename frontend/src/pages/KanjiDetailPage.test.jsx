import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { expect, it, vi } from 'vitest';
import api from '../lib/api';
import KanjiDetailPage from './KanjiDetailPage';

vi.mock('../lib/api', () => ({ default: { get: vi.fn() } }));

it('opens an example word with its reading and bilingual meanings and closes it', async () => {
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute('open', ''); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute('open'); };
  api.get.mockResolvedValue({ data: { items: [{ id: 'word', kana: 'まいにち', romaji: 'mainichi', english_meaning: 'every day', indonesian_meaning: 'setiap hari' }] } });
  const item = { character: '毎', jlpt_level: 'N5', on_readings: [], kun_readings: [], example_words: ['毎日'] };
  render(<MemoryRouter initialEntries={[{ pathname: '/kanji/毎', state: { item } }]}><Routes><Route path="/kanji/:character" element={<KanjiDetailPage />} /></Routes></MemoryRouter>);
  await userEvent.click(screen.getByRole('button', { name: '毎日' }));
  expect(await screen.findByText('まいにち')).toBeInTheDocument();
  expect(screen.getByText('every day')).toBeInTheDocument();
  expect(screen.getByText('setiap hari')).toBeInTheDocument();
  await userEvent.click(screen.getByRole('button', { name: 'Close' }));
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
});

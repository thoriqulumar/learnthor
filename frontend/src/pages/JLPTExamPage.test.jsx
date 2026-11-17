import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { expect, it, vi } from 'vitest';
import api from '../lib/api';
import JLPTExamPage from './JLPTExamPage';

vi.mock('../lib/api', () => ({ default: { get: vi.fn(), post: vi.fn() } }));

it('keeps reading last and submits unanswered questions without showing early results', async () => {
  api.get.mockResolvedValue({ data: { token: 'signed-paper', questions: [
    { id: 'v1', category: 'vocabulary', prompt_en: 'Choose a reading', options: ['a', 'b'] },
    { id: 'r1', category: 'reading', prompt_en: 'Read the passage', passage: '日本語', options: ['c', 'd'] },
  ] } });
  api.post.mockResolvedValue({ data: { score: 50, correct: 1, total: 2, results: [
    { question_id: 'v1', correct: true, correct_index: 0, explanation_en: 'Correct reading' },
    { question_id: 'r1', correct: false, correct_index: 1, explanation_en: 'Reading explanation' },
  ] } });
  render(<MemoryRouter initialEntries={['/learn/n5/exam']}><Routes><Route path="/learn/:level/exam" element={<JLPTExamPage />} /></Routes></MemoryRouter>);
  await screen.findByText('Choose a reading');
  expect(screen.getAllByRole('heading', { level: 2 }).map(node => node.textContent)).toEqual(['01 / Vocabulary', '05 / Reading / Dokkai']);
  expect(screen.queryByText('50%')).not.toBeInTheDocument();
  await userEvent.click(screen.getAllByRole('radio')[0]);
  await userEvent.click(screen.getByRole('button', { name: 'Finish quiz' }));
  expect(api.post).toHaveBeenCalledWith('/jlpt/exam/finish', { token: 'signed-paper', answers: [
    { question_id: 'v1', selected_index: 0 }, { question_id: 'r1', selected_index: -1 },
  ] });
  expect(await screen.findByText('50%')).toBeInTheDocument();
});

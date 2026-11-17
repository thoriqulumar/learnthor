import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

import api from '../lib/api';
import SectionQuizPage from './SectionQuizPage';

vi.mock('../lib/api', () => ({ default: { get: vi.fn(), post: vi.fn() } }));

const questions = [
  { vocab_item_id: 'word-1', type: 'kanji_to_kana', prompt: 'あ', options: [{ id: 'word-1', text: 'a' }, { id: 'word-2', text: 'i' }] },
  { vocab_item_id: 'word-2', type: 'kana_to_kanji', prompt: 'i', options: [{ id: 'word-1', text: 'あ' }, { id: 'word-2', text: 'い' }] },
];

function renderPage() {
  return render(
    <MemoryRouter initialEntries={[{ pathname: '/sections/section-1/quiz', state: { sectionTitle: 'Hiragana' } }]}>
      <Routes><Route path="/sections/:sectionId/quiz" element={<SectionQuizPage />} /></Routes>
    </MemoryRouter>,
  );
}

describe('SectionQuizPage', () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
    api.get.mockResolvedValue({ data: { questions, timer_seconds: 1800 } });
    api.post.mockResolvedValue({ data: { score: 50, correct: 1, total: 2 } });
  });

  it('keeps answers hidden and submits selections only when finished', async () => {
    renderPage();
    await screen.findByText('あ');
    expect(screen.getByText('30:00')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Aa' }));
    expect(screen.queryByText(/Correct/)).not.toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    fireEvent.click(screen.getAllByRole('button', { name: /Finish Test/ })[0]);
    await waitFor(() => expect(api.post).toHaveBeenCalledWith('/sections/section-1/quiz/finish', {
      answers: [{ vocab_item_id: 'word-1', skill_type: 'kanji_to_kana', selected_option_id: 'word-1' }],
    }));
    expect(await screen.findByText('50%')).toBeInTheDocument();
  });

  it('submits automatically when the timer expires', async () => {
    vi.useFakeTimers();
    renderPage();
    await vi.waitFor(() => expect(screen.getByText('あ')).toBeInTheDocument());
    await vi.advanceTimersByTimeAsync(30 * 60 * 1000);
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith('/sections/section-1/quiz/finish', { answers: [] }));
  });
});

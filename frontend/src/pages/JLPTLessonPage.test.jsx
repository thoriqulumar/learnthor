import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import api from '../lib/api';
import JLPTLessonPage from './JLPTLessonPage';

vi.mock('../lib/api', () => ({ default: { get: vi.fn(), post: vi.fn() } }));

const LESSON = {
  id: 'l1', level: 'N5', category: 'particles', title_en: 'Core particles', title_id: 'Partikel dasar', summary_en: 'Summary', summary_id: 'Ringkasan',
  content: [{ heading_en: 'Topic', heading_id: 'Topik', body_en: 'Explanation', body_id: 'Penjelasan', examples: [] }],
  questions: [{ id: 'q1', prompt_en: 'Choose one', prompt_id: 'Pilih satu', options: ['は', 'を'] }],
};

describe('JLPTLessonPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.get.mockResolvedValue({ data: LESSON });
    api.post.mockResolvedValue({ data: { score: 100, correct: 1, total: 1, best_score: 100, results: [{ question_id: 'q1', selected_index: 1, correct_index: 1, correct: true, explanation_en: 'Correct', explanation_id: 'Benar' }] } });
  });

  it('submits answers and shows the server-graded result', async () => {
    const user = userEvent.setup();
    render(<MemoryRouter initialEntries={['/learn/n5/lessons/l1']}><Routes><Route path="/learn/:level/lessons/:lessonId" element={<JLPTLessonPage />} /></Routes></MemoryRouter>);
    await waitFor(() => expect(screen.getByText('Core particles')).toBeInTheDocument());
    await user.click(screen.getByLabelText('を'));
    await user.click(screen.getByRole('button', { name: 'Finish quiz' }));
    await waitFor(() => expect(screen.getByText('100%')).toBeInTheDocument());
    expect(api.post).toHaveBeenCalledWith('/jlpt/lessons/l1/finish', { answers: [{ question_id: 'q1', selected_index: 1 }] });
  });
});

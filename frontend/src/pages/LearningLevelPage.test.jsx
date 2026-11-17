import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import api from '../lib/api';
import { AuthProvider } from '../context/AuthContext';
import LearningLevelPage from './LearningLevelPage';

vi.mock('../lib/api', () => ({ default: { get: vi.fn() } }));

describe('LearningLevelPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const payload = btoa(JSON.stringify({ exp: 9999999999 }));
    localStorage.setItem('token', `x.${payload}.y`);
    localStorage.setItem('user', JSON.stringify({ role: 'student', subscription_tier: 'premium' }));
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: { sections: [{ id: 's5', title: 'JLPT N5 Vocabulary', modules_total: 1, modules_passed: 0, mastery_pct: 0, modules: [] }] } });
      return Promise.resolve({ data: { lessons: [{ id: 'l1', category: 'particles', title_en: 'Core particles', title_id: 'Partikel dasar', summary_en: 'Summary', summary_id: 'Ringkasan', locked: false, attempt_count: 0 }] } });
    });
  });

  it('shows JLPT skill categories and the level vocabulary route', async () => {
    render(<MemoryRouter initialEntries={['/learn/n5']}><AuthProvider><Routes><Route path="/learn/:level" element={<LearningLevelPage />} /></Routes></AuthProvider></MemoryRouter>);
    await waitFor(() => expect(screen.getByText('JLPT N5 Vocabulary')).toBeInTheDocument());
    expect(screen.getByText('Reading / Dokkai')).toBeInTheDocument();
    expect(screen.getByText('JLPT N5 Vocabulary')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /Particles/ }));
    expect(screen.getByText('Core particles')).toBeInTheDocument();
    expect(screen.queryByText('JLPT N5 Vocabulary')).not.toBeInTheDocument();
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

import api from '../lib/api';
import { AuthProvider } from '../context/AuthContext';
import HomePage from './HomePage';

vi.mock('../lib/api', () => ({
  default: { get: vi.fn() },
}));

const SECTIONS = {
  sections: [
    {
      id: 's1',
      title: 'Hiragana',
      sort_order: 1,
      locked: false,
      mastery_pct: 100,
      modules_total: 1,
      modules_passed: 1,
      modules: [
        {
          id: 'm1',
          title: 'Vowels',
          sort_order: 1,
          vocab_count: 5,
          locked: false,
          high_score: 95,
          passed: true,
          pass_threshold: 90,
          skill_scores: {},
        },
      ],
    },
    {
      id: 's2',
      title: 'JLPT N4 Vocabulary',
      sort_order: 8,
      locked: true,
      is_premium: true,
      mastery_pct: 0,
      modules_total: 1,
      modules_passed: 0,
      modules: [
        {
          id: 'm2',
          title: 'N4 Page 1',
          sort_order: 100,
          vocab_count: 20,
          locked: true,
          high_score: 0,
          passed: false,
          pass_threshold: 90,
          skill_scores: {},
        },
      ],
    },
  ],
};

function seedUser(role = 'student', tier = 'free') {
  const payload = btoa(JSON.stringify({ exp: 9999999999 }));
  localStorage.setItem('token', `x.${payload}.y`);
  localStorage.setItem(
    'user',
    JSON.stringify({ username: 'u', role, subscription_tier: tier, preferred_gloss: 'both' }),
  );
}

function renderHome() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <HomePage />
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe('HomePage', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('groups the learning route into level cards', async () => {
    seedUser();
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: SECTIONS });
      if (url === '/reviews/stats')
        return Promise.resolve({ data: { due: 0, new: 0, learning: 0, review: 0, relearning: 0 } });
      return Promise.reject(new Error('unexpected ' + url));
    });

    renderHome();
    await waitFor(() => expect(screen.getByText('Beginner')).toBeInTheDocument());
    expect(screen.getByText('JLPT N5')).toBeInTheDocument();
    expect(screen.getByText('JLPT N4')).toBeInTheDocument();
    expect(screen.getByText('JLPT N3')).toBeInTheDocument();
  });

  it('shows lock overlay on locked sections for students', async () => {
    seedUser('student');
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: SECTIONS });
      return Promise.resolve({ data: { due: 0 } });
    });

    renderHome();
    await waitFor(() => screen.getByText('Beginner'));
    expect(screen.getByRole('link', { name: /JLPT N4/i })).toHaveClass('is-locked');
    expect(
      screen.getAllByText(/Upgrade to Premium/i).length,
    ).toBeGreaterThan(0);
  });

  it('premium user sees premium sections unlocked', async () => {
    seedUser('student', 'premium');
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: {
        sections: SECTIONS.sections.map(section => section.id === 's2' ? { ...section, locked: false } : section),
      } });
      return Promise.resolve({ data: { due: 0 } });
    });

    renderHome();
    await waitFor(() => screen.getByText('Beginner'));
    expect(screen.getByRole('link', { name: /JLPT N4/i })).not.toHaveClass('is-locked');
    expect(screen.queryByText('👑 Upgrade to Premium')).not.toBeInTheDocument();
  });

  it('hides the daily review widget when nothing is due', async () => {
    seedUser();
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: SECTIONS });
      return Promise.resolve({ data: { due: 0 } });
    });

    renderHome();
    await waitFor(() => screen.getByText('Beginner'));
    expect(screen.queryByText('Daily Review')).not.toBeInTheDocument();
  });

  it('shows the daily review widget with the due count', async () => {
    seedUser();
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: SECTIONS });
      return Promise.resolve({
        data: { due: 7, new: 2, learning: 3, review: 2, relearning: 0 },
      });
    });

    renderHome();
    await waitFor(() => screen.getByText('Beginner'));
    await waitFor(() =>
      expect(screen.getByText('Daily Review')).toBeInTheDocument(),
    );
    expect(screen.getByText(/7 cards due/)).toBeInTheDocument();
  });

  it('admin bypasses section locks', async () => {
    seedUser('admin');
    api.get.mockImplementation((url) => {
      if (url === '/sections') return Promise.resolve({ data: SECTIONS });
      return Promise.resolve({ data: { due: 0 } });
    });

    renderHome();
    await waitFor(() => screen.getByText('Beginner'));
    expect(screen.queryByText('Locked')).not.toBeInTheDocument();
  });
});

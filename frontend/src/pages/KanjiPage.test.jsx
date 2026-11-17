import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import api from '../lib/api';
import KanjiPage from './KanjiPage';

vi.mock('../lib/api', () => ({ default: { get: vi.fn() } }));

const ITEMS = [
  { character: '日', jlpt_level: 'N5', stroke_count: 4, on_readings: ['ニチ'], kun_readings: ['ひ'], english_meaning: 'day, sun', indonesian_meaning: 'hari, matahari' },
  { character: '猫', jlpt_level: 'N3', stroke_count: 11, on_readings: ['ビョウ'], kun_readings: ['ねこ'], english_meaning: 'cat', indonesian_meaning: 'kucing' },
];

describe('KanjiPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.get.mockResolvedValue({ data: { count: 2, items: ITEMS } });
  });

  it('filters by level and bilingual search text', async () => {
    const user = userEvent.setup();
    render(<MemoryRouter><KanjiPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText('日')).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: 'N3' }));
    expect(screen.queryByText('日')).not.toBeInTheDocument();
    expect(screen.getByText('猫')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'All' }));
    await user.type(screen.getByPlaceholderText('Kanji, reading, or meaning'), 'matahari');
    expect(screen.getByText('日')).toBeInTheDocument();
    expect(screen.queryByText('猫')).not.toBeInTheDocument();
  });
});

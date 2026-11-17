import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import MenuPage from './MenuPage';

describe('MenuPage', () => {
  it('links to the learning route and kanji library', () => {
    render(<MemoryRouter><MenuPage /></MemoryRouter>);
    expect(screen.getByRole('link', { name: /Learning route/i })).toHaveAttribute('href', '/learn');
    expect(screen.getByRole('link', { name: /Kanji/i })).toHaveAttribute('href', '/kanji');
  });
});

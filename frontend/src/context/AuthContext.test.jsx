import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import api from '../lib/api';
import { AuthProvider, useAuth } from './AuthContext';

vi.mock('../lib/api', () => ({
  default: { post: vi.fn() },
}));

// A tiny consumer that exercises the context API.
function Probe() {
  const { user, login, logout, refreshUser } = useAuth();
  return (
    <div>
      <p data-testid="username">{user ? user.username : 'anon'}</p>
      <p data-testid="gloss">{user?.preferred_gloss ?? 'none'}</p>
      <button onClick={() => login('a@b.c', 'pw')}>login</button>
      <button onClick={logout}>logout</button>
      <button onClick={() => refreshUser({ preferred_gloss: 'id' })}>
        setGloss
      </button>
    </div>
  );
}

function renderProbe() {
  return render(
    <AuthProvider>
      <Probe />
    </AuthProvider>,
  );
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('starts anonymous with empty storage', async () => {
    renderProbe();
    await waitFor(() =>
      expect(screen.getByTestId('username')).toHaveTextContent('anon'),
    );
  });

  it('restores a stored session on mount', async () => {
    // A JWT whose payload decodes to a far-future exp.
    const payload = btoa(JSON.stringify({ exp: 9999999999 }));
    localStorage.setItem('token', `x.${payload}.y`);
    localStorage.setItem(
      'user',
      JSON.stringify({ username: 'stored', preferred_gloss: 'both' }),
    );

    renderProbe();
    await waitFor(() =>
      expect(screen.getByTestId('username')).toHaveTextContent('stored'),
    );
  });

  it('login stores token + user', async () => {
    api.post.mockResolvedValue({
      data: {
        token: 'tok',
        user: { username: 'logged', preferred_gloss: 'both' },
      },
    });

    renderProbe();
    await waitFor(() => screen.getByTestId('username'));
    await userEvent.click(screen.getByText('login'));

    await waitFor(() =>
      expect(screen.getByTestId('username')).toHaveTextContent('logged'),
    );
    expect(localStorage.getItem('token')).toBe('tok');
    expect(JSON.parse(localStorage.getItem('user')).username).toBe('logged');
  });

  it('logout clears state and storage', async () => {
    api.post.mockResolvedValue({
      data: { token: 'tok', user: { username: 'logged' } },
    });

    renderProbe();
    await userEvent.click(screen.getByText('login'));
    await waitFor(() =>
      expect(screen.getByTestId('username')).toHaveTextContent('logged'),
    );

    await userEvent.click(screen.getByText('logout'));
    expect(screen.getByTestId('username')).toHaveTextContent('anon');
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
  });

  it('refreshUser patches state and persists to storage', async () => {
    api.post.mockResolvedValue({
      data: {
        token: 'tok',
        user: { username: 'logged', preferred_gloss: 'both' },
      },
    });

    renderProbe();
    await userEvent.click(screen.getByText('login'));
    await waitFor(() =>
      expect(screen.getByTestId('gloss')).toHaveTextContent('both'),
    );

    await userEvent.click(screen.getByText('setGloss'));
    expect(screen.getByTestId('gloss')).toHaveTextContent('id');
    expect(JSON.parse(localStorage.getItem('user')).preferred_gloss).toBe('id');
  });
});

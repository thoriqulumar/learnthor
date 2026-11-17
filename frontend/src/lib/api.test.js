import { describe, it, expect, vi, beforeEach } from 'vitest';

// Capture the interceptors registered on the created axios instance so we can
// invoke them directly — no HTTP needed.
const requestHandlers = [];
const responseHandlers = [];

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      interceptors: {
        request: { use: (fn) => requestHandlers.push(fn) },
        response: { use: (ok, err) => responseHandlers.push({ ok, err }) },
      },
    })),
  },
}));

// Importing api.js registers its interceptors on our mock instance.
await import('./api');

describe('api request interceptor', () => {
  beforeEach(() => localStorage.clear());

  it('attaches the JWT from localStorage', () => {
    localStorage.setItem('token', 'abc123');
    const config = { headers: {} };
    const result = requestHandlers[0](config);
    expect(result.headers.Authorization).toBe('Bearer abc123');
  });

  it('leaves headers untouched without a token', () => {
    const config = { headers: {} };
    const result = requestHandlers[0](config);
    expect(result.headers.Authorization).toBeUndefined();
  });
});

describe('api response interceptor', () => {
  beforeEach(() => {
    localStorage.clear();
    // jsdom stubs navigation; replace location with a plain object.
    delete window.location;
    window.location = { href: '' };
  });

  it('passes successful responses through', () => {
    const res = { status: 200 };
    expect(responseHandlers[0].ok(res)).toBe(res);
  });

  it('clears the token and redirects to /login on 401', async () => {
    localStorage.setItem('token', 'abc123');
    const err = { response: { status: 401 } };
    await expect(responseHandlers[0].err(err)).rejects.toBe(err);
    expect(localStorage.getItem('token')).toBeNull();
    expect(window.location.href).toBe('/login');
  });

  it('does not clear the token on other errors', async () => {
    localStorage.setItem('token', 'abc123');
    const err = { response: { status: 500 } };
    await expect(responseHandlers[0].err(err)).rejects.toBe(err);
    expect(localStorage.getItem('token')).toBe('abc123');
    expect(window.location.href).toBe('');
  });
});

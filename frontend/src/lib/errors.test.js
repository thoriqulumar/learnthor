import { describe, it, expect } from 'vitest';
import { friendlyError, friendlyErrorParts } from './errors';

function apiError(status, code) {
  return { response: { status, data: code ? { code } : {} } };
}

describe('friendlyError', () => {
  it('maps known backend codes to friendly copy', () => {
    const err = apiError(400, 'INVALID_CREDENTIALS');
    expect(friendlyError(err)).toBe('Email or password is incorrect.');
    expect(friendlyError(err, 'id')).toBe('Email atau kata sandi salah.');
  });

  it('maps UPGRADE_REQUIRED to an upgrade prompt', () => {
    const err = apiError(403, 'UPGRADE_REQUIRED');
    expect(friendlyError(err)).toMatch(/Premium/);
    expect(friendlyError(err, 'id')).toMatch(/Premium/);
  });

  it('falls back by HTTP status for unknown codes', () => {
    expect(friendlyError(apiError(429))).toMatch(/Too many attempts/);
    expect(friendlyError(apiError(404))).toMatch(/couldn't find/);
    expect(friendlyError(apiError(500))).toMatch(/on our side/);
    expect(friendlyError(apiError(503))).toMatch(/on our side/);
  });

  it('handles network failures with no response object', () => {
    expect(friendlyError(new Error('Network Error'))).toMatch(/check your connection/);
    expect(friendlyError(undefined)).toMatch(/check your connection/);
    expect(friendlyError(null, 'id')).toMatch(/koneksi/);
  });

  it('unknown errors get a generic message', () => {
    expect(friendlyError(apiError(418))).toBe('Something went wrong. Please try again.');
  });

  it('friendlyErrorParts returns both languages', () => {
    const parts = friendlyErrorParts(apiError(403, 'UPGRADE_REQUIRED'));
    expect(parts.en).toBeTruthy();
    expect(parts.id).toBeTruthy();
    expect(parts.en).not.toBe(parts.id);
  });
});
